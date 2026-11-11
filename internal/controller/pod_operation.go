package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const podOperationAnnotation = "l2sm.l2sm.k8s.local/operation"
const podOperationTTL = 2 * time.Minute

type podOperation struct {
	ID    string      `json:"id"`
	Until metav1.Time `json:"until"`
}

// Kubernetes CAS serializes SDN changes for one pod across reconcile requests
// and controller processes. The lease bounds recovery after a process crash.
// Each operation uses a one-minute deadline; SDN HTTP calls have a 10s timeout.
func acquirePodOperation(ctx context.Context, store allocationStore, key client.ObjectKey) (context.Context, func(), error) {
	id := string(uuid.NewUUID())
	err := updatePod(ctx, store, key, func(pod *corev1.Pod) error {
		if raw := pod.Annotations[podOperationAnnotation]; raw != "" {
			var current podOperation
			if err := json.Unmarshal([]byte(raw), &current); err != nil {
				return err
			}
			if time.Now().Before(current.Until.Time) {
				return fmt.Errorf("pod %s has an active network operation; retry", pod.Name)
			}
		}
		if pod.Annotations == nil {
			pod.Annotations = map[string]string{}
		}
		raw, _ := json.Marshal(podOperation{ID: id, Until: metav1.NewTime(time.Now().Add(podOperationTTL))})
		pod.Annotations[podOperationAnnotation] = string(raw)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	operationCtx, cancel := context.WithTimeout(ctx, time.Minute)
	release := func() {
		cancel()
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		err := updatePod(cleanup, store, key, func(pod *corev1.Pod) error {
			var current podOperation
			if raw := pod.Annotations[podOperationAnnotation]; raw != "" {
				if err := json.Unmarshal([]byte(raw), &current); err != nil {
					return err
				}
			}
			if current.ID == id {
				delete(pod.Annotations, podOperationAnnotation)
			}
			return nil
		})
		if err = client.IgnoreNotFound(err); err != nil {
			log.FromContext(ctx).Error(err, "could not release pod operation")
		}
	}
	return operationCtx, release, nil
}
