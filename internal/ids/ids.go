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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// Assuming your types are in this package
	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
)

// GenerateExternalResources orchestrates the creation of the ConfigMap and Deployment
func GenerateExternalResources(idsRules *l2smv1.IdsRules) ([]client.Object, error) {
	resArray := []client.Object{}

	// 1. Create the ConfigMap containing the rules
	// We pass the custom sources defined in the CR
	cm, err := constructConfigMap(idsRules.CustomRuleSources)
	if err != nil {
		return nil, fmt.Errorf("failed to construct configmap: %w", err)
	}
	resArray = append(resArray, cm)

	// 2. Create the Suricata Deployment
	// We pass the ConfigMap name AND the custom sources so we can mount the refs
	suri := generateSuricataDeployment(cm.Name, idsRules.CustomRuleSources)
	resArray = append(resArray, suri)

	return resArray, nil
}

// constructConfigMap aggregates rules and creates the K8s object
func constructConfigMap(customRuleSources []l2smv1.IDSRuleSource) (*corev1.ConfigMap, error) {
	var rulesBuilder strings.Builder

	rulesBuilder.WriteString(`
# ---------------------------------------------------------------------------
# SYN SCAN DETECTION
# ---------------------------------------------------------------------------

# Internal SYN scanning to common ports
alert tcp $HOME_NET any -> $HOME_NET [7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible Syn Scan Technique attempted from internal host"; flow:to_server, stateless; flags:S; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000001; rev:1;)

# Internal SYN scanning to NON-common ports (inverse selection)
alert tcp $HOME_NET any -> $HOME_NET ![7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible Syn Scan Technique attempted from internal host"; flow:to_server, stateless; flags:S; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000002; rev:1;)

# External SYN scanning
alert tcp $EXTERNAL_NET any -> any any (msg:"Possible Syn Scan Technique attempted from the internet"; flow:to_server, stateless; flags:S; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_port_scan_from_the_internet; sid:40000003; rev:1;)

# ---------------------------------------------------------------------------
# NULL SCAN DETECTION
# ---------------------------------------------------------------------------

# Internal NULL scanning to common ports
alert tcp $HOME_NET any -> $HOME_NET [7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible Null Scan Attempt from internal host"; flow:to_server, stateless; flags:0; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000004; rev:1;)

# Internal NULL scanning to NON-common ports
alert tcp $HOME_NET any -> $HOME_NET ![7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible Null Scan Technique from internal host"; flow:to_server, stateless; flags:0; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000005; rev:1;)

# External NULL scanning
alert tcp $EXTERNAL_NET any -> any any (msg:"Possible Null Scan attempt from internet"; flow:to_server, stateless; flags:0; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_port_scan_from_the_internet; sid:40000006; rev:1;)

# ---------------------------------------------------------------------------
# FIN SCAN DETECTION
# ---------------------------------------------------------------------------

# Internal FIN scanning to common ports
alert tcp $HOME_NET any -> $HOME_NET [7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible FIN Scan Attempt from internal host"; flow:to_server, stateless; flags:F; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000007; rev:1;)

# Internal FIN scanning to NON-common ports
alert tcp $HOME_NET any -> $HOME_NET ![7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible FIN Scan Technique from internal host"; flow:to_server, stateless; flags:F; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:40000008; rev:1;)

# External FIN scanning
alert tcp $EXTERNAL_NET any -> any any (msg:"Possible FIN Scan attempt from internet"; flow:to_server, stateless; flags:F; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_port_scan_from_the_internet; sid:40000009; rev:1;)

# ---------------------------------------------------------------------------
# XMAS SCAN DETECTION
# ---------------------------------------------------------------------------

# Internal XMAS scanning to common ports
alert tcp $HOME_NET any -> $HOME_NET [7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible XMAS Scan Attempt from internal host"; flow:to_server, stateless; flags:FPU; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:400000011; rev:1;)

# Internal XMAS scanning to NON-common ports
alert tcp $HOME_NET any -> $HOME_NET ![7,9,13,21:23,25:26,37,53,79:81,88,106,110:111,113,119,135,139,143:144,179,199,389,427,443:445,465,513:515,543:544,548,554,587,631,646,873,990,993,995,1025:1029,1110,1433,1720,1723,1755,1900,2000:2001,2049,2121,2717,3000,3128,3306,3389,3986,4899,5000,5009,5051,5060,5101,5190,5357,5432,5631,5666,5800,5900,6000:6001,6646,7070,8000,8008:8009,8080:8081,8443,8888,9100,9999:10000,32768,49152:49157] (msg:"Possible XMAS Scan Technique from internal host"; flow:to_server, stateless; flags:FPU; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_internal_port_scan; sid:400000012; rev:1;)

# External XMAS scanning
alert tcp $EXTERNAL_NET any -> any any (msg:"Possible XMAS Scan attempt from internet"; flow:to_server, stateless; flags:FPU; window:1024; detection_filter:track by_src, count 10, seconds 15; classtype:attempted_port_scan_from_the_internet; sid:400000013; rev:1;)
		`)

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
			Name:      "suricata-rules-cfg",
			Namespace: "default",
		},
		Data: map[string]string{
			"suricata.rules": rulesBuilder.String(),
		},
	}

	return configMap, nil
}

// generateSuricataDeployment creates the deployment definition
func generateSuricataDeployment(generatedConfigMapName string, customRuleSources []l2smv1.IDSRuleSource) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{"app": "suricata-ids"}

	// This makes sure the Pod runs as Root to allow packet capture capabilities
	// or you can use specific Capabilities like NET_ADMIN + NET_RAW
	privileged := true

	// Construct the Projected Volume Sources
	// This allows us to merge the generated inline rules AND any external ConfigMaps (like the portscan one)
	// into a single directory: /var/lib/suricata/rules/
	projectedSources := []corev1.VolumeProjection{
		{
			ConfigMap: &corev1.ConfigMapProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: generatedConfigMapName,
				},
			},
		},
	}

	// Add any user-provided ConfigMapRefs to the volume projection
	for _, source := range customRuleSources {
		if source.ConfigMapRef != nil {
			projectedSources = append(projectedSources, corev1.VolumeProjection{
				ConfigMap: &corev1.ConfigMapProjection{
					LocalObjectReference: *source.ConfigMapRef,
				},
			})
		}
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "suricata-ids",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"k8s.v1.cni.cncf.io/networks": `[
                            {
                                "name": "overlay-sample-veth10",
                                "ips": ["192.168.0.1/24"]
                            }
                        ]`,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "suricata",
							Image: "jasonish/suricata:latest",
							// Command args to listen specifically on the Multus interface (net1)
							// New Args: Run Suricata in the background, then tail the log file to stdout
							Command: []string{"/bin/bash", "-c"},
							Args: []string{
								// Start Suricata as a background process (&)
								// Then tail the log file so it streams to the container's stdout
								"suricata -D -i net1 && touch /var/log/suricata/fast.log && tail -f /var/log/suricata/fast.log -s 1 ---disable-inotify",
							},
							SecurityContext: &corev1.SecurityContext{
								// Suricata needs privileges to capture packets
								Privileged: &privileged,
								Capabilities: &corev1.Capabilities{
									Add: []corev1.Capability{"NET_ADMIN", "NET_RAW", "IPC_LOCK"},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rules-volume",
									MountPath: "/var/lib/suricata/rules", // All ConfigMaps will be merged here
									ReadOnly:  true,
								},
							},
						},
					},
					NodeName: "l2sm-test-control-plane",
					Volumes: []corev1.Volume{
						{
							Name: "rules-volume",
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: projectedSources,
								},
							},
						},
					},
				},
			},
		},
	}
}

// Helper to parse resource quantities cleanly
func parseQuantity(q string) resource.Quantity {
	// Requires "k8s.io/apimachinery/pkg/api/resource"
	val, _ := resource.ParseQuantity(q)
	return val
}
