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

package utils

import (
	"fmt"
	"hash/fnv"
)

type SwitchType string

const (
	SlicePacketSwitch SwitchType = "sps"
	NetworkEdgeDevice SwitchType = "ned"
)

func GenerateSwitchPodName(resourceName, nodeName string, switchType SwitchType) string {
	hash := fnv.New32() // Using FNV hash for a compact hash, but still 32 bits
	hash.Write([]byte(fmt.Sprintf("%s%s", resourceName, nodeName)))
	sum := hash.Sum32()
	// Encode the hash as a base32 string and take the first 4 characters
	return fmt.Sprintf("%s-%04x", switchType, sum) // H
}
func GenerateReplicaSetName(switchPodName string) string {
	return switchPodName
}
func GenerateServiceName(switchPodName string) string {
	// Encode the hash as a base32 string and take the first 4 characters
	return fmt.Sprintf("%s-svc", switchPodName) // H
}

func GenerateLPMNetworkName(overlayName string) string {
	return fmt.Sprintf("lpm-%s", overlayName)
}
