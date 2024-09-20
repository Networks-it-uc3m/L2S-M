package main

import (
	"errors"
	"flag"
	"fmt"
	"os/exec"
	"regexp"

	"l2sm.local/ovs-switch/pkg/ovs"
)

// Script that takes two required arguments:
// the first one is the name in the cluster of the node where the script is running
// the second one is the path to the configuration file, in reference to the code.
func main() {

	vethNumber, controllerIP, switchName, err := takeArguments()

	if err != nil {
		fmt.Println("Error with the arguments. Error:", err)
		return
	}

	fmt.Println("Initializing switch, connected to controller: ", controllerIP)

	bridge, err := initializeSwitch(switchName, controllerIP)

	if err != nil {

		fmt.Println("Could not initialize switch. Error:", err)
		return
	}

	fmt.Println("Switch initialized and connected to the controller.")

	// Set all virtual interfaces up, and connect them to the tunnel bridge:
	for i := 1; i <= vethNumber; i++ {
		veth := fmt.Sprintf("net%d", i)
		if err := bridge.AddPort(veth); err != nil {
			fmt.Println("Error:", err)
		}
	}
	fmt.Printf("Switch initialized, current state: ", bridge)
}

func takeArguments() (int, string, string, error) {

	vethNumber := flag.Int("n_veths", 0, "number of pod interfaces that are going to be attached to the switch")
	controllerIP := flag.String("controller_ip", "", "ip where the SDN controller is listening using the OpenFlow13 protocol. Required")
	switchName := flag.String("switch_name", "", "name of the switch that will be used to set a custom datapath id. If not set, a randome datapath will be assigned")
	flag.Parse()

	switch {
	case *controllerIP == "":
		return 0, "", "", errors.New("controller IP is not defined")
	}

	return *vethNumber, *controllerIP, *switchName, nil
}

func initializeSwitch(switchName, controllerIP string) (ovs.Bridge, error) {

	re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	if !re.MatchString(controllerIP) {
		out, _ := exec.Command("host", controllerIP).Output()
		controllerIP = re.FindString(string(out))
	}

	controller := fmt.Sprintf("tcp:%s:6633", controllerIP)

	datapathId := ovs.GenerateDatapathID(switchName)
	bridge, err := ovs.NewBridge(ovs.Bridge{Name: "brtun", Controller: controller, Protocol: "OpenFlow13", DatapathId: datapathId})

	return bridge, err
}
