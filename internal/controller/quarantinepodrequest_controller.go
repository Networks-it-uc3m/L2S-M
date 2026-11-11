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
	"reflect"
	"time"

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
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// QuarantinePodRequestReconciler reconciles a QuarantinePodRequest object
type QuarantinePodRequestReconciler struct {
	client.Client
	APIReader      client.Reader
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
	if err := (allocationStore{Client: r.Client, Reader: r.APIReader}).reader().Get(ctx, req.NamespacedName, quarantineRequest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if condition := meta.FindStatusCondition(quarantineRequest.Status.Conditions, "Available"); condition != nil && condition.Status == metav1.ConditionTrue && condition.ObservedGeneration == quarantineRequest.Generation {
		return ctrl.Result{}, nil
	}

	sourceNetwork, result, err := r.resolveSourceNetwork(ctx, quarantineRequest)
	if err != nil || sourceNetwork == nil {
		return result, err
	}

	targetNetwork := &l2smv1.L2Network{}
	if err := (allocationStore{Client: r.Client, Reader: r.APIReader}).reader().Get(ctx, client.ObjectKey{Name: quarantineRequest.Spec.TargetL2Network, Namespace: req.Namespace}, targetNetwork); err != nil {
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
	if err := (allocationStore{Client: r.Client, Reader: r.APIReader}).reader().List(ctx, podList, &client.ListOptions{Namespace: req.Namespace, LabelSelector: podSelector}); err != nil {
		return ctrl.Result{}, err
	}

	var matchedPods int32
	var movedPods int32
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.GetDeletionTimestamp() != nil {
			continue
		}

		moved, err := r.movePodToTargetNetwork(ctx, pod, sourceNetwork, targetNetwork, string(quarantineRequest.UID))
		if err != nil {
			logger.Error(err, "could not quarantine pod", "pod", fmt.Sprintf("%s/%s", pod.Namespace, pod.Name), "sourceL2Network", sourceNetwork.Name, "targetL2Network", targetNetwork.Name)
			if statusErr := r.setQuarantineStatus(ctx, quarantineRequest, metav1.ConditionFalse, "PodMoveFailed", err.Error(), sourceNetwork.Name, targetNetwork.Name, matchedPods, movedPods); statusErr != nil {
				return ctrl.Result{}, statusErr
			}
			return ctrl.Result{RequeueAfter: time.Second}, err
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
	if err := (allocationStore{Client: r.Client, Reader: r.APIReader}).reader().List(ctx, sourceNetworks, &client.ListOptions{Namespace: request.Namespace, LabelSelector: sourceSelector}); err != nil {
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

const pendingMoveAnnotation = "l2sm.l2sm.k8s.local/pending-move"
const completedMoveAnnotation = "l2sm.l2sm.k8s.local/completed-move"

type pendingMove struct {
	RequestID string `json:"requestID,omitempty"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	SourceUID string `json:"sourceUID"`
	TargetUID string `json:"targetUID"`
}

func (r *QuarantinePodRequestReconciler) movePodToTargetNetwork(ctx context.Context, pod *corev1.Pod, sourceNetwork, targetNetwork *l2smv1.L2Network, requestIDs ...string) (bool, error) {
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	key := client.ObjectKeyFromObject(pod)
	operationCtx, release, err := acquirePodOperation(ctx, store, key)
	if err != nil {
		return false, err
	}
	defer release()
	ctx = operationCtx
	if err := store.reader().Get(ctx, key, pod); err != nil {
		return false, err
	}
	networks, err := networkannotation.ExtractNetworks(pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION], pod.Namespace)
	if err != nil {
		return false, err
	}
	sourceIndex := networkAnnotationIndex(networks, sourceNetwork.Name)
	journal := pendingMove{Source: sourceNetwork.Name, Target: targetNetwork.Name, SourceUID: string(sourceNetwork.UID), TargetUID: string(targetNetwork.UID)}
	previous := pod.Annotations[pendingMoveAnnotation]
	if len(requestIDs) > 0 {
		journal.RequestID = requestIDs[0]
	} else if previous != "" {
		var unfinished pendingMove
		if err := json.Unmarshal([]byte(previous), &unfinished); err != nil {
			return false, err
		}
		journal.RequestID = unfinished.RequestID
	}
	encoded, _ := json.Marshal(journal)
	desiredJournal := string(encoded)

	if previous != "" && previous != desiredJournal {
		return false, fmt.Errorf("pod %s has another pending network move", pod.Name)
	}
	if sourceIndex == -1 {
		if previous == "" {
			// Preserve counts when another reconcile completed this request's
			// move after our previous attempt stopped midway through a batch.
			completed := journal.RequestID != "" && pod.Annotations[completedMoveAnnotation] == desiredJournal && networkAnnotationIndex(networks, targetNetwork.Name) >= 0
			return completed, nil
		}
		// A previous attempt changed the pod annotation but failed during status
		// updates. Its durable journal keeps that unfinished move discoverable.
		sourceIndex = networkAnnotationIndex(networks, targetNetwork.Name)
	}
	if sourceIndex == -1 {
		return false, fmt.Errorf("pending move has no matching pod interface")
	}
	attachments, err := networkannotation.ExtractNetworks(pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY], pod.Namespace)
	if err != nil || len(attachments) != len(networks) {
		return false, fmt.Errorf("pod has invalid network attachment mapping: %v", err)
	}
	port, err := utils.GetPortNumberFromNetAttachDef(attachments[sourceIndex].Name)
	if err != nil {
		return false, err
	}
	// Persist intent before external effects. Concurrent moves of the same pod
	// compare the journal on a fresh resource version instead of racing blindly.
	if err := updatePod(ctx, store, key, func(current *corev1.Pod) error {
		if current.DeletionTimestamp != nil {
			return fmt.Errorf("pod is deleting")
		}
		if current.Annotations[pendingMoveAnnotation] != "" && current.Annotations[pendingMoveAnnotation] != desiredJournal {
			return fmt.Errorf("another network move is in progress")
		}
		if current.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] != pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] {
			return fmt.Errorf("pod membership changed; retry move")
		}
		current.Annotations[pendingMoveAnnotation] = desiredJournal
		return nil
	}); err != nil {
		return false, err
	}
	// Reserve target addresses before forwarding changes; refuse collisions with
	// existing target members rather than overwriting their status entries.
	ips := attachments[sourceIndex].IPAddresses
	if _, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(targetNetwork), pod.Name, podAllocationID(pod), ips); err != nil {
		// Nothing external has changed; remove our intent so a permanent address
		// collision does not prevent a different request from moving the pod.
		_ = updatePod(ctx, store, key, func(current *corev1.Pod) error {
			if current.Annotations[pendingMoveAnnotation] == desiredJournal {
				delete(current.Annotations, pendingMoveAnnotation)
			}
			return nil
		})
		return false, err
	}
	ofPort := fmt.Sprintf("of:%s/%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: pod.Spec.NodeName, ProviderName: l2smv1.OVERLAY_PROVIDER})), port)
	if err := r.InternalClient.DetachPodFromNetwork(sourceNetwork.Spec.Type, sdnclient.VnetPayload{NetworkId: sourceNetwork.Name, Port: []string{ofPort}}); err != nil {
		return false, err
	}
	if err := r.InternalClient.AttachPodToNetwork(targetNetwork.Spec.Type, sdnclient.VnetPayload{NetworkId: targetNetwork.Name, Port: []string{ofPort}}); err != nil {
		return false, err
	}
	if err := updatePod(ctx, store, key, func(current *corev1.Pod) error {
		if current.Annotations[pendingMoveAnnotation] != desiredJournal {
			return fmt.Errorf("move journal changed")
		}
		currentNetworks, err := networkannotation.ExtractNetworks(current.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION], current.Namespace)
		if err != nil {
			return err
		}
		index := networkAnnotationIndex(currentNetworks, sourceNetwork.Name)
		if index >= 0 {
			currentNetworks[index].Name = targetNetwork.Name
		}
		value := networkannotation.MultusAnnotationToString(currentNetworks)
		current.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] = value
		current.Annotations[attachedNetworksAnnotation] = value
		return nil
	}); err != nil {
		return false, err
	}
	if err := store.releaseIPs(ctx, client.ObjectKeyFromObject(sourceNetwork), pod.Name, podAllocationID(pod)); err != nil {
		return false, err
	}
	if err := updatePod(ctx, store, key, func(current *corev1.Pod) error {
		if current.Annotations[pendingMoveAnnotation] == desiredJournal {
			current.Annotations[completedMoveAnnotation] = desiredJournal
			delete(current.Annotations, pendingMoveAnnotation)
		}
		return nil
	}); err != nil {
		return false, err
	}
	return true, nil
}

func (r *QuarantinePodRequestReconciler) setQuarantineStatus(ctx context.Context, request *l2smv1.QuarantinePodRequest, conditionStatus metav1.ConditionStatus, reason, message, sourceNetworkName, targetNetworkName string, matchedPodCount, movedPodCount int32) error {
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	return retry.RetryOnConflict(allocationBackoff, func() error {
		current := &l2smv1.QuarantinePodRequest{}
		if err := store.reader().Get(ctx, client.ObjectKeyFromObject(request), current); err != nil {
			return client.IgnoreNotFound(err)
		}
		if current.Generation != request.Generation || current.UID != request.UID {
			return fmt.Errorf("quarantine request changed during reconciliation")
		}
		before := current.DeepCopy().Status
		current.Status.ObservedGeneration = current.Generation
		current.Status.SourceL2NetworkName = sourceNetworkName
		current.Status.TargetL2NetworkName = targetNetworkName
		current.Status.MatchedPodCount = matchedPodCount
		current.Status.MovedPodCount = movedPodCount
		meta.SetStatusCondition(&current.Status.Conditions, metav1.Condition{Type: "Available", Status: conditionStatus, ObservedGeneration: current.Generation, Reason: reason, Message: message})
		if reflect.DeepEqual(current.Status, before) {
			return nil
		}
		return r.Status().Update(ctx, current)
	})
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
	r.APIReader = mgr.GetAPIReader()
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
