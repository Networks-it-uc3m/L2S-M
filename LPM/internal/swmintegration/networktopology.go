// This file includes work that is derived from 2023 Siemens AG's project
// located at https://gitlab.eclipse.org/eclipse-research-labs/codeco-project/scheduling-and-workload-migration-swm/qos-scheduler, licensed under the Apache License, Version 2.0.

package swmintegration

import (
	"encoding/json"
	"fmt"
	"hash/fnv"

	"github.com/Networks-it-uc3m/LPM/pkg/collector"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type NodeType string

const (
	COMPUTE_NODE NodeType = "COMPUTE"
	NETWORK_NODE NodeType = "NETWORK"
)

type TopologyNodeSpec struct {
	Name string `json:"name,omitempty"`

	Type NodeType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=NodeType"`
}

type TopologyLinkCapabilities struct {
	BandWidthBits string `json:"bandWidthBits,omitempty"`

	LatencyNanos string `json:"latencyNanos,omitempty"`

	OtherCapabilities map[string]string `json:"otherCapabilities"`
}

type TopologyLinkSpec struct {
	Name   string `json:"name"`
	Source string `json:"source"`

	Target       string                   `json:"target"`
	Capabilities TopologyLinkCapabilities `json:"capabilities"`
}

type TopologyPathSpec struct {
	Nodes []string `json:"nodes"`
}

type NetworkTopologySpec struct {
	NetworkImplementation string `json:"networkImplementation"`

	PhysicalBase string `json:"physicalBase"`

	Nodes []TopologyNodeSpec `json:"nodes,omitempty"`

	Links []TopologyLinkSpec `json:"links,omitempty"`
}

type NetworkTopology struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              NetworkTopologySpec `json:"spec,omitempty"`
}

type NetworkTopologyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkTopology `json:"items"`
}

func (networkTopology NetworkTopology) GetUnstructuredData() *unstructured.Unstructured {

	jsonData, err := json.Marshal(networkTopology)
	if err != nil {
		panic(err)
	}

	var objMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &objMap); err != nil {
		panic(err)
	}
	unstructuredObj := &unstructured.Unstructured{Object: objMap}

	unstructuredObj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "qos-scheduler.siemens.com",
		Version: "v1alpha1",
		Kind:    "NetworkTopology",
	})

	return unstructuredObj

}

func GenerateTopologyFromMetrics(metricArray []collector.MetricData) (NetworkTopology, error) {
	networkTopology := NetworkTopology{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "qos-scheduler.siemens.com/v1alpha1",
			Kind:       "NetworkTopology",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "l2sm-overlay",
		},
		Spec: NetworkTopologySpec{
			NetworkImplementation: "L2SM",
			PhysicalBase:          "K8s",
		},
	}

	nodeMap := make(map[string]bool)
	linkMap := make(map[string]*TopologyLinkSpec)

	for _, metric := range metricArray {
		// Ensure node entries are unique and create them if they don't exist
		if !nodeMap[metric.SourceNodeName] {
			networkTopology.Spec.Nodes = append(networkTopology.Spec.Nodes, TopologyNodeSpec{
				Name: metric.SourceNodeName,
				Type: COMPUTE_NODE,
			})
			networkTopology.Spec.Links = append(networkTopology.Spec.Links, TopologyLinkSpec{
				Source: metric.SourceNodeName,
				Target: metric.SourceNodeName,
				Name:   linkHash(TopologyLinkSpec{Source: metric.SourceNodeName, Target: metric.SourceNodeName}),
				Capabilities: TopologyLinkCapabilities{
					OtherCapabilities: make(map[string]string),
				}})
			nodeMap[metric.SourceNodeName] = true
		}

		if !nodeMap[metric.TargetNodeName] {

			networkTopology.Spec.Nodes = append(networkTopology.Spec.Nodes, TopologyNodeSpec{
				Name: metric.TargetNodeName,
				Type: COMPUTE_NODE,
			})
			networkTopology.Spec.Links = append(networkTopology.Spec.Links, TopologyLinkSpec{
				Source: metric.TargetNodeName,
				Target: metric.TargetNodeName,
				Name:   linkHash(TopologyLinkSpec{Source: metric.TargetNodeName, Target: metric.TargetNodeName}),
				Capabilities: TopologyLinkCapabilities{
					OtherCapabilities: make(map[string]string),
				}})
			nodeMap[metric.TargetNodeName] = true
		}

		linkKey := linkHash(TopologyLinkSpec{Source: metric.SourceNodeName, Target: metric.TargetNodeName})
		link, exists := linkMap[linkKey]

		if !exists {
			link = &TopologyLinkSpec{
				Name:   linkKey,
				Source: metric.SourceNodeName,
				Target: metric.TargetNodeName,
				Capabilities: TopologyLinkCapabilities{
					OtherCapabilities: make(map[string]string),
				},
			}
			linkMap[linkKey] = link
		}

		if metric.Value != 0 {
			switch metric.Name {
			case "net_rtt_ms":
				latencyNanos := fmt.Sprintf("%f", metric.Value) // save in miliseconds
				link.Capabilities.LatencyNanos = latencyNanos
			case "net_throughput_kbps":
				bandWidthBits := fmt.Sprintf("%fM", metric.Value*0.0009765625) // From kbps to Mbps
				link.Capabilities.BandWidthBits = bandWidthBits
			default:
				//fmt.Printf("Metric not found: %s\n", metric.Name)
			}
		}

	}

	// Convert the link map to a slice for the topology spec
	for _, link := range linkMap {
		networkTopology.Spec.Links = append(networkTopology.Spec.Links, *link)
	}

	return networkTopology, nil
}

func linkHash(link TopologyLinkSpec) string {
	hashString := fmt.Sprintf("%s%s", link.Source, link.Target)
	hash := fnv.New32() // Using FNV hash for a compact hash, but still 32 bits
	hash.Write([]byte(hashString))
	sum := hash.Sum32()
	// Encode the hash as a base32 string and take the first 4 characters
	return fmt.Sprintf("%04x", sum) // H
}
