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

func GenerateSwitchName(resourceName, nodeName string, switchType SwitchType) string {
	hash := fnv.New32() // Using FNV hash for a compact hash, but still 32 bits
	hash.Write([]byte(fmt.Sprintf("%s%s", resourceName, nodeName)))
	sum := hash.Sum32()
	// Encode the hash as a base32 string and take the first 4 characters
	return fmt.Sprintf("%s-%04x", switchType, sum) // H
}
func GenerateReplicaSetName(switchName string) string {
	return switchName
}
func GenerateServiceName(switchName string) string {
	// Encode the hash as a base32 string and take the first 4 characters
	return fmt.Sprintf("%s-svc", switchName) // H
}

func GenerateLPMNetworkName(overlayName string) string {
	return fmt.Sprintf("lpm-%s", overlayName)
}
