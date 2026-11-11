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
	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"math/rand"
	"net/http"
	"time"

	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mpod.kb.io,sideEffects=NoneOnDryRun,admissionReviewVersions=v1
type PodAnnotator struct {
	Client            client.Client
	APIReader         client.Reader
	Decoder           *admission.Decoder
	SwitchesNamespace string
}

func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	if err := a.Decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if pod.DeletionTimestamp != nil {
		return admission.Allowed("pod deletion")
	}
	raw, uses := pod.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION]
	if !uses || pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY] != "" {
		return admission.Allowed("no allocation required")
	}
	if req.DryRun != nil && *req.DryRun {
		// Admission reserves real resources. A dry-run must never mutate them.
		return admission.Denied("L2S-M attachment allocation requires a non-dry-run request")
	}
	if pod.Namespace == "" {
		pod.Namespace = req.Namespace
	}
	if pod.Name == "" {
		return admission.Denied("L2S-M allocation requires a pod name")
	}
	store := allocationStore{Client: a.Client, Reader: a.APIReader}
	if pod.Spec.NodeName == "" {
		nodes := &corev1.NodeList{}
		if err := store.reader().List(ctx, nodes); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if len(nodes.Items) == 0 {
			return admission.Denied("no available nodes")
		}
		pod.Spec.NodeName = nodes.Items[rand.Intn(len(nodes.Items))].Name
	}
	networks, err := networkannotation.ExtractNetworks(raw, pod.Namespace)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	// An admission ID distinguishes a new pod incarnation from delayed cleanup
	// of another pod with the same namespace/name.
	id := string(req.UID)
	if id == "" {
		return admission.Denied("missing admission request UID")
	}
	pod.Annotations[allocationIDAnnotation] = id
	type acquired struct {
		network client.ObjectKey
		port    client.ObjectKey
		owner   portOwner
	}
	var held []acquired
	committed := false
	defer func() {
		if committed {
			return
		}
		// Request cancellation must not prevent rollback. Orphan reconciliation
		// handles failures of these best-effort compensations.
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, claim := range held {
			if err := store.releaseIPs(cleanup, claim.network, pod.Name, id); err != nil {
				log.FromContext(ctx).Error(err, "allocation rollback failed")
			}
			if err := store.releasePort(cleanup, claim.port, pod.Spec.NodeName, claim.owner); err != nil {
				log.FromContext(ctx).Error(err, "port rollback failed")
			}
		}
	}()
	var attachments []networkannotation.NetworkAnnotation
	seen := map[client.ObjectKey]bool{}
	for _, network := range networks {
		namespace := network.Namespace
		if namespace == "" {
			namespace = pod.Namespace
		}
		if namespace != pod.Namespace {
			return admission.Denied("cross-namespace L2Network attachment is not supported")
		}
		key := client.ObjectKey{Namespace: namespace, Name: network.Name}
		if seen[key] {
			return admission.Denied("duplicate L2Network attachment")
		}
		seen[key] = true
		// Validate the network before reserving a scarce attachment.
		n := &l2smv1.L2Network{}
		if err := store.reader().Get(ctx, key, n); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		owner := podPortOwner(pod.Namespace, pod.Name, id, network.Name)
		nad, err := store.reservePort(ctx, a.SwitchesNamespace, pod.Spec.NodeName, owner)
		if err != nil {
			return admission.Errored(http.StatusServiceUnavailable, err)
		}
		held = append(held, acquired{key, client.ObjectKeyFromObject(nad), owner})
		ips, err := store.reserveIPs(ctx, key, pod.Name, id, network.IPAddresses)
		if err != nil {
			return admission.Errored(http.StatusServiceUnavailable, err)
		}
		attachment := networkannotation.NetworkAnnotation{Name: nad.Name, Namespace: nad.Namespace, IPAddresses: ips}
		if len(ips) == 0 {
			attachment.GenerateIPv6Address()
		}
		attachments = append(attachments, attachment)
	}
	pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY] = networkannotation.MultusAnnotationToString(attachments)
	encoded, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	committed = true
	return admission.PatchResponseFromRaw(req.Object.Raw, encoded)
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
