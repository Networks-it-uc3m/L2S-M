package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io
type PodAnnotator struct {
	Client            client.Client
	Decoder           *admission.Decoder
	SwitchesNamespace string
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
	if pod.Spec.NodeName == "" {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("pod hasn't got a node assigned to it"))
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
		networks, err := extractNetworks(annot, a.SwitchesNamespace)
		if err != nil {
			log.Error(err, "L2S-M Network annotations could not be extracted")
			return admission.Errored(http.StatusInternalServerError, err)
		}

		if _, err := GetL2Networks(ctx, a.Client, networks); err != nil {
			log.Info("Pod's network annotation incorrect. L2Network not attached.")
			// return admission.Allowed("Pod's network annotation incorrect. L2Network not attached.")

		}

		// We get the available network attachment definitions. These are interfaces attached to the switches, so
		// by using labelling, we can know which interfaces the switch has.
		var multusAnnotations []NetworkAnnotation
		netAttachDefs := GetFreeNetAttachDefs(ctx, a.Client, a.SwitchesNamespace, netAttachDefLabel)

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
			newAnnotation := NetworkAnnotation{Name: netAttachDef.Name, IPAdresses: network.IPAdresses, Namespace: a.SwitchesNamespace}
			netAttachDef.Labels[netAttachDefLabel] = "true"

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
