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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/dnsinterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/L2S-M/internal/nedinterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

// L2NetworkReconciler reconciles a L2Network object
type L2NetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// Manages interactions with the onos SDN Controller.
	InternalClient    sdnclient.Client
	SwitchesNamespace string
}

//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=l2networks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=l2networks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=l2networks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the L2Network object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *L2NetworkReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// log := r.Log.WithValues("l2network", req.NamespacedName)

	// Fetch the L2Network instance
	network := &l2smv1.L2Network{}

	err := r.Get(ctx, req.NamespacedName, network)
	if err != nil {
		logger.Error(err, "unable to fetch L2Network")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if network.GetDeletionTimestamp() != nil {
		if utils.ContainsString(network.GetFinalizers(), l2smFinalizer) {
			// The object is being deleted
			if err := r.InternalClient.DeleteNetwork(network.Spec.Type, network.Name); err != nil {
				// If fail to delete the external dependency here, return with error
				// so that it can be retried
				logger.Error(err, "couldn't delete network in sdn controller")
				return ctrl.Result{}, err
			}

			// Remove our finalizer from the list and update it.
			network.SetFinalizers(utils.RemoveString(network.GetFinalizers(), l2smFinalizer))
			if err := r.Update(ctx, network); err != nil {
				logger.Error(err, "couldn't remove finalizer to l2network")
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !utils.ContainsString(network.GetFinalizers(), l2smFinalizer) {
		err := r.InternalClient.CreateNetwork(network.Spec.Type, sdnclient.VnetPayload{NetworkId: network.Name})
		if err != nil {
			logger.Error(err, "failed to create network")
			r.updateControllerStatus(ctx, network, l2smv1.OfflineStatus)

			return ctrl.Result{}, err
		}
		logger.Info("Network created in SDN controller", "NetworkID", network.Name)
		r.updateControllerStatus(ctx, network, l2smv1.OnlineStatus)
		network.Status.AssignedIPs = make(map[string]string)
		network.SetFinalizers(append(network.GetFinalizers(), l2smFinalizer))
		if err := r.Update(ctx, network); err != nil {
			return ctrl.Result{}, err
		}
		// If network is inter domain
		if network.Spec.Provider != nil {
			provStatus, err := interDomainReconcile(network, logger)
			if err != nil {
				logger.Error(err, "failed to connect to provider")
			}
			network.Status.ProviderConnectivity = &provStatus

			// Update the status in the Kubernetes API
			if statusUpdateErr := r.Status().Update(ctx, network); statusUpdateErr != nil {
				logger.Error(statusUpdateErr, "unable to update L2Network provider status")
				return ctrl.Result{}, statusUpdateErr
			}

			logger.Info("Attaching NED to internal Overlay for new network")

			// First we get information from the NED, required to perform the next operations.
			// The info we need is the node name it is residing in.
			ned, err := nedinterface.GetNetworkEdgeDevice(ctx, r.Client, network.Spec.Provider.Name)

			if err != nil {
				logger.Error(err, "error getting NED")
				return ctrl.Result{}, nil

			}
			// Then, we create the connection between the NED and the l2sm-switch, in the internal SDN Controller
			nedNetworkAttachDef, err := r.ConnectInternalSwitchToNED(ctx, network.Name, ned.Spec.NodeConfig.NodeName)
			if err != nil {
				logger.Error(err, "error connecting NED")
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
				logger.Error(err, "error attaching NED to the l2network")

				return ctrl.Result{}, nil
			}
			logger.Info("Connected overlay to inter-domain network")

			dnsinterface.AddServerToLocalCoreDNS(r.Client, network.Name, network.Spec.Provider.Domain, network.Spec.Provider.DNSPort)

		}
	}

	// exists, err := r.InternalClient.CheckNetworkExists(network.Spec.Type, network.Name)
	// if err != nil {
	// 	logger.Error(err, "failed to check network existence")
	// 	// Update the status to Unknown due to connection issues

	// 	// Update the status in the Kubernetes API
	// 	r.updateControllerStatus(ctx, network, l2smv1.UnknownStatus)
	// 	return ctrl.Result{}, err
	// }

	// if !exists {
	// 	err := r.InternalClient.CreateNetwork(network.Spec.Type, sdnclient.VnetPayload{NetworkId: network.Name})
	// 	if err != nil {
	// 		logger.Error(err, "failed to create network")
	// 		r.updateControllerStatus(ctx, network, l2smv1.OfflineStatus)

	// 		return ctrl.Result{}, err
	// 	}
	// 	logger.Info("Network created in SDN controller", "NetworkID", network.Name)
	// } else {
	// 	logger.Info("Network already exists in SDN controller, no action needed", "NetworkID", network.Name)
	// }
	// if statusUpdateErr := r.updateControllerStatus(ctx, network, l2smv1.OnlineStatus); statusUpdateErr != nil {
	// 	logger.Error(statusUpdateErr, "unable to update L2Network provider status")
	// 	return ctrl.Result{}, statusUpdateErr
	// }
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *L2NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	// Initialize the InternalClient with the base URL of the SDN controller
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:%s/onos", env.GetControllerIP(), env.GetControllerPort()), Username: "karaf", Password: "karaf"}

	r.InternalClient, err = sdnclient.NewClient(sdnclient.InternalType, clientConfig)
	if err != nil {
		r.Log.Error(err, "failed to initiate session with sdn controller")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.L2Network{}). // Watch for changes to primary resource L2Network
		Complete(r)
}

func interDomainReconcile(network *l2smv1.L2Network, log logr.Logger) (l2smv1.ConnectivityStatus, error) {

	if network.Spec.Provider == nil {
		return l2smv1.UnknownStatus, errors.New("ext-vnet doesn't have a provider specified")
	}

	providerAddress := fmt.Sprintf("%s:%s", network.Spec.Provider.Domain, utils.DefaultIfEmpty(network.Spec.Provider.SDNPort, "30808"))
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s/onos", providerAddress), Username: "karaf", Password: "karaf"}

	externalClient, err := sdnclient.NewClient(sdnclient.InternalType, clientConfig)

	if err != nil {
		return l2smv1.OfflineStatus, fmt.Errorf("could not initialize session with external provider: %v", err)
	}

	exists, err := externalClient.CheckNetworkExists(network.Spec.Type, network.Name)
	if err != nil {
		log.Error(err, "failed to check network existence")

		return l2smv1.UnknownStatus, err
	}

	if !exists {
		err := externalClient.CreateNetwork(network.Spec.Type, sdnclient.VnetPayload{NetworkId: network.Name})
		if err != nil {
			log.Error(err, "failed to create network")
			return l2smv1.OfflineStatus, err
		}
		log.Info("Network created in Provider controller", "NetworkID", network.Name)
	} else {
		log.Info("Network already exists in Provider controller, no action needed", "NetworkID", network.Name)
	}
	return l2smv1.OnlineStatus, nil

}

func (r *L2NetworkReconciler) updateControllerStatus(ctx context.Context, network *l2smv1.L2Network, status l2smv1.ConnectivityStatus) error {

	network.Status.InternalConnectivity = &status

	return r.Status().Update(ctx, network)

}

func (r *L2NetworkReconciler) ConnectInternalSwitchToNED(ctx context.Context, networkName, nedNodeName string) (nettypes.NetworkAttachmentDefinition, error) {

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

// CreateNEDConnection is a method that given the name of the network and the
func (r *L2NetworkReconciler) CreateNewNEDConnection(network *l2smv1.L2Network, nedNetworkAttachDef string, ned l2smv1.NetworkEdgeDevice) error {

	providerAddress := fmt.Sprintf("%s:%s", network.Spec.Provider.Domain, utils.DefaultIfEmpty(network.Spec.Provider.SDNPort, "30808"))
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s/onos", providerAddress), Username: "karaf", Password: "karaf"}

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
