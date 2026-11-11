package controller

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const allocationIDAnnotation = "l2sm.l2sm.k8s.local/allocation-id"
const reservationPrefix = "l2sm.l2sm.k8s.local/port-"

var allocationBackoff = wait.Backoff{Steps: 16, Duration: 5 * time.Millisecond, Factor: 1.5, Jitter: .1, Cap: 200 * time.Millisecond}

type allocationStore struct {
	client.Client
	Reader client.Reader
}

func (a allocationStore) reader() client.Reader {
	if a.Reader != nil {
		return a.Reader
	}
	return a.Client
}

type portOwner struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	ID        string `json:"id"`
	Scope     string `json:"scope,omitempty"`
}
type portReservation struct {
	Owner     portOwner   `json:"owner"`
	Node      string      `json:"node"`
	CreatedAt metav1.Time `json:"createdAt"`
}

func portKey(node string) string {
	return fmt.Sprintf("%s%x", reservationPrefix, sha256.Sum256([]byte(node)))[:len(reservationPrefix)+32]
}
func podPortOwner(podNamespace, podName, id, network string) portOwner {
	return portOwner{Kind: "Pod", Namespace: podNamespace, Name: podName, ID: id, Scope: network}
}
func networkPortOwner(n *l2smv1.L2Network) portOwner {
	return portOwner{Kind: "L2Network", Namespace: n.Namespace, Name: n.Name, ID: string(n.UID)}
}
func (a allocationStore) updateNetwork(ctx context.Context, key client.ObjectKey, change func(*l2smv1.L2Network) error) error {
	return retry.RetryOnConflict(allocationBackoff, func() error {
		n := &l2smv1.L2Network{}
		if err := a.reader().Get(ctx, key, n); err != nil {
			return err
		}
		before := n.DeepCopy().Status
		if err := change(n); err != nil {
			return err
		}
		if reflect.DeepEqual(before, n.Status) {
			return nil
		}
		expected := len(n.Status.PodAllocations)
		if err := a.Status().Update(ctx, n); err != nil {
			return err
		}
		if expected > 0 && len(n.Status.PodAllocations) != expected {
			return fmt.Errorf("L2Network CRD does not preserve podAllocations; install the updated CRD before running this controller")
		}
		return nil
	})
}

// reservePort uses resourceVersion as a compare-and-swap. Both IDS and pod
// admission MUST use this operation before exposing an attachment to consumers.
func (a allocationStore) reservePort(ctx context.Context, namespace, node string, owner portOwner) (*nettypes.NetworkAttachmentDefinition, error) {
	var chosen *nettypes.NetworkAttachmentDefinition
	if owner.ID == "" {
		return nil, fmt.Errorf("port reservation requires an incarnation ID")
	}
	err := retry.RetryOnConflict(allocationBackoff, func() error {
		list := &nettypes.NetworkAttachmentDefinitionList{}
		if err := a.reader().List(ctx, list, client.InNamespace(namespace), client.MatchingLabels{"app": "l2sm"}); err != nil {
			return err
		}
		sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].Name < list.Items[j].Name })
		label, key := networkannotation.NET_ATTACH_LABEL_PREFIX+node, portKey(node)
		// Find our existing reservation first: admission/reconcile retries reuse it.
		for i := range list.Items {
			n := &list.Items[i]
			if raw := n.Annotations[key]; raw != "" {
				var r portReservation
				if err := json.Unmarshal([]byte(raw), &r); err != nil {
					return fmt.Errorf("invalid reservation on %s: %w", n.Name, err)
				}
				if r.Owner == owner {
					if n.DeletionTimestamp != nil {
						return fmt.Errorf("reserved attachment %s is deleting", n.Name)
					}
					chosen = n.DeepCopy()
					return nil
				}
			}
		}
		var conflict error
		for i := range list.Items {
			n := &list.Items[i]
			// Fresh overlay NADs have no used-<node> label until first allocation.
			used := n.Labels[label]
			if used == "true" || n.Annotations[key] != "" || n.DeletionTimestamp != nil {
				continue
			}
			base := n.DeepCopy()
			if n.Annotations == nil {
				n.Annotations = map[string]string{}
			}
			raw, _ := json.Marshal(portReservation{Owner: owner, Node: node, CreatedAt: metav1.Now()})
			n.Annotations[key], n.Labels[label] = string(raw), "true"
			// Optimistic merge patch avoids overwriting reservations on other nodes.
			err := a.Patch(ctx, n, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
			if apierrors.IsConflict(err) {
				conflict = err
				continue
			}
			if err != nil {
				return err
			}
			chosen = n.DeepCopy()
			return nil
		}
		if conflict != nil {
			return conflict
		}
		return fmt.Errorf("no free overlay attachment on node %s", node)
	})
	return chosen, err
}

// Releasing requires an exact owner match. A delayed cleanup cannot free a port
// that has since been allocated to a different pod or network incarnation.
func (a allocationStore) releasePort(ctx context.Context, key client.ObjectKey, node string, owner portOwner) error {
	err := retry.RetryOnConflict(allocationBackoff, func() error {
		n := &nettypes.NetworkAttachmentDefinition{}
		if err := a.reader().Get(ctx, key, n); err != nil {
			return err
		}
		raw := n.Annotations[portKey(node)]
		if raw == "" {
			return nil
		} // Legacy reservations are never freed blindly.
		var r portReservation
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return err
		}
		if r.Owner != owner {
			return nil
		}
		base := n.DeepCopy()
		delete(n.Annotations, portKey(node))
		if n.Labels == nil {
			n.Labels = map[string]string{}
		}
		n.Labels[networkannotation.NET_ATTACH_LABEL_PREFIX+node] = "false"
		return a.Patch(ctx, n, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
	})
	return client.IgnoreNotFound(err)
}

func (a allocationStore) reserveIPs(ctx context.Context, key client.ObjectKey, name, id string, requested []string, migrating ...bool) ([]string, error) {
	var ips []string
	err := a.updateNetwork(ctx, key, func(n *l2smv1.L2Network) error {
		if n.DeletionTimestamp != nil && (len(migrating) == 0 || !migrating[0]) {
			return fmt.Errorf("network %s is deleting", n.Name)
		}
		if previous, ok := n.Status.PodAllocations[name]; ok {
			if previous.ID != id {
				return fmt.Errorf("pod %s already has a reservation from another admission", name)
			}
			ips = append([]string(nil), previous.IPAddresses...)
			return nil
		}
		ips = append([]string(nil), requested...)
		if len(ips) == 0 && n.Spec.NetworkCIDR != "" {
			_, subnet, err := net.ParseCIDR(n.Spec.NetworkCIDR)
			if err != nil {
				return err
			}
			mask, _ := subnet.Mask.Size()
			pool := n.Spec.PodAddressRange
			if pool == "" {
				pool = n.Spec.NetworkCIDR
			}
			next, _, err := GetNextAvailableIP(pool, n.Status.LastAssignedIP, n.Status.AssignedIPs)
			if err != nil {
				return err
			}
			ips = []string{fmt.Sprintf("%s/%d", next, mask)}
			n.Status.LastAssignedIP = next
		}
		if n.Status.AssignedIPs == nil {
			n.Status.AssignedIPs = map[string]string{}
		}
		for _, cidr := range ips {
			ip, _, err := net.ParseCIDR(cidr)
			if err != nil {
				return fmt.Errorf("invalid pod address %q: %w", cidr, err)
			}
			if previous, exists := n.Status.AssignedIPs[ip.String()]; exists && previous != name {
				return fmt.Errorf("address %s already reserved by %s", ip, previous)
			}
			n.Status.AssignedIPs[ip.String()] = name
		}
		if n.Status.PodAllocations == nil {
			n.Status.PodAllocations = map[string]l2smv1.PodAllocation{}
		}
		n.Status.PodAllocations[name] = l2smv1.PodAllocation{ID: id, IPAddresses: ips, CreatedAt: metav1.Now()}
		// A legacy pod was already counted by the old admission webhook.
		if len(migrating) == 0 || !migrating[0] {
			n.Status.ConnectedPodCount++
		}
		return nil
	})
	return ips, err
}
func (a allocationStore) releaseIPs(ctx context.Context, key client.ObjectKey, name, id string) error {
	return client.IgnoreNotFound(a.updateNetwork(ctx, key, func(n *l2smv1.L2Network) error {
		entry, exists := n.Status.PodAllocations[name]
		if exists && entry.ID != id {
			return nil
		}
		removed := exists
		// Legacy maps can be cleaned using their recorded pod owner, but must never
		// remove another pod's address when repairing old inconsistent state.
		for ip, owner := range n.Status.AssignedIPs {
			if owner == name {
				delete(n.Status.AssignedIPs, ip)
				removed = true
			}
		}
		delete(n.Status.PodAllocations, name)
		if removed && n.Status.ConnectedPodCount > 0 {
			n.Status.ConnectedPodCount--
		}
		return nil
	}))
}

func conflict(resource, name string) error {
	return apierrors.NewConflict(schema.GroupResource{Group: l2smv1.GroupVersion.Group, Resource: resource}, name, fmt.Errorf("allocation changed"))
}

func (a allocationStore) releaseOwnedPorts(ctx context.Context, namespace string, owner portOwner) error {
	list := &nettypes.NetworkAttachmentDefinitionList{}
	if err := a.reader().List(ctx, list, client.InNamespace(namespace), client.MatchingLabels{"app": "l2sm"}); err != nil {
		return err
	}
	for _, nad := range list.Items {
		for key, raw := range nad.Annotations {
			if !strings.HasPrefix(key, reservationPrefix) {
				continue
			}
			var reservation portReservation
			if err := json.Unmarshal([]byte(raw), &reservation); err != nil {
				return err
			}
			actual := reservation.Owner
			if actual.Kind == owner.Kind && actual.Namespace == owner.Namespace && actual.Name == owner.Name && actual.ID == owner.ID {
				if err := a.releasePort(ctx, client.ObjectKeyFromObject(&nad), reservation.Node, actual); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

const attachedNetworksAnnotation = "l2sm.l2sm.k8s.local/attached-networks"

func podAllocationID(pod *corev1.Pod) string {
	if id := pod.Annotations[allocationIDAnnotation]; id != "" {
		return id
	}
	return string(pod.UID)
}
func updatePod(ctx context.Context, store allocationStore, key client.ObjectKey, change func(*corev1.Pod) error) error {
	return retry.RetryOnConflict(allocationBackoff, func() error {
		pod := &corev1.Pod{}
		if err := store.reader().Get(ctx, key, pod); err != nil {
			return err
		}
		base := pod.DeepCopy()
		if err := change(pod); err != nil {
			return err
		}
		return store.Patch(ctx, pod, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
	})
}

// Adopt an existing legacy attachment only after verifying all its consumers.
// Switch pods intentionally expose every overlay NAD and are not consumers of
// the workload-side reservation. Conflicting legacy attachments require repair.
func (a allocationStore) adoptPort(ctx context.Context, key client.ObjectKey, node string, owner portOwner, allowed func(*corev1.Pod) bool) error {
	return retry.RetryOnConflict(allocationBackoff, func() error {
		nad := &nettypes.NetworkAttachmentDefinition{}
		if err := a.reader().Get(ctx, key, nad); err != nil {
			return err
		}
		if raw := nad.Annotations[portKey(node)]; raw != "" {
			var reservation portReservation
			if err := json.Unmarshal([]byte(raw), &reservation); err != nil {
				return err
			}
			if reservation.Owner == owner {
				return nil
			}
			return fmt.Errorf("attachment %s on %s is reserved by %s/%s", nad.Name, node, reservation.Owner.Namespace, reservation.Owner.Name)
		}
		pods := &corev1.PodList{}
		if err := a.reader().List(ctx, pods); err != nil {
			return err
		}
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Spec.NodeName != node {
				continue
			}
			switchPod := false
			for _, container := range pod.Spec.Containers {
				if container.Name == "l2sm-switch" {
					switchPod = true
				}
			}
			if switchPod {
				continue
			}
			raw := pod.Annotations[networkannotation.MULTUS_ANNOTATION_KEY]
			if raw == "" {
				continue
			}
			attachments, err := networkannotation.ExtractNetworks(raw, pod.Namespace)
			if err != nil {
				return err
			}
			for _, attachment := range attachments {
				if attachment.Name == nad.Name && attachment.Namespace == nad.Namespace && !allowed(pod) {
					return fmt.Errorf("legacy attachment collision: %s/%s and %s/%s share %s on %s", owner.Namespace, owner.Name, pod.Namespace, pod.Name, nad.Name, node)
				}
			}
		}
		base := nad.DeepCopy()
		if nad.Annotations == nil {
			nad.Annotations = map[string]string{}
		}
		if nad.Labels == nil {
			nad.Labels = map[string]string{}
		}
		raw, _ := json.Marshal(portReservation{Owner: owner, Node: node, CreatedAt: metav1.Now()})
		nad.Annotations[portKey(node)] = string(raw)
		nad.Labels[networkannotation.NET_ATTACH_LABEL_PREFIX+node] = "true"
		return a.Patch(ctx, nad, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{}))
	})
}
