package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

// ContainsString checks if a string is present in a slice.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString removes a string from a slice.
func RemoveString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func Int32Ptr(i int32) *int32 { return &i }

func GenerateHash(obj runtime.Object) string {
	// Serializer that handles runtime.Objects specifically for Kubernetes
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: false, Pretty: false, Strict: true})

	// Create a buffer to hold the JSON data
	var b bytes.Buffer

	// Encode the object to JSON; handle runtime objects appropriately
	err := s.Encode(obj, &b)
	if err != nil {
		return ""
	}

	// Compute the SHA-1 hash of the JSON representation
	hash := sha1.Sum(b.Bytes())
	return hex.EncodeToString(hash[:5])
}

func SpecToJson(obj runtime.Object) bytes.Buffer {
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, nil, nil, json.SerializerOptions{Yaml: false, Pretty: false, Strict: true})

	// Create a buffer to hold the JSON data
	var b bytes.Buffer

	// Encode the object to JSON; handle runtime objects appropriately
	s.Encode(obj, &b)

	return b
}
