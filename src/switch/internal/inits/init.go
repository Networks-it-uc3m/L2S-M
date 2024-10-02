package inits

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	topo "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
	"github.com/Networks-it-uc3m/l2sm-switch/pkg/ovs"
)

func InitializeSwitch(switchName, controllerIP string) (ovs.Bridge, error) {

	re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	if !re.MatchString(controllerIP) {
		out, _ := exec.Command("host", controllerIP).Output()
		controllerIP = re.FindString(string(out))
	}

	controller := fmt.Sprintf("tcp:%s:6633", controllerIP)

	datapathId := ovs.GenerateDatapathID(switchName)
	bridge, err := ovs.NewBridge(ovs.Bridge{Name: switchName, Controller: controller, Protocol: "OpenFlow13", DatapathId: datapathId})

	return bridge, err
}

func ReadFile(configDir string, dataStruct interface{}) error {

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

/*
*
Example:

	        {
	            "Name": "l2sm1",
	            "nodeIP": "10.1.14.53",
				"neighborNodes":["10.4.2.3","10.4.2.5"]
			}
*/
func ConnectToNeighbors(bridge ovs.Bridge, node topo.Node) error {
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
