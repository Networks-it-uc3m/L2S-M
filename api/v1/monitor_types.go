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

	ExportMetrics *ExportMetricSpec `json:"exportMetric"`

	NetworkCIDR *string `json:"networkCIDR,omitempty"`
}

const SWM_METHOD = "codeco-swm"
const SWM_NT_NAMESPACE_OPTION = "nt_namespace"

type ExportMetricSpec struct {
	// Method to export the metrics. Reserved names include: "codeco-swm", which includes an interface for the
	// codeco swm crd. Interface must be implemented
	// for the method, so this must be designed beforehand.
	//+kubebuilder:default="default"
	Method string `json:"method,omitempty"`

	//+kubebuilder:default:="default"
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// Additional configuration parameters that may be set by developer to implement different kinds
	// of flexible key-value pairs. In the case of the codeco-swm, "namespace" is included.
	Config map[string]string `json:"config,omitempty"`
}
