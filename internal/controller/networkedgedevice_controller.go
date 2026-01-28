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
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/lpminterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	talpav1 "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NetworkEdgeDeviceReconciler reconciles a NetworkEdgeDevice object
type NetworkEdgeDeviceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	// name of our custom finalizer
	l2smFinalizer      = "l2sm.operator.io/finalizer"
	replicaSetOwnerKey = ".metadata.controller"
	apiGVStr           = l2smv1.GroupVersion.String()
)

// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NetworkEdgeDevice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *NetworkEdgeDeviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	netEdgeDevice := &l2smv1.NetworkEdgeDevice{}

	if err := r.Get(ctx, req.NamespacedName, netEdgeDevice); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if netEdgeDevice.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(netEdgeDevice, l2smFinalizer) {
			controllerutil.AddFinalizer(netEdgeDevice, l2smFinalizer)
			if err := r.Update(ctx, netEdgeDevice); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Network Edge Device created", "NetworkEdgeDevice", netEdgeDevice.Name)

		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(netEdgeDevice, l2smFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			// if err := r.deleteExternalResources(ctx, netEdgeDevice); err != nil {
			// 	// if fail to delete the external dependency here, return with error
			// 	// so that it can be retried.
			// 	return ctrl.Result{}, err
			// }

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(netEdgeDevice, l2smFinalizer)
			if err := r.Update(ctx, netEdgeDevice); err != nil {
				return ctrl.Result{}, err
			}

		}
		log.Info("Network Edge Device deleted", "NetworkEdgeDevice", netEdgeDevice.Name)
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var switchReplicaSets appsv1.ReplicaSetList
	if err := r.List(ctx, &switchReplicaSets, client.InNamespace(req.Namespace), client.MatchingFields{replicaSetOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child ReplicaSets")
		return ctrl.Result{}, err
	}

	if len(switchReplicaSets.Items) == 0 {
		if err := r.createExternalResources(ctx, netEdgeDevice); err != nil {
			log.Error(err, "unable to create ReplicaSet")
			return ctrl.Result{}, err
		}
		log.Info("NED Launched")
		return ctrl.Result{RequeueAfter: time.Second * 20}, nil
	} else {
		if err := r.reconcileConfigMap(ctx, netEdgeDevice); err != nil {
			log.Error(err, "unable to reconcile configmap")
			return ctrl.Result{}, err
		}
	}

	// 	//b, _ := json.Marshal(netEdgeDevice.Spec.Neighbors)

	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NetworkEdgeDeviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.ReplicaSet{}, replicaSetOwnerKey, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		replicaSet := rawObj.(*appsv1.ReplicaSet)
		owner := metav1.GetControllerOf(replicaSet)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "NetworkEdgeDevice" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.ConfigMap{}, replicaSetOwnerKey, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		configMap := rawObj.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "NetworkEdgeDevice" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.NetworkEdgeDevice{}).
		Owns(&appsv1.ReplicaSet{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

// func (r *NetworkEdgeDeviceReconciler) deleteExternalResources(ctx context.Context, netEdgeDevice *l2smv1.NetworkEdgeDevice) error {

//		return nil
//	}
func (r *NetworkEdgeDeviceReconciler) createExternalResources(ctx context.Context, netEdgeDevice *l2smv1.NetworkEdgeDevice) error {
	var extResources []client.Object

	// Create a ConfigMap to store the neighbors JSON
	configMap, err := constructConfigMapForNED(netEdgeDevice, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not construct the config map for the network edge device: %v", err)
	}
	extResources = append(extResources, configMap)

	replicaSet, err := constructReplicaSetforNED(netEdgeDevice, configMap.Name)
	if err != nil {
		return fmt.Errorf("could not construct replicaset for network edge device: %v", err)
	}

	if netEdgeDevice.Spec.Monitor != nil {

		monCont, monCMs, err := lpminterface.BuildMonitoringCollectorResources()
	}
	extResources = append(extResources, replicaSet)

	for _, obj := range extResources {
		if err := controllerutil.SetControllerReference(netEdgeDevice, obj, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference to obj %s: %w", obj.GetName(), err)
		}
		if err := r.Client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %s %s: %w", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
	}

	return nil
}

func constructReplicaSetforNED(netEdgeDevice *l2smv1.NetworkEdgeDevice, configmapName string) (*appsv1.ReplicaSet, error) {

	name := utils.GenerateReplicaSetName(utils.GenerateSwitchName(netEdgeDevice.Name, netEdgeDevice.Spec.NodeConfig.NodeName, utils.NetworkEdgeDevice))
	// Define volume mounts to be added to each container
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "configurations",
			MountPath: "/etc/l2sm/",
			ReadOnly:  true,
		},
	}

	// Update containers to include the volume mount
	containers := make([]corev1.Container, len(netEdgeDevice.Spec.SwitchTemplate.Spec.Containers))
	for i, container := range netEdgeDevice.Spec.SwitchTemplate.Spec.Containers {
		container.VolumeMounts = append(container.VolumeMounts, volumeMounts...)
		containers[i] = container
	}

	// Define the volume using the created ConfigMap
	volumes := []corev1.Volume{
		{
			Name: "configurations",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  "neighbors.json",
							Path: "neighbors.json",
						},
						{
							Key:  "config.json",
							Path: "config.json",
						},
					},
				},
			},
		},
	}

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   netEdgeDevice.Namespace,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "l2sm",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "l2sm",
					},
				},
				Spec: corev1.PodSpec{
					InitContainers: netEdgeDevice.Spec.SwitchTemplate.Spec.InitContainers,
					Containers:     containers,
					Volumes:        volumes,
					HostNetwork:    netEdgeDevice.Spec.SwitchTemplate.Spec.HostNetwork,
					NodeSelector: map[string]string{
						corev1.LabelHostname: netEdgeDevice.Spec.NodeConfig.NodeName,
					},
					Tolerations: []corev1.Toleration{
						{Operator: corev1.TolerationOpExists},
					},
				},
			},
		},
	}

	for k, v := range netEdgeDevice.Spec.SwitchTemplate.Annotations {
		replicaSet.Annotations[k] = v
	}
	for k, v := range netEdgeDevice.Spec.SwitchTemplate.Labels {
		replicaSet.Labels[k] = v
	}
	return replicaSet, nil
}
func (r *NetworkEdgeDeviceReconciler) reconcileConfigMap(ctx context.Context, netEdgeDevice *l2smv1.NetworkEdgeDevice) error {

	configMap, err := constructConfigMapForNED(netEdgeDevice, r.Scheme)
	if err != nil {
		return fmt.Errorf("could not construct the config map for the network edge device: %v", err)
	}

	if err := r.Client.Patch(ctx, configMap, client.Apply, client.FieldOwner("yo"), client.ForceOwnership); err != nil {
		return fmt.Errorf("failed to apply config map: %w", err)
	}

	return nil
}

func constructConfigMapForNED(netEdgeDevice *l2smv1.NetworkEdgeDevice, scheme *runtime.Scheme) (*corev1.ConfigMap, error) {
	neighbors := make([]string, len(netEdgeDevice.Spec.Neighbors))
	for i, neighbor := range netEdgeDevice.Spec.Neighbors {
		neighbors[i] = neighbor.Domain
	}
	nedName := dp.GetSwitchName(dp.DatapathParams{NodeName: netEdgeDevice.Spec.NodeConfig.NodeName, ProviderName: netEdgeDevice.Spec.Provider.Name})

	nedConfig, err := json.Marshal(talpav1.Settings{
		ControllerIP:   netEdgeDevice.Spec.Provider.Domain,
		ControllerPort: netEdgeDevice.Spec.Provider.OFPort,
		NodeName:       netEdgeDevice.Spec.NodeConfig.NodeName,
		SwitchName:     nedName})
	if err != nil {
		return nil, err
	}
	nedNeighbors, err := json.Marshal(talpav1.Node{Name: netEdgeDevice.Spec.NodeConfig.NodeName, NodeIP: netEdgeDevice.Spec.NodeConfig.IPAddress, NeighborNodes: neighbors})
	if err != nil {
		return nil, err
	}
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", netEdgeDevice.Name),
			Namespace: netEdgeDevice.Namespace,
		},
		Data: map[string]string{
			"neighbors.json": string(nedNeighbors),
			"config.json":    string(nedConfig),
		},
	}
	return configMap, nil
}
