/*******************************************************************************
 * Copyright 2024  Universidad Carlos III de Madrid
 * 
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.  You may obtain a copy
 * of the License at
 * 
 *   http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 * License for the specific language governing permissions and limitations under
 * the License.
 * 
 * SPDX-License-Identifier: Apache-2.0
 ******************************************************************************/
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

	configDir, nodeName, err := takeArguments()

	if err != nil {
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	err = createVxlans(configDir, nodeName)

	if err != nil {
		fmt.Println("Vxlans not created: ", err)
		return
	}
}

func takeArguments() (string, string, error) {
	configDir := os.Args[len(os.Args)-1]

	nodeName := flag.String("node_name", "", "name of the node the script is executed in. Required.")

	flag.Parse()

	switch {
	case *nodeName == "":
		return "", "", errors.New("node name is not defined")
	case configDir == "":
		return "", "", errors.New("config directory is not defined")
	}

	return configDir, *nodeName, nil
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
							"options:key=flow",
							fmt.Sprintf("options:remote_ip=%s", neighborIP),
							fmt.Sprintf("options:local_ip=%s", nodeIP),
							"options:dst_port=7000",
						}
						_, err := exec.Command("ovs-vsctl", commandArgs...).Output()
						if err != nil {
							return fmt.Errorf("could not create vxlan between node %s and node %s", node.Name, neighbor)
						} else {
							fmt.Printf("Created vxlan between node %s and node %s.\n", node.Name, neighbor)
						}
					}
					vxlanNumber++
				}

			}
		}
	}
	return nil
}
