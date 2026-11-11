// Copyright 2024 Universidad Carlos III de Madrid
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/dnsinterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	APIReader client.Reader
	Log       logr.Logger

	Scheme            *runtime.Scheme
	SwitchesNamespace string
	InternalClient    sdnclient.Client
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	pod := &corev1.Pod{}
	if err := store.reader().Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] == "" || pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY] == "" {
		return ctrl.Result{}, nil
	}
	// Finished pods need no operation lease; avoiding metadata writes here also
	// prevents our own lock annotations from causing a reconcile loop.
	if pod.DeletionTimestamp == nil {
		if raw := pod.Annotations[pendingMoveAnnotation]; raw != "" {
			var move pendingMove
			if err := json.Unmarshal([]byte(raw), &move); err != nil {
				return ctrl.Result{}, err
			}
			source, target := &l2smv1.L2Network{}, &l2smv1.L2Network{}
			if err := store.reader().Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: move.Source}, source); err != nil {
				return ctrl.Result{}, err
			}
			if err := store.reader().Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: move.Target}, target); err != nil {
				return ctrl.Result{}, err
			}
			if string(source.UID) != move.SourceUID || string(target.UID) != move.TargetUID {
				return ctrl.Result{}, fmt.Errorf("pending move refers to a replaced network")
			}
			mover := &QuarantinePodRequestReconciler{Client: r.Client, APIReader: r.APIReader, InternalClient: r.InternalClient}
			_, err := mover.movePodToTargetNetwork(ctx, pod, source, target)
			return ctrl.Result{}, err
		}
		if pod.Annotations[attachedNetworksAnnotation] == pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] && slices.Contains(pod.Finalizers, l2smFinalizer) {
			return ctrl.Result{}, nil
		}
	}
	operationCtx, release, err := acquirePodOperation(ctx, store, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	defer release()
	ctx = operationCtx
	if err := store.reader().Get(ctx, req.NamespacedName, pod); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	networks, err := networkannotation.ExtractNetworks(pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION], pod.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	attachments, err := networkannotation.ExtractNetworks(pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY], pod.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(networks) != len(attachments) {
		return ctrl.Result{}, fmt.Errorf("pod has mismatched network/attachment counts")
	}
	// Upgrade existing pods without freeing legacy reservations blindly. Verify
	// exclusivity before giving the attachment a durable incarnation owner.
	if pod.Annotations[allocationIDAnnotation] == "" {
		id := podAllocationID(pod)
		if id == "" {
			return ctrl.Result{}, fmt.Errorf("pod has no allocation identity")
		}
		for i, network := range networks {
			owner := podPortOwner(pod.Namespace, pod.Name, id, network.Name)
			key := client.ObjectKey{Namespace: attachments[i].Namespace, Name: attachments[i].Name}
			if err := store.adoptPort(ctx, key, pod.Spec.NodeName, owner, func(other *corev1.Pod) bool { return other.UID == pod.UID }); err != nil {
				return ctrl.Result{}, err
			}
			if _, err := store.reserveIPs(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: network.Name}, pod.Name, id, attachments[i].IPAddresses, true); err != nil {
				if pod.DeletionTimestamp == nil || !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		}
		if err := updatePod(ctx, store, req.NamespacedName, func(current *corev1.Pod) error { current.Annotations[allocationIDAnnotation] = id; return nil }); err != nil {
			return ctrl.Result{}, err
		}
		pod.Annotations[allocationIDAnnotation] = id
	}
	if pod.DeletionTimestamp != nil {
		if !slices.Contains(pod.Finalizers, l2smFinalizer) {
			return ctrl.Result{}, nil
		}
		// Keep ownership until the old containers have stopped using the interface.
		for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
			if status.State.Running != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
		}
		for i, network := range networks {
			port, err := utils.GetPortNumberFromNetAttachDef(attachments[i].Name)
			if err != nil {
				return ctrl.Result{}, err
			}
			ofPort := fmt.Sprintf("of:%s/%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: pod.Spec.NodeName, ProviderName: l2smv1.OVERLAY_PROVIDER})), port)
			exists, err := r.InternalClient.CheckNetworkExists(l2smv1.NetworkTypeVnet, network.Name)
			if err != nil {
				return ctrl.Result{}, err
			}
			if exists {
				if err := r.InternalClient.DetachPodFromNetwork("vnets", sdnclient.VnetPayload{NetworkId: network.Name, Port: []string{ofPort}}); err != nil {
					return ctrl.Result{}, err
				}
			}
			if err := store.releaseIPs(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: network.Name}, pod.Name, pod.Annotations[allocationIDAnnotation]); err != nil {
				return ctrl.Result{}, err
			}
		}
		if raw := pod.Annotations[pendingMoveAnnotation]; raw != "" {
			var move pendingMove
			if err := json.Unmarshal([]byte(raw), &move); err != nil {
				return ctrl.Result{}, err
			}
			index := networkAnnotationIndex(networks, move.Source)
			if index < 0 {
				index = networkAnnotationIndex(networks, move.Target)
			}
			if index < 0 {
				return ctrl.Result{}, fmt.Errorf("cannot clean pending move: interface not found")
			}
			port, err := utils.GetPortNumberFromNetAttachDef(attachments[index].Name)
			if err != nil {
				return ctrl.Result{}, err
			}
			ofPort := fmt.Sprintf("of:%s/%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: pod.Spec.NodeName, ProviderName: l2smv1.OVERLAY_PROVIDER})), port)
			for _, name := range []string{move.Source, move.Target} {
				exists, err := r.InternalClient.CheckNetworkExists(l2smv1.NetworkTypeVnet, name)
				if err != nil {
					return ctrl.Result{}, err
				}
				if exists {
					if err := r.InternalClient.DetachPodFromNetwork("vnets", sdnclient.VnetPayload{NetworkId: name, Port: []string{ofPort}}); err != nil {
						return ctrl.Result{}, err
					}
				}
				if err := store.releaseIPs(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: name}, pod.Name, podAllocationID(pod)); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
		if err := store.releaseOwnedPorts(ctx, r.SwitchesNamespace, podPortOwner(pod.Namespace, pod.Name, pod.Annotations[allocationIDAnnotation], "")); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, updatePod(ctx, store, req.NamespacedName, func(current *corev1.Pod) error {
			current.Finalizers = utils.RemoveString(current.Finalizers, l2smFinalizer)
			return nil
		})
	}
	if !slices.Contains(pod.Finalizers, l2smFinalizer) {
		if err := updatePod(ctx, store, req.NamespacedName, func(current *corev1.Pod) error {
			if !slices.Contains(current.Finalizers, l2smFinalizer) {
				current.Finalizers = append(current.Finalizers, l2smFinalizer)
			}
			return nil
		}); err != nil {
			return ctrl.Result{}, err
		}
	}
	if pod.Annotations[pendingMoveAnnotation] != "" {
		return ctrl.Result{}, nil
	} // Quarantine owns this transition.
	desired := pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION]
	if pod.Annotations[attachedNetworksAnnotation] == desired {
		return ctrl.Result{}, nil
	}
	for i, network := range networks {
		port, err := utils.GetPortNumberFromNetAttachDef(attachments[i].Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		ofPort := fmt.Sprintf("of:%s/%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: pod.Spec.NodeName, ProviderName: l2smv1.OVERLAY_PROVIDER})), port)
		if err := r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPayload{NetworkId: network.Name, Port: []string{ofPort}}); err != nil {
			return ctrl.Result{}, err
		}
		n := &l2smv1.L2Network{}
		if err := store.reader().Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: network.Name}, n); err != nil {
			return ctrl.Result{}, err
		}
		if n.Spec.Provider != nil && len(attachments[i].IPAddresses) > 0 {
			name := pod.Name
			if pod.Labels[L2SM_PODNAME_LABEL] != "" {
				name = pod.Labels[L2SM_PODNAME_LABEL]
			}
			if err := CreateDNSEntry(n, name, attachments[i].IPAddresses[0]); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, updatePod(ctx, store, req.NamespacedName, func(current *corev1.Pod) error {
		if current.Annotations[pendingMoveAnnotation] != "" || current.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] != desired {
			return conflict("pods", current.Name)
		}
		current.Annotations[attachedNetworksAnnotation] = desired
		return nil
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.APIReader = mgr.GetAPIReader()
	var err error
	// Initialize the InternalClient with the base URL of the SDN controller
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:%s/onos", env.GetControllerIP(), env.GetControllerPort()), Username: "karaf", Password: "karaf"}

	r.InternalClient, err = sdnclient.NewClient(sdnclient.InternalType, clientConfig)
	if err != nil {
		r.Log.Error(err, "failed to initiate session with sdn controller")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

func CreateDNSEntry(network *l2smv1.L2Network, podName, podCIDR string) error {
	// We create a DNS Client for registring this pod in an external DNS
	providerAddress := fmt.Sprintf("%s:%s", network.Spec.Provider.Domain, utils.DefaultIfEmpty(network.Spec.Provider.DNSGRPCPort, "30818"))

	dnsClient := dnsinterface.DNSClient{ServerAddress: providerAddress, Scope: "inter"}

	ip, _, err := net.ParseCIDR(podCIDR)
	if err != nil {
		return fmt.Errorf("could not parse pod cidr: %v", err)

	}
	err = dnsClient.AddDNSEntry(podName, network.Name, ip.To4().String())

	if err != nil {
		return fmt.Errorf("could not add dns entry in remote server: %v", err)
	}

	return nil
}
