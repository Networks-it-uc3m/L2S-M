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

package controller

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net/netip"
	"strings"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/talpainterface"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ERROR_ANNOTATION   = "l2sm/error"
	L2SM_PODNAME_LABEL = "l2sm/app"
)

func GetL2Networks(ctx context.Context, c client.Client, networks []networkannotation.NetworkAnnotation) ([]l2smv1.L2Network, error) {
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

// TODO: join methods together
func GetL2NetworksMap(ctx context.Context, c client.Client, networks []networkannotation.NetworkAnnotation) (map[string]l2smv1.L2Network, error) {
	// List all L2Networks
	result := make(map[string]l2smv1.L2Network)
	l2Networks := &l2smv1.L2NetworkList{}
	if err := c.List(ctx, l2Networks); err != nil {
		return result, err
	}

	// Create a map of existing L2Network names to L2Network objects for quick lookup
	existingNetworks := make(map[string]l2smv1.L2Network)
	for _, network := range l2Networks.Items {
		existingNetworks[network.Name] = network
	}

	// Collect the L2Networks that match the requested networks
	for _, net := range networks {
		if l2net, exists := existingNetworks[net.Name]; exists {
			result[net.Name] = l2net
		} else {
			return result, fmt.Errorf("network %s doesn't exist", net.Name)
		}
	}

	return result, nil
}

func GetFreeNetAttachDefs(ctx context.Context, c client.Client, switchesNamespace, label string) nettypes.NetworkAttachmentDefinitionList {

	// We define the network attachment definition list that will be later filled.
	freeNetAttachDef := &nettypes.NetworkAttachmentDefinitionList{}

	// We specify which net-attach-def we want. We want the ones that are specific to l2sm, in the overlay namespace and available in the desired node.
	nodeSelector := labels.NewSelector()

	nodeRequirement, _ := labels.NewRequirement(label, selection.NotIn, []string{"true"})
	l2smRequirement, _ := labels.NewRequirement("app", selection.Equals, []string{"l2sm"})

	nodeSelector = nodeSelector.Add(*l2smRequirement, *nodeRequirement)

	listOptions := client.ListOptions{LabelSelector: nodeSelector, Namespace: switchesNamespace}

	// We get the net-attach-def with the corresponding list options
	c.List(ctx, freeNetAttachDef, &listOptions)
	return *freeNetAttachDef

}

func (r *PodReconciler) DetachNetAttachDef(ctx context.Context, multusNetAttachDef networkannotation.NetworkAnnotation, namespace string) error {

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

func GenerateAnnotations(overlayName, nodeName, providerName string, ammount int) string {

	annotationsString := []string{}
	var newAnnotation string
	for i := 1; i <= ammount; i++ {
		newAnnotation = fmt.Sprintf(`{"name": "%s-veth%d", "ips": ["fe80::58d0:b8ff:fe%s:%s/64"], "interface":"%s"}`,
			overlayName, i,
			fmt.Sprintf("%02d", i), Generate4byteChunk(),
			talpainterface.GetIfName(nodeName, l2smv1.OVERLAY_PROVIDER, i))
		annotationsString = append(annotationsString, newAnnotation)
	}

	return "[" + strings.Join(annotationsString, ",") + "]"
}

func Generate4byteChunk() string {

	// Generating the interface ID (64 bits)
	interfaceID := rand.Uint64() & 0xffff

	// Formatting to a 4 character hexadecimal string
	interfaceIDHex := fmt.Sprintf("%04x", interfaceID)

	return interfaceIDHex

}

func CreateErrorEvent(
	ctx context.Context,
	client client.Client,
	pod *corev1.Pod,
	message string,
	reason string,
) {
	logger := log.FromContext(ctx)

	// 1) Annotate the Pod with the error
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Annotations[ERROR_ANNOTATION] = message

	// 2) Create a k8s Event so that `kubectl describe pod` shows it
	//    (Requires your ServiceAccount to have permission to create events).
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "l2sm-",
			Namespace:    pod.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:            "Pod",
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			UID:             pod.UID,
			APIVersion:      "v1",
			ResourceVersion: pod.ResourceVersion,
		},
		Reason:         reason,
		Message:        message,
		Type:           corev1.EventTypeWarning,
		Source:         corev1.EventSource{Component: "l2sm"},
		FirstTimestamp: metav1.Now(),
	}

	if err := client.Create(ctx, event); err != nil {
		logger.Error(err, "Failed to create event for Pod", "pod", pod.Name)
	}

}

// GetNextAvailableIP generates the next available IP dynamically and returns it with the subnet mask.
func GetNextAvailableIP(networkCIDR, lastAssignedIP string, assignedIPs map[string]string) (string, string, error) {
	prefix, err := netip.ParsePrefix(networkCIDR)
	if err != nil {
		return "", "", fmt.Errorf("invalid CIDR: %s", networkCIDR)
	}
	prefix = prefix.Masked()
	first := prefix.Addr()
	// Exclude network/broadcast addresses for ordinary IPv4 subnets. /31 and
	// /32 retain their point-to-point/host semantics.
	var broadcast netip.Addr
	if first.Is4() && prefix.Bits() < 31 {
		raw := first.As4()
		number := binary.BigEndian.Uint32(raw[:]) | (^uint32(0) >> prefix.Bits())
		binary.BigEndian.PutUint32(raw[:], number)
		broadcast = netip.AddrFrom4(raw)
		first = first.Next()
	}
	candidate := first
	if previous, err := netip.ParseAddr(lastAssignedIP); err == nil && prefix.Contains(previous) {
		candidate = previous.Next()
	}
	// At most the existing allocations plus reserved boundaries can precede a
	// free address. Wrap to reuse released addresses; Addr.Next handles carry.
	for attempt := 0; attempt <= len(assignedIPs)+2; attempt++ {
		if !candidate.IsValid() || !prefix.Contains(candidate) || candidate == broadcast {
			candidate = first
		}
		if !prefix.Contains(candidate) || candidate == broadcast {
			break
		}
		if _, used := assignedIPs[candidate.String()]; !used {
			return candidate.String(), fmt.Sprintf("/%d", prefix.Bits()), nil
		}
		candidate = candidate.Next()
	}
	return "", "", fmt.Errorf("no available IPs in CIDR: %s", networkCIDR)
}
