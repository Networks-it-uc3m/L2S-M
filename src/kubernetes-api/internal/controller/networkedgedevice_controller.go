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
	"encoding/json"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	"l2sm.k8s.local/controllermanager/internal/utils"
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
	replicaSetOwnerKey = ".metadata.controller"
	apiGVStr           = l2smv1.GroupVersion.String()
)

// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=networkedgedevices/finalizers,verbs=update
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
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

	// name of our custom finalizer
	l2smFinalizer := "l2sm.operator.io/finalizer"

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
			if err := r.deleteExternalResources(ctx, netEdgeDevice); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried.
				return ctrl.Result{}, err
			}

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

		//b, _ := json.Marshal(netEdgeDevice.Spec.Neighbors)

	}

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
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.NetworkEdgeDevice{}).
		Owns(&appsv1.ReplicaSet{}).
		Complete(r)
}

func (r *NetworkEdgeDeviceReconciler) deleteExternalResources(ctx context.Context, netEdgeDevice *l2smv1.NetworkEdgeDevice) error {

	return nil
}
func (r *NetworkEdgeDeviceReconciler) createExternalResources(ctx context.Context, netEdgeDevice *l2smv1.NetworkEdgeDevice) error {
	// Convert netEdgeDevice.Spec.Neighbors to JSON
	neighborsJSON, err := json.Marshal(netEdgeDevice.Spec.Neighbors)
	if err != nil {
		return err
	}

	// Create a ConfigMap to store the neighbors JSON

	constructConfigMapForNED := func(netEdgeDevice *l2smv1.NetworkEdgeDevice) (*corev1.ConfigMap, error) {

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-neighbors", netEdgeDevice.Name),
				Namespace: netEdgeDevice.Namespace,
			},
			Data: map[string]string{
				"neighbors.json": string(neighborsJSON),
			},
		}
		if err := controllerutil.SetControllerReference(netEdgeDevice, configMap, r.Scheme); err != nil {
			return nil, err
		}
		return configMap, nil
	}

	configMap, err := constructConfigMapForNED(netEdgeDevice)

	// Create the ConfigMap in Kubernetes
	if err := r.Client.Create(ctx, configMap); err != nil {
		return err
	}

	constructReplicaSetforNED := func(netEdgeDevice *l2smv1.NetworkEdgeDevice) (*appsv1.ReplicaSet, error) {
		name := fmt.Sprintf("%s-%s", netEdgeDevice.Name, utils.GenerateHash(netEdgeDevice))

		// Define volume mounts to be added to each container
		volumeMounts := []corev1.VolumeMount{
			{
				Name:      "neighbors",
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
				Name: "neighbors",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configMap.Name,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "neighbors.json",
								Path: "neighbors.json",
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
		if err := controllerutil.SetControllerReference(netEdgeDevice, replicaSet, r.Scheme); err != nil {
			return nil, err
		}
		return replicaSet, nil
	}

	replicaSet, err := constructReplicaSetforNED(netEdgeDevice)
	if err != nil {
		return err
	}

	if err = r.Client.Create(ctx, replicaSet); err != nil {
		return err
	}

	return nil
}
