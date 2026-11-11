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
	"io"
	"net/http"
	"strings"
	"testing"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type interOverlayTransport func(*http.Request) (*http.Response, error)

func (f interOverlayTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestCcipsInterOverlayLifecycle(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := l2smv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	overlay := &l2smv1.CcipsInterOverlay{
		ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: "default"},
		Spec: l2smv1.CcipsInterOverlaySpec{
			ControllerIP: "192.0.2.10",
			Clusters:     []l2smv1.Cluster{{Name: "east", Address: "192.0.2.1"}, {Name: "west", Address: "192.0.2.2"}},
			Tunnels:      []l2smv1.Tunnel{{EndpointA: "east", EndpointB: "west"}},
		},
	}
	// No Kubernetes Nodes exist: endpoints must come from the cluster entries.
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(overlay).Build()
	r := &CcipsInterOverlayReconciler{Client: c, Scheme: scheme}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: overlay.Name, Namespace: overlay.Namespace}}
	posts, deletes := 0, 0
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	http.DefaultTransport = interOverlayTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "192.0.2.10:5000" {
			t.Fatalf("unexpected controller: %s", req.URL)
		}
		body := `{}`
		switch req.Method {
		case http.MethodPost:
			posts++
			if req.URL.Path != "/ccips" {
				t.Fatalf("unexpected path: %s", req.URL.Path)
			}
			var payload TunnelRequest
			if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if len(payload.Nodes) != 2 || payload.Nodes[0] != (RequestNode{IPData: "192.0.2.1", IPControl: "192.0.2.1"}) || payload.Nodes[1] != (RequestNode{IPData: "192.0.2.2", IPControl: "192.0.2.2"}) {
				t.Fatalf("unexpected endpoints: %+v", payload.Nodes)
			}
			if len(payload.EncAlg) != 1 || payload.EncAlg[0] != "aes-cbc" || len(payload.IntAlg) != 1 || payload.IntAlg[0] != "sha2-256" || payload.SoftLifetime.NTime != 15 || payload.HardLifetime.NTime != 30 {
				t.Fatalf("unexpected settings: %+v", payload)
			}
			body = `{"id":"remote-tunnel"}`
		case http.MethodDelete:
			deletes++
			if req.URL.Path != "/ccips/remote-tunnel" {
				t.Fatalf("unexpected delete: %s", req.URL)
			}
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	reconcile := func() {
		t.Helper()
		if _, err := r.Reconcile(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
	reconcile() // Install the finalizer before external side effects.
	if posts != 0 {
		t.Fatal("created tunnel before persisting finalizer")
	}
	reconcile()
	reconcile()
	if posts != 1 {
		t.Fatalf("expected one POST, got %d", posts)
	}
	cm, stored, err := r.getStoredTunnels(ctx, overlay)
	if err != nil {
		t.Fatal(err)
	}
	tunnel := stored[tunnelKey("east", "west")]
	if tunnel.ID != "remote-tunnel" || tunnel.State != "Created" {
		t.Fatalf("unexpected stored tunnel: %+v", tunnel)
	}
	if len(cm.OwnerReferences) != 1 || cm.OwnerReferences[0].Kind != "CcipsInterOverlay" {
		t.Fatalf("unexpected ownership: %+v", cm.OwnerReferences)
	}
	if err := c.Get(ctx, req.NamespacedName, overlay); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, overlay); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if deletes != 1 {
		t.Fatalf("expected one DELETE, got %d", deletes)
	}
	if err := c.Get(ctx, req.NamespacedName, overlay); !apierrors.IsNotFound(err) {
		t.Fatalf("resource still exists after cleanup: %v", err)
	}
	_, stored, err = r.getStoredTunnels(ctx, overlay)
	if err != nil || len(stored) != 0 {
		t.Fatalf("tunnels remain after cleanup: %v, %v", stored, err)
	}
}
