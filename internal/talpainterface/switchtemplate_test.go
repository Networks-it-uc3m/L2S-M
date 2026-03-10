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

package talpainterface

import "testing"

func TestDefaultSwitchTemplateNED(t *testing.T) {
	template, err := DefaultSwitchTemplate(SwitchTemplateModeNED)
	if err != nil {
		t.Fatalf("DefaultSwitchTemplate returned error: %v", err)
	}

	if !template.Spec.HostNetwork {
		t.Fatalf("expected hostNetwork to be enabled")
	}

	container := template.Spec.Containers[0]
	if got, want := len(container.Args), 1; got != want {
		t.Fatalf("expected %d args, got %d", want, got)
	}
	if got, want := container.Args[0], SwitchTemplateModeNED; got != want {
		t.Fatalf("expected arg %q, got %q", want, got)
	}
	if len(container.Env) != 0 {
		t.Fatalf("expected no env vars, got %d", len(container.Env))
	}
}

func TestDefaultSwitchTemplateSPS(t *testing.T) {
	template, err := DefaultSwitchTemplate(SwitchTemplateModeSPS)
	if err != nil {
		t.Fatalf("DefaultSwitchTemplate returned error: %v", err)
	}

	if template.Spec.HostNetwork {
		t.Fatalf("expected hostNetwork to be disabled")
	}

	container := template.Spec.Containers[0]
	if got, want := len(container.Args), 2; got != want {
		t.Fatalf("expected %d args, got %d", want, got)
	}
	if got, want := container.Args[0], "sps-init"; got != want {
		t.Fatalf("expected first arg %q, got %q", want, got)
	}
	if got, want := container.Args[1], "--node_name=$(NODENAME)"; got != want {
		t.Fatalf("expected second arg %q, got %q", want, got)
	}
	if got, want := len(container.Env), 1; got != want {
		t.Fatalf("expected %d env var, got %d", want, got)
	}
	if got, want := container.Env[0].Name, "NODENAME"; got != want {
		t.Fatalf("expected env var %q, got %q", want, got)
	}
	if container.Env[0].ValueFrom == nil || container.Env[0].ValueFrom.FieldRef == nil {
		t.Fatalf("expected fieldRef env var source")
	}
	if got, want := container.Env[0].ValueFrom.FieldRef.FieldPath, "spec.nodeName"; got != want {
		t.Fatalf("expected field path %q, got %q", want, got)
	}
}

func TestDefaultSwitchTemplateInvalidMode(t *testing.T) {
	if _, err := DefaultSwitchTemplate("invalid"); err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}
