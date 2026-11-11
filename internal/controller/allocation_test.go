package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func allocationFixture(t *testing.T, ports int) (allocationStore, *l2smv1.L2Network) {
	t.Helper()
	scheme := runtime.NewScheme()
	for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1.AddToScheme, l2smv1.AddToScheme, nettypes.AddToScheme} {
		if err := add(scheme); err != nil {
			t.Fatal(err)
		}
	}
	n := &l2smv1.L2Network{ObjectMeta: metav1.ObjectMeta{Name: "source", Namespace: "default", UID: "network-uid"}, Spec: l2smv1.L2NetworkSpec{Type: l2smv1.NetworkTypeVnet, NetworkCIDR: "10.0.0.0/24"}}
	objects := []client.Object{n}
	for i := 1; i <= ports; i++ {
		objects = append(objects, &nettypes.NetworkAttachmentDefinition{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("overlay-veth%d", i), Namespace: "default", Labels: map[string]string{"app": "l2sm", "used-worker": "false", "used-other": "false"}}})
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&l2smv1.L2Network{}, &l2smv1.QuarantinePodRequest{}).WithObjects(objects...).Build()
	wrapped := &atomicFakeClient{Client: c}
	return allocationStore{Client: wrapped, Reader: wrapped}, n
}

func TestAllocationConcurrentPortsAndIPs(t *testing.T) {
	store, n := allocationFixture(t, 20)
	ctx := context.Background()
	type result struct {
		port string
		ip   string
		err  error
	}
	results := make(chan result, 15)
	gate := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < 15; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-gate
			name, id := fmt.Sprintf("pod-%d", i), fmt.Sprintf("id-%d", i)
			nad, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", name, id, n.Name))
			if err != nil {
				results <- result{err: err}
				return
			}
			ips, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(n), name, id, nil)
			if err != nil {
				results <- result{err: err}
				return
			}
			results <- result{nad.Name, ips[0], nil}
		}(i)
	}
	close(gate)
	workers.Wait()
	close(results)
	ports, ips := map[string]bool{}, map[string]bool{}
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		if ports[result.port] || ips[result.ip] {
			t.Fatalf("duplicate allocation %+v (port=%v IP=%v)", result, ports[result.port], ips[result.ip])
		}
		ports[result.port] = true
		ips[result.ip] = true
	}
	current := &l2smv1.L2Network{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(n), current); err != nil {
		t.Fatal(err)
	}
	if current.Status.ConnectedPodCount != 15 || len(current.Status.AssignedIPs) != 15 || len(current.Status.PodAllocations) != 15 {
		t.Fatalf("lost updates: %+v", current.Status)
	}
}

func TestAllocationRetryAndOwnerSafeRelease(t *testing.T) {
	store, n := allocationFixture(t, 1)
	ctx := context.Background()
	owner := podPortOwner("default", "pod", "attempt-1", n.Name)
	first, err := store.reservePort(ctx, "default", "worker", owner)
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.reservePort(ctx, "default", "worker", owner)
	if err != nil || again.Name != first.Name {
		t.Fatalf("retry allocated a new port: %v", err)
	}
	if _, err := store.reservePort(ctx, "default", "other", podPortOwner("default", "other", "other", n.Name)); err != nil {
		t.Fatal(err)
	} // same NAD on another switch is valid
	ips, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(n), "pod", "attempt-1", nil)
	if err != nil {
		t.Fatal(err)
	}
	againIPs, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(n), "pod", "attempt-1", nil)
	if err != nil || ips[0] != againIPs[0] {
		t.Fatal("retry allocated another IP")
	}
	if err := store.releaseOwnedPorts(ctx, "default", podPortOwner("default", "pod", "older-attempt", n.Name)); err != nil {
		t.Fatal(err)
	}
	if err := store.releaseIPs(ctx, client.ObjectKeyFromObject(n), "pod", "older-attempt"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "other", "new", n.Name)); err == nil {
		t.Fatal("stale cleanup released live port")
	}
	if _, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(n), "other", "new", ips); err == nil {
		t.Fatal("static IP collision accepted")
	}
	if err := store.releaseOwnedPorts(ctx, "default", owner); err != nil {
		t.Fatal(err)
	}
	if err := store.releaseIPs(ctx, client.ObjectKeyFromObject(n), "pod", "attempt-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "other", "new", n.Name)); err != nil {
		t.Fatal(err)
	}
}

func admissionRequest(t *testing.T, name, id, networks string) admission.Request {
	t.Helper()
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Annotations: map[string]string{networkannotation.L2SM_NETWORK_ANNOTATION: networks}}, Spec: corev1.PodSpec{NodeName: "worker", Containers: []corev1.Container{{Name: "test", Image: "alpine"}}}}
	raw, err := json.Marshal(pod)
	if err != nil {
		t.Fatal(err)
	}
	return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{UID: types.UID(id), Namespace: "default", Object: runtime.RawExtension{Raw: raw}}}
}

func TestAllocationAdmissionFailureRollsBack(t *testing.T) {
	store, n := allocationFixture(t, 2)
	handler := &PodAnnotator{Client: store.Client, APIReader: store.Reader, SwitchesNamespace: "default", Decoder: admission.NewDecoder(store.Scheme())}
	response := handler.Handle(context.Background(), admissionRequest(t, "pod", "admission-1", `[{"name":"source"},{"name":"missing"}]`))
	if response.Allowed {
		t.Fatal("partial admission failure was allowed")
	}
	current := &l2smv1.L2Network{}
	if err := store.Get(context.Background(), client.ObjectKeyFromObject(n), current); err != nil {
		t.Fatal(err)
	}
	if current.Status.ConnectedPodCount != 0 || len(current.Status.AssignedIPs) != 0 {
		t.Fatalf("IP reservation leaked: %+v", current.Status)
	}
	if _, err := store.reservePort(context.Background(), "default", "worker", podPortOwner("default", "other", "id", n.Name)); err != nil {
		t.Fatal(err)
	}
}

func TestAllocationAdmissionDryRunHasNoSideEffects(t *testing.T) {
	store, n := allocationFixture(t, 1)
	handler := &PodAnnotator{Client: store.Client, APIReader: store.Reader, SwitchesNamespace: "default", Decoder: admission.NewDecoder(store.Scheme())}
	req := admissionRequest(t, "pod", "attempt", n.Name)
	dry := true
	req.DryRun = &dry
	if response := handler.Handle(context.Background(), req); response.Allowed {
		t.Fatal("unsupported allocating dry-run should explain rejection")
	}
	if _, err := store.reservePort(context.Background(), "default", "worker", podPortOwner("default", "other", "new", n.Name)); err != nil {
		t.Fatal("dry-run reserved a port", err)
	}
}

func TestAllocationRecoversAbandonedAdmissionButPreservesLivePod(t *testing.T) {
	store, n := allocationFixture(t, 3)
	ctx := context.Background()
	for _, name := range []string{"abandoned", "live", "recent"} {
		owner := podPortOwner("default", name, name, n.Name)
		if _, err := store.reservePort(ctx, "default", "worker", owner); err != nil {
			t.Fatal(err)
		}
		if _, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(n), name, name, nil); err != nil {
			t.Fatal(err)
		}
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "live", Namespace: "default", Annotations: map[string]string{allocationIDAnnotation: "live"}}}
	if err := store.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}
	// Make only the first two claims older than the admission grace.
	past := metav1.NewTime(time.Now().Add(-3 * time.Minute))
	if err := store.updateNetwork(ctx, client.ObjectKeyFromObject(n), func(n *l2smv1.L2Network) error {
		for _, name := range []string{"abandoned", "live"} {
			entry := n.Status.PodAllocations[name]
			entry.CreatedAt = past
			n.Status.PodAllocations[name] = entry
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	nads := &nettypes.NetworkAttachmentDefinitionList{}
	if err := store.List(ctx, nads); err != nil {
		t.Fatal(err)
	}
	for i := range nads.Items {
		nad := &nads.Items[i]
		var reservation portReservation
		if err := json.Unmarshal([]byte(nad.Annotations[portKey("worker")]), &reservation); err != nil {
			t.Fatal(err)
		}
		if reservation.Owner.Name == "recent" {
			continue
		}
		reservation.CreatedAt = past
		raw, _ := json.Marshal(reservation)
		nad.Annotations[portKey("worker")] = string(raw)
		if err := store.Update(ctx, nad); err != nil {
			t.Fatal(err)
		}
	}
	recovery := &AllocationRecovery{Client: store.Client, APIReader: store.Reader, SwitchesNamespace: "default"}
	if err := recovery.sweep(ctx, time.Now()); err != nil {
		t.Fatal(err)
	}
	current := &l2smv1.L2Network{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(n), current); err != nil {
		t.Fatal(err)
	}
	if _, exists := current.Status.PodAllocations["abandoned"]; exists {
		t.Fatal("abandoned IP not recovered")
	}
	if len(current.Status.PodAllocations) != 2 || current.Status.ConnectedPodCount != 2 {
		t.Fatalf("live/recent reservations lost: %+v", current.Status)
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "replacement", "new", n.Name)); err != nil {
		t.Fatal(err)
	}
}

// Inject a real resourceVersion conflict after another writer allocates an IP.
// The retry must recompute from the latest status rather than reuse the old IP.
type conflictingStatusClient struct {
	client.Client
	once sync.Once
}
type conflictingStatusWriter struct {
	client.SubResourceWriter
	parent *conflictingStatusClient
}

func (c *conflictingStatusClient) Status() client.SubResourceWriter {
	return &conflictingStatusWriter{c.Client.Status(), c}
}
func (w *conflictingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	injected := false
	w.parent.once.Do(func() {
		current := &l2smv1.L2Network{}
		_ = w.parent.Client.Get(ctx, client.ObjectKeyFromObject(obj), current)
		current.Status.AssignedIPs = map[string]string{"10.0.0.1": "racer"}
		current.Status.LastAssignedIP = "10.0.0.1"
		current.Status.ConnectedPodCount = 1
		_ = w.parent.Client.Status().Update(ctx, current)
		injected = true
	})
	if injected {
		return conflict("l2networks", obj.GetName())
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}
func TestAllocationConflictRecomputesIP(t *testing.T) {
	store, n := allocationFixture(t, 1)
	store.Client = &conflictingStatusClient{Client: store.Client}
	ips, err := store.reserveIPs(context.Background(), client.ObjectKeyFromObject(n), "pod", "id", nil)
	if err != nil || len(ips) != 1 || ips[0] != "10.0.0.2/24" {
		t.Fatalf("stale allocation reused after conflict: %v %v", ips, err)
	}
}

// controller-runtime v0.17's fake tracker checks resourceVersion separately
// from its write. Serialize fake writes to model the API server's atomic CAS;
// reads remain concurrent, so stale-read conflicts still exercise retry logic.
type atomicFakeClient struct {
	client.Client
	mu sync.Mutex
}

func (c *atomicFakeClient) Patch(ctx context.Context, obj client.Object, p client.Patch, opts ...client.PatchOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Client.Patch(ctx, obj, p, opts...)
}
func (c *atomicFakeClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Client.Update(ctx, obj, opts...)
}
func (c *atomicFakeClient) Status() client.SubResourceWriter {
	return &atomicFakeStatus{SubResourceWriter: c.Client.Status(), parent: c}
}

type atomicFakeStatus struct {
	client.SubResourceWriter
	parent *atomicFakeClient
}

func (w *atomicFakeStatus) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	w.parent.mu.Lock()
	defer w.parent.mu.Unlock()
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

func TestAllocationAddressReuseAndOctetCarry(t *testing.T) {
	for _, example := range []struct {
		pool, last, want string
		assigned         map[string]string
	}{
		{"10.0.0.0/16", "10.0.0.255", "10.0.1.0", nil},
		{"10.0.0.0/30", "10.0.0.2", "10.0.0.1", map[string]string{"10.0.0.2": "busy"}},
		{"10.0.0.1/32", "", "10.0.0.1", nil},
	} {
		got, _, err := GetNextAvailableIP(example.pool, example.last, example.assigned)
		if err != nil || got != example.want {
			t.Fatalf("%s after %s: got %s (%v), want %s", example.pool, example.last, got, err, example.want)
		}
	}
	if _, _, err := GetNextAvailableIP("10.0.0.0/30", "10.0.0.2", map[string]string{"10.0.0.1": "one", "10.0.0.2": "two"}); err == nil {
		t.Fatal("exhausted pool allocated a duplicate or broadcast address")
	}
}

func TestAllocationFreshOverlayWithoutUsageLabels(t *testing.T) {
	store, n := allocationFixture(t, 1)
	ctx := context.Background()
	nad := &nettypes.NetworkAttachmentDefinition{}
	key := client.ObjectKey{Namespace: "default", Name: "overlay-veth1"}
	if err := store.Get(ctx, key, nad); err != nil {
		t.Fatal(err)
	}
	nad.Labels = map[string]string{"app": "l2sm", "overlay": "sample"}
	if err := store.Update(ctx, nad); err != nil {
		t.Fatal(err)
	}
	firstOwner := podPortOwner("default", "one", "first", n.Name)
	first, err := store.reservePort(ctx, "default", "worker", firstOwner)
	if err != nil {
		t.Fatalf("fresh unlabelled NAD rejected: %v", err)
	}
	if first.Labels["used-worker"] != "true" {
		t.Fatal("first allocation did not set usage label")
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "two", "second", n.Name)); err == nil {
		t.Fatal("claimed port allocated twice")
	}
	if _, err := store.reservePort(ctx, "default", "other", podPortOwner("default", "two", "second", n.Name)); err != nil {
		t.Fatalf("missing usage label on another node rejected: %v", err)
	}
	if err := store.releasePort(ctx, key, "worker", firstOwner); err != nil {
		t.Fatal(err)
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "replacement", "third", n.Name)); err != nil {
		t.Fatalf("released port could not be reused: %v", err)
	}
}
