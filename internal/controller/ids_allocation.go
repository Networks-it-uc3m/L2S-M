package controller

import (
	"context"
	"fmt"
	"strings"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/ids"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *L2NetworkReconciler) ensureIDS(ctx context.Context, n *l2smv1.L2Network) error {
	if n.Spec.Ids == nil || !n.Spec.Ids.Enabled {
		return nil
	}
	if n.Spec.Ids.Namespace != "" && n.Spec.Ids.Namespace != n.Namespace {
		return fmt.Errorf("IDS namespace must match its owning L2Network namespace")
	}
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	existing := &appsv1.Deployment{}
	err := store.reader().Get(ctx, client.ObjectKey{Namespace: n.Namespace, Name: utils.GenerateIdsDeployname(n.Name)}, existing)
	if err == nil {
		if !metav1.IsControlledBy(existing, n) {
			return fmt.Errorf("IDS deployment %s is owned by another resource", existing.Name)
		}
		if existing.Spec.Template.Spec.NodeName != n.Spec.Ids.Node {
			return fmt.Errorf("changing the IDS node requires recreating its deployment")
		}
		attachments, err := networkannotation.ExtractNetworks(existing.Spec.Template.Annotations[networkannotation.MULTUS_ANNOTATION_KEY], existing.Namespace)
		if err != nil || len(attachments) != 1 {
			return fmt.Errorf("IDS deployment has invalid attachment mapping: %v", err)
		}
		key := client.ObjectKey{Namespace: attachments[0].Namespace, Name: attachments[0].Name}
		return store.adoptPort(ctx, key, n.Spec.Ids.Node, networkPortOwner(n), func(pod *corev1.Pod) bool {
			for _, ref := range pod.OwnerReferences {
				if ref.Kind != "ReplicaSet" {
					continue
				}
				rs := &appsv1.ReplicaSet{}
				if err := store.reader().Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: ref.Name}, rs); err == nil && rs.UID == ref.UID && metav1.IsControlledBy(rs, existing) {
					return true
				}
			}
			return false
		})
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	nad, err := store.reservePort(ctx, r.SwitchesNamespace, n.Spec.Ids.Node, networkPortOwner(n))
	if err != nil {
		return err
	}

	// Reserve first, then touch ONOS, then create resources. A failure retains the
	// network-owned claim so the next reconcile resumes on exactly the same port.
	port, err := utils.GetPortNumberFromNetAttachDef(nad.Name)
	if err != nil {
		return err
	}
	ofPort := fmt.Sprintf("of:%s/%s", dp.GenerateID(dp.GetSwitchName(dp.DatapathParams{NodeName: n.Spec.Ids.Node, ProviderName: l2smv1.OVERLAY_PROVIDER})), port)
	if err := r.InternalClient.SetUpMirrorPort(n.Spec.Type, sdnclient.VnetPayload{NetworkId: n.Name, MirrorPort: ofPort}); err != nil {
		return err
	}
	attachment := networkannotation.NetworkAnnotation{Name: nad.Name, Namespace: nad.Namespace}
	attachment.GenerateIPv6Address()
	resources, err := ids.GenerateExternalResources(n, networkannotation.MultusAnnotationToString([]networkannotation.NetworkAnnotation{attachment}))
	if err != nil {
		return err
	}
	for _, obj := range resources {
		if err := controllerutil.SetControllerReference(n, obj, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, obj); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return err
			}
			// ConfigMap creation may have succeeded before a previous crash.
			existing := obj.DeepCopyObject().(client.Object)
			if err := store.reader().Get(ctx, client.ObjectKeyFromObject(obj), existing); err != nil {
				return err
			}
			if !metav1.IsControlledBy(existing, n) {
				return fmt.Errorf("IDS resource %s has a different owner", obj.GetName())
			}
		}
	}
	return nil
}

func (r *L2NetworkReconciler) deleteIDS(ctx context.Context, n *l2smv1.L2Network) (bool, error) {
	store := allocationStore{Client: r.Client, Reader: r.APIReader}
	// Do not depend on the current IDS spec: it may have been disabled/changed.
	deployments := &appsv1.DeploymentList{}
	if err := store.reader().List(ctx, deployments, client.InNamespace(n.Namespace)); err != nil {
		return false, err
	}
	for i := range deployments.Items {
		dep := &deployments.Items[i]
		if !metav1.IsControlledBy(dep, n) {
			continue
		}
		policy := metav1.DeletePropagationForeground
		if err := r.Delete(ctx, dep, &client.DeleteOptions{PropagationPolicy: &policy}); client.IgnoreNotFound(err) != nil {
			return false, err
		}
		return false, nil
	}
	pods := &corev1.PodList{}
	if err := store.reader().List(ctx, pods, client.InNamespace(n.Namespace), client.MatchingLabels{"l2sm/component": "ids"}); err != nil {
		return false, err
	}
	// Foreground deletion normally guarantees this; keep the reservation if pods
	// are still visible after an interrupted/deferred garbage collection.
	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, utils.GenerateIdsDeployname(n.Name)+"-") {
			return false, nil
		}
	}

	return true, nil
}
