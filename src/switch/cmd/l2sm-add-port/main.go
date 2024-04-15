package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"ovs-switch/pkg/ovs"
)

type Node struct {
	Name          string `json:"name"`
	NodeIP        string `json:"nodeIP"`
	NeighborNodes []Node `json:"neighborNodes"`
}

// Script that takes two required arguments:
// the first one is the name in the cluster of the node where the script is running
// the second one is the path to the configuration file, in reference to the code.
func main() {

	portName, err := takeArguments()

	bridge := ovs.FromName("brtun")

	if err != nil {
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	bridge.AddPort(portName)

	if err != nil {
		fmt.Println("Port not added: ", err)
		return
	}
}

func takeArguments() (string, error) {

	portName := flag.String("port_name", "", "port you want to add. Required.")

	flag.Parse()

	if *portName == "" {
		return "", errors.New("port name is not defined")

	}

	return *portName, nil
}

func createTopology(bridge ovs.Bridge, nodes []Node, nodeName string) error {

	// Search for the corresponding node in the configuration, according to the first passed parameter.
	// Once the node is found, create a bridge for every neighbour node defined.
	// The bridge is created with the nodeIp and neighborNodeIP and VNI. The VNI is generated in the l2sm-controller thats why its set to 'flow'.
	for _, node := range nodes {
		if node.Name == nodeName {
			//nodeIP := strings.TrimSpace(node.NodeIP)
			connectToNeighbors(bridge, node)
		}
	}
	return nil
}

func readFile(configDir string) ([]Node, error) {
	/// Read file and save in memory the JSON info
	data, err := ioutil.ReadFile(configDir)
	if err != nil {
		fmt.Println("No input file was found.", err)
		return nil, err
	}

	var nodes []Node
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		return nil, err
	}

	return nodes, nil

}

func connectToNeighbors(bridge ovs.Bridge, node Node) error {
	for vxlanNumber, neighbor := range node.NeighborNodes {
		vxlanId := fmt.Sprintf("vxlan%d", vxlanNumber)
		err := bridge.CreateVxlan(ovs.Vxlan{VxlanId: vxlanId, LocalIp: node.NodeIP, RemoteIp: neighbor.NodeIP, UdpPort: "7000"})

		if err != nil {
			return fmt.Errorf("could not create vxlan between node %s and node %s", node.Name, neighbor)
		} else {
			fmt.Printf("Created vxlan between node %s and node %s.\n", node.Name, neighbor)
		}
	}
	return nil
}
