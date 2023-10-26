package main

import (
	"encoding/json"
	"errors"
	"flag"
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

	configDir, vhostNumber, nodeName, controllerIP, err := takeArguments()

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	err = initializeSwitch(controllerIP)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Switch initialized and connected to the controller.")

	// Set all virtual interfaces up, and connect them to the tunnel bridge:
	for i := 1; i <= vhostNumber; i++ {
		vhost := fmt.Sprintf("vhost%d", i)
		cmd := exec.Command("ip", "link", "set", vhost, "up") // i.e: ip link set vhost1 up
		if err := cmd.Run(); err != nil {
			fmt.Println("Error:", err)
		}
		exec.Command("ovs-vsctl", "add-port", "brtun", vhost).Run() // i.e: ovs-vsctl add-port brtun vhost1
	}

	err = createVxlans(configDir, nodeName)

	if err != nil {
		fmt.Println("Error:", err)
		return
	}
}

func takeArguments() (string, int, string, string, error) {
	configDir := os.Args[len(os.Args)-1]

	vhostNumber := flag.Int("n_vpods", 0, "number of pod interfaces that are going to be attached to the switch")
	nodeName := flag.String("node_name", "", "name of the node the script is executed in. Required.")
	controllerIP := flag.String("controller_ip", "", "ip where the SDN controller is listening using the OpenFlow13 protocol. Required")

	flag.Parse()

	switch {
	case *nodeName == "":
		return "", 0, "", "", errors.New("Node name is not defined")
	case configDir == "":
		return "", 0, "", "", errors.New("Config directory is not defined")
	case *controllerIP == "":
		return "", 0, "", "", errors.New("Controller IP is not defined")
	}

	return configDir, *vhostNumber, *nodeName, *controllerIP, nil
}

func initializeSwitch(controllerIP string) error {

	var err error

	err = exec.Command("ovs-vsctl", "add-br", "brtun").Run()

	if err != nil {
		return err
	}

	err = exec.Command("ip", "link", "set", "brtun", "up").Run()

	if err != nil {
		return err
	}

	err = exec.Command("ovs-vsctl", "set", "bridge", "brtun", "OpenFlow13").Run()

	if err != nil {
		return err
	}

	target := fmt.Sprintf("tcp:%s:6633", controllerIP)

	err = exec.Command("ovs-vsctl", "set-controller", "brtun", target).Run()

	if err != nil {
		return err
	}
	return nil
}

func createVxlans(configDir, nodeName string) error {

	/// Read file and save in memory the JSON info
	data, err := ioutil.ReadFile(configDir)
	if err != nil {
		fmt.Println("Error reading input file:", err)
		return err
	}

	var nodes []Node
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		return err
	}

	// Search for the corresponding node in the configuration, according to the first passed parameter.
	// Once the node is found, create a bridge for every neighbour node defined.
	// The bridge is created with the nodeIp, neighborNodeIP and VNI. The VNI is generated according to the position of the node in the Json file. The first node will have the number 5
	// as a reference, the second a 6, and so on. And if a bridge between node 1 and node 2 is generated, it will have a vni of 5006, the two references with two 0s in between.
	// Another example would be node 3 of the cluster and node 9. Node 3 will have the reference 7 (5+3-1), and the Node 9 the reference 13(5 + 9 -1), resulting in the VNI 70013.
	// There's up to 2 ^ 24 possible vnis that are reduced to (2 ^24)/100 because of this measure (2 decimal digits are lost). So in total, a number of 167.772 virtual networks can be created.
	nodeVniRef := 5
	for _, node := range nodes {
		if node.Name == nodeName {
			nodeIP := strings.TrimSpace(node.NodeIP)
			for _, neighbor := range node.NeighborNodes {
				neighborVniRef := 5
				for _, n := range nodes {
					if n.Name == neighbor {
						var vni string
						if nodeVniRef < neighborVniRef {
							vni = fmt.Sprintf("%d00%d", nodeVniRef, neighborVniRef)

						} else {
							vni = fmt.Sprintf("%d00%d", neighborVniRef, nodeVniRef)
						}
						neighborIP := strings.TrimSpace(n.NodeIP)
						commandArgs := []string{
							"add-port",
							"brtun",
							fmt.Sprintf("vxlan%d", neighborVniRef),
							"--",
							"set", "interface",
							fmt.Sprintf("vxlan%d", neighborVniRef),
							"type=vxlan",
							fmt.Sprintf("options:key=%s", vni),
							fmt.Sprintf("options:remote_ip=%s", neighborIP),
							fmt.Sprintf("options:local_ip=%s", nodeIP),
							"options:dst_port=7000",
						}
						_, err := exec.Command("ovs-vsctl", commandArgs...).Output()
						if err != nil {
							return errors.New(fmt.Sprintf("Could not create vxlan between node %s and node %s.", node.Name, neighbor))
						} else {
							fmt.Println(fmt.Sprintf("Created vxlan between node %s and node %s.", node.Name, neighbor))
						}
					}
					neighborVniRef++
				}

			}
		}
		nodeVniRef++
	}
	return nil
}
