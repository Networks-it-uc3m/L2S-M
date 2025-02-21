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
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=update,versions=v1,name=mpod.kb.io
type PodAnnotator struct {
	Client            client.Client
	Decoder           *admission.Decoder
	SwitchesNamespace string
}

func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	log := log.FromContext(ctx)
	log.Info("Webhook: registering pod")
	// First we decode the pod
	pod := &corev1.Pod{}
	err := a.Decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Error decoding pod")
		return admission.Errored(http.StatusBadRequest, err)
	}
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		return admission.Allowed("Allowing pod's deletion")
	}

	if pod.Spec.NodeName == "" {
		// List available nodes.
		nodeList := &corev1.NodeList{}
		if err := a.Client.List(ctx, nodeList); err != nil {
			log.Error(err, "Error listing nodes")
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if len(nodeList.Items) == 0 {
			return admission.Errored(http.StatusInternalServerError, fmt.Errorf("no available nodes"))
		}

		// Seed the random number generator.

		selectedNode := nodeList.Items[rand.Intn(len(nodeList.Items))].Name
		log.Info("Selected node", "node", selectedNode)
	}
	if _, ok := pod.Annotations[ERROR_ANNOTATION]; ok {
		return admission.Allowed("Already errored creation")
	}
	// Check if the pod has the annotation l2sm/networks. This webhook operation only will happen if so. Else, it will just
	// let the creation begin.
	if annot, ok := pod.Annotations[L2SM_NETWORK_ANNOTATION]; ok {
		if _, ok := pod.Annotations[MULTUS_ANNOTATION_KEY]; ok {
			return admission.Allowed("Pod already using multus cni plugin")
		}
		netAttachDefLabel := NET_ATTACH_LABEL_PREFIX + pod.Spec.NodeName
		// We extract which networks the user intends to attach the pod to. If there is any error, or the
		// Networks aren't created, the pod will be set as errored, until a network is created.
		netAnnotations, err := extractNetworks(annot, a.SwitchesNamespace)
		if err != nil {
			log.Error(err, "L2S-M Network annotations could not be extracted")
			return admission.Errored(http.StatusInternalServerError, err)
		}

		networkResources, err := GetL2NetworksMap(ctx, a.Client, netAnnotations)
		if err != nil {
			log.Info("Pod's network annotation incorrect. L2Network not attached.")
			// return admission.Allowed("Pod's network annotation incorrect. L2Network not attached.")

		}

		// We get the available network attachment definitions. These are interfaces attached to the switches, so
		// by using labelling, we can know which interfaces the switch has.
		var multusAnnotations []NetworkAnnotation
		netAttachDefs := GetFreeNetAttachDefs(ctx, a.Client, a.SwitchesNamespace, netAttachDefLabel)

		// If there are no available network attachment definitions, we can't attach the pod to the desired networks
		// So, we launch an error.
		if len(netAttachDefs.Items) < len(netAnnotations) {
			msg := fmt.Sprintf("No interfaces available for node %s", pod.Spec.NodeName)
			return patchErrorPod(req, pod, &log, msg)
		}
		// Now we create the multus annotations, by using the network attachment definition name
		// And the desired IP address.
		for index, netAnnot := range netAnnotations {

			network, ok := networkResources[netAnnot.Name]
			if !ok {
				log.Error(err, "Could not retrieve l2network")
			}
			var assignIPAddr []string
			netAttachDef := &netAttachDefs.Items[index]
			newAnnotation := NetworkAnnotation{Name: netAttachDef.Name, Namespace: a.SwitchesNamespace}

			if len(netAnnot.IPAdresses) != 0 {

				assignIPAddr = netAnnot.IPAdresses
			} else {

				if network.Spec.NetworkCIDR != "" {
					addressRange := network.Spec.NetworkCIDR
					_, subnet, err := net.ParseCIDR(addressRange)
					subnetMask := fmt.Sprintf("/%d", subnet)

					if err != nil {
						log.Error(err, "NetworkCIDR couldn't be parsed correctly", "network", network.Name)
					}

					if network.Spec.PodAddressRange != "" {
						addressRange = network.Spec.PodAddressRange
					}
					nextIP, _, err := GetNextAvailableIP(addressRange, network.Status.LastAssignedIP, network.Status.AssignedIPs)

					network.Status.LastAssignedIP = nextIP
					assignIPAddr = append(assignIPAddr, nextIP+subnetMask)
					if err != nil {
						log.Error(err, "No available IP addresses for network", "network", network.Name)
					}
				}

			}
			if len(assignIPAddr) != 0 {
				newAnnotation.IPAdresses = assignIPAddr

			}
			network.Status.ConnectedPodCount++
			if network.Status.AssignedIPs == nil {
				network.Status.AssignedIPs = make(map[string]string)
			}

			// Now safely assign the IP to the pod
			network.Status.AssignedIPs[network.Status.LastAssignedIP] = pod.Name
			netAttachDef.Labels[netAttachDefLabel] = "true"

			err = a.Client.Update(ctx, netAttachDef)
			if err != nil {
				log.Error(err, "Could not update network attachment definition")

			}
			err = a.Client.Status().Update(ctx, &network)
			if err != nil {
				log.Error(err, "Could not update l2network status")

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

// patchPod marshals the mutated pod and returns a patch response.
func patchErrorPod(req admission.Request, pod *corev1.Pod, logger *logr.Logger, errorMessage string) admission.Response {
	pod.Annotations[ERROR_ANNOTATION] = errorMessage

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "Error marshaling Pod")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
