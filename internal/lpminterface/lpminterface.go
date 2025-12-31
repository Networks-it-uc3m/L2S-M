package lpminterface

import (
	"encoding/json"
	"fmt"

	"encoding/json"
	"fmt"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

func BuildMonitoringResources(overlay *l2smv1.Overlay) (*corev1.Container, *corev1.ConfigMap) {

	c := &corev1.Container{}
	cm := &corev1.ConfigMap{}
	return c, cm
}

func BuildMonitoringExporter(networkTopologyNamespace, exporterName string, targets []string) (*appsv1.Deployment, *corev1.ConfigMap, *corev1.Service, error) {

	appName := fmt.Sprintf("prometheus-%s", exporterName)
	saName := "default"

	// 1. Generate Prometheus Config (Scrape Targets)

	// Join targets for YAML array
	targetsJson, _ := json.Marshal(targets) // simple trick to format ["a","b"] correctly

	promConfigContent := fmt.Sprintf(`global:
  scrape_interval: 15s
  evaluation_interval: 15s
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: %s`, string(targetsJson))

	// 2. Create ConfigMap
	cmName := fmt.Sprintf("prometheus-config-%s", exporterName)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:   cmName,
			Labels: map[string]string{"app": appName},
		},
		Data: map[string]string{
			"prometheus.yml": promConfigContent,
		},
	}

	// 3. Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appName,
			Labels: map[string]string{"app": appName},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: saName,
					Containers: []corev1.Container{
						{
							Name:  "prometheus",
							Image: "prom/prometheus:v2.30.3",
							Args: []string{
								"--config.file=/etc/prometheus/prometheus.yml",
								"--storage.tsdb.path=/prometheus",
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9090},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "config-volume", MountPath: "/etc/prometheus"},
								{Name: "data-volume", MountPath: "/prometheus"},
							},
						},
						{
							Name:            "exporter",
							Image:           "alexdecb/lpm-exporter:1.1",
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:  "TOPOLOGY_NAMESPACE",
									Value: networkTopologyNamespace,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
						{
							Name: "data-volume",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	// 4. Create Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appName,
			Labels: map[string]string{"app": appName},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": appName},
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     9090,
				},
			},
		},
	}

	return deployment, configMap, service, nil
}
