package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
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
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	fmt.Println("initializing switch, connected to controller: ", controllerIP)
	err = initializeSwitch(controllerIP)

	if err != nil {
		fmt.Println("Could not initialize switch. Error:", err)
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
		fmt.Println("Vxlans not created: ", err)
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

	re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	if !re.MatchString(controllerIP) {
		out, _ := exec.Command("host", controllerIP).Output()
		controllerIP = re.FindString(string(out))
	}

	var err error

	err = exec.Command("ovs-vsctl", "add-br", "brtun").Run()

	if err != nil {
		return errors.New("Could not create brtun interface")
	}

	err = exec.Command("ip", "link", "set", "brtun", "up").Run()

	if err != nil {
		return errors.New("Could not set brtun interface up")
	}

	err = exec.Command("ovs-vsctl", "set", "bridge", "brtun", "protocols=OpenFlow13").Run()

	if err != nil {
		return errors.New("Couldnt set brtun messaing protocol to OpenFlow13")
	}

	target := fmt.Sprintf("tcp:%s:6633", controllerIP)

	err = exec.Command("ovs-vsctl", "set-controller", "brtun", target).Run()

	if err != nil {
		return errors.New("Could not connect to controller")
	}
	return nil
}

func createVxlans(configDir, nodeName string) error {

	/// Read file and save in memory the JSON info
	data, err := ioutil.ReadFile(configDir)
	if err != nil {
		fmt.Println("No input file was found.", err)
		return err
	}

	var nodes []Node
	err = json.Unmarshal(data, &nodes)
	if err != nil {
		return err
	}

	// Search for the corresponding node in the configuration, according to the first passed parameter.
	// Once the node is found, create a bridge for every neighbour node defined.
	// The bridge is created with the nodeIp and neighborNodeIP and VNI. The VNI is generated in the l2sm-controller thats why its set to 'flow'.
	for _, node := range nodes {
		if node.Name == nodeName {
			nodeIP := strings.TrimSpace(node.NodeIP)
			for _, neighbor := range node.NeighborNodes {
				vxlanNumber := 1
				for _, n := range nodes {
					if n.Name == neighbor {
						neighborIP := strings.TrimSpace(n.NodeIP)
						commandArgs := []string{
							"add-port",
							"brtun",
							fmt.Sprintf("vxlan%d", vxlanNumber),
							"--",
							"set", "interface",
							fmt.Sprintf("vxlan%d", vxlanNumber),
							"type=vxlan",
							fmt.Sprintf("options:key=flow"),
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
					vxlanNumber++
				}

			}
		}
	}
	return nil
}
