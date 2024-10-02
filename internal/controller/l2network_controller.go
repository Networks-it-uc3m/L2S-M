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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

// L2NetworkReconciler reconciles a L2Network object
type L2NetworkReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// Manages interactions with the onos SDN Controller.
	InternalClient sdnclient.Client
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
	log := log.FromContext(ctx)

	// log := r.Log.WithValues("l2network", req.NamespacedName)

	// Fetch the L2Network instance
	network := &l2smv1.L2Network{}

	err := r.Get(ctx, req.NamespacedName, network)
	if err != nil {
		log.Error(err, "unable to fetch L2Network")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the object is being deleted
	if network.GetDeletionTimestamp() != nil {
		if utils.ContainsString(network.GetFinalizers(), l2smFinalizer) {
			// The object is being deleted
			if err := r.InternalClient.DeleteNetwork(network.Spec.Type, network.Name); err != nil {
				// If fail to delete the external dependency here, return with error
				// so that it can be retried
				log.Error(err, "couldn't delete network in sdn controller")
				return ctrl.Result{}, err
			}

			// Remove our finalizer from the list and update it.
			network.SetFinalizers(utils.RemoveString(network.GetFinalizers(), l2smFinalizer))
			if err := r.Update(ctx, network); err != nil {
				log.Error(err, "couldn't remove finalizer to l2network")
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
			log.Error(err, "failed to create network")
			r.updateControllerStatus(ctx, network, l2smv1.OfflineStatus)

			return ctrl.Result{}, err
		}
		log.Info("Network created in SDN controller", "NetworkID", network.Name)
		r.updateControllerStatus(ctx, network, l2smv1.OnlineStatus)
		network.SetFinalizers(append(network.GetFinalizers(), l2smFinalizer))
		if err := r.Update(ctx, network); err != nil {
			return ctrl.Result{}, err
		}
	}

	// If network is inter domain
	if network.Spec.Provider != nil {
		provStatus, err := interDomainReconcile(network, log)
		if err != nil {
			log.Error(err, "failed to connect to provider")
		}
		network.Status.ProviderConnectivity = &provStatus

		// Update the status in the Kubernetes API
		if statusUpdateErr := r.Status().Update(ctx, network); statusUpdateErr != nil {
			log.Error(statusUpdateErr, "unable to update L2Network provider status")
			return ctrl.Result{}, statusUpdateErr
		}
	}

	// exists, err := r.InternalClient.CheckNetworkExists(network.Spec.Type, network.Name)
	// if err != nil {
	// 	log.Error(err, "failed to check network existence")
	// 	// Update the status to Unknown due to connection issues

	// 	// Update the status in the Kubernetes API
	// 	r.updateControllerStatus(ctx, network, l2smv1.UnknownStatus)
	// 	return ctrl.Result{}, err
	// }

	// if !exists {
	// 	err := r.InternalClient.CreateNetwork(network.Spec.Type, sdnclient.VnetPayload{NetworkId: network.Name})
	// 	if err != nil {
	// 		log.Error(err, "failed to create network")
	// 		r.updateControllerStatus(ctx, network, l2smv1.OfflineStatus)

	// 		return ctrl.Result{}, err
	// 	}
	// 	log.Info("Network created in SDN controller", "NetworkID", network.Name)
	// } else {
	// 	log.Info("Network already exists in SDN controller, no action needed", "NetworkID", network.Name)
	// }
	// if statusUpdateErr := r.updateControllerStatus(ctx, network, l2smv1.OnlineStatus); statusUpdateErr != nil {
	// 	log.Error(statusUpdateErr, "unable to update L2Network provider status")
	// 	return ctrl.Result{}, statusUpdateErr
	// }

	log.Info("something in the rain")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *L2NetworkReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var err error

	r.Log.Info("this is the controller ip", os.Getenv("CONTROLLER_IP"))
	fmt.Println(os.Getenv("CONTROLLER_IP"))
	fmt.Println(os.Getenv("CONTROLLER_PORT"))

	// Initialize the InternalClient with the base URL of the SDN controller
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s:%s/onos", os.Getenv("CONTROLLER_IP"), os.Getenv("CONTROLLER_PORT")), Username: "karaf", Password: "karaf"}

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
	clientConfig := sdnclient.ClientConfig{BaseURL: fmt.Sprintf("http://%s/onos", network.Spec.Provider.Domain), Username: "karaf", Password: "karaf"}

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
