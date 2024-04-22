package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"ovs-switch/pkg/ovs"
)

type Node struct {
	Name          string   `json:"name"`
	NodeIP        string   `json:"nodeIP"`
	NeighborNodes []string `json:"neighborNodes"`
}

type Link struct {
	EndpointNodeA string `json:"endpointA"`
	EndpointNodeB string `json:"endpointB"`
}

type Topology struct {
	Nodes []Node `json:"Nodes"`
	Links []Link `json:"Links"`
}

// Script that takes two required arguments:
// the first one is the name in the cluster of the node where the script is running
// the second one is the path to the configuration file, in reference to the code.
func main() {

	//configDir, _, fileType, err := takeArguments()

	configDir, nodeName, fileType, err := takeArguments()

	bridge := ovs.FromName("brtun")

	if err != nil {
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	switch fileType {
	case "topology":
		var topology Topology

		err = readFile(configDir, &topology)

		if err != nil {
			fmt.Println("Error with the provided file. Error:", err)
			return
		}

		fmt.Println(topology)
		err = createTopology(bridge, topology, nodeName)

	case "neighbors":
		var node Node
		err := readFile(configDir, &node)

		if err != nil {
			fmt.Println("Error with the provided file. Error:", err)
			return
		}

		err = connectToNeighbors(bridge, node)
	}

	if err != nil {
		fmt.Println("Vxlans not created: ", err)
		return
	}
}

func takeArguments() (string, string, string, error) {
	configDir := os.Args[len(os.Args)-1]

	nodeName := flag.String("node_name", "", "name of the node the script is executed in. Required.")

	fileType := flag.String("file_type", "topology", "type of filed passed as an argument. Can either be topology or neighbors. Default: topology.")

	flag.Parse()

	switch {
	case *nodeName == "":
		return "", "", "", errors.New("node name is not defined")
	case *fileType != "topology" && *fileType != "neighbors":
		return "", "", "", errors.New("file type not supported. Available types: 'topology' and 'neighbors'")
	case configDir == "":
		return "", "", "", errors.New("config directory is not defined")
	}

	return configDir, *nodeName, *fileType, nil
}

/**
Example:
{
    "Nodes": [
        {
            "name": "l2sm1",
            "nodeIP": "10.1.14.53"
        },
        {
            "name": "l2sm2",
            "nodeIP": "10.1.14.90"
        }
    ],
    "Links": [
        {
            "endpointA": "l2sm1",
            "endpointB": "l2sm2"
        }
    ]
}

*/
func createTopology(bridge ovs.Bridge, topology Topology, nodeName string) error {

	nodeMap := make(map[string]string)
	for _, node := range topology.Nodes {
		nodeMap[node.Name] = node.NodeIP
	}

	localIp := nodeMap[nodeName]

	for vxlanNumber, link := range topology.Links {
		vxlanId := fmt.Sprintf("vxlan%d", vxlanNumber)
		var remoteIp string
		switch nodeName {
		case link.EndpointNodeA:
			remoteIp = nodeMap[link.EndpointNodeB]
		case link.EndpointNodeB:
			remoteIp = nodeMap[link.EndpointNodeA]
		default:
			break
		}
		err := bridge.CreateVxlan(ovs.Vxlan{VxlanId: vxlanId, LocalIp: localIp, RemoteIp: remoteIp, UdpPort: "7000"})

		if err != nil {
			return fmt.Errorf("could not create vxlan between node %s and node %s", link.EndpointNodeA, link.EndpointNodeB)
		} else {
			fmt.Printf("Created vxlan between node %s and node %s.\n", link.EndpointNodeA, link.EndpointNodeB)
		}

	}
	return nil

}

func readFile(configDir string, dataStruct interface{}) error {

	/// Read file and save in memory the JSON info
	data, err := os.ReadFile(configDir)
	if err != nil {
		fmt.Println("No input file was found.", err)
		return err
	}

	err = json.Unmarshal(data, &dataStruct)
	if err != nil {
		return err
	}

	return nil

}

/**
Example:

        {
            "Name": "l2sm1",
            "nodeIP": "10.1.14.53",
			"neighborNodes":["10.4.2.3","10.4.2.5"]
		}
*/
func connectToNeighbors(bridge ovs.Bridge, node Node) error {
	for vxlanNumber, neighborIp := range node.NeighborNodes {
		vxlanId := fmt.Sprintf("vxlan%d", vxlanNumber)
		err := bridge.CreateVxlan(ovs.Vxlan{VxlanId: vxlanId, LocalIp: node.NodeIP, RemoteIp: neighborIp, UdpPort: "7000"})

		if err != nil {
			return fmt.Errorf("could not create vxlan with neighbor %s", neighborIp)
		} else {
			fmt.Printf("Created vxlan with neighbor %s", neighborIp)
		}
	}
	return nil
}
