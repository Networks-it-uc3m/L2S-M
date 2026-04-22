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
	"math/rand"
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

// CcipsOverlayReconciler reconciles a CcipsOverlay object
type CcipsOverlayReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type Lifetime struct {
	NTime int `json:"nTime"`
}

type RequestNode struct {
	IPData    string `json:"ipData"`
	IPControl string `json:"ipControl"`
}

type TunnelRequest struct {
	Nodes        []RequestNode `json:"nodes"`
	EncAlg       []string      `json:"encAlg"`
	IntAlg       []string      `json:"intAlg"`
	SoftLifetime Lifetime      `json:"softLifetime"`
	HardLifetime Lifetime      `json:"hardLifetime"`
}

type StoredTunnel struct {
	ID        string `json:"id"`
	EndpointA string `json:"endpointA"`
	EndpointB string `json:"endpointB"`
}

type StoredTunnelMap map[string]StoredTunnel

type CreateTunnelResponse struct {
	ID string `json:"id"`
}

// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsoverlays,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsoverlays/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=l2sm.l2sm.k8s.local,resources=ccipsoverlays/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CcipsOverlay object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile

func tunnelKey(a, b string) string {
	return a + "__" + b
}

func randomID(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func configMapName(ccips *l2smv1.CcipsOverlay) string {
	return ccips.Name + "-tunnels"
}

func (r *CcipsOverlayReconciler) getStoredTunnels(
	ctx context.Context,
	ccips *l2smv1.CcipsOverlay,
) (*corev1.ConfigMap, StoredTunnelMap, error) {

	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      configMapName(ccips),
		Namespace: ccips.Namespace,
	}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName(ccips),
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

func (r *CcipsOverlayReconciler) saveStoredTunnels(
	ctx context.Context,
	ccips *l2smv1.CcipsOverlay,
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
	if cm.CreationTimestamp.IsZero() {
		return r.Create(ctx, cm)
	}
	return r.Update(ctx, cm)

}

func (r *CcipsOverlayReconciler) deleteOneStoredTunnel(
	ctx context.Context,
	ccips *l2smv1.CcipsOverlay,
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

		// log.Info("DEBUG: would send tunnel delete request",
		// 	"key", key,
		// 	"endpointA", tunnel.EndpointA,
		// 	"endpointB", tunnel.EndpointB,
		// 	"tunnelID", tunnel.ID,
		// 	"url", url,
		// )

		// --- REAL HTTP DELETE (disabled for now) ---
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

		// Since delete succeeded (or is simulated), remove it from stored state
		delete(storedTunnels, key)

		if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
			return false, err
		}

		// log.Info("deleted one stored tunnel entry", "key", key)

		return len(storedTunnels) == 0, nil
	}

	return true, nil
}

func (r *CcipsOverlayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	ccips := &l2smv1.CcipsOverlay{}
	if err := r.Get(ctx, req.NamespacedName, ccips); err != nil {
		log.Error(err, "unable to fetch CcipsOverlay")
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

	// Resolve node IPs
	ipAddr := make(map[string]string)

	for _, nodeName := range ccips.Spec.Nodes {
		node := &corev1.Node{}
		if err := r.Get(ctx, types.NamespacedName{Name: nodeName}, node); err != nil {
			log.Error(err, "unable to fetch Node", "node", nodeName)
			return ctrl.Result{}, err
		}

		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				ipAddr[nodeName] = addr.Address
				break
			}
		}

		if _, ok := ipAddr[nodeName]; !ok {
			return ctrl.Result{}, fmt.Errorf("node %s has no InternalIP", nodeName)
		}

		log.Info("resolved node IP", "node", nodeName, "ip", ipAddr[nodeName])
	}

	// Build and send one request per tunnel
	for _, tunnel := range ccips.Spec.Tunnels {
		endpointA := tunnel.EndpointA
		endpointB := tunnel.EndpointB

		key := tunnelKey(endpointA, endpointB)
		// Skip if already stored
		if _, exists := storedTunnels[key]; exists {
			log.Info("tunnel already stored, skipping", "key", key)
			continue
		}

		ipA, ok := ipAddr[endpointA]
		if !ok || ipA == "" {
			return ctrl.Result{}, fmt.Errorf("missing IP for endpointA %s", endpointA)
		}

		ipB, ok := ipAddr[endpointB]
		if !ok || ipB == "" {
			return ctrl.Result{}, fmt.Errorf("missing IP for endpointB %s", endpointB)
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

		// jsonBody, err := json.MarshalIndent(reqBody, "", "  ")
		// if err != nil {
		// 	return ctrl.Result{}, err
		// }
		// // Fake response ID (simulate controller response)
		// fakeID := randomID(10)
		// // Log everything
		// log.Info("DEBUG: would send tunnel request",
		// 	"endpointA", endpointA,
		// 	"endpointB", endpointB,
		// 	"request", string(jsonBody),
		// 	"fakeID", fakeID,
		// )

		//UNCOMMENT FROM HERE TO ENABLE HTTP REQUEST SEND

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
		}

	}

	if err := r.saveStoredTunnels(ctx, ccips, cm, storedTunnels); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CcipsOverlayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&l2smv1.CcipsOverlay{}).
		Named("ccipsoverlay").
		Complete(r)
}
