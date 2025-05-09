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

package env

import (
	"os"
)

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func GetDNSPortNumber() string {
	return getEnv("DNS_PORT_NUMBER", "30053")
}

func GetDNSGRPCPortNumber() string {
	return getEnv("GRPC_DNS_PORT_NUMBER", "30818")
}

func GetSwitchesNamespace() string {
	return getEnv("SWITCHES_NAMESPACE", "")
}

func GetControllerIP() string {
	return getEnv("CONTROLLER_IP", "l2sm-controller-service.l2sm-system.svc.cluster.local")
}

func GetControllerPort() string {
	return getEnv("CONTROLLER_PORT", "8181")
}
func GetIntraConfigmapNamespace() string {
	return getEnv("INTRA_CONFIGMAP_NAMESPACE", "kube-system")

}

func GetIntraConfigmapName() string {
	return getEnv("INTRA_CONFIGMAP_NAME", "coredns")

}
