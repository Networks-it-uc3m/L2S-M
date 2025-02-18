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

func GetSwitchesNamespace() string {
	return getEnv("SWITCHES_NAMESPACE", "l2sm-system")
}

func GetControllerIP() string {
	return getEnv("CONTROLLER_IP", "l2sm-controller-service.l2sm-system.svc.cluster.local")
}

func GetControllerPort() string {
	return getEnv("CONTROLLER_PORT", "8181")
}
