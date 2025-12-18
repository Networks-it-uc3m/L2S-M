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
