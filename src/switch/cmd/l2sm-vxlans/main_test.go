package main

import (
	"encoding/json"
	"testing"

	"l2sm.local/ovs-switch/pkg/ovs"
)

func TestCreateTopology(t *testing.T) {
	bridge := ovs.FromName("brtun")

	config := `{
		"Nodes": [
			{"name": "netma-test-2", "nodeIP": "10.244.1.8"},
			{"name": "netma-test-3", "nodeIP": "10.244.2.10"},
			{"name": "netma-test-1", "nodeIP": "10.244.0.4"}
		],
		"Links": [
			{"endpointA": "netma-test-2", "endpointB": "netma-test-3"},
			{"endpointA": "netma-test-2", "endpointB": "netma-test-1"},
			{"endpointA": "netma-test-3", "endpointB": "netma-test-1"}
		]
	}`

	var topology Topology
	err := json.Unmarshal([]byte(config), &topology)
	if err != nil {
		t.Fatalf("Error unmarshalling config: %v", err)
	}

	nodeName := "netma-test-1"
	err = createTopology(bridge, topology, nodeName)
	if err != nil {
		t.Fatalf("Error creating topology: %v", err)
	}

	expectedCommands := []string{
		"ovs-vsctl add-port brtun vxlan1 -- set interface vxlan1 type=vxlan options:key=flow options:remote_ip=10.244.1.8 options:local_ip=10.244.0.4 options:dst_port=7000",
		"ovs-vsctl add-port brtun vxlan2 -- set interface vxlan2 type=vxlan options:key=flow options:remote_ip=10.244.2.10 options:local_ip=10.244.0.4 options:dst_port=7000",
	}

	if len(bridge.Commands) != len(expectedCommands) {
		t.Fatalf("Expected %d commands, got %d", len(expectedCommands), len(bridge.Commands))
	}

	for i, command := range expectedCommands {
		if bridge.Commands[i] != command {
			t.Errorf("Expected command %q, got %q", command, bridge.Commands[i])
		}
	}
}
