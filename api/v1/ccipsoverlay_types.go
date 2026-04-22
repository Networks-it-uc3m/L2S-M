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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Link represents a bidirectional connection between two nodes in the topology.
type Tunnel struct {
	// EndpointA is the name of the first node in the link.
	EndpointA string `json:"endpointA"`
	// EndpointB is the name of the second node in the link.
	EndpointB string `json:"endpointB"`
}

// CcipsOverlaySpec defines the desired state of CcipsOverlay
type CcipsOverlaySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of CcipsOverlay. Edit ccipsoverlay_types.go to remove/update
	// +optional
	// Foo   *string `json:"foo,omitempty"`
	// Node1 *string `json:"node1"`
	// Node2 *string `json:"node2"`

	ControllerIP string `json:"controllerIp"`

	Nodes []string `json:"nodes"`
	// Links is a list of connections between the defined nodes.
	Tunnels []Tunnel `json:"tunnels"`
}

// CcipsOverlayStatus defines the observed state of CcipsOverlay.
type CcipsOverlayStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the CcipsOverlay resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CcipsOverlay is the Schema for the ccipsoverlays API
type CcipsOverlay struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of CcipsOverlay
	// +required
	Spec CcipsOverlaySpec `json:"spec"`

	// status defines the observed state of CcipsOverlay
	// +optional
	Status CcipsOverlayStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CcipsOverlayList contains a list of CcipsOverlay
type CcipsOverlayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []CcipsOverlay `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CcipsOverlay{}, &CcipsOverlayList{})
}
