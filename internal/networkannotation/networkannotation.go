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

package networkannotation

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
)

const (
	MULTUS_ANNOTATION_KEY   = "k8s.v1.cni.cncf.io/networks"
	NET_ATTACH_LABEL_PREFIX = "used-"
	L2SM_NETWORK_ANNOTATION = "l2sm/networks"
)

type NetworkAnnotation struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace,omitempty"`
	IPAddresses []string `json:"ips,omitempty"`
	IfName      string   `json:"ifname,omitempty"`
}

func MultusAnnotationToString(multusAnnotations []NetworkAnnotation) string {
	jsonData, err := json.Marshal(multusAnnotations)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func ExtractNetworks(annotations, namespace string) ([]NetworkAnnotation, error) {

	var networks []NetworkAnnotation
	err := json.Unmarshal([]byte(annotations), &networks)
	if err != nil {
		// If unmarshalling fails, treat as comma-separated list
		names := strings.Split(annotations, ",")

		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				networks = append(networks, NetworkAnnotation{Name: name})
			}
		}
	}

	// Iterate over the networks to add the namespace
	for i := range networks {
		// if len(networks[i].IPAddresses) == 0 {
		// 	// Call GenerateIPv6Address if IPAddresses are missing
		// 	networks[i].GenerateIPv6Address()
		// }
		networks[i].Namespace = namespace
	}
	return networks, nil
}

func (network *NetworkAnnotation) GenerateIPv6Address() {

	// Generating the interface ID (64 bits)
	interfaceID := rand.Uint64()

	// Formatting to a 16 character hexadecimal string
	interfaceIDHex := fmt.Sprintf("%016x", interfaceID)

	// Constructing the full IPv6 address in the fe80::/64 range
	ipv6Address := fmt.Sprintf("fe80::%s:%s:%s:%s/64",
		interfaceIDHex[:4], interfaceIDHex[4:8], interfaceIDHex[8:12], interfaceIDHex[12:])

	network.IPAddresses = append(network.IPAddresses, ipv6Address)

}
