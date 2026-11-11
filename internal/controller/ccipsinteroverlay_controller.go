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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CcipsInterOverlayReconciler reconciles a CcipsInterOverlay object
type CcipsInterOverlayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsinteroverlays,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsinteroverlays/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsinteroverlays/finalizers,verbs=update

func interConfigMapName(ccips *l2smv1.CcipsInterOverlay) string {
	return ccips.Name + "-inter-tunnels"
}

func (r *CcipsInterOverlayReconciler) getStoredTunnels(
	ctx context.Context,
	ccips *l2smv1.CcipsInterOverlay,
) (*corev1.ConfigMap, StoredTunnelMap, error) {

	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      interConfigMapName(ccips),
		Namespace: ccips.Namespace,
	}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      interConfigMapName(ccips),
					Namespace: ccips.Namespace,
				},
				Data: map[string]string{},
			}
			return cm, StoredTunnelMap{}, nil
		}
		return nil, nil, err
	}
	stored := StoredTunnelMap{}
	if raw, ok := cm.Data["tunnels.json"]; ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			return nil, nil, err
		}
	}
	return cm, stored, nil
}

func (r *CcipsInterOverlayReconciler) saveStoredTunnels(
	ctx context.Context,
	ccips *l2smv1.CcipsInterOverlay,
	cm *corev1.ConfigMap,
	stored StoredTunnelMap,
) error {

	raw, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["tunnels.json"] = string(raw)
	if err := controllerutil.SetControllerReference(ccips, cm, r.Scheme); err != nil {
		return err
	}
	if cm.ResourceVersion == "" {
		return r.Create(ctx, cm)
	}
	return r.Update(ctx, cm)

}

func (r *CcipsInterOverlayReconciler) deleteOneStoredTunnel(
	ctx context.Context,
	ccips *l2smv1.CcipsInterOverlay,
	log logr.Logger,
) (bool, error) {
	cm, storedTunnels, err := r.getStoredTunnels(ctx, ccips)
	if err != nil {
		return false, err
	}

	// Nothing left to delete
	if len(storedTunnels) == 0 {
		log.Info("no stored tunnels left to delete")
		return true, nil
	}

	// Pick one tunnel
	for key, tunnel := range storedTunnels {
		if tunnel.ID == "" {
			log.Info("stored tunnel has empty ID, removing from configmap", "key", key)
			delete(storedTunnels, key)
			if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
				return false, err
			}
			return len(storedTunnels) == 0, nil
		}

		url := fmt.Sprintf("http://%s:5000/ccips/%s", ccips.Spec.ControllerIP, tunnel.ID)

		httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
		if err != nil {
			return false, err
		}

		httpReq.Header.Set("Accept", "application/json")

		httpClient := &http.Client{Timeout: 60 * time.Second}
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return false, err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return false, readErr
		}

		log.Info("server delete response",
			"url", url,
			"status", resp.StatusCode,
			"body", string(body),
		)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return false, fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, string(body))
		}

		// Remove the stored entry after the external tunnel has been deleted.
		delete(storedTunnels, key)

		if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
			return false, err
		}

		return len(storedTunnels) == 0, nil
	}

	return true, nil
}

func (r *CcipsInterOverlayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ccips := &l2smv1.CcipsInterOverlay{}
	if err := r.Get(ctx, req.NamespacedName, ccips); err != nil {
		log.Error(err, "unable to fetch CcipsInterOverlay")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 1. Deletion path
	if !ccips.GetDeletionTimestamp().IsZero() {

		if controllerutil.ContainsFinalizer(ccips, l2smFinalizer) {
			done, err := r.deleteOneStoredTunnel(ctx, ccips, log)
			if err != nil {
				log.Error(err, "failed to delete external tunnel")
				return ctrl.Result{}, err
			}
			if !done {
				return ctrl.Result{Requeue: true}, nil
			}
			controllerutil.RemoveFinalizer(ccips, l2smFinalizer)
			if err := r.Update(ctx, ccips); err != nil {
				log.Error(err, "couldn't remove finalizer from ccips")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 2. Ensure finalizer exists
	if !controllerutil.ContainsFinalizer(ccips, l2smFinalizer) {
		controllerutil.AddFinalizer(ccips, l2smFinalizer)
		if err := r.Update(ctx, ccips); err != nil {
			return ctrl.Result{}, err
		}

		// Return here and let the next reconcile do the real work
		return ctrl.Result{}, nil
	}

	cm, storedTunnels, err := r.getStoredTunnels(ctx, ccips)

	if err != nil {
		return ctrl.Result{}, err
	}

	allStored := true

	for _, tunnel := range ccips.Spec.Tunnels {
		key := tunnelKey(tunnel.EndpointA, tunnel.EndpointB)
		if _, exists := storedTunnels[key]; !exists {
			allStored = false
			break
		}
	}

	if allStored {
		log.Info("all tunnels already stored, skipping")
		return ctrl.Result{}, nil
	}

	// 3. Normal reconcile logic starts here

	// Resolve cluster names to the explicitly configured CCIPs agent addresses.
	ipAddr := make(map[string]string)
	for _, cluster := range ccips.Spec.Clusters {
		if cluster.Name == "" || cluster.Address == "" {
			return ctrl.Result{}, fmt.Errorf("cluster name and CCIPs agent address must be non-empty")
		}
		if _, exists := ipAddr[cluster.Name]; exists {
			return ctrl.Result{}, fmt.Errorf("duplicate cluster name %s", cluster.Name)
		}
		ipAddr[cluster.Name] = cluster.Address
	}

	// Build and send one request per tunnel
	for _, tunnel := range ccips.Spec.Tunnels {
		endpointA := tunnel.EndpointA
		endpointB := tunnel.EndpointB

		key := tunnelKey(endpointA, endpointB)
		// Skip if already stored
		if existing, exists := storedTunnels[key]; exists {
			switch existing.State {
			case "Created":
				log.Info("tunnel already created, skipping", "key", key, "id", existing.ID)
				continue
			case "Pending":
				log.Info("tunnel creation already in progress, skipping", "key", key)
				continue
			}
		}

		ipA, ok := ipAddr[endpointA]
		if !ok || ipA == "" {
			return ctrl.Result{}, fmt.Errorf("missing IP for endpointA %s", endpointA)
		}

		ipB, ok := ipAddr[endpointB]
		if !ok || ipB == "" {
			return ctrl.Result{}, fmt.Errorf("missing IP for endpointB %s", endpointB)
		}

		//Save tunnel to pending state before sending request, to avoid creating duplicates if reconcile is triggered again before we update the stored state with the new tunnel ID
		storedTunnels[key] = StoredTunnel{
			EndpointA: endpointA,
			EndpointB: endpointB,
			State:     "Pending",
		}

		if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
			return ctrl.Result{}, err
		}

		reqBody := TunnelRequest{
			Nodes: []RequestNode{
				{IPData: ipA, IPControl: ipA},
				{IPData: ipB, IPControl: ipB},
			},
			EncAlg:       []string{"aes-cbc"},
			IntAlg:       []string{"sha2-256"},
			SoftLifetime: Lifetime{NTime: 15},
			HardLifetime: Lifetime{NTime: 30},
		}

		url := fmt.Sprintf("http://%s:5000/ccips", ccips.Spec.ControllerIP)

		jsonBody, err := json.Marshal(reqBody)

		if err != nil {
			return ctrl.Result{}, err
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))

		if err != nil {
			return ctrl.Result{}, err
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(httpReq)

		if err != nil {
			log.Error(err, "failed to send request", "url", url)
			return ctrl.Result{}, err
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return ctrl.Result{}, readErr
		}

		log.Info("server response",
			"url", url,
			"status", resp.StatusCode,
			"body", string(body),
			"endpointA", endpointA,
			"endpointB", endpointB,
		)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return ctrl.Result{}, fmt.Errorf(
				"controller returned status %d for tunnel %s-%s: %s",
				resp.StatusCode, endpointA, endpointB, string(body),
			)
		}

		var createResp CreateTunnelResponse
		if err := json.Unmarshal(body, &createResp); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse controller response for tunnel %s-%s: %w", endpointA, endpointB, err)
		}

		if createResp.ID == "" {
			return ctrl.Result{}, fmt.Errorf("controller response missing id for tunnel %s-%s: %s", endpointA, endpointB, string(body))
		}

		storedTunnels[key] = StoredTunnel{
			ID:        createResp.ID,
			EndpointA: endpointA,
			EndpointB: endpointB,
			State:     "Created",
		}

	}

	if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CcipsInterOverlayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.CcipsInterOverlay{}).
		Named("ccipsinteroverlay").
		Complete(r)
}
