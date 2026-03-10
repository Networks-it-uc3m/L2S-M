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

import (
	"fmt"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	talpaImageName    = "alexdecb/talpa"
	talpaImageVersion = "1.3.8"

	SwitchTemplateModeNED = "ned"
	SwitchTemplateModeSPS = "sps"
)

// DefaultSwitchTemplate returns the default switch template for the provided mode.
func DefaultSwitchTemplate(mode string) (*l2smv1.SwitchTemplateSpec, error) {
	container := defaultSwitchContainer()

	spec := l2smv1.SwitchPodSpec{
		Containers: []corev1.Container{container},
	}

	switch mode {
	case SwitchTemplateModeNED:
		spec.HostNetwork = true
		spec.Containers[0].Args = []string{SwitchTemplateModeNED}
	case SwitchTemplateModeSPS:
		spec.Containers[0].Args = []string{"sps-init", "--node_name=$(NODENAME)"}
		spec.Containers[0].Env = []corev1.EnvVar{
			{
				Name: "NODENAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
		}
	default:
		return nil, fmt.Errorf("unsupported switch template mode: %s", mode)
	}

	return &l2smv1.SwitchTemplateSpec{Spec: spec}, nil
}

func defaultSwitchContainer() corev1.Container {
	return corev1.Container{
		Name:            "l2sm-switch",
		Image:           fmt.Sprintf("%s:%s", talpaImageName, talpaImageVersion),
		ImagePullPolicy: corev1.PullAlways,
		Resources:       corev1.ResourceRequirements{},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
	}
}
