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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type NeighborSpec struct {

	// Name of the cluster the link is going to be made upon.
	Node string `json:"node"`

	// Domain where the neighbor's NED switch can be reached at. Must be a valid IP Address or Domain name, reachable from the node the NED
	// is going to be deployed at.
	Domain string `json:"domain"`
}

type SwitchPodSpec struct {
	// List of volumes that can be mounted by containers belonging to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	// +listType=map
	// +listMapKey=name
	Volumes []corev1.Volume `json:"volumes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=volumes"`
	// List of initialization containers belonging to the pod.
	// Init containers are executed in order prior to containers being started. If any
	// init container fails, the pod is considered to have failed and is handled according
	// to its restartPolicy. The name for an init container or normal container must be
	// unique among all containers.
	// Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.
	// The resourceRequirements of an init container are taken into account during scheduling
	// by finding the highest request/limit for each resource type, and then using the max of
	// of that value or the sum of the normal containers. Limits are applied to init containers
	// in a similar fashion.
	// Init containers cannot currently be added or removed.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	InitContainers []corev1.Container `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,20,rep,name=initContainers"`
	// List of containers belonging to the pod.
	// Containers cannot currently be added or removed.
	// There must be at least one container in a Pod.
	// Cannot be updated.
	// +patchMergeKey=name
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=name
	Containers []corev1.Container `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`
	// Host networking requested for this pod. Use the host's network namespace.
	// If this option is set, the ports that will be used must be specified.
	// Default to false.
	// +k8s:conversion-gen=false
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty" protobuf:"varint,11,opt,name=hostNetwork"`
}
type SwitchTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec SwitchPodSpec `json:"spec,omitempty"`
}

type NodeConfigSpec struct {
	NodeName string `json:"nodeName"`

	IPAddress string `json:"ipAddress"`
}

// NetworkEdgeDeviceSpec defines the desired state of NetworkEdgeDevice
type NetworkEdgeDeviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The SDN Controller that manages the overlay network. Must specify a domain and a name.
	Provider *ProviderSpec `json:"provider"`

	// Node Configuration
	NodeConfig *NodeConfigSpec `json:"nodeConfig"`

	// Field exclusive to the multi-domain overlay type. If specified in other  types of overlays, the reosurce will launch an error and won't be created.
	Neighbors []NeighborSpec `json:"neighbors,omitempty"`

	// Template describes the virtual switch pod that will be created.
	SwitchTemplate *SwitchTemplateSpec `json:"switchTemplate"`

	// Available pod range. The pod specified will run a local grpc server and the next one will be used for the VXLAN creation
}

// NetworkEdgeDeviceStatus defines the observed state of NetworkEdgeDevice
type NetworkEdgeDeviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status of the overlay. Is available when switches are connected between them and with the network Controller.
	// +kubebuilder:default=Unavailable
	Availability *ConnectivityStatus `json:"availability"`

	ConnectedNeighbors []NeighborSpec `json:"connectedNeighbors,omitempty"`

	OpenflowId string `json:"openflowId,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NetworkEdgeDevice is the Schema for the networkedgedevices API
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.availability",description="Availability status of the overlay"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
type NetworkEdgeDevice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkEdgeDeviceSpec   `json:"spec,omitempty"`
	Status NetworkEdgeDeviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NetworkEdgeDeviceList contains a list of NetworkEdgeDevice
type NetworkEdgeDeviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkEdgeDevice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkEdgeDevice{}, &NetworkEdgeDeviceList{})
}
