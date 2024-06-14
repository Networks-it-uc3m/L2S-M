package basicclient

import (
	"context"
	"encoding/json"

	"github.com/Networks-it-uc3m/LPM/pkg/exporterclient"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type BasicClient struct {
	SchemaGVR     schema.GroupVersionResource
	DynamicClient *dynamic.DynamicClient
}

func (basicClient *BasicClient) NewClient() error {

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := dynamic.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	schemaGVR := schema.GroupVersionResource{Group: "group", Version: "version", Resource: "resource"}

	basicClient = &BasicClient{SchemaGVR: schemaGVR, DynamicClient: clientset}

	return nil
}

func (basicClient *BasicClient) ExportCRD(namespace string, structCRD exporterclient.StructCustomResourceDefinition) {
	jsonData, err := json.Marshal(structCRD)
	if err != nil {
		panic(err)
	}

	// Unmarshal JSON into a map[string]interface{} to prepare for unstructured conversion
	var objMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &objMap); err != nil {
		panic(err)
	}

	// Create an unstructured.Unstructured object from the map
	unstructuredObj := &unstructured.Unstructured{Object: objMap}
	unstructuredObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "group",
		Version: "version",
		Kind:    "Resource",
	})

	basicClient.DynamicClient.Resource(basicClient.SchemaGVR).Namespace(namespace).Create(context.Background(), unstructuredObj, metav1.CreateOptions{})

}
