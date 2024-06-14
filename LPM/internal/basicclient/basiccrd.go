package basicclient

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type BasicCRD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              BasicSpec `json:"spec,omitempty"`
}

type BasicSpec struct {
}

func (basicCR BasicCRD) GetUnstructuredData() *unstructured.Unstructured {

	jsonData, err := json.Marshal(basicCR)
	if err != nil {
		panic(err)
	}

	// Unmarshal JSON into a map[string]interface{} to prepare for unstructured conversion
	var objMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &objMap); err != nil {
		panic(err)
	}
	unstructuredObj := &unstructured.Unstructured{Object: objMap}

	unstructuredObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "group",
		Version: "version",
		Kind:    "Kind",
	})

	// Create an unstructured.Unstructured object from the map
	return unstructuredObj
}
