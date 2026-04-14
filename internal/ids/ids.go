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
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Assuming your types are in this package
	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/networkannotation"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

// GenerateExternalResources orchestrates the creation of the ConfigMap and Deployment.
func GenerateExternalResources(network *l2smv1.L2Network, netAttachString string) ([]client.Object, error) {
	if network == nil {
		return nil, fmt.Errorf("network is nil")
	}
	if network.Spec.Ids == nil {
		return nil, fmt.Errorf("ids configuration is nil")
	}

	resArray := []client.Object{}

	// if a namespace was specified, we use it. if none was, we use the network namespace
	namespace := network.Spec.Ids.Namespace
	if namespace == "" {
		namespace = network.Namespace
	}
	// 1. Create the ConfigMap containing the rules
	// We pass the custom sources defined in the CR
	cm, err := constructConfigMap(network, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to construct configmap: %w", err)
	}
	resArray = append(resArray, cm)

	// 2. Create the Suricata Deployment
	// We pass the ConfigMap name AND the custom sources so we can mount the refs
	suri := generateSuricataDeployment(network.Spec.Ids, network.Name, netAttachString, namespace)
	resArray = append(resArray, suri)

	return resArray, nil
}

// constructConfigMap aggregates rules and creates the K8s object
func constructConfigMap(network *l2smv1.L2Network, namespace string) (*corev1.ConfigMap, error) {
	var rulesBuilder strings.Builder
	idsRules := network.Spec.Ids
	customRuleSources := idsRules.CustomRuleSources
	homeNetCIDR := idsRules.HomeNetCIDR
	if len(homeNetCIDR) == 0 {
		rulesBuilder.WriteString("var HOME_NET any\n")
	} else {
		rulesBuilder.WriteString(fmt.Sprintf("var HOME_NET [%s]\n", strings.Join(homeNetCIDR, ",")))
	}
	rulesBuilder.WriteString("var EXTERNAL_NET !$HOME_NET\n")

	// Iterate over the API sources (Logic placeholder for future expansion)
	for _, source := range customRuleSources {
		if source.Inline != "" {
			rulesBuilder.WriteString(fmt.Sprintf("\n# Source: %s\n", source.Name))
			rulesBuilder.WriteString(source.Inline + "\n")
		}
		// ConfigMapRef is handled in the Deployment volume projection, not here
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GenerateIdsCMName(network.Name),
			Namespace: namespace,
		},
		Data: map[string]string{
			"suricata.rules": rulesBuilder.String(),
		},
	}

	return configMap, nil
}

// generateSuricataDeployment creates the deployment definition
func generateSuricataDeployment(idsRules *l2smv1.IdsRules, networkName, netAttachAnnotation, namespace string) *appsv1.Deployment {
	labels := map[string]string{
		"app":            "suricata-ids",
		"l2sm/component": "ids",
	}

	// This makes sure the Pod runs as Root to allow packet capture capabilities
	privileged := true

	volumes := []corev1.Volume{
		{
			Name: "active-rules",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "generated-rules",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: utils.GenerateIdsCMName(networkName),
					},
				},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "active-rules",
			MountPath: "/var/lib/suricata/rules",
		},
		{
			Name:      "generated-rules",
			MountPath: "/var/lib/suricata/generated-rules",
			ReadOnly:  true,
		},
	}

	// Mount every referenced ConfigMap in its own directory so rule keys from
	// different ConfigMaps cannot collide in a projected volume.
	for i, source := range idsRules.CustomRuleSources {
		if source.ConfigMapRef != nil && source.ConfigMapRef.Name != "" {
			volumeName := fmt.Sprintf("custom-rules-%d", i)
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: *source.ConfigMapRef,
					},
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: fmt.Sprintf("/var/lib/suricata/custom-rules/%d", i),
				ReadOnly:  true,
			})
		}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.GenerateIdsDeployname(networkName),
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						networkannotation.MULTUS_ANNOTATION_KEY: netAttachAnnotation,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "suricata",
							Image: "jasonish/suricata:latest",
							// Command args to listen specifically on the Multus interface (net1)
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								// Merge generated inline rules with every file from referenced rule ConfigMaps.
								"set -eu; rules=/var/lib/suricata/rules/suricata.rules; cat /var/lib/suricata/generated-rules/suricata.rules > \"$rules\"; for dir in /var/lib/suricata/custom-rules/*; do [ -d \"$dir\" ] || continue; find \"$dir\" -maxdepth 1 -type f | sort | while read -r file; do printf '\\n# Source file: %s\\n' \"$file\" >> \"$rules\"; cat \"$file\" >> \"$rules\"; printf '\\n' >> \"$rules\"; done; done; suricata -D -i net1 && touch /var/log/suricata/fast.log && tail -f /var/log/suricata/fast.log -s 1",
							},
							SecurityContext: &corev1.SecurityContext{
								// Suricata needs privileges to capture packets
								Privileged: &privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN", "NET_RAW", "IPC_LOCK"},
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					NodeName: idsRules.Node,
					Volumes:  volumes,
				},
			},
		},
	}
}
