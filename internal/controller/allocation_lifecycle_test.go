package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type failingDeploymentClient struct {
	client.Client
	fail bool
}

func (c *failingDeploymentClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	if _, ok := obj.(*appsv1.Deployment); ok && c.fail {
		c.fail = false
		return fmt.Errorf("injected deployment failure")
	}
	return c.Client.Create(ctx, obj, opts...)
}

type checkingMirrorSDN struct {
	fakeQuarantineSDNClient
	store   allocationStore
	t       *testing.T
	network *l2smv1.L2Network
	calls   int
}

func (c *checkingMirrorSDN) SetUpMirrorPort(_ l2smv1.NetworkType, _ any) error {
	c.calls++
	list := &nettypes.NetworkAttachmentDefinitionList{}
	if err := c.store.List(context.Background(), list); err != nil {
		c.t.Fatal(err)
	}
	found := false
	for _, nad := range list.Items {
		var reservation portReservation
		if raw := nad.Annotations[portKey("worker")]; raw != "" {
			if err := json.Unmarshal([]byte(raw), &reservation); err != nil {
				c.t.Fatal(err)
			}
			if reservation.Owner == networkPortOwner(c.network) && nad.Labels["used-worker"] == "true" {
				found = true
			}
		}
	}
	if !found {
		c.t.Fatal("IDS configured before owned reservation")
	}

	return nil
}
func TestAllocationIDSResumesAfterPartialCreation(t *testing.T) {
	store, n := allocationFixture(t, 3)
	ctx := context.Background()
	n.Spec.Ids = &l2smv1.IdsRules{Enabled: true, Node: "worker", Profile: "suricata"}
	if err := store.Get(ctx, client.ObjectKeyFromObject(n), n); err != nil {
		t.Fatal(err)
	}
	n.Spec.Ids = &l2smv1.IdsRules{Enabled: true, Node: "worker", Profile: "suricata"}
	n.Finalizers = []string{l2smFinalizer}
	if err := store.Update(ctx, n); err != nil {
		t.Fatal(err)
	}
	failing := &failingDeploymentClient{Client: store.Client, fail: true}
	sdn := &checkingMirrorSDN{store: store, t: t, network: n}
	r := &L2NetworkReconciler{Client: failing, APIReader: store.Reader, Scheme: store.Scheme(), InternalClient: sdn, SwitchesNamespace: "default"}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(n)}
	if _, err := r.Reconcile(ctx, req); err == nil {
		t.Fatal("failed IDS creation was ignored")
	}
	// Another pod cannot steal the network's pending IDS claim.
	other, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "pod", "id", n.Name))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal("IDS retry failed", err)
	}
	dep := &appsv1.Deployment{}
	if err := store.Get(ctx, client.ObjectKey{Namespace: "default", Name: n.Name + "-ids"}, dep); err != nil {
		t.Fatal(err)
	}
	attachments, err := networkannotation.ExtractNetworks(dep.Spec.Template.Annotations[networkannotation.MULTUS_ANNOTATION_KEY], "default")
	if err != nil || len(attachments) != 1 {
		t.Fatal(err)
	}
	if attachments[0].Name == other.Name {
		t.Fatal("IDS/pod port collision")
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if sdn.calls != 2 {
		t.Fatalf("completed IDS unexpectedly configured again: %d", sdn.calls)
	}
}

// Fail the source status write only after the pod membership annotation changed.
// The next reconciliation must discover and finish that interrupted move.
type failingSourceClient struct {
	client.Client
	fail bool
}
type failingSourceStatus struct {
	client.SubResourceWriter
	parent *failingSourceClient
}

func (c *failingSourceClient) Status() client.SubResourceWriter {
	return &failingSourceStatus{c.Client.Status(), c}
}
func (w *failingSourceStatus) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	if obj.GetName() == "source" && w.parent.fail {
		w.parent.fail = false
		return fmt.Errorf("injected source status failure")
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}
func quarantineFixture(t *testing.T) (allocationStore, *l2smv1.L2Network, *l2smv1.L2Network, *corev1.Pod) {
	t.Helper()
	store, source := allocationFixture(t, 3)
	ctx := context.Background()
	target := &l2smv1.L2Network{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "default", UID: "target-uid"}, Spec: source.Spec}
	if err := store.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	ips, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(source), "attacker", "attempt", nil)
	if err != nil {
		t.Fatal(err)
	}
	nad, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "attacker", "attempt", source.Name))
	if err != nil {
		t.Fatal(err)
	}
	raw := networkannotation.MultusAnnotationToString([]networkannotation.NetworkAnnotation{{Name: nad.Name, Namespace: "default", IPAddresses: ips}})
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "attacker", Namespace: "default", Annotations: map[string]string{allocationIDAnnotation: "attempt", networkannotation.L2SM_NETWORK_ANNOTATION: source.Name, networkannotation.MULTUS_ANNOTATION_KEY: raw}}, Spec: corev1.PodSpec{NodeName: "worker"}}
	if err := store.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}
	return store, source, target, pod
}
func TestAllocationQuarantineResumesAfterAnnotationAndDoesNotDoubleCount(t *testing.T) {
	store, source, target, pod := quarantineFixture(t)
	ctx := context.Background()
	failing := &failingSourceClient{Client: store.Client, fail: true}
	sdn := &fakeQuarantineSDNClient{}
	r := &QuarantinePodRequestReconciler{Client: failing, APIReader: store.Reader, InternalClient: sdn}
	if _, err := r.movePodToTargetNetwork(ctx, pod, source, target); err == nil {
		t.Fatal("injected status failure ignored")
	}
	current := &corev1.Pod{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(pod), current); err != nil {
		t.Fatal(err)
	}
	if current.Annotations[pendingMoveAnnotation] == "" {
		t.Fatal("interrupted move lost journal")
	}
	if moved, err := r.movePodToTargetNetwork(ctx, current, source, target); err != nil || !moved {
		t.Fatalf("move not resumed: %v %v", moved, err)
	}
	if moved, err := r.movePodToTargetNetwork(ctx, current, source, target); err != nil || moved {
		t.Fatalf("completed move repeated: %v %v", moved, err)
	}
	for _, example := range []struct {
		network *l2smv1.L2Network
		count   int
	}{{source, 0}, {target, 1}} {
		n := &l2smv1.L2Network{}
		if err := store.Get(ctx, client.ObjectKeyFromObject(example.network), n); err != nil {
			t.Fatal(err)
		}
		if n.Status.ConnectedPodCount != example.count || len(n.Status.PodAllocations) != example.count || len(n.Status.AssignedIPs) != example.count {
			t.Fatalf("non-idempotent bookkeeping: %+v", n.Status)
		}
	}
}
func TestAllocationQuarantineRejectsTargetIPCollisionBeforeForwarding(t *testing.T) {
	store, source, target, pod := quarantineFixture(t)
	ctx := context.Background()
	if _, err := store.reserveIPs(ctx, client.ObjectKeyFromObject(target), "resident", "other", []string{"10.0.0.1/24"}); err != nil {
		t.Fatal(err)
	}
	sdn := &fakeQuarantineSDNClient{}
	r := &QuarantinePodRequestReconciler{Client: store.Client, APIReader: store.Reader, InternalClient: sdn}
	if _, err := r.movePodToTargetNetwork(ctx, pod, source, target); err == nil {
		t.Fatal("target IP collision accepted")
	}
	if len(sdn.calls) != 0 {
		t.Fatal("forwarding changed before collision was rejected")
	}
	current := &corev1.Pod{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(pod), current); err != nil {
		t.Fatal(err)
	}
	if current.Annotations[pendingMoveAnnotation] != "" || current.Annotations[networkannotation.L2SM_NETWORK_ANNOTATION] != source.Name {
		t.Fatal("failed reservation changed membership or blocked subsequent moves")
	}
}
func TestAllocationPodAttachmentRetriesAfterSDNFailure(t *testing.T) {
	store, _, _, pod := quarantineFixture(t)
	ctx := context.Background()
	sdn := &flakyAttachSDN{fail: true}
	r := &PodReconciler{Client: store.Client, APIReader: store.Reader, InternalClient: sdn, SwitchesNamespace: "default"}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pod)}
	if _, err := r.Reconcile(ctx, req); err == nil {
		t.Fatal("SDN failure ignored")
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	current := &corev1.Pod{}
	if err := store.Get(ctx, req.NamespacedName, current); err != nil {
		t.Fatal(err)
	}
	if current.Annotations[attachedNetworksAnnotation] == "" {
		t.Fatal("retry skipped attach because finalizer already existed")
	}
}

type flakyAttachSDN struct {
	fakeQuarantineSDNClient
	fail bool
}

func (s *flakyAttachSDN) AttachPodToNetwork(kind l2smv1.NetworkType, payload interface{}) error {
	if s.fail {
		s.fail = false
		return fmt.Errorf("injected attach failure")
	}
	return s.fakeQuarantineSDNClient.AttachPodToNetwork(kind, payload)
}

func TestAllocationPodDeletionReleasesMembershipAndPort(t *testing.T) {
	store, source, _, pod := quarantineFixture(t)
	ctx := context.Background()
	pod.Finalizers = []string{l2smFinalizer}
	if err := store.Update(ctx, pod); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, pod); err != nil {
		t.Fatal(err)
	}
	r := &PodReconciler{Client: store.Client, APIReader: store.Reader, InternalClient: &fakeQuarantineSDNClient{}, SwitchesNamespace: "default"}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pod)}); err != nil {
		t.Fatal(err)
	}
	current := &corev1.Pod{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(pod), current); !apierrors.IsNotFound(err) {
		t.Fatalf("pod finalizer remained: %v", err)
	}
	n := &l2smv1.L2Network{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(source), n); err != nil {
		t.Fatal(err)
	}
	if n.Status.ConnectedPodCount != 0 || len(n.Status.AssignedIPs) != 0 {
		t.Fatal("deleted pod allocation not released")
	}
}

func TestAllocationPodOperationsAreExclusiveAndOldReleaseIsHarmless(t *testing.T) {
	store, _, _, pod := quarantineFixture(t)
	ctx := context.Background()
	key := client.ObjectKeyFromObject(pod)
	_, releaseOld, err := acquirePodOperation(ctx, store, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, release, err := acquirePodOperation(ctx, store, key); err == nil {
		release()
		t.Fatal("concurrent operations on one pod were allowed")
	}
	if err := updatePod(ctx, store, key, func(p *corev1.Pod) error {
		var lease podOperation
		if err := json.Unmarshal([]byte(p.Annotations[podOperationAnnotation]), &lease); err != nil {
			return err
		}
		lease.Until = metav1.NewTime(time.Now().Add(-time.Minute))
		raw, _ := json.Marshal(lease)
		p.Annotations[podOperationAnnotation] = string(raw)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_, releaseNew, err := acquirePodOperation(ctx, store, key)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseNew()
	releaseOld()
	if _, release, err := acquirePodOperation(ctx, store, key); err == nil {
		release()
		t.Fatal("old holder released the new operation lease")
	}
}

func TestAllocationLegacyCollisionIsNotAdopted(t *testing.T) {
	store, n := allocationFixture(t, 1)
	ctx := context.Background()
	for _, name := range []string{"old-one", "old-two"} {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Annotations: map[string]string{networkannotation.MULTUS_ANNOTATION_KEY: `[{"name":"overlay-veth1","namespace":"default"}]`}}, Spec: corev1.PodSpec{NodeName: "worker"}}
		if err := store.Create(ctx, pod); err != nil {
			t.Fatal(err)
		}
	}
	key := client.ObjectKey{Namespace: "default", Name: "overlay-veth1"}
	if err := store.adoptPort(ctx, key, "worker", podPortOwner("default", "old-one", "uid", n.Name), func(pod *corev1.Pod) bool { return pod.Name == "old-one" }); err == nil {
		t.Fatal("conflicting legacy consumers adopted as exclusive ownership")
	}
}

func TestAllocationNetworkDeletionWaitsForIDSPods(t *testing.T) {
	store, n := allocationFixture(t, 1)
	ctx := context.Background()
	if err := store.Get(ctx, client.ObjectKeyFromObject(n), n); err != nil {
		t.Fatal(err)
	}
	n.Spec.Ids = &l2smv1.IdsRules{Enabled: true, Node: "worker", Profile: "suricata"}
	n.Finalizers = []string{l2smFinalizer}
	if err := store.Update(ctx, n); err != nil {
		t.Fatal(err)
	}
	r := &L2NetworkReconciler{Client: store.Client, APIReader: store.Reader, Scheme: store.Scheme(), SwitchesNamespace: "default", InternalClient: &fakeQuarantineSDNClient{}}
	if err := r.ensureIDS(ctx, n); err != nil {
		t.Fatal(err)
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: n.Name + "-ids-pod", Namespace: "default", Labels: map[string]string{"l2sm/component": "ids"}}}
	if err := store.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, n); err != nil {
		t.Fatal(err)
	}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(n)}
	for i := 0; i < 2; i++ {
		result, err := r.Reconcile(ctx, req)
		if err != nil || result.RequeueAfter == 0 {
			t.Fatalf("IDS deletion did not wait: %+v %v", result, err)
		}
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "new", "id", "other")); err == nil {
		t.Fatal("IDS port released while its pod still exists")
	}
	if err := store.Delete(ctx, pod); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := store.reservePort(ctx, "default", "worker", podPortOwner("default", "new", "id", "other")); err != nil {
		t.Fatal("network deletion leaked IDS port", err)
	}
}

func TestAllocationPodReconciliationCompletesMoveWithoutRequest(t *testing.T) {
	store, source, target, pod := quarantineFixture(t)
	ctx := context.Background()
	failing := &failingSourceClient{Client: store.Client, fail: true}
	mover := &QuarantinePodRequestReconciler{Client: failing, APIReader: store.Reader, InternalClient: &fakeQuarantineSDNClient{}}
	if _, err := mover.movePodToTargetNetwork(ctx, pod, source, target); err == nil {
		t.Fatal("injected interrupted move did not fail")
	}
	// No QuarantinePodRequest exists. The pod's durable journal is sufficient.
	reconciler := &PodReconciler{Client: store.Client, APIReader: store.Reader, InternalClient: &fakeQuarantineSDNClient{}, SwitchesNamespace: "default"}
	if _, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(pod)}); err != nil {
		t.Fatal(err)
	}
	current := &corev1.Pod{}
	if err := store.Get(ctx, client.ObjectKeyFromObject(pod), current); err != nil {
		t.Fatal(err)
	}
	if current.Annotations[pendingMoveAnnotation] != "" {
		t.Fatal("move was not recovered from pod journal")
	}
	if err := store.Get(ctx, client.ObjectKeyFromObject(source), source); err != nil {
		t.Fatal(err)
	}
	if source.Status.ConnectedPodCount != 0 {
		t.Fatal("source membership not repaired")
	}
}

func TestAllocationCompletedMoveReceiptPreservesRequestCounts(t *testing.T) {
	store, source, target, pod := quarantineFixture(t)
	ctx := context.Background()
	sdn := &fakeQuarantineSDNClient{}
	mover := &QuarantinePodRequestReconciler{Client: store.Client, APIReader: store.Reader, InternalClient: sdn}
	for i := 0; i < 2; i++ {
		moved, err := mover.movePodToTargetNetwork(ctx, pod, source, target, "request-uid")
		if err != nil || !moved {
			t.Fatalf("request lost completed move: %v %v", moved, err)
		}
	}
	if len(sdn.calls) != 2 {
		t.Fatal("retry repeated SDN operations instead of using receipt")
	}
	moved, err := mover.movePodToTargetNetwork(ctx, pod, source, target, "other-request")
	if err != nil || moved {
		t.Fatalf("another request claimed the completed move: %v %v", moved, err)
	}
}

func TestAllocationIDSRetryWithoutSwitchesNamespace(t *testing.T) {
	store, n := allocationFixture(t, 1)
	ctx := context.Background()
	n.Spec.Ids = &l2smv1.IdsRules{Enabled: true, Node: "worker", Profile: "suricata"}
	r := &L2NetworkReconciler{Client: store.Client, APIReader: store.Reader, Scheme: store.Scheme(), InternalClient: &fakeQuarantineSDNClient{}}
	if err := r.ensureIDS(ctx, n); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureIDS(ctx, n); err != nil {
		t.Fatalf("IDS retry lost explicit NAD namespace with unset SWITCHES_NAMESPACE: %v", err)
	}
	dep := &appsv1.Deployment{}
	if err := store.Get(ctx, client.ObjectKey{Namespace: n.Namespace, Name: n.Name + "-ids"}, dep); err != nil {
		t.Fatal(err)
	}
	var attachments []networkannotation.NetworkAnnotation
	if err := json.Unmarshal([]byte(dep.Spec.Template.Annotations[networkannotation.MULTUS_ANNOTATION_KEY]), &attachments); err != nil {
		t.Fatal(err)
	}
	attachments[0].Namespace = ""
	dep.Spec.Template.Annotations[networkannotation.MULTUS_ANNOTATION_KEY] = networkannotation.MultusAnnotationToString(attachments)
	if err := store.Update(ctx, dep); err != nil {
		t.Fatal(err)
	}
	if err := r.ensureIDS(ctx, n); err != nil {
		t.Fatalf("IDS retry did not default omitted namespace to deployment namespace: %v", err)
	}
}
