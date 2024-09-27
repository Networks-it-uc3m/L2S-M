package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NET_ATTACH_LABEL_PREFIX = "used-"
	L2SM_NETWORK_ANNOTATION = "l2sm/networks"
	MULTUS_ANNOTATION_KEY   = "k8s.v1.cni.cncf.io/networks"
)

type NetworkAnnotation struct {
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace,omitempty"`
	IPAdresses []string `json:"ips,omitempty"`
}

func extractNetworks(annotations, namespace string) ([]NetworkAnnotation, error) {

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

	// Iterate over the networks to check if any IPAddresses are missing
	for i := range networks {
		if len(networks[i].IPAdresses) == 0 {
			// Call GenerateIPv6Address if IPAddresses are missing
			networks[i].GenerateIPv6Address()
		}
		networks[i].Namespace = namespace
	}
	return networks, nil
}

func GetL2Networks(ctx context.Context, c client.Client, networks []NetworkAnnotation) ([]l2smv1.L2Network, error) {
	// List all L2Networks
	l2Networks := &l2smv1.L2NetworkList{}
	if err := c.List(ctx, l2Networks); err != nil {
		return []l2smv1.L2Network{}, err
	}

	// Create a map of existing L2Network names to L2Network objects for quick lookup
	existingNetworks := make(map[string]l2smv1.L2Network)
	for _, network := range l2Networks.Items {
		existingNetworks[network.Name] = network
	}

	// Collect the L2Networks that match the requested networks
	var result l2smv1.L2NetworkList
	for _, net := range networks {
		if l2net, exists := existingNetworks[net.Name]; exists {
			result.Items = append(result.Items, l2net)
		} else {
			return result.Items, fmt.Errorf("network %s doesn't exist", net.Name)
		}
	}

	return result.Items, nil
}

func GetFreeNetAttachDefs(ctx context.Context, c client.Client, switchesNamespace, label string) nettypes.NetworkAttachmentDefinitionList {

	// We define the network attachment definition list that will be later filled.
	freeNetAttachDef := &nettypes.NetworkAttachmentDefinitionList{}

	// We specify which net-attach-def we want. We want the ones that are specific to l2sm, in the overlay namespace and available in the desired node.
	nodeSelector := labels.NewSelector()

	nodeRequirement, _ := labels.NewRequirement(label, selection.NotIn, []string{"true"})
	l2smRequirement, _ := labels.NewRequirement("app", selection.Equals, []string{"l2sm"})

	nodeSelector.Add(*nodeRequirement)
	nodeSelector.Add(*l2smRequirement)

	listOptions := client.ListOptions{LabelSelector: nodeSelector, Namespace: switchesNamespace}

	// We get the net-attach-def with the corresponding list options
	c.List(ctx, freeNetAttachDef, &listOptions)
	return *freeNetAttachDef

}

func (network *NetworkAnnotation) GenerateIPv6Address() {
	rand.Seed(time.Now().UnixNano())

	// Generating the interface ID (64 bits)
	interfaceID := rand.Uint64()

	// Formatting to a 16 character hexadecimal string
	interfaceIDHex := fmt.Sprintf("%016x", interfaceID)

	// Constructing the full IPv6 address in the fe80::/64 range
	ipv6Address := fmt.Sprintf("fe80::%s:%s:%s:%s/64",
		interfaceIDHex[:4], interfaceIDHex[4:8], interfaceIDHex[8:12], interfaceIDHex[12:])

	network.IPAdresses = append(network.IPAdresses, ipv6Address)

}
func multusAnnotationToString(multusAnnotations []NetworkAnnotation) string {
	jsonData, err := json.Marshal(multusAnnotations)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (r *PodReconciler) DetachNetAttachDef(ctx context.Context, multusNetAttachDef NetworkAnnotation, namespace string) error {

	netAttachDef := &nettypes.NetworkAttachmentDefinition{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      multusNetAttachDef.Name,
		Namespace: namespace,
	}, netAttachDef)
	if err != nil {
		return err
	}
	err = r.Delete(ctx, netAttachDef)
	return err

}