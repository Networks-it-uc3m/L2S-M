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
	"fmt"
	"net"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// QuarantinePodRequestReconciler reconciles a QuarantinePodRequest object
type QuarantinePodRequestReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	InternalClient sdnclient.Client
}

// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=quarantinepodrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=quarantinepodrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=quarantinepodrequests/finalizers,verbs=update
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=l2networks,verbs=get;list;watch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=l2networks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *QuarantinePodRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	quarantineRequest := &l2smv1.QuarantinePodRequest{}
	if err := r.Get(ctx, req.NamespacedName, quarantineRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sourceNetwork, result, err := r.resolveSourceNetwork(ctx, quarantineRequest)
	if err != nil {
		return result, err
	}

	targetNetwork := &l2smv1.L2Network{}
	if err := r.Get(ctx, client.ObjectKey{Name: quarantineRequest.Spec.TargetL2Network, Namespace: req.Namespace}, targetNetwork); err != nil {
		if apierrors.IsNotFound(err) {
			statusErr := r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "TargetL2NetworkNotFound", fmt.Sprintf("target L2Network %q does not exist", quarantineRequest.Spec.TargetL2Network), sourceNetwork.Name, quarantineRequest.Spec.TargetL2Network, 0, 0)
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, err
	}

	if sourceNetwork.Name == targetNetwork.Name {
		return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "InvalidRequest", "source and target L2Network are the same", sourceNetwork.Name, targetNetwork.Name, 0, 0)
	}

	if r.InternalClient == nil {
		return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "SDNClientNotConfigured", "internal SDN client is not configured", sourceNetwork.Name, targetNetwork.Name, 0, 0)
	}

	exists, err := r.InternalClient.CheckNetworkExists(targetNetwork.Spec.Type, targetNetwork.Name)
	if err != nil {
		return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "TargetL2NetworkCheckFailed", fmt.Sprintf("could not check target L2Network in SDN controller: %v", err), sourceNetwork.Name, targetNetwork.Name, 0, 0)
	}
	if !exists {
		return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "TargetL2NetworkUnavailable", fmt.Sprintf("target L2Network %q does not exist in the SDN controller", targetNetwork.Name), sourceNetwork.Name, targetNetwork.Name, 0, 0)
	}

	podSelector, err := metav1.LabelSelectorAsSelector(&quarantineRequest.Spec.Selector.PodLabelSelector)
	if err != nil {
		return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "InvalidPodSelector", fmt.Sprintf("invalid pod selector: %v", err), sourceNetwork.Name, targetNetwork.Name, 0, 0)
	}

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, &client.ListOptions{Namespace: req.Namespace, LabelSelector: podSelector}); err != nil {
		return ctrl.Result{}, err
	}

	var matchedPods int32
	var movedPods int32
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		moved, err := r.movePodToTargetNetwork(ctx, pod, sourceNetwork, targetNetwork)
		if err != nil {
			logger.Error(err, "could not quarantine pod", "pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name), "sourceL2Network", sourceNetwork.Name, "targetL2Network", targetNetwork.Name)
			return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "PodMoveFailed", err.Error(), sourceNetwork.Name, targetNetwork.Name, matchedPods, movedPods)
		}
		if moved {
			matchedPods++
			movedPods++
		}
	}

	return ctrl.Result{}, r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionTrue, "PodsMoved", fmt.Sprintf("moved %d pod(s) from %q to %q", movedPods, sourceNetwork.Name, targetNetwork.Name), sourceNetwork.Name, targetNetwork.Name, matchedPods, movedPods)
}

func (r *QuarantinePodRequestReconciler) resolveSourceNetwork(ctx context.Context, request *l2smv1.QuarantinePodRequest) (*l2smv1.L2Network, ctrl.Result, error) {
	sourceSelector, err := metav1.LabelSelectorAsSelector(&request.Spec.Selector.L2NetworkSelector)
	if err != nil {
		return nil, ctrl.Result{}, r.setQuarantineStatus(ctx, request, metav1.ConditionFalse, "InvalidL2NetworkSelector", fmt.Sprintf("invalid L2Network selector: %v", err), "", request.Spec.TargetL2Network, 0, 0)
	}

	sourceNetworks := &l2smv1.L2NetworkList{}
	if err := r.List(ctx, sourceNetworks, &client.ListOptions{Namespace: request.Namespace, LabelSelector: sourceSelector}); err != nil {
		return nil, ctrl.Result{}, err
	}

	switch len(sourceNetworks.Items) {
	case 0:
		return nil, ctrl.Result{}, r.setQuarantineStatus(ctx, request, metav1.ConditionFalse, "SourceL2NetworkNotFound", "L2Network selector did not match any source L2Network", "", request.Spec.TargetL2Network, 0, 0)
	case 1:
		return &sourceNetworks.Items[0], ctrl.Result{}, nil
	default:
		return nil, ctrl.Result{}, r.setQuarantineStatus(ctx, request, metav1.ConditionFalse, "MultipleSourceL2Networks", fmt.Sprintf("L2Network selector matched %d source L2Networks; expected exactly one", len(sourceNetworks.Items)), "", request.Spec.TargetL2Network, 0, 0)
	}
}

func (r *QuarantinePodRequestReconciler) movePodToTargetNetwork(ctx context.Context, pod *corev1.Pod, sourceNetwork, targetNetwork *l2smv1.L2Network) (bool, error) {
	l2smNetworksRaw, ok := pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION]
	if !ok {
		return false, nil
	}
	multusNetworksRaw, ok := pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY]
	if !ok {
		return false, nil
	}

	l2smNetworks, err := networkannotation.ExtractNetworks(l2smNetworksRaw, pod.Namespace)
	if err != nil {
		return false, fmt.Errorf("could not extract pod L2Network annotations: %w", err)
	}
	sourceIndex := networkAnnotationIndex(l2smNetworks, sourceNetwork.Name)
	if sourceIndex == -1 {
		return false, nil
	}

	multusNetworks, err := networkannotation.ExtractNetworks(multusNetworksRaw, pod.Namespace)
	if err != nil {
		return false, fmt.Errorf("could not extract pod Multus annotations: %w", err)
	}
	if len(multusNetworks) != len(l2smNetworks) {
		return false, fmt.Errorf("pod has mismatched l2sm and Multus annotation counts: %d l2sm networks, %d Multus networks", len(l2smNetworks), len(multusNetworks))
	}

	portNumber, err := utils.GetPortNumberFromNetAttachDef(multusNetworks[sourceIndex].Name)
	if err != nil {
		return false, fmt.Errorf("could not get port number from network attachment definition %q: %w", multusNetworks[sourceIndex].Name, err)
	}
	ofID := fmt.Sprintf("of:%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: pod.Spec.NodeName, ProviderName: l2smv1.OVERLAY_PROVIDER})))
	ofPort := fmt.Sprintf("%s/%s", ofID, portNumber)

	sourcePayload := sdnclient.VnetPayload{NetworkId: sourceNetwork.Name, Port: []string{ofPort}}
	if err := r.InternalClient.DetachPodFromNetwork(sourceNetwork.Spec.Type, sourcePayload); err != nil {
		return false, fmt.Errorf("could not detach pod %s/%s port %s from source L2Network %q: %w", pod.Namespace, pod.Name, ofPort, sourceNetwork.Name, err)
	}

	targetPayload := sdnclient.VnetPayload{NetworkId: targetNetwork.Name, Port: []string{ofPort}}
	if err := r.InternalClient.AttachPodToNetwork(targetNetwork.Spec.Type, targetPayload); err != nil {
		return false, fmt.Errorf("could not attach pod %s/%s port %s to target L2Network %q: %w", pod.Namespace, pod.Name, ofPort, targetNetwork.Name, err)
	}

	l2smNetworks[sourceIndex].Name = targetNetwork.Name
	pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] = networkannotation.MultusAnnotationToString(l2smNetworks)
	if err := r.Update(ctx, pod); err != nil {
		return false, fmt.Errorf("could not update pod network annotation: %w", err)
	}

	if err := r.updateNetworkStatuses(ctx, sourceNetwork, targetNetwork, pod.Name, multusNetworks[sourceIndex].IPAddresses); err != nil {
		return false, err
	}

	return true, nil
}

func (r *QuarantinePodRequestReconciler) updateNetworkStatuses(ctx context.Context, sourceNetwork, targetNetwork *l2smv1.L2Network, podName string, ipAddresses []string) error {
	if sourceNetwork.Status.ConnectedPodCount > 0 {
		sourceNetwork.Status.ConnectedPodCount--
	}
	targetNetwork.Status.ConnectedPodCount++

	for _, podCIDR := range ipAddresses {
		ip, _, err := net.ParseCIDR(podCIDR)
		if err != nil {
			continue
		}
		ipString := ip.String()
		delete(sourceNetwork.Status.AssignedIPs, ipString)
		if targetNetwork.Status.AssignedIPs == nil {
			targetNetwork.Status.AssignedIPs = map[string]string{}
		}
		targetNetwork.Status.AssignedIPs[ipString] = podName
	}

	if err := r.Status().Update(ctx, sourceNetwork); err != nil {
		return fmt.Errorf("could not update source L2Network status: %w", err)
	}
	if err := r.Status().Update(ctx, targetNetwork); err != nil {
		return fmt.Errorf("could not update target L2Network status: %w", err)
	}

	return nil
}

func (r *QuarantinePodRequestReconciler) setQuarantineStatus(ctx context.Context, request *l2smv1.QuarantinePodRequest, conditionStatus metav1.ConditionStatus, reason, message, sourceNetworkName, targetNetworkName string, matchedPodCount, movedPodCount int32) error {
	request.Status.ObservedGeneration = request.Generation
	request.Status.SourceL2NetworkName = sourceNetworkName
	request.Status.TargetL2NetworkName = targetNetworkName
	request.Status.MatchedPodCount = matchedPodCount
	request.Status.MovedPodCount = movedPodCount
	meta.SetStatusCondition(&request.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             conditionStatus,
		ObservedGeneration: request.Generation,
		Reason:             reason,
		Message:            message,
	})

	if err := r.Status().Update(ctx, request); err != nil {
		return client.IgnoreNotFound(err)
	}
	return nil
}

func networkAnnotationIndex(networks []networkannotation.NetworkAnnotation, networkName string) int {
	for i := range networks {
		if networks[i].Name == networkName {
			return i
		}
	}
	return -1
}

// SetupWithManager sets up the controller with the Manager.
func (r *QuarantinePodRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.InternalClient == nil {
		clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:%s/onos", env.GetControllerIP(), env.GetControllerPort()), Username: "karaf", Password: "karaf"}
		internalClient, err := sdnclient.NewClient(sdnclient.InternalType, clientConfig)
		if err != nil {
			return err
		}
		r.InternalClient = internalClient
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.QuarantinePodRequest{}).
		Named("quarantinepodrequest").
		Complete(r)
}
