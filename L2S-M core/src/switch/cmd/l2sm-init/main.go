/*******************************************************************************
 * Copyright 2024  Charles III University of Madrid
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
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"regexp"
)

// Script that takes two required arguments:
// the first one is the name in the cluster of the node where the script is running
// the second one is the path to the configuration file, in reference to the code.
func main() {

	vethNumber, controllerIP, err := takeArguments()

	if err != nil {
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	fmt.Println("Initializing switch, connected to controller: ", controllerIP)
	err = initializeSwitch(controllerIP)

	if err != nil {
		fmt.Println("Could not initialize switch. Error:", err)
		return
	}

	fmt.Println("Switch initialized and connected to the controller.")

	// Set all virtual interfaces up, and connect them to the tunnel bridge:
	for i := 1; i <= vethNumber; i++ {
		veth := fmt.Sprintf("net%d", i)
		cmd := exec.Command("ip", "link", "set", veth, "up") // i.e: ip link set veth1 up
		if err := cmd.Run(); err != nil {
			fmt.Println("Error:", err)
		}
		exec.Command("ovs-vsctl", "add-port", "brtun", veth).Run() // i.e: ovs-vsctl add-port brtun veth1
	}
}

func takeArguments() (int, string, error) {

	vethNumber := flag.Int("n_veths", 0, "number of pod interfaces that are going to be attached to the switch")
	controllerIP := flag.String("controller_ip", "", "ip where the SDN controller is listening using the OpenFlow13 protocol. Required")

	flag.Parse()

	switch {
	case *controllerIP == "":
		return 0, "", errors.New("controller IP is not defined")
	}

	return *vethNumber, *controllerIP, nil
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
		return errors.New("could not create brtun interface")
	}

	err = exec.Command("ip", "link", "set", "brtun", "up").Run()

	if err != nil {
		return errors.New("could not set brtun interface up")
	}

	err = exec.Command("ovs-vsctl", "set", "bridge", "brtun", "protocols=OpenFlow13").Run()

	if err != nil {
		return errors.New("could not set brtun messaing protocol to OpenFlow13")
	}

	target := fmt.Sprintf("tcp:%s:6633", controllerIP)

	err = exec.Command("ovs-vsctl", "set-controller", "brtun", target).Run()

	if err != nil {
		return errors.New("could not connect to controller")
	}
	return nil
}
