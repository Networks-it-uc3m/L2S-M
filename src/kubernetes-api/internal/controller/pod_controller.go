/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	"l2sm.k8s.local/controllermanager/internal/sdnclient"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Log logr.Logger

	Scheme            *runtime.Scheme
	SwitchesNamespace string
	InternalClient    sdnclient.Client
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

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
	// logger := log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.Get(ctx, req.NamespacedName, pod)
	if err != nil {
		// Error reading the object - requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the 'l2sm/network' annotation is present. If not, we are not interested in this pod.
	if _, ok := pod.GetAnnotations()[L2SM_NETWORK_ANNOTATION]; !ok {
		return ctrl.Result{}, nil

	}

	// Ensure the Multus annotation is correctly formatted and present

	// Check if the object is being deleted
	// if pod.GetDeletionTimestamp() != nil {
	// 	if utils.ContainsString(pod.GetFinalizers(), "pod.finalizers.l2sm.k8s.local") {
	// 		// If the pod is being deleted, we should free the interface, both the net-attach-def crd and the openflow port.
	// 		// This is done for each interface in the pod.
	// 		multusAnnotations, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]

	// 		if !ok {
	// 			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
	// 			return ctrl.Result{}, nil
	// 		}

	// 		multusNetAttachDefinitions, err := extractNetworks(pod.Annotations[multusAnnotations], r.SwitchesNamespace)

	// 		if err != nil {
	// 			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
	// 			return ctrl.Result{}, nil
	// 		}

	// 		for _, multusNetAttachDef := range multusNetAttachDefinitions {

	// 			// We liberate the specific attachment from the node, so it can be used again
	// 			r.DetachNetAttachDef(ctx, multusNetAttachDef, r.SwitchesNamespace)

	// 			// We liberate the port in the onos app
	// 			// r.InternalClient.DetachPodFromNetwork("vnet",multusNetAttachDef)
	// 		}

	// 		// Remove our finalizer from the list and update it.
	// 		pod.SetFinalizers(utils.RemoveString(pod.GetFinalizers(), "pod.finalizers.l2sm.k8s.local"))
	// 		if err := r.Update(ctx, pod); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 		// Stop reconciliation as the item is being deleted
	// 		return ctrl.Result{}, nil
	// 	}

	// 	// If it's not getting deleted, then let's check if it's been created or not
	// 	// Add finalizer for this CR
	// 	if !utils.ContainsString(pod.GetFinalizers(), "pod.finalizers.l2sm.k8s.local") {

	// 		networks, err := extractNetworks(pod.Annotations[L2SM_NETWORK_ANNOTATION], r.SwitchesNamespace)

	// 		if created, _ := r.verifyNetworksAreCreated(ctx, networks); !created {

	// 			logger.Error(nil, "Pod's network annotation incorrect. L2Network not attached.")
	// 			// return admission.Allowed("Pod's network annotation incorrect. L2Network not attached.")
	// 			return ctrl.Result{}, err

	// 		}
	// 		// Add the pod interfaces to the sdn controller
	// 		multusAnnotations, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]

	// 		if !ok {
	// 			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
	// 			return ctrl.Result{}, nil
	// 		}

	// 		multusNetAttachDefinitions, err := extractNetworks(pod.Annotations[multusAnnotations], r.SwitchesNamespace)

	// 		if err != nil {
	// 			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
	// 			return ctrl.Result{}, nil
	// 		}

	// 		fmt.Println(multusNetAttachDefinitions)

	// 		// ofID := r.GetOpenflowId(ctx, pod.Spec.NodeName)

	// 		// for index, network := range networks {

	// 		// 	portNumber, _ := utils.GetPortNumberFromNetAttachDef(multusNetAttachDefinitions[index].Name)
	// 		// 	ofPort := fmt.Sprintf("%s/%s", ofID, portNumber)

	// 		// 	r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: network.Name, Port: []string{ofPort}})
	// 		// }

	// 		// We add the finalizers now that the pod has been added to the network and we want to keep track of it
	// 		pod.SetFinalizers(append(pod.GetFinalizers(), "pod.finalizers.l2sm.k8s.local"))
	// 		if err := r.Update(ctx, pod); err != nil {
	// 			return ctrl.Result{}, err
	// 		}
	// 	}

	// }

	// r.GetOpenflowId(ctx,pod.Spec.NodeName)

	// for _, network := range networks {

	// 	ofPort := fmt.Sprintf("%s/%s",ofID,portNumber)
	// 	sdnclient.VnetPortPayload{NetworkId: network.Name, Port: ofPort}

	// 	r.InternalClient.AttachPodToNetwork()
	// }
	return ctrl.Result{}, nil

}

// func (r *PodReconciler) GetOpenflowId(ctx context.Context, namespace string) (string,error) {
// 	return "",nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	// Initialize the InternalClient with the base URL of the SDN controller
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:8181/onos", os.Getenv("CONTROLLER_IP")), Username: "karaf", Password: "karaf"}

	r.InternalClient, err = sdnclient.NewClient(sdnclient.InternalType, clientConfig)
	if err != nil {
		r.Log.Error(err, "failed to initiate session with sdn controller")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

func (r *PodReconciler) verifyNetworksAreCreated(ctx context.Context, networks []NetworkAnnotation) (bool, error) {
	// List all L2Networks
	l2Networks := &l2smv1.L2NetworkList{}
	if err := r.List(ctx, l2Networks); err != nil {
		return false, err
	}

	// Create a map of existing L2Network names for quick lookup
	existingNetworks := make(map[string]struct{})
	for _, network := range l2Networks.Items {
		existingNetworks[network.Name] = struct{}{}
	}

	// Verify if each annotated network exists
	for _, net := range networks {
		if _, exists := existingNetworks[net.Name]; !exists {
			return false, nil
		}

	}

	return true, nil
}

func (r *PodReconciler) DetachNetAttachDef(ctx context.Context, multusNetAttachDef NetworkAnnotation, namespace string) error {

	netAttachDef := &nettypes.NetworkAttachmentDefinition{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      multusNetAttachDef.Name,
		Namespace: namespace,
	}, netAttachDef)
	if err != nil {
		return err
	}
	err = r.Delete(ctx, netAttachDef)
	return err

}

func (r *PodReconciler) GetOpenflowId(ctx context.Context, nodename string) (string, error) {
	return "", nil
	// // Define the list options with the namespace
	// listOptions := &client.ListOptions{
	//     Namespace: r.SwitchesNamespace,
	// }

	// // List all pods in the namespace
	// podList := &corev1.PodList{}
	// if err := r.List(ctx, podList, listOptions); err != nil {
	//     return "", fmt.Errorf("failed to list pods: %w", err)
	// }

	// // Iterate through the pods to find the one on the specified node
	// var switchPod *corev1.Pod
	// for _, pod := range podList.Items {
	//     if pod.Spec.NodeName == nodename {
	//         switchPod = &pod
	//         break
	//     }
	// }

	// if switchPod == nil {
	//     return "", fmt.Errorf("pod not found on node: %s", nodename)
	// }

	// // Check if the OpenFlow ID is already annotated
	// if openflowID, exists := switchPod.Annotations["openflow-id"]; exists && openflowID != "" {
	//     return openflowID, nil
	// }

	// // Retrieve the OpenFlow ID (replace this with your actual logic)
	// openflowID, err := r.InternalClient.RetrieveOpenflowID(switchPod.)
	// if err != nil {
	//     return "", fmt.Errorf("failed to retrieve openflow id: %w", err)
	// }

	// // Annotate the pod with the OpenFlow ID
	// if switchPod.Annotations == nil {
	//     switchPod.Annotations = make(map[string]string)
	// }
	// switchPod.Annotations["openflow-id"] = openflowID

	// // Update the pod with the new annotation
	// if err := r.Update(ctx, switchPod); err != nil {
	//     return "", fmt.Errorf("failed to update pod with openflow id: %w", err)
	// }

	// return openflowID, nil
}
