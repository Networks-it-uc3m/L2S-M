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

package ids

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
)

func TestGenerateExternalResourcesMountsConfigMapRuleSources(t *testing.T) {
	network := &l2smv1.L2Network{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitored-network",
			Namespace: "default",
		},
		Spec: l2smv1.L2NetworkSpec{
			Ids: &l2smv1.IdsRules{
				Enabled:     true,
				Node:        "node-a",
				HomeNetCIDR: []string{"10.10.0.0/24"},
				CustomRuleSources: []l2smv1.IDSRuleSource{
					{
						Name: "external-rules",
						ConfigMapRef: &corev1.LocalObjectReference{
							Name: "external-suricata-rules",
						},
					},
					{
						Name:   "inline-rules",
						Inline: `alert icmp any any -> any any (msg:"test"; sid:5000001; rev:1;)`,
					},
				},
			},
		},
	}

	objects, err := GenerateExternalResources(network, `[{"name":"ids-net"}]`)
	if err != nil {
		t.Fatalf("GenerateExternalResources returned error: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected 2 generated objects, got %d", len(objects))
	}

	cm, ok := objects[0].(*corev1.ConfigMap)
	if !ok {
		t.Fatalf("expected first object to be ConfigMap, got %T", objects[0])
	}
	if !strings.Contains(cm.Data["suricata.rules"], "var HOME_NET [10.10.0.0/24]") {
		t.Fatalf("generated rules do not define HOME_NET: %q", cm.Data["suricata.rules"])
	}
	if !strings.Contains(cm.Data["suricata.rules"], "sid:5000001") {
		t.Fatalf("generated rules do not include inline rule: %q", cm.Data["suricata.rules"])
	}

	deployment, ok := objects[1].(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected second object to be Deployment, got %T", objects[1])
	}

	podSpec := deployment.Spec.Template.Spec
	if !hasEmptyDirVolume(podSpec.Volumes, "active-rules") {
		t.Fatalf("deployment does not create active rules volume: %#v", podSpec.Volumes)
	}
	if !hasConfigMapVolume(podSpec.Volumes, "generated-rules", "monitored-network-ids-cm") {
		t.Fatalf("deployment does not mount generated rules ConfigMap: %#v", podSpec.Volumes)
	}
	if !hasConfigMapVolume(podSpec.Volumes, "custom-rules-0", "external-suricata-rules") {
		t.Fatalf("deployment does not mount referenced rules ConfigMap: %#v", podSpec.Volumes)
	}

	container := podSpec.Containers[0]
	if !hasWritableVolumeMount(container.VolumeMounts, "active-rules", "/var/lib/suricata/rules") {
		t.Fatalf("container does not mount writable active rules directory: %#v", container.VolumeMounts)
	}
	if !hasVolumeMount(container.VolumeMounts, "generated-rules", "/var/lib/suricata/generated-rules") {
		t.Fatalf("container does not mount generated rules: %#v", container.VolumeMounts)
	}
	if !hasVolumeMount(container.VolumeMounts, "custom-rules-0", "/var/lib/suricata/custom-rules/0") {
		t.Fatalf("container does not mount external rules: %#v", container.VolumeMounts)
	}
	if len(container.Args) != 1 || !strings.Contains(container.Args[0], "/var/lib/suricata/custom-rules/*") {
		t.Fatalf("container command does not merge custom rule sources: %#v", container.Args)
	}
	if !strings.Contains(container.Args[0], "/var/lib/suricata/rules/suricata.rules") {
		t.Fatalf("container command does not write the active Suricata rules file: %#v", container.Args)
	}
	if !strings.Contains(container.Args[0], "suricata -D -i net1 && touch /var/log/suricata/fast.log") {
		t.Fatalf("container command does not launch Suricata with the expected command: %#v", container.Args)
	}
}

func hasEmptyDirVolume(volumes []corev1.Volume, volumeName string) bool {
	for _, volume := range volumes {
		if volume.Name == volumeName && volume.EmptyDir != nil {
			return true
		}
	}
	return false
}

func hasConfigMapVolume(volumes []corev1.Volume, volumeName, configMapName string) bool {
	for _, volume := range volumes {
		if volume.Name == volumeName && volume.ConfigMap != nil && volume.ConfigMap.Name == configMapName {
			return true
		}
	}
	return false
}

func hasWritableVolumeMount(mounts []corev1.VolumeMount, volumeName, mountPath string) bool {
	for _, mount := range mounts {
		if mount.Name == volumeName && mount.MountPath == mountPath && !mount.ReadOnly {
			return true
		}
	}
	return false
}

func hasVolumeMount(mounts []corev1.VolumeMount, volumeName, mountPath string) bool {
	for _, mount := range mounts {
		if mount.Name == volumeName && mount.MountPath == mountPath && mount.ReadOnly {
			return true
		}
	}
	return false
}
