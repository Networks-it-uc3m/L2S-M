package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	jsonpatch "github.com/evanphx/json-patch"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sync"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Concurrent allocation against the API server", func() {
	It("keeps port/IP reservations exclusive and persists retry bookkeeping", func() {
		ctx := context.Background()
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "allocation-"}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, ns)).To(Succeed()) })
		network := &l2smv1.L2Network{ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: ns.Name}, Spec: l2smv1.L2NetworkSpec{Type: l2smv1.NetworkTypeVnet, NetworkCIDR: "10.0.0.0/24"}}
		Expect(k8sClient.Create(ctx, network)).To(Succeed())
		for i := 1; i <= 20; i++ {
			nad := &nettypes.NetworkAttachmentDefinition{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("overlay-veth%d", i), Namespace: ns.Name, Labels: map[string]string{"app": "l2sm"}}}
			Expect(k8sClient.Create(ctx, nad)).To(Succeed())
		}
		store := allocationStore{Client: k8sClient, Reader: k8sClient}
		type result struct {
			port, ip string
			err      error
		}
		results := make(chan result, 15)
		gate := make(chan struct{})
		var workers sync.WaitGroup
		for i := 0; i < 15; i++ {
			workers.Add(1)
			go func(i int) {
				defer workers.Done()
				<-gate
				name := fmt.Sprintf("pod-%d", i)
				owner := podPortOwner(ns.Name, name, name, network.Name)
				if i%3 == 0 {
					owner = portOwner{Kind: "L2Network", Namespace: ns.Name, Name: fmt.Sprintf("ids-%d", i), ID: fmt.Sprintf("network-%d", i)}
				}
				port, err := store.reservePort(ctx, ns.Name, "worker", owner)
				if err != nil {
					results <- result{err: err}
					return
				}
				ips, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(network), name, name, nil)
				if err != nil {
					results <- result{err: err}
					return
				}
				results <- result{port.Name, ips[0], nil}
			}(i)
		}
		close(gate)
		workers.Wait()
		close(results)
		ports, ips := map[string]bool{}, map[string]bool{}
		for result := range results {
			Expect(result.err).NotTo(HaveOccurred())
			Expect(ports[result.port]).To(BeFalse(), "port collision")
			Expect(ips[result.ip]).To(BeFalse(), "IP collision")
			ports[result.port] = true
			ips[result.ip] = true
		}
		current := &l2smv1.L2Network{}
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(network), current)).To(Succeed())
		Expect(current.Status.PodAllocations).To(HaveLen(15))
		Expect(current.Status.AssignedIPs).To(HaveLen(15))
		Expect(current.Status.ConnectedPodCount).To(Equal(15))
	})
})

var _ = Describe("Five-network admission and IDS allocation", func() {
	It("reserves fifteen different ports while ten pods and five IDS deployments are created together", func() {
		ctx := context.Background()
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "five-networks-"}}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		DeferCleanup(func() { Expect(k8sClient.Delete(ctx, ns)).To(Succeed()) })
		for i := 1; i <= 20; i++ {
			nad := &nettypes.NetworkAttachmentDefinition{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("overlay-veth%d", i), Namespace: ns.Name, Labels: map[string]string{"app": "l2sm"}}}
			Expect(k8sClient.Create(ctx, nad)).To(Succeed())
		}
		networks := make([]*l2smv1.L2Network, 5)
		for i := range networks {
			n := &l2smv1.L2Network{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("network-%d", i), Namespace: ns.Name}, Spec: l2smv1.L2NetworkSpec{Type: l2smv1.NetworkTypeVnet, NetworkCIDR: "10.0.0.0/24", Ids: &l2smv1.IdsRules{Enabled: true, Node: "worker", Profile: "suricata"}}}
			Expect(k8sClient.Create(ctx, n)).To(Succeed())
			networks[i] = n
		}
		gate := make(chan struct{})
		results := make(chan error, 15)
		var workers sync.WaitGroup
		for _, network := range networks {
			n := network
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-gate
				r := &L2NetworkReconciler{Client: k8sClient, APIReader: k8sClient, Scheme: k8sClient.Scheme(), SwitchesNamespace: ns.Name, InternalClient: &fakeQuarantineSDNClient{}}
				results <- r.ensureIDS(ctx, n)
			}()
			for _, role := range []string{"ping", "pong"} {
				name := n.Name + "-" + role
				raw, _ := json.Marshal(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns.Name, Annotations: map[string]string{networkannotation.L2SM_NETWORK_ANNOTATION: n.Name}}, Spec: corev1.PodSpec{NodeName: "worker", Containers: []corev1.Container{{Name: "alpine", Image: "alpine:latest"}}}})
				request := admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{UID: types.UID(name), Namespace: ns.Name, Object: runtime.RawExtension{Raw: raw}}}
				workers.Add(1)
				go func(req admission.Request) {
					defer workers.Done()
					<-gate
					handler := &PodAnnotator{Client: k8sClient, APIReader: k8sClient, SwitchesNamespace: ns.Name, Decoder: admission.NewDecoder(k8sClient.Scheme())}
					response := handler.Handle(ctx, req)
					if !response.Allowed {
						results <- fmt.Errorf("admission %s: %v", req.UID, response.Result)
						return
					}
					patchRaw, err := json.Marshal(response.Patches)
					if err != nil {
						results <- err
						return
					}
					patch, err := jsonpatch.DecodePatch(patchRaw)
					if err != nil {
						results <- err
						return
					}
					encoded, err := patch.Apply(req.Object.Raw)
					if err != nil {
						results <- err
						return
					}
					pod := &corev1.Pod{}
					if err := json.Unmarshal(encoded, pod); err != nil {
						results <- err
						return
					}
					results <- k8sClient.Create(ctx, pod)
				}(request)
			}
		}
		close(gate)
		workers.Wait()
		close(results)
		for err := range results {
			Expect(err).NotTo(HaveOccurred())
		}
		ports := map[string]bool{}
		checkPort := func(raw string) {
			attachments, err := networkannotation.ExtractNetworks(raw, ns.Name)
			Expect(err).NotTo(HaveOccurred())
			Expect(attachments).To(HaveLen(1))
			name := attachments[0].Name
			Expect(ports[name]).To(BeFalse(), "duplicate switch port %s", name)
			ports[name] = true
		}
		pods := &corev1.PodList{}
		Expect(k8sClient.List(ctx, pods, client.InNamespace(ns.Name))).To(Succeed())
		Expect(pods.Items).To(HaveLen(10))
		for _, pod := range pods.Items {
			checkPort(pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY])
		}
		deps := &appsv1.DeploymentList{}
		Expect(k8sClient.List(ctx, deps, client.InNamespace(ns.Name))).To(Succeed())
		Expect(deps.Items).To(HaveLen(5))
		for _, dep := range deps.Items {
			checkPort(dep.Spec.Template.Annotations[networkannotation.MULTUS_ANNOTATION_KEY])
		}
		Expect(ports).To(HaveLen(15))
		// Development mode leaves SWITCHES_NAMESPACE unset. Reconcile the
		// existing IDS deployments again through the real API client.
		retryController := &L2NetworkReconciler{Client: k8sClient, APIReader: k8sClient, Scheme: k8sClient.Scheme(), InternalClient: &fakeQuarantineSDNClient{}}
		for _, network := range networks {
			Expect(retryController.ensureIDS(ctx, network)).To(Succeed())
		}

		for _, network := range networks {
			current := &l2smv1.L2Network{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(network), current)).To(Succeed())
			Expect(current.Status.ConnectedPodCount).To(Equal(2))
			Expect(current.Status.AssignedIPs).To(HaveLen(2))
			Expect(current.Status.PodAllocations).To(HaveLen(2))
		}
	})
})
