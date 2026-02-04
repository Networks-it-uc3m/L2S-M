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

package lpminterface

import (
	"fmt"
	"strconv"

	"encoding/json"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	lpmv1 "github.com/Networks-it-uc3m/LPM/api/v1"
	talpav1 "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultCollectorPort          = 8090
	defaultRTTIntervalSeconds     = 10
	defaultThroughputIntervalSecs = 20
	defaultJitterIntervalSeconds  = 5
	defaultSpreadfactor           = "0.2"
	defaultIPStart                = 2
	defaultNetworkCIDR            = "10.0.0.0/24"
	collectorConfigKey            = "config.json"
	collectorVolumeName           = "lpm-collector-config"
	lpmImage                      = "alexdecb/lpm"
	lpmVersion                    = "1.2.3"
	collectorMountedCfgName       = "lpm-config.json"
	collectorMountPath            = "/etc/lpm/lpm-config.json"
)

// ExporterStrategy defines how to build the resources
type ExporterStrategy interface {
	BuildResources(saName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error)
}

func NewExporter(m, ns string, o map[string]string) ExporterStrategy {
	if m == l2smv1.SWM_METHOD {
		// todo: nt is created in ns for some reason, check what is happening
		return &swmStrategy{Namespace: ns, NetworkTopologyNamespace: o[l2smv1.SWM_NT_NAMESPACE_OPTION]}
	}
	return &regularStrategy{Namespace: ns}
}

// SWMStrategy implements the SWM logic
type swmStrategy struct {
	Namespace                string
	NetworkTopologyNamespace string
}

func (s *swmStrategy) BuildResources(saName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error) {
	// Call internal logic for SWM
	return s.buildSWMExporterInternal(saName, s.Namespace, "swm-lpm", targets)
}

// RegularStrategy implements the default logic
type regularStrategy struct {
	Namespace string
}

func (s *regularStrategy) BuildResources(saName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error) {
	// Call internal logic for Regular
	return s.buildRegularExporterInternal(saName, "lpm", targets)
}

// CollectorBuildOptions controls address/interval defaults and image settings.
type CollectorBuildOptions struct {
	// IPs are computed as: fmt.Sprintf("%s%d", IPPrefix, IPStart+index)
	// Example: prefix "10.0.0.", start 2 => index0=10.0.0.2, index1=10.0.0.3 ...
	NetworkCIDR *string
	IpCidr      *string
	IPStart     *int

	SpreadFactor             *string
	RTTIntervalSeconds       *int
	ThroughputIntervalSecs   *int
	JitterIntervalSeconds    *int
	CollectorImage           *string
	CollectorImagePullPolicy *corev1.PullPolicy
	CollectorName            *string
}

func GenerateLPMPorts(nodes []string, overlayProvider string) []string {

	fmt.Println(len(nodes))
	fmt.Println(nodes)
	lpmPorts := make([]string, 0, len(nodes)) // len=0, cap=len(nodes)
	for _, n := range nodes {
		p := fmt.Sprintf("of:%s/%d", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: n, ProviderName: overlayProvider})), talpav1.RESERVED_PROBE_ID)
		lpmPorts = append(lpmPorts, p)
	}
	fmt.Println(len(lpmPorts))
	fmt.Println(lpmPorts)
	return lpmPorts
}

// BuildMonitoringCollectorResources builds:
//  1. the collector sidecar container spec (volume mount included)
//  2. a list of ConfigMaps (one per node) with LPM NodeConfig encoded in config.json
//  3. a lookup map nodeName -> configMapName to make RS loop integration trivial
func BuildMonitoringCollectorResources(
	overlay *l2smv1.Overlay,
	opts CollectorBuildOptions,
) (*corev1.Container, []*corev1.ConfigMap, error) {

	if overlay == nil {
		return nil, nil, fmt.Errorf("overlay is nil")
	}
	if len(overlay.Spec.Topology.Nodes) == 0 {
		return nil, nil, fmt.Errorf("overlay topology has no nodes")
	}

	if opts.NetworkCIDR == nil || *opts.NetworkCIDR == "" {
		v := defaultNetworkCIDR
		opts.NetworkCIDR = &v
	}
	if opts.IPStart == nil {
		v := defaultIPStart
		opts.IPStart = &v
	}

	if opts.RTTIntervalSeconds == nil {
		v := defaultRTTIntervalSeconds
		opts.RTTIntervalSeconds = &v
	}
	if opts.ThroughputIntervalSecs == nil {
		v := defaultThroughputIntervalSecs
		opts.ThroughputIntervalSecs = &v
	}
	if opts.JitterIntervalSeconds == nil {
		v := defaultJitterIntervalSeconds
		opts.JitterIntervalSeconds = &v
	}

	if opts.CollectorName == nil || *opts.CollectorName == "" {
		v := "lpm-collector"
		opts.CollectorName = &v
	}

	if opts.CollectorImagePullPolicy == nil {
		v := corev1.PullIfNotPresent
		opts.CollectorImagePullPolicy = &v
	}

	if opts.CollectorImage == nil || *opts.CollectorImage == "" {
		v := fmt.Sprintf("%s:%s", lpmImage, lpmVersion)

		opts.CollectorImage = &v
	}

	if opts.SpreadFactor == nil || *opts.SpreadFactor == "" {
		v := defaultSpreadfactor
		opts.SpreadFactor = &v
	}
	nodes := overlay.Spec.Topology.Nodes

	allocated, mask, err := utils.AllocateIPv4s(*opts.NetworkCIDR, *opts.IPStart, len(nodes))
	if err != nil {
		return nil, nil, fmt.Errorf("monitoring CIDR allocation failed: %w", err)
	}

	// Map node -> IP by index (stable ordering depends on overlay.Spec.Topology.Nodes order)
	nodeIP := make(map[string]string, len(nodes))
	for i, n := range nodes {
		nodeIP[n] = allocated[i]
	}

	var configMaps []*corev1.ConfigMap

	sf, err := strconv.ParseFloat(*opts.SpreadFactor, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("spread factor not inputted as float64, please input correct field in crd.")
	}
	for _, node := range nodes {
		// Build neighbour metrics list: all nodes except self
		neigh := make([]lpmv1.MetricConfiguration, 0, len(nodes)-1)
		for _, other := range nodes {
			if other == node {
				continue
			}
			neigh = append(neigh, lpmv1.MetricConfiguration{
				Name:       other,
				IP:         nodeIP[other],
				RTT:        *opts.RTTIntervalSeconds,
				Throughput: *opts.ThroughputIntervalSecs,
				Jitter:     *opts.JitterIntervalSeconds,
			})
		}

		switchName := utils.GenerateSwitchName(overlay.Name, node, utils.SlicePacketSwitch)
		// Use LPM API types as requested
		cfg := lpmv1.NodeConfig{
			NodeName:              node,
			IpAddress:             fmt.Sprintf("%s%s", nodeIP[node], mask),
			MetricsNeighbourNodes: neigh,
			SpreadFactor:          sf,
		}

		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, nil, fmt.Errorf("marshal node config for %q: %w", node, err)
		}

		cmName := GenerateConfigmapName(utils.GenerateReplicaSetName(switchName))

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: overlay.Namespace,
				Labels: map[string]string{
					"app":     "l2sm",
					"overlay": overlay.Name,
				},
			},
			Data: map[string]string{
				collectorConfigKey: string(b),
			},
		}

		configMaps = append(configMaps, cm)
	}

	// Collector sidecar container: mounts /etc/l2sm/lpm-conf.json (SubPath) from a CM volume.
	// The volume itself must be attached per-ReplicaSet (because CM differs per node).
	collectorContainer := &corev1.Container{
		Name:            *opts.CollectorName,
		Image:           *opts.CollectorImage,
		ImagePullPolicy: *opts.CollectorImagePullPolicy,
		Args: []string{
			"collector",
			fmt.Sprintf("--config_file=%s", collectorMountPath),
		},
		Ports: []corev1.ContainerPort{
			{ContainerPort: defaultCollectorPort, Name: "lpm"},
		}, VolumeMounts: []corev1.VolumeMount{
			{
				Name:      collectorVolumeName,
				MountPath: collectorMountPath,
				SubPath:   collectorMountedCfgName,
			},
		},
	}

	return collectorContainer, configMaps, nil
}
func GenerateConfigmapName(rsName string) string {
	return fmt.Sprintf("%s-config", rsName)
}

func AddLPMConfigMapToSps(ps *corev1.PodSpec) {
	for i := range ps.Containers {
		if ps.Containers[i].Name != "l2sm-switch" {
			continue
		}

		for _, vm := range ps.Containers[i].VolumeMounts {
			if vm.MountPath == collectorMountPath {
				return
			}
		}

		ps.Containers[i].Args = append(ps.Containers[i].Args, "--monitor_file", collectorMountPath)
		ps.Containers[i].VolumeMounts = append(ps.Containers[i].VolumeMounts, corev1.VolumeMount{
			Name:      collectorVolumeName,
			MountPath: collectorMountPath,
			SubPath:   collectorMountedCfgName,
		})
		return
	}
}

// AttachCollectorConfigToReplicaSet patches the Pod template with the right per-node CM.
// Call this inside your ReplicaSet loop (once you know the node and its cmName).
func AttachCollectorConfigToReplicaSet(ps *corev1.PodSpec, rsName string) error {
	if ps == nil || rsName == "" {
		return fmt.Errorf("replicaset fields are empty, could not attach resource")
	}
	cmName := GenerateConfigmapName(rsName)
	// Ensure volume exists / updated
	vol := corev1.Volume{
		Name: collectorVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Items: []corev1.KeyToPath{
					{Key: collectorConfigKey, Path: collectorMountedCfgName},
				},
			},
		},
	}

	// Upsert volume by name
	found := false
	for i := range ps.Volumes {
		if ps.Volumes[i].Name == collectorVolumeName {
			ps.Volumes[i] = vol
			found = true
			break
		}
	}
	if !found {
		ps.Volumes = append(ps.Volumes, vol)
	}
	return nil
}

func (exp *swmStrategy) buildSWMExporterInternal(serviceAccount, networkTopologyNamespace, exporterName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error) {

	appName := fmt.Sprintf("prometheus-%s", exporterName)

	// 1. Generate Prometheus Config (Scrape Targets)

	// Join targets for YAML array
	targetsJson, _ := json.Marshal(targets) // simple trick to format ["a","b"] correctly

	promConfigContent := fmt.Sprintf(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: %s`, string(targetsJson))

	// 2. Create ConfigMap
	cmName := fmt.Sprintf("prometheus-config-%s", exporterName)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: exp.Namespace,
			Labels:    map[string]string{"app": appName},
		},
		Data: map[string]string{
			"prometheus.yml": promConfigContent,
		},
	}

	// 3. Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: exp.Namespace,
			Labels:    map[string]string{"app": appName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount,
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: "prom/prometheus:v2.30.3",
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus",
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9090},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/prometheus"},
								{Name: "data-volume", MountPath: "/prometheus"},
							},
						},
						{
							Name:            "exporter",
							Image:           fmt.Sprintf("%s:%s", lpmImage, lpmVersion),
							ImagePullPolicy: corev1.PullAlways,
							Args: []string{
								"exporter",
								"--target",
								"swm",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "TOPOLOGY_NAMESPACE",
									Value: networkTopologyNamespace,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// 4. Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus-lpm-network",
			Namespace: exp.Namespace,
			Labels:    map[string]string{"app": appName},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": appName},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     9090,
				},
			},
		},
	}

	return deployment, configMap, service, nil
}

func (exp *regularStrategy) buildRegularExporterInternal(serviceAccount, exporterName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error) {
	appName := fmt.Sprintf("prometheus-%s", exporterName)

	// 1. Generate Prometheus Config (Scrape Targets)

	// Join targets for YAML array
	targetsJson, _ := json.Marshal(targets) // simple trick to format ["a","b"] correctly

	promConfigContent := fmt.Sprintf(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: %s`, string(targetsJson))

	// 2. Create ConfigMap
	cmName := fmt.Sprintf("prometheus-config-%s", exporterName)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: exp.Namespace,
			Labels:    map[string]string{"app": appName},
		},
		Data: map[string]string{
			"prometheus.yml": promConfigContent,
		},
	}

	// 3. Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Labels:    map[string]string{"app": appName},
			Namespace: exp.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount,
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: "prom/prometheus:v2.30.3",
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus",
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9090},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/prometheus"},
								{Name: "data-volume", MountPath: "/prometheus"},
							},
						},
						{
							Name:            "exporter",
							Image:           fmt.Sprintf("%s:%s", lpmImage, lpmVersion),
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"exporter"},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// 4. Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Labels:    map[string]string{"app": appName},
			Namespace: exp.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": appName},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     9090,
				},
			},
		},
	}

	return deployment, configMap, service, nil
}

func BuildNEDMonitoringResources(
	ned *l2smv1.NetworkEdgeDevice,
	opts CollectorBuildOptions,
) (*corev1.Container, []*corev1.ConfigMap, error) {

	if ned == nil {
		return nil, nil, fmt.Errorf("ned is nil")
	}

	if opts.IpCidr == nil || *opts.IpCidr == "" {
		return nil, nil, fmt.Errorf("monitoring set, but no ip specified for ned probe interface")
	}

	if opts.RTTIntervalSeconds == nil {
		v := defaultRTTIntervalSeconds
		opts.RTTIntervalSeconds = &v
	}
	if opts.ThroughputIntervalSecs == nil {
		v := defaultThroughputIntervalSecs
		opts.ThroughputIntervalSecs = &v
	}
	if opts.JitterIntervalSeconds == nil {
		v := defaultJitterIntervalSeconds
		opts.JitterIntervalSeconds = &v
	}

	if opts.CollectorName == nil || *opts.CollectorName == "" {
		v := "lpm-collector"
		opts.CollectorName = &v
	}

	if opts.CollectorImagePullPolicy == nil {
		v := corev1.PullIfNotPresent
		opts.CollectorImagePullPolicy = &v
	}

	if opts.CollectorImage == nil || *opts.CollectorImage == "" {
		v := fmt.Sprintf("%s:%s", lpmImage, lpmVersion)

		opts.CollectorImage = &v
	}

	if opts.SpreadFactor == nil || *opts.SpreadFactor == "" {
		v := defaultSpreadfactor
		opts.SpreadFactor = &v
	}
	neighs := ned.Spec.Neighbors

	var configMaps []*corev1.ConfigMap

	sf, err := strconv.ParseFloat(*opts.SpreadFactor, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("spread factor not inputted as float64, please input correct field in crd.")
	}
	var conf []lpmv1.MetricConfiguration
	for _, neigh := range neighs {

		conf = append(conf, lpmv1.MetricConfiguration{
			Name:       neigh.Node,
			IP:         *neigh.LpmIp,
			RTT:        *opts.RTTIntervalSeconds,
			Throughput: *opts.ThroughputIntervalSecs,
			Jitter:     *opts.JitterIntervalSeconds,
		})
	}

	switchName := utils.GenerateSwitchName(ned.Name, ned.Spec.NodeConfig.NodeName, utils.NetworkEdgeDevice)
	// Use LPM API types as requested
	cfg := lpmv1.NodeConfig{
		NodeName:              ned.Spec.NodeConfig.NodeName,
		IpAddress:             *opts.IpCidr,
		MetricsNeighbourNodes: conf,
		SpreadFactor:          sf,
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal node config for ned %q: %w", ned.Name, err)
	}

	cmName := GenerateConfigmapName(utils.GenerateReplicaSetName(switchName))

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: ned.Namespace,
			Labels: map[string]string{
				"app": "l2sm",
				"ned": ned.Name,
			},
		},
		Data: map[string]string{
			collectorConfigKey: string(b),
		},
	}

	configMaps = append(configMaps, cm)

	// Collector sidecar container: mounts /etc/l2sm/lpm-conf.json (SubPath) from a CM volume.
	// The volume itself must be attached per-ReplicaSet (because CM differs per node).
	collectorContainer := &corev1.Container{
		Name:            *opts.CollectorName,
		Image:           *opts.CollectorImage,
		ImagePullPolicy: *opts.CollectorImagePullPolicy,
		Args: []string{
			"collector",
			fmt.Sprintf("--config_file=%s", collectorMountPath),
		},
		Ports: []corev1.ContainerPort{
			{ContainerPort: defaultCollectorPort, Name: "lpm"},
		}, VolumeMounts: []corev1.VolumeMount{
			{
				Name:      collectorVolumeName,
				MountPath: collectorMountPath,
				SubPath:   collectorMountedCfgName,
			},
		},
	}

	return collectorContainer, configMaps, nil
}
