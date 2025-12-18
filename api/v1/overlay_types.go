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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Link represents a bidirectional connection between two nodes in the topology.
type Link struct {
	// EndpointA is the name of the first node in the link.
	EndpointA string `json:"endpointA"`
	// EndpointB is the name of the second node in the link.
	EndpointB string `json:"endpointB"`
}

// TopologySpec defines the physical or logical structure of the network.
type TopologySpec struct {
	// Nodes is a list of node names where switches will be deployed.
	Nodes []string `json:"nodes"`
	// Links is a list of connections between the defined nodes.
	Links []Link `json:"links,omitempty"`
}

// ConfigMapKeySelector selects a key from a ConfigMap.
type ConfigMapKeySelector struct {
	// Name of the ConfigMap.
	Name string `json:"name"`
	// Key within the ConfigMap that contains the script (e.g., "measure.sh").
	Key string `json:"key"`
}

// Metric defines a specific network measurement task.
type MetricSpec struct {
	// Name identifies the metric.
	// Reserved names: "rtt", "jitter", "throughput".
	// If a reserved name is used, the internal Go implementation is used by default.
	// If a custom name is used, 'scriptSource' is required.
	Name string `json:"name"`

	// Interval specifies the time in minutes between measurements.
	// If not set (nil), the metric runs in "continuous mode", consuming the live stream of the measurement tool.
	// +optional
	Interval *int `json:"interval,omitempty"`

	// ScriptSource points to a ConfigMap containing a shell script to execute for this metric.
	// If provided, this script overrides the internal implementation (even for reserved names).
	// The script must print the measurement value to stdout.
	// +optional
	ScriptSource *ConfigMapKeySelector `json:"scriptSource,omitempty"`
}

// MonitorSpec configures the L2S-M Performance Measurement module.
type MonitorSpec struct {
	// Metrics is the list of measurements to perform on the overlay network.
	// Supports built-in metrics (rtt, jitter, throughput) and custom script-based metrics.
	Metrics []MetricSpec `json:"metrics"`

	// SpreadFactor determines how metric execution is distributed over time to avoid congestion.
	// A higher value spreads execution more widely.
	// +kubebuilder:default:="0.2"
	SpreadFactor string `json:"spreadFactor,omitempty"`
}

// OverlaySpec defines the desired state of Overlay
type OverlaySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The SDN Controller that manages the overlay network. Must specify a domain and a name.
	Provider *ProviderSpec `json:"provider"`

	// Topology represents the desired topology, it's represented by the 'Nodes' field, a list of nodes where the switches are going to be deployed and a list of bidirectional links,
	// selecting the nodes that are going to be linked.
	Topology *TopologySpec `json:"topology,omitempty"`

	// Template describes the virtual switch pod that will be created.
	SwitchTemplate *SwitchTemplateSpec `json:"switchTemplate"`

	// Interface number specifies how many interfaces the switch should have predefined (if used with multus)
	//+kubebuilder:default:value=10
	InterfaceNumber int `json:"interfaceNumber,omitempty"`

	// Monitor enables the performance measurement probing mechanism.
	// If omitted, no metrics are collected.
	// +optional
	Monitor *MonitorSpec `json:"monitor,omitempty"`
}

// MetricValue holds the latest measurement for a specific metric.
type MetricValue struct {
	// Name of the metric (e.g., "rtt", "jitter", "custom-loss").
	Name string `json:"name"`

	// Value is the latest measurement as a float (e.g., 12.5).
	Value string `json:"value"`
}

// LinkStatus defines the observed state of a specific link between two nodes.
type LinkStatus struct {
	// SourceNode is the name of the node where the measurement originated.
	SourceNode string `json:"sourceNode"`

	// TargetNode is the name of the neighbor node being measured.
	TargetNode string `json:"targetNode"`

	// Status indicates if the link is "Up", "Down", or "Degraded".
	Status string `json:"status"`

	// Metrics contains the list of latest measurements for this link.
	Metrics []MetricValue `json:"metrics,omitempty"`
}

// OverlayStatus defines the observed state of Overlay
type OverlayStatus struct {
	// LinkMetrics holds the performance data for every monitored link.
	// +optional
	LinkMetrics *[]LinkStatus `json:"linkMetrics,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Overlay is the Schema for the overlays API
type Overlay struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OverlaySpec   `json:"spec,omitempty"`
	Status OverlayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OverlayList contains a list of Overlay
type OverlayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Overlay `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Overlay{}, &OverlayList{})
}
