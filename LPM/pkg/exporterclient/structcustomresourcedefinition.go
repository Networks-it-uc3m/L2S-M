package exporterclient

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type StructCustomResourceDefinition interface {
	GetUnstructuredData() *unstructured.Unstructured
}
