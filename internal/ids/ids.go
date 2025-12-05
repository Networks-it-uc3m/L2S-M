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
	// We pass the ConfigMap name so the deployment knows what volume to mount
	suri := generateSuricataDeployment(cm.Name)
	resArray = append(resArray, suri)

	return resArray, nil
}

// constructConfigMap aggregates rules and creates the K8s object
func constructConfigMap(customRuleSources []l2smv1.IDSRuleSource) (*corev1.ConfigMap, error) {
	var rulesBuilder strings.Builder

	rulesBuilder.WriteString("# Hardcoded Demo Rule\n")
	rulesBuilder.WriteString("alert icmp any any -> any any (msg:\"ICMP Packet found\"; sid:1000001; rev:1;)\n")

	// Iterate over the API sources (Logic placeholder for future expansion)
	for _, source := range customRuleSources {
		if source.Inline != "" {
			rulesBuilder.WriteString(fmt.Sprintf("\n# Source: %s\n", source.Name))
			rulesBuilder.WriteString(source.Inline + "\n")
		}
		// Note: ConfigMapRef handling and URL downloading would happen here in a real implementation
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
			"local.rules": rulesBuilder.String(),
		},
	}

	return configMap, nil
}

// generateSuricataDeployment creates the deployment definition
func generateSuricataDeployment(configMapName string) *appsv1.Deployment {
	replicas := int32(1)
	labels := map[string]string{"app": "suricata-ids"}

	// This makes sure the Pod runs as Root to allow packet capture capabilities
	// or you can use specific Capabilities like NET_ADMIN + NET_RAW
	privileged := true

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
						// MULTUS INTEGRATION: Mount the secondary interface
						"k8s.v1.cni.cncf.io/networks": "overlay-sample-veth10",
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
								"suricata -D -i net1 && tail -f /var/log/suricata/fast.log",
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
									MountPath: "/var/lib/suricata/rules", // Standard path for jasonish image
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
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
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
