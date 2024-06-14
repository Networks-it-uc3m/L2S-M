package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	NET_ATTACH_LABEL_PREFIX = "used-"
	L2SM_NETWORK_ANNOTATION = "l2sm/networks"
	MULTUS_ANNOTATION_KEY   = "k8s.v1.cni.cncf.io/networks"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io
type PodAnnotator struct {
	Client            client.Client
	Decoder           *admission.Decoder
	SwitchesNamespace string
}

type NetworkAnnotation struct {
	Name       string   `json:"name"`
	Namespace  string   `json:"namespace,omitempty"`
	IPAdresses []string `json:"ips,omitempty"`
}

func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := log.FromContext(ctx)
	log.Info("Registering pod")

	// First we decode the pod
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Error decoding pod")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Check if the pod has the annotation l2sm/networks. This webhook operation only will happen if so. Else, it will just
	// let the creation begin.
	if annot, ok := pod.Annotations[L2SM_NETWORK_ANNOTATION]; ok {
		netAttachDefLabel := NET_ATTACH_LABEL_PREFIX + pod.Spec.NodeName
		// We extract which networks the user intends to attach the pod to. If there is any error, or the
		// Networks aren't created, the pod will be set as errored, until a network is created.
		networks, err := extractNetworks(annot, a.SwitchesNamespace)
		if err != nil {
			log.Error(err, "L2S-M Network annotations could not be extracted")
			return admission.Errored(http.StatusInternalServerError, err)
		}

		if created, _ := a.verifyNetworksAreCreated(ctx, networks); !created {
			log.Info("Pod's network annotation incorrect. L2Network not attached.")
			// return admission.Allowed("Pod's network annotation incorrect. L2Network not attached.")

		}

		// We get the available network attachment definitions. These are interfaces attached to the switches, so
		// by using labelling, we can know which interfaces the switch has.
		var multusAnnotations []NetworkAnnotation
		netAttachDefs := a.getFreeNetAttachDefs(ctx, netAttachDefLabel)

		// If there are no available network attachment definitions, we can't attach the pod to the desired networks
		// So, we launch an error.
		if len(netAttachDefs.Items) < len(networks) {
			log.Info(fmt.Sprintf("No interfaces available for node %s", pod.Spec.NodeName))
			return admission.Allowed("No interfaces available for node")
		}

		// Now we create the multus annotations, by using the network attachment definition name
		// And the desired IP address.
		for index, network := range networks {

			netAttachDef := &netAttachDefs.Items[index]
			newAnnotation := NetworkAnnotation{Name: netAttachDef.Name, IPAdresses: network.IPAdresses}
			netAttachDef.Labels[netAttachDefLabel] = "true"
			log.Info(fmt.Sprintf("updating network attachment definition_ ", netAttachDef))

			err = a.Client.Update(ctx, netAttachDef)
			if err != nil {
				log.Error(err, "Could not update network attachment definition")

			}
			multusAnnotations = append(multusAnnotations, newAnnotation)
		}
		pod.Annotations[MULTUS_ANNOTATION_KEY] = multusAnnotationToString(multusAnnotations)

		// pod.Annotations["k8s.v1.cni.cncf.io/networks"] = `[{"name": "veth10","ips": ["10.0.0.1/24"]}]`
		log.Info("Pod assigned to the l2networks")

		marshaledPod, err := json.Marshal(pod)
		if err != nil {
			log.Error(err, "Error marshaling pod")
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)

	}
	return admission.Allowed("Pod not using l2sm networks")

}

func (a *PodAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.Decoder = d
	return nil
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

func (a *PodAnnotator) verifyNetworksAreCreated(ctx context.Context, networks []NetworkAnnotation) (bool, error) {
	// List all L2Networks
	l2Networks := &l2smv1.L2NetworkList{}
	if err := a.Client.List(ctx, l2Networks); err != nil {
		return false, err
	}

	// Create a map of existing L2Network names for quick lookup
	existingNetworks := make(map[string]struct{})
	for _, network := range l2Networks.Items {
		existingNetworks[network.Name] = struct{}{}
	}

	// Verify if each annotated network exists
	for _, net := range networks {
		if _, exists := existingNetworks[net.Name]; !exists {
			return false, nil
		}

	}

	return true, nil
}

func (a *PodAnnotator) getFreeNetAttachDefs(ctx context.Context, nodeName string) nettypes.NetworkAttachmentDefinitionList {

	// We define the network attachment definition list that will be later filled.
	freeNetAttachDef := &nettypes.NetworkAttachmentDefinitionList{}

	// We specify which net-attach-def we want. We want the ones that are specific to l2sm, in the overlay namespace and available in the desired node.
	nodeSelector := labels.NewSelector()

	nodeRequirement, _ := labels.NewRequirement(fmt.Sprintf("%s%s", NET_ATTACH_LABEL_PREFIX, nodeName), selection.NotIn, []string{"true"})
	l2smRequirement, _ := labels.NewRequirement("app", selection.Equals, []string{"l2sm"})

	nodeSelector.Add(*nodeRequirement)
	nodeSelector.Add(*l2smRequirement)

	listOptions := client.ListOptions{LabelSelector: nodeSelector, Namespace: a.SwitchesNamespace}

	// We get the net-attach-def with the corresponding list options
	a.Client.List(ctx, freeNetAttachDef, &listOptions)
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