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
	"errors"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	"l2sm.k8s.local/controllermanager/internal/nedinterface"
	"l2sm.k8s.local/controllermanager/internal/sdnclient"
	"l2sm.k8s.local/controllermanager/internal/utils"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	logger := log.FromContext(ctx)

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
	if pod.GetDeletionTimestamp() != nil {
		if utils.ContainsString(pod.GetFinalizers(), l2smFinalizer) {
			logger.Info("L2S-M Pod deleted: detaching l2network")

			// If the pod is being deleted, we should free the interface, both the net-attach-def crd and the openflow port.
			// This is done for each interface in the pod.
			multusAnnotations, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]

			if !ok {
				logger.Error(nil, "Error detaching the pod from the network attachment definitions")
				return ctrl.Result{}, nil
			}

			multusNetAttachDefinitions, err := extractNetworks(pod.Annotations[multusAnnotations], r.SwitchesNamespace)

			if err != nil {
				logger.Error(nil, "Error detaching the pod from the network attachment definitions")
				return ctrl.Result{}, nil
			}

			for _, multusNetAttachDef := range multusNetAttachDefinitions {

				fmt.Println(multusNetAttachDef)
				// We liberate the specific attachment from the node, so it can be used again
				//r.DetachNetAttachDef(ctx, multusNetAttachDef, r.SwitchesNamespace)

				// We liberate the port in the onos app
				//r.InternalClient.DetachPodFromNetwork("vnet",multusNetAttachDef)
			}

			// Remove our finalizer from the list and update it.
			pod.SetFinalizers(utils.RemoveString(pod.GetFinalizers(), l2smFinalizer))
			if err := r.Update(ctx, pod); err != nil {
				return ctrl.Result{}, err
			}
			// Stop reconciliation as the item is being deleted
			return ctrl.Result{}, nil
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// If it's not getting deleted, then let's check if it's been created or not
	// Add finalizer for this CR
	if !utils.ContainsString(pod.GetFinalizers(), l2smFinalizer) {
		// We add the finalizers now that the pod has been added to the network and we want to keep track of it
		pod.SetFinalizers(append(pod.GetFinalizers(), l2smFinalizer))
		if err := r.Update(ctx, pod); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("L2S-M Pod created: attaching to l2network")

		networkAnnotations, err := extractNetworks(pod.Annotations[L2SM_NETWORK_ANNOTATION], r.SwitchesNamespace)

		if err != nil {
			logger.Error(err, "l2 networks could not be extracted from the pods annotations")
		}
		networks, err := GetL2Networks(ctx, r.Client, networkAnnotations)
		if err != nil {

			logger.Error(nil, "Pod's network annotation incorrect. L2Network not attached.")
			// return admission.Allowed("Pod's network annotation incorrect. L2Network not attached.")
			return ctrl.Result{}, err

		}
		// Add the pod interfaces to the sdn controller
		multusAnnotations, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]

		if !ok {
			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
			return ctrl.Result{}, nil
		}

		multusNetAttachDefinitions, err := extractNetworks(multusAnnotations, r.SwitchesNamespace)

		if err != nil || len(multusNetAttachDefinitions) != len(networkAnnotations) {
			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
			return ctrl.Result{}, nil
		}

		ofID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(pod.Spec.NodeName))

		for index, network := range networks {
			portNumber, _ := utils.GetPortNumberFromNetAttachDef(multusNetAttachDefinitions[index].Name)
			ofPort := fmt.Sprintf("%s/%s", ofID, portNumber)

			err = r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: network.Name, Port: []string{ofPort}})
			if err != nil {
				logger.Error(err, "Error attaching pod to the l2network")
				return ctrl.Result{}, nil
			}

			// If the L2Network is of type inter-domain (has a provider), attach the associated NED
			// and communicate with it
			if network.Spec.Provider != nil {

				gatewayNodeName, err := r.CreateNewNEDConnection(network)
				if err != nil {
					return ctrl.Result{}, nil
				}

				err = r.ConnectGatewaySwitchToNED(ctx, network.Name, gatewayNodeName)
				if err != nil {
					return ctrl.Result{}, nil
				}

			}
		}

	}

	return ctrl.Result{}, nil

}

// func (r *PodReconciler) GetOpenflowId(ctx context.Context, namespace string) (string,error) {
// 	return "",nil
// }

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error
	// Initialize the InternalClient with the base URL of the SDN controller
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:%s/onos", os.Getenv("CONTROLLER_IP"), os.Getenv("CONTROLLER_PORT")), Username: "karaf", Password: "karaf"}

	r.InternalClient, err = sdnclient.NewClient(sdnclient.InternalType, clientConfig)
	if err != nil {
		r.Log.Error(err, "failed to initiate session with sdn controller")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

func (r *PodReconciler) CreateNewNEDConnection(network l2smv1.L2Network) (string, error) {

	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s/onos/v1", network.Spec.Provider.Domain), Username: "karaf", Password: "karaf"}

	externalClient, err := sdnclient.NewClient(sdnclient.InternalType, clientConfig)

	if err != nil {
		return "", err
		// logger.Error(err, "no connection could be made with external sdn controller")
	}
	gatewayNodeName, nedPortNumber := nedinterface.GetConnectionInfo()

	nedOFID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(fmt.Sprintf("%s%s", gatewayNodeName, network.Spec.Provider.Name)))
	nedOFPort := fmt.Sprintf("%s/%s", nedOFID, nedPortNumber)

	err = externalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: network.Name, Port: []string{nedOFPort}})
	if err != nil {
		return "", errors.Join(err, errors.New("could not update network attachment definition"))

	}
	return gatewayNodeName, nil
}

func (r *PodReconciler) ConnectGatewaySwitchToNED(ctx context.Context, networkName, gatewayNodeName string) error {

	var err error
	netAttachDefLabel := NET_ATTACH_LABEL_PREFIX + gatewayNodeName
	netAttachDefs := GetFreeNetAttachDefs(ctx, r.Client, r.SwitchesNamespace, netAttachDefLabel)

	if len(netAttachDefs.Items) == 0 {
		err = errors.New("no interfaces available in control plane node")
		//logger.Error(err, fmt.Sprintf("No interfaces available for node %s", gatewayNodeName))
		return err
	}

	netAttachDef := &netAttachDefs.Items[0]

	portNumber, _ := utils.GetPortNumberFromNetAttachDef(netAttachDef.Name)

	gatewayOFID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(gatewayNodeName))

	gatewayOFPort := fmt.Sprintf("%s/%s", gatewayOFID, portNumber)

	err = r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: networkName, Port: []string{gatewayOFPort}})

	if err != nil {
		return err
		//logger.Error(err, "could not make a connection between the gateway switch and the NED. Internal SDN controller error.")
	}

	netAttachDef.Labels[netAttachDefLabel] = "true"
	err = r.Client.Update(ctx, netAttachDef)
	if err != nil {
		return err
		//.Error(err, "Could not update network attachment definition")

	}

	return nil
}
