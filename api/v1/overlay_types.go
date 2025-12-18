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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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
}

// OverlayStatus defines the observed state of Overlay
type OverlayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ConnectedNeighbors []NeighborSpec `json:"connectedNeighbors,omitempty"`
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
