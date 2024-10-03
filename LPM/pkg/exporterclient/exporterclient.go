package exporterclient

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

type ExporterClient interface {
	NewClient() error
	ExportCRD(namespace string, crd StructCustomResourceDefinition)
	GetSchemaGVR() schema.GroupVersionResource
	GetDynamicClient() dynamic.DynamicClient
}
