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
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	talpav1 "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
)

// OverlayReconciler reconciles a Overlay object
type OverlayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var setOwnerKeyOverlay = ".metadata.controller.overlay"
var OVERLAY_PROVIDER = "l2sm-controller"

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays/finalizers,verbs=update
// +kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=overlays/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Overlay object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *OverlayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	overlay := &l2smv1.Overlay{}

	if err := r.Get(ctx, req.NamespacedName, overlay); err != nil {
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// name of our custom finalizer

	// examine DeletionTimestamp to determine if object is under deletion
	if overlay.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(overlay, l2smFinalizer) {
			controllerutil.AddFinalizer(overlay, l2smFinalizer)
			if err := r.Update(ctx, overlay); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("Overlay created", "Overlay", overlay.Name)

		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(overlay, l2smFinalizer) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(ctx, overlay); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried.
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(overlay, l2smFinalizer)
			if err := r.Update(ctx, overlay); err != nil {
				return ctrl.Result{}, err
			}

		}
		log.Info("Overlay deleted", "Overlay", overlay.Name)
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var switchReplicaSets appsv1.ReplicaSetList
	if err := r.List(ctx, &switchReplicaSets, client.InNamespace(req.Namespace), client.MatchingFields{setOwnerKeyOverlay: req.Name}); err != nil {
		log.Error(err, "unable to list child ReplicaSets")
		return ctrl.Result{}, err
	}

	if len(switchReplicaSets.Items) == 0 {
		if err := r.createExternalResources(ctx, overlay); err != nil {
			log.Error(err, "unable to create ReplicaSet")
			return ctrl.Result{}, err
		}
		log.Info("Overlay Launched")
		return ctrl.Result{RequeueAfter: time.Second * 20}, nil
	}
	// else {

	// 	//b, _ := json.Marshal(netEdgeDevice.Spec.Neighbors)

	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OverlayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.ReplicaSet{}, setOwnerKeyOverlay, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		replicaSet := rawObj.(*appsv1.ReplicaSet)
		owner := metav1.GetControllerOf(replicaSet)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "Overlay" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.ConfigMap{}, setOwnerKeyOverlay, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		configMap := rawObj.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "Overlay" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, setOwnerKeyOverlay, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		service := rawObj.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "Overlay" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &nettypes.NetworkAttachmentDefinition{}, setOwnerKeyOverlay, func(rawObj client.Object) []string {
		// grab the replica set object, extract the owner...
		netAttachDef := rawObj.(*nettypes.NetworkAttachmentDefinition)
		owner := metav1.GetControllerOf(netAttachDef)
		if owner == nil {
			return nil
		}
		// ...make sure it's a ReplicaSet...
		if owner.APIVersion != apiGVStr || owner.Kind != "Overlay" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.Overlay{}).
		Owns(&appsv1.ReplicaSet{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&nettypes.NetworkAttachmentDefinition{}).
		Complete(r)
}

func (r *OverlayReconciler) deleteExternalResources(ctx context.Context, overlay *l2smv1.Overlay) error {
	opts := []client.DeleteAllOfOption{
		client.InNamespace(overlay.Namespace),
		client.MatchingLabels{"overlay": overlay.Name},
	}
	r.Client.DeleteAllOf(ctx, &nettypes.NetworkAttachmentDefinition{}, opts...)
	return nil
}

type OverlayConfigJson struct {
	ControllerIp string `json:"ControllerIp"`
}

type TopologySwitchJson struct {
	Nodes []NodeJson    `json:"Nodes"`
	Links []l2smv1.Link `json:"Links"`
}

type NodeJson struct {
	Name   string `json:"name"`
	NodeIP string `json:"nodeIP"`
}

// createExternalResources is the main orchestrator.
// It calls helpers to build objects, then creates them in the cluster.
func (r *OverlayReconciler) createExternalResources(ctx context.Context, overlay *l2smv1.Overlay) error {

	var extResources []client.Object

	// 1. Build and Create the ConfigMap
	configMap, err := r.buildTopologyConfigMap(overlay)
	if err != nil {
		return fmt.Errorf("failed to build config map: %w", err)
	}
	extResources = append(extResources, configMap)
	// 2. Build and Create Network Attachment Definitions
	netAttachDefs, err := r.buildNetworkAttachmentDefinitions(overlay)
	if err != nil {
		return fmt.Errorf("failed to build net attach defs: %w", err)
	}
	for _, nad := range netAttachDefs {
		extResources = append(extResources, nad)
	}

	// 3. Build and Create Node Resources (ReplicaSets and Services)
	replicaSets, services, err := r.buildNodeResources(overlay, configMap.Name)
	if err != nil {
		return fmt.Errorf("failed to build node resources: %w", err)
	}
	for _, rs := range replicaSets {
		extResources = append(extResources, rs)
	}

	for _, svc := range services {
		extResources = append(extResources, svc)

	}

	if overlay.Spec.Monitor != nil {
		var targets []string
		for _, node := range overlay.Spec.Topology.Nodes {
			// Target the service created in buildNodeResources (port 8090 for lpm-collector)
			switchName := utils.GenerateSwitchName(overlay.Name, node)
			serviceName := utils.GenerateServiceName(switchName)
			// Format: "service-name:8090"
			targets = append(targets, fmt.Sprintf("'%s:8090'", serviceName))
		}

		var lpmStrategy lpminterface.ExporterStrategy

		// Decide strategy
		if overlay.Spec.Monitor.ExportMetrics.Method == l2smv1.SWM_METHOD {
			lpmStrategy = &lpminterface.SWMStrategy{

				Namespace: overlay.Spec.Monitor.ExportMetrics.Config[l2smv1.SWM_NAMESPACE_OPTION],
			}
		} else {
			lpmStrategy = &lpminterface.RegularStrategy{}
		}

		// Execute strategy
		exporterDeployment, exporterConfig, exporterService, err := lpmStrategy.BuildResources(overlay.Spec.Monitor.ExportMetrics.ServiceAccount, targets)
		if err != nil {
			return fmt.Errorf("failed to build monitoring resources: %w", err)
		}
		extResources = append(extResources, exporterDeployment, exporterConfig, exporterService)
	}

	for _, obj := range extResources {
		if err := r.Client.Create(ctx, obj); err != nil {
			return fmt.Errorf("failed to create %s %s: %w", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

// buildTopologyConfigMap constructs the ConfigMap containing topology.json and config.json
func (r *OverlayReconciler) buildTopologyConfigMap(overlay *l2smv1.Overlay) (*corev1.ConfigMap, error) {
	// Construct Topology Object
	topologySwitch := talpav1.Topology{}

	// Populate Nodes
	for _, nodeName := range overlay.Spec.Topology.Nodes {
		node := talpav1.Node{
			Name:   nodeName,
			NodeIP: utils.GenerateServiceName(utils.GenerateSwitchName(overlay.ObjectMeta.Name, nodeName)),
		}
		topologySwitch.Nodes = append(topologySwitch.Nodes, node)
	}

	// Populate Links
	for _, overlayLink := range overlay.Spec.Topology.Links {
		link := talpav1.Link{
			EndpointNodeA: overlayLink.EndpointA,
			EndpointNodeB: overlayLink.EndpointB,
		}
		topologySwitch.Links = append(topologySwitch.Links, link)
	}

	// Construct Settings Object
	overlayConfig := talpav1.Settings{
		ControllerIP:     overlay.Spec.Provider.Domain,
		ControllerPort:   overlay.Spec.Provider.OFPort,
		InterfacesNumber: overlay.Spec.InterfaceNumber,
		ProviderName:     OVERLAY_PROVIDER,
	}

	// Marshal JSONs
	topologyJSON, err := json.Marshal(topologySwitch)
	if err != nil {
		return nil, err
	}
	configJSON, err := json.Marshal(overlayConfig)
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-topology", overlay.Name),
			Namespace: overlay.Namespace,
		},
		Data: map[string]string{
			"topology.json": string(topologyJSON),
			"config.json":   string(configJSON),
		},
	}

	if err := controllerutil.SetControllerReference(overlay, configMap, r.Scheme); err != nil {
		return nil, err
	}
	return configMap, nil
}

// buildNetworkAttachmentDefinitions creates the Multus definitions for the interfaces
func (r *OverlayReconciler) buildNetworkAttachmentDefinitions(overlay *l2smv1.Overlay) ([]*nettypes.NetworkAttachmentDefinition, error) {
	var defs []*nettypes.NetworkAttachmentDefinition

	for i := 1; i <= overlay.Spec.InterfaceNumber; i++ {
		configSpec := fmt.Sprintf(`{
            "cniVersion": "0.3.0",
            "type": "bridge",
            "bridge": "%sbr%d",
            "mtu": 1400,
            "device": "%s-veth%d",
            "ipam": { "type":"static" }
        }`, "", i, overlay.Name, i)

		nad := &nettypes.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-veth%d", overlay.Name, i),
				Namespace: overlay.Namespace,
				Labels:    map[string]string{"app": "l2sm", "overlay": overlay.Name},
			},
			Spec: nettypes.NetworkAttachmentDefinitionSpec{
				Config: configSpec,
			},
		}

		if err := controllerutil.SetControllerReference(overlay, nad, r.Scheme); err != nil {
			return nil, err
		}
		defs = append(defs, nad)
	}
	return defs, nil
}

// buildNodeResources constructs the ReplicaSets and Headless Services for every node in the topology
func (r *OverlayReconciler) buildNodeResources(overlay *l2smv1.Overlay, configMapName string) ([]*appsv1.ReplicaSet, []*corev1.Service, error) {
	var replicaSets []*appsv1.ReplicaSet
	var services []*corev1.Service

	// Prepare Volume Mounts (shared across all containers)
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/etc/l2sm/",
			ReadOnly:  true,
		},
	}

	// Update containers to include the volume mount
	containers := make([]corev1.Container, len(overlay.Spec.SwitchTemplate.Spec.Containers))
	for i, container := range overlay.Spec.SwitchTemplate.Spec.Containers {
		// Deep copy or append to new slice to avoid mutating original spec if shared
		c := container
		c.VolumeMounts = append(c.VolumeMounts, volumeMounts...)
		containers[i] = c
	}

	// Define Volumes
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configMapName},
					Items: []corev1.KeyToPath{
						{Key: "topology.json", Path: "topology.json"},
						{Key: "config.json", Path: "config.json"},
					},
				},
			},
		},
	}

	switchInterfacesAnnotations := GenerateAnnotations(overlay.Name, overlay.Spec.InterfaceNumber)

	for _, node := range overlay.Spec.Topology.Nodes {

		if overlay.Spec.Monitor != nil {
			monCont, monConfMap := r.buildMonitoringResources(overlay)
			containers = append(containers, monCont)
		}
		switchName := utils.GenerateSwitchName(overlay.Name, node)
		serviceName := utils.GenerateServiceName(switchName)

		// 1. Create ReplicaSet
		replicaSet := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:        switchName,
				Namespace:   overlay.Namespace,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
			},
			Spec: appsv1.ReplicaSetSpec{
				Replicas: utils.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": switchName},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": switchName},
						Annotations: map[string]string{
							MULTUS_ANNOTATION_KEY: switchInterfacesAnnotations,
						},
					},
					Spec: corev1.PodSpec{
						InitContainers: overlay.Spec.SwitchTemplate.Spec.InitContainers,
						Containers:     containers,
						Volumes:        volumes,
						HostNetwork:    overlay.Spec.SwitchTemplate.Spec.HostNetwork,
						NodeSelector:   map[string]string{corev1.LabelHostname: node},
						Tolerations:    []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
					},
				},
			},
		}

		// Apply Template Annotations/Labels
		for k, v := range overlay.Spec.SwitchTemplate.Annotations {
			replicaSet.Annotations[k] = v
		}
		for k, v := range overlay.Spec.SwitchTemplate.Labels {
			replicaSet.Labels[k] = v
		}

		if err := controllerutil.SetControllerReference(overlay, replicaSet, r.Scheme); err != nil {
			return nil, nil, err
		}
		replicaSets = append(replicaSets, replicaSet)

		// 2. Create Service
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: overlay.Namespace,
				Labels:    map[string]string{"app": switchName},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "None",
				Selector:  map[string]string{"app": switchName},
				Ports: []corev1.ServicePort{
					{Name: "http", Port: 80},
				},
			},
		}

		if err := controllerutil.SetControllerReference(overlay, service, r.Scheme); err != nil {
			return nil, nil, err
		}
		services = append(services, service)
	}

	return replicaSets, services, nil
}
