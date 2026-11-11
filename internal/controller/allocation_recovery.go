package controller

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Admission writes reservations before the API server persists the Pod. If a
// later webhook rejects that Pod, no Pod event can trigger cleanup. This sweep
// also handles process crashes between the reservation and admission response.
// The grace exceeds Kubernetes' maximum admission webhook timeout (30 seconds).
const abandonedAdmissionGrace = 2 * time.Minute

type AllocationRecovery struct {
	Client            client.Client
	APIReader         client.Reader
	SwitchesNamespace string
}

func (*AllocationRecovery) NeedLeaderElection() bool { return true }
func (r *AllocationRecovery) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.sweep(ctx, time.Now()); err != nil {
				log.FromContext(ctx).Error(err, "allocation recovery failed")
			}
		}
	}
}
func (r *AllocationRecovery) sweep(ctx context.Context, now time.Time) error {
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	networks := &l2smv1.L2NetworkList{}
	if err := store.reader().List(ctx, networks); err != nil {
		return err
	}
	for _, network := range networks.Items {
		for name, reservation := range network.Status.PodAllocations {
			if now.Sub(reservation.CreatedAt.Time) < abandonedAdmissionGrace {
				continue
			}
			pod := &corev1.Pod{}
			err := store.reader().Get(ctx, client.ObjectKey{Namespace: network.Namespace, Name: name}, pod)
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			if err == nil && podAllocationID(pod) == reservation.ID {
				continue
			}
			if err := store.releaseIPs(ctx, client.ObjectKeyFromObject(&network), name, reservation.ID); err != nil {
				return err
			}
		}
	}
	nads := &nettypes.NetworkAttachmentDefinitionList{}
	if err := store.reader().List(ctx, nads, client.InNamespace(r.SwitchesNamespace), client.MatchingLabels{"app": "l2sm"}); err != nil {
		return err
	}
	for _, nad := range nads.Items {
		for key, raw := range nad.Annotations {
			if !strings.HasPrefix(key, reservationPrefix) {
				continue
			}
			var reservation portReservation
			if err := json.Unmarshal([]byte(raw), &reservation); err != nil {
				return err
			}
			if now.Sub(reservation.CreatedAt.Time) < abandonedAdmissionGrace {
				continue
			}
			owner := reservation.Owner
			switch owner.Kind {
			case "Pod":
				pod := &corev1.Pod{}
				err := store.reader().Get(ctx, client.ObjectKey{Namespace: owner.Namespace, Name: owner.Name}, pod)
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}
				if err == nil && podAllocationID(pod) == owner.ID {
					continue
				}
			case "L2Network":
				network := &l2smv1.L2Network{}
				err := store.reader().Get(ctx, client.ObjectKey{Namespace: owner.Namespace, Name: owner.Name}, network)
				if err != nil && !apierrors.IsNotFound(err) {
					return err
				}
				if err == nil && string(network.UID) == owner.ID {
					continue
				}
				// Network deletion normally waits for foreground IDS garbage collection.
				// Be conservative if someone force-removed a finalizer.
				pods := &corev1.PodList{}
				if err := store.reader().List(ctx, pods, client.InNamespace(owner.Namespace), client.MatchingLabels{"l2sm/component": "ids"}); err != nil {
					return err
				}
				inUse := false
				for _, pod := range pods.Items {
					if strings.HasPrefix(pod.Name, owner.Name+"-ids-") {
						inUse = true
					}
				}
				if inUse {
					continue
				}
			default:
				continue
			}
			if err := store.releasePort(ctx, client.ObjectKeyFromObject(&nad), reservation.Node, owner); err != nil {
				return err
			}
		}
	}
	return nil
}
