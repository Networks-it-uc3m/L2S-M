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

// NetworkType represents the type of network being configured.
// +kubebuilder:validation:Enum=ext-vnet;vnet;vlink
// +kubebuilder:pruning:PreserveUnknownFields
type NetworkType string

const (
	NetworkTypeExtVnet NetworkType = "ext-vnet"
	NetworkTypeVnet    NetworkType = "vnet"
	NetworkTypeVlink   NetworkType = "vlink"
)

// +kubebuilder:validation:Enum=Available;Unavailable;Unknown
type ConnectivityStatus string

const (
	OnlineStatus  ConnectivityStatus = "Available"
	OfflineStatus ConnectivityStatus = "Unavailable"
	UnknownStatus ConnectivityStatus = "Unknown"
)

// ProviderSpec defines the provider's name and domain. This is used in the inter-cluster scenario, to allow managing of the network in the external environment by this certified SDN provider.
type ProviderSpec struct {
	Name   string   `json:"name"`
	Domain []string `json:"domain"`

	//+kubebuilder:default:value="30808"
	SDNPort string `json:"sdnPort,omitempty"`

	// DNS service configuration
	//+kubebuilder:default:="30053"
	// DNS protocol port (used for DNS queries via tools like dig)
	DNSPort string `json:"dnsPort,omitempty"`

	//+kubebuilder:default:="30818"
	// gRPC management port for DNS service (used for adding/modifying DNS entries)
	DNSGRPCPort string `json:"dnsGrpcPort,omitempty"`

	//+kubebuilder:default:value="6633"
	OFPort string `json:"ofPort,omitempty"`
}

// IDSRuleSource defines where the IDS should fetch signatures from.
type IDSRuleSource struct {
	// Name is a friendly identifier for this rule set
	Name string `json:"name"`

	// URL allows fetching a remote ruleset (e.g., specific version of ET Open).
	// The controller will need logic to download and cache this.
	// +optional
	URL string `json:"url,omitempty"`

	// Inline allows the user to paste a raw Suricata rule directly in the YAML.
	// Example: "alert tcp any any -> any 80 (msg:\"test rule\"; sid:100001; rev:1;)"
	// +optional
	Inline string `json:"inline,omitempty"`

	// ConfigMapRef allows loading rules from a Kubernetes ConfigMap.
	// Useful for large, organization-specific rule sets managed separately.
	// +optional
	ConfigMapRef *corev1.LocalObjectReference `json:"configMapRef,omitempty"`
}

// IdsRules defines the configuration for the Intrusion Detection System
type IdsRules struct {
	// Enabled allows toggling the IDS on/off without deleting the config.
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// HomeNetCIDR overrides the SURICATA $HOME_NET variable.
	// If empty, the controller should default this to the L2Network's NetworkCIDR.
	// +optional
	HomeNetCIDR []string `json:"homeNetCidr,omitempty"`

	// UseEmergingThreatsOpen is a high-level boolean to automatically enable
	// the standard "ET Open" ruleset without needing to manually configure URLs.
	// +kubebuilder:default:=true
	UseEmergingThreatsOpen bool `json:"useEmergingThreatsOpen"`

	// CustomRuleSources allows adding specific rule files or inline rules.
	// +optional
	CustomRuleSources []IDSRuleSource `json:"customRuleSources,omitempty"`

	// IgnorePorts allows whitelisting specific traffic flow from inspection
	// to improve performance or reduce false positives (BPF Filter generation).
	// +optional
	IgnorePorts []int32 `json:"ignorePorts,omitempty"`
}

// L2NetworkSpec defines the desired state of L2Network
type L2NetworkSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NetworkType represents the type of network being configured.
	Type NetworkType `json:"type"`

	// Config is an optional field that is meant to be used as additional configuration depending on the type of network. Check each type of network for specific configuration definitions.
	Config *string `json:"config,omitempty"`

	// Provider is an optional field representing a provider spec. Check the provider spec definition for more details
	Provider *ProviderSpec `json:"provider,omitempty"`

	// NetworkCIDR defines the overall network CIDR used for routing pod interfaces.
	// This value represents the broader network segment that encompasses all pod IPs,
	// e.g. 10.101.0.0/16.
	NetworkCIDR string `json:"networkCIDR,omitempty"`

	// PodAddressRange specifies the specific pool of IP addresses that can be assigned to pods.
	// This range should be a subset of the overall network CIDR, e.g. 10.101.2.0/24.
	PodAddressRange string `json:"podAddressRange,omitempty"`

	// Ids configures the intrusion detection system.
	// +optional
	Ids *IdsRules `json:"ids,omitempty"`
}

// L2NetworkStatus defines the observed state of L2Network
type L2NetworkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=0
	ConnectedPodCount int `json:"connectedPodCount,omitempty"`
	// Last assigned IP, used for sequential allocation
	LastAssignedIP string `json:"lastAssignedIP,omitempty"`

	// Existing Pods in the network
	AssignedIPs map[string]string `json:"assignedIPs,omitempty"`

	// Status of the connectivity to the internal SDN Controller. If there is no connection, internal l2sm-switches won't forward traffic
	// +kubebuilder:default=Unavailable
	InternalConnectivity *ConnectivityStatus `json:"internalConnectivity"`

	// Status of the connectivity to the external provider SDN Controller. If there is no connectivity, the exisitng l2sm-ned in the cluster won't forward packages to the external clusters.
	ProviderConnectivity *ConnectivityStatus `json:"providerConnectivity,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AVAILABILITY",type="string",JSONPath=".status.internalConnectivity",description="Internal SDN Controller Connectivity"
// +kubebuilder:printcolumn:name="CONNECTED_PODS",type="integer",JSONPath=".status.connectedPodCount",description="Number of pods in the network"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// L2Network is the Schema for the l2networks API
type L2Network struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   L2NetworkSpec   `json:"spec,omitempty"`
	Status L2NetworkStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// L2NetworkList contains a list of L2Network
type L2NetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []L2Network `json:"items"`
}

func init() {
	SchemeBuilder.Register(&L2Network{}, &L2NetworkList{})
}
