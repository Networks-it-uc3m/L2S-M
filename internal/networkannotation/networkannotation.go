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
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace,omitempty"`
	IPAdresses []string `json:"ips,omitempty"`
	IfName     string   `json:"ifname,omitempty"`
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
		// if len(networks[i].IPAdresses) == 0 {
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

	network.IPAdresses = append(network.IPAdresses, ipv6Address)

}
