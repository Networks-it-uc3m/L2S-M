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

// QuarantinePodSelector identifies pods attached to an L2Network.
type QuarantinePodSelector struct {
	// PodLabelSelector selects the pods to move.
	// +required
	PodLabelSelector metav1.LabelSelector `json:"podLabelSelector"`

	// L2NetworkSelector selects the source L2Network where matching pods must
	// currently be attached.
	// +required
	L2NetworkSelector metav1.LabelSelector `json:"l2NetworkSelector"`
}

// QuarantinePodRequestSpec defines the desired state of QuarantinePodRequest
type QuarantinePodRequestSpec struct {
	// Selector identifies the pods to quarantine and the L2Network they are
	// currently attached to.
	// +required
	Selector QuarantinePodSelector `json:"selector"`

	// TargetL2Network names the L2Network where selected pods should be moved.
	// +required
	// +kubebuilder:validation:MinLength=1
	TargetL2Network string `json:"targetL2Network"`
}

// QuarantinePodRequestStatus defines the observed state of QuarantinePodRequest.
type QuarantinePodRequestStatus struct {
	// ObservedGeneration is the most recent generation reconciled by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// SourceL2NetworkName is the source L2Network selected by the request.
	// +optional
	SourceL2NetworkName string `json:"sourceL2NetworkName,omitempty"`

	// TargetL2NetworkName is the target L2Network used by the request.
	// +optional
	TargetL2NetworkName string `json:"targetL2NetworkName,omitempty"`

	// MatchedPodCount is the number of pods matching the request selector.
	// +optional
	MatchedPodCount int32 `json:"matchedPodCount,omitempty"`

	// MovedPodCount is the number of pods moved to the target L2Network.
	// +optional
	MovedPodCount int32 `json:"movedPodCount,omitempty"`

	// conditions represent the current state of the QuarantinePodRequest resource.
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

// QuarantinePodRequest is the Schema for the quarantinepodrequests API
type QuarantinePodRequest struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of QuarantinePodRequest
	// +required
	Spec QuarantinePodRequestSpec `json:"spec"`

	// status defines the observed state of QuarantinePodRequest
	// +optional
	Status QuarantinePodRequestStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// QuarantinePodRequestList contains a list of QuarantinePodRequest
type QuarantinePodRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []QuarantinePodRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&QuarantinePodRequest{}, &QuarantinePodRequestList{})
}
