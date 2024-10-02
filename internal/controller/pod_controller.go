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

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/nedinterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
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

	// Check if the pod is being deleted
	if pod.GetDeletionTimestamp() != nil {
		if utils.ContainsString(pod.GetFinalizers(), l2smFinalizer) {
			logger.Info("L2S-M Pod deleted: detaching l2network")

			// If the pod is being deleted, we should free the interface, both the net-attach-def crd and the openflow port.
			// This is done for each interface in the pod.
			// multusAnnotations, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]

			// if !ok {
			// 	logger.Error(nil, "Error detaching the pod from the network attachment definitions")
			// }

			// multusNetAttachDefinitions, err := extractNetworks(pod.Annotations[multusAnnotations], r.SwitchesNamespace)

			// if err != nil {
			// 	logger.Error(nil, "Error detaching the pod from the network attachment definitions")
			// }

			// for _, multusNetAttachDef := range multusNetAttachDefinitions {

			// 	fmt.Println(multusNetAttachDef)
			// 	// We liberate the specific attachment from the node, so it can be used again
			// 	//r.DetachNetAttachDef(ctx, multusNetAttachDef, r.SwitchesNamespace)

			// 	// We liberate the port in the onos app
			// 	//r.InternalClient.DetachPodFromNetwork("vnet",multusNetAttachDef)
			// }

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

	// Check if the pod has a finalizer attached to it. If not, we asume this pod is being created,
	// so we attach the l2network to it.
	if !utils.ContainsString(pod.GetFinalizers(), l2smFinalizer) {
		// We add the finalizers now that the pod has been added to the network and we want to keep track of it
		pod.SetFinalizers(append(pod.GetFinalizers(), l2smFinalizer))
		if err := r.Update(ctx, pod); err != nil {
			return ctrl.Result{}, err
		}
		logger.Info("L2S-M Pod created: attaching to l2network")

		// We extract the network names and ip adresses desired for our pod.
		networkAnnotations, err := extractNetworks(pod.Annotations[L2SM_NETWORK_ANNOTATION], r.SwitchesNamespace)

		// If there's an error, probably the user did input wrongfully the networks, in this case throw an error.
		if err != nil {
			logger.Error(err, "l2 networks could not be extracted from the pods annotations")
			return ctrl.Result{}, err
		}

		// We get an array of the existing L2Networks the pod's being associated to.
		networks, err := GetL2Networks(ctx, r.Client, networkAnnotations)

		// If there's an error, it mayu be that the network is not yet created ornot available. In this case,
		// we don't let the pod create itself, and wait until the l2network is created and/or available
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

		// We get which interfaces are we using inside the pod, so we can later attach them to the sdn controller
		multusNetAttachDefinitions, err := extractNetworks(multusAnnotations, r.SwitchesNamespace)

		// If there are not the same number of multus annotations as networks, we need to throw an error as we can't
		// reach the desired state for the user.
		if err != nil || len(multusNetAttachDefinitions) != len(networkAnnotations) {
			logger.Error(nil, "Error detaching the pod from the network attachment definitions")
			return ctrl.Result{}, nil
		}

		// Now, for every network, we make a call to the sdn controller, asking for the attachment to the switch
		// We get the openflow ID of the switch that this pod is connected to.
		ofID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(pod.Spec.NodeName))

		for index, network := range networks {
			// We get the port number based on the name of the multus annotation. veth1 -> port num 1.
			portNumber, err := utils.GetPortNumberFromNetAttachDef(multusNetAttachDefinitions[index].Name)

			if err != nil {
				// If there is an error, it must be that the name is not compliant, so we can't be certain of which
				// port we are trying to attach.
				return ctrl.Result{}, fmt.Errorf("could not get port number from the multus network annotation: %v. Can't attach pod to network", err)
			}
			ofPort := fmt.Sprintf("%s/%s", ofID, portNumber)

			// we inform the sdn controller of this new port attachment
			err = r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: network.Name, Port: []string{ofPort}})
			if err != nil {
				logger.Error(err, "Error attaching pod to the l2network")
				return ctrl.Result{}, nil
			}
			// If the L2Network is of type inter-domain (has a provider), attach the associated NED
			// and communicate with it
			if network.Spec.Provider != nil {
				logger.Info("Attaching pod to the external sdn controller")
				// First we get information from the NED, required to perform the next operations.
				// The info we need is the node name it is residing in.
				ned, err := nedinterface.GetNetworkEdgeDevice(ctx, r.Client, network.Spec.Provider.Name)

				if err != nil {
					fmt.Printf("error getting NED: %v", err)
					return ctrl.Result{}, nil

				}
				// Then, we create the connection between the NED and the l2sm-switch, in the internal SDN Controller
				nedNetworkAttachDef, err := r.ConnectInternalSwitchToNED(ctx, network.Name, ned.Spec.NodeConfig.NodeName)
				if err != nil {
					fmt.Printf("error connecting NED: %v", err)
					return ctrl.Result{}, nil
				}
				// We attach the ned to this new network, connecting with the IDCO SDN Controller. We need
				// The Network name so we can know which network to attach the port to.
				// The multus network attachment definition that will be used as a bridge between the internal switch and the NED.
				bridgeName, err := utils.GetPortNumberFromNetAttachDef(nedNetworkAttachDef.Name)
				if err != nil {
					// If there is an error, it must be that the name is not compliant, so we can't be certain of which
					// port we are trying to attach.
					return ctrl.Result{}, fmt.Errorf("could not get port number from the multus network annotation: %v. Can't attach pod to network", err)
				}
				err = r.CreateNewNEDConnection(network, fmt.Sprintf("br%s", bridgeName), ned)
				if err != nil {
					fmt.Printf("error attaching NED to the l2network: %v", err)

					return ctrl.Result{}, nil
				}
				logger.Info("Connected pod to inter-domain network")

			}
		}

	}

	return ctrl.Result{}, nil

}

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

// CreateNEDConnection is a method that given the name of the network and the
func (r *PodReconciler) CreateNewNEDConnection(network l2smv1.L2Network, nedNetworkAttachDef string, ned l2smv1.NetworkEdgeDevice) error {

	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s/onos", network.Spec.Provider.Domain), Username: "karaf", Password: "karaf"}

	fmt.Println(clientConfig)
	externalClient, err := sdnclient.NewClient(sdnclient.InternalType, clientConfig)

	if err != nil {
		return fmt.Errorf("no connection could be made with external sdn controller: %s", err)

	}
	// AddPort returns the port number to attach so we can talk directly with the IDCO
	// It needs to know which exiting interface to add to the network
	nedPortNumber, err := nedinterface.AttachInterface(fmt.Sprintf("%s:50051", ned.Spec.NodeConfig.IPAddress), nedNetworkAttachDef)

	if err != nil {
		return fmt.Errorf("no connection could be made with ned: %v", err)
	}

	nedOFID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(utils.GetBridgeName(utils.BridgeParams{NodeName: ned.Spec.NodeConfig.NodeName, ProviderName: network.Spec.Provider.Name})))
	nedOFPort := fmt.Sprintf("%s/%s", nedOFID, nedPortNumber)

	err = externalClient.AttachPodToNetwork(network.Spec.Type, sdnclient.VnetPortPayload{NetworkId: network.Name, Port: []string{nedOFPort}})
	if err != nil {
		return errors.Join(err, errors.New("could not update network attachment definition"))

	}
	return nil
}

func (r *PodReconciler) ConnectInternalSwitchToNED(ctx context.Context, networkName, nedNodeName string) (nettypes.NetworkAttachmentDefinition, error) {

	// We get a free interface in the node name of the NED, this way we can interconnect the NED with the l2sm switch
	var err error
	netAttachDefLabel := NET_ATTACH_LABEL_PREFIX + nedNodeName
	netAttachDefs := GetFreeNetAttachDefs(ctx, r.Client, r.SwitchesNamespace, netAttachDefLabel)

	if len(netAttachDefs.Items) == 0 {
		err = errors.New("no interfaces available in control plane node")
		//logger.Error(err, fmt.Sprintf("No interfaces available for node %s", gatewayNodeName))
		return nettypes.NetworkAttachmentDefinition{}, err
	}

	netAttachDef := &netAttachDefs.Items[0]

	portNumber, _ := utils.GetPortNumberFromNetAttachDef(netAttachDef.Name)

	internalSwitchOFID := fmt.Sprintf("of:%s", utils.GenerateDatapathID(nedNodeName))

	internalSwitchOFPort := fmt.Sprintf("%s/%s", internalSwitchOFID, portNumber)

	err = r.InternalClient.AttachPodToNetwork("vnets", sdnclient.VnetPortPayload{NetworkId: networkName, Port: []string{internalSwitchOFPort}})

	if err != nil {
		return nettypes.NetworkAttachmentDefinition{}, fmt.Errorf("could not make a connection between the internal switch and the NED. Internal SDN controller error: %s", err)

	}

	netAttachDef.Labels[netAttachDefLabel] = "true"
	err = r.Client.Update(ctx, netAttachDef)
	if err != nil {
		return nettypes.NetworkAttachmentDefinition{}, fmt.Errorf("could not update network attachment definition: %s", err)

	}

	return *netAttachDef, nil
}
