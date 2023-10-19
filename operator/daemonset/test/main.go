package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Node struct {
	Name          string   `json:"name"`
	NodeIP        string   `json:"nodeIP"`
	NeighborNodes []string `json:"neighborNodes"`
}

// Script that takes two required arguments:
// the first one is the name in the cluster of the node where the script is running
// the second one is the path to the configuration file, in reference to the code.
func main() {

	// Read file and save in memory the JSON info
	data, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		fmt.Println("Error reading input file:", err)
		return
	}

	var nodes []Node
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return
	}

	// Search for the corresponding node in the configuration, according to the first passed parameter.
	// Once the node is found, create a bridge for every neighbour node defined.
	// The bridge is created with the nodeIp, neighborNodeIP and VNI. The VNI is generated according to the position of the node in the Json file. The first node will have the number 5
	// as a reference, the second a 6, and so on. And if a bridge between node 1 and node 2 is generated, it will have a vni of 5006, the two references with two 0s in between.
	// Another example would be node 3 of the cluster and node 9. Node 3 will have the reference 7 (5+3-1), and the Node 9 the reference 13(5 + 9 -1), resulting in the VNI 70013.
	// There's up to 2 ^ 24 possible vnis that are reduced to (2 ^24)/100 because of this measure (2 decimal digits are lost). So in total, a number of 167.772 virtual networks can be created.
	nodeVniRef := 5
	for _, node := range nodes {
		if node.Name == os.Args[1] {
			nodeIP := strings.TrimSpace(node.NodeIP)
			neighborVniRef := 5
			for _, neighbor := range node.NeighborNodes {
				for _, n := range nodes {
					if n.Name == neighbor {
						vni := fmt.Sprintf("%d00%d", nodeVniRef, neighborVniRef)
						neighborIP := strings.TrimSpace(n.NodeIP)
						command := fmt.Sprintf("ovs-vsctl add-port brtun vxlan%d -- set interface vxlan1 type=vxlan options:key=%s options:remote_ip=%s options:local_ip=%s options:dst_port=7000", neighborVniRef, vni, neighborIP, nodeIP)
						exec.Command(command).Output()
						if err != nil {
							fmt.Print(fmt.Errorf("Could not create vxlan between node %s and node %s. OVS responds: %w", node.Name, neighbor, err))
						}
					}
					neighborVniRef++
				}
			}
		}
		nodeVniRef++
	}
}
