package ovs

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// TODO: Abstract exec client as a separate entity, that doesnt use CLI. The following presented hardcoded way is not clean.

type Port struct {
	Name   string
	Status string
}

type Bridge struct {
	Controller string
	Name       string
	Protocol   string
	DatapathId string
	Ports      []Port
}

type Vxlan struct {
	VxlanId  string
	LocalIp  string
	RemoteIp string
	UdpPort  string
}

func FromName(bridgeName string) Bridge {

	bridge := Bridge{Name: bridgeName}

	bridge.getPorts()

	return bridge
}

func NewBridge(bridgeConf Bridge) (Bridge, error) {

	var err error

	bridge := Bridge{}

	err = exec.Command("ovs-vsctl", "add-br", bridgeConf.Name).Run()

	if err != nil {
		return bridge, errors.New("could not create brtun interface")
	}

	bridge.Name = bridgeConf.Name

	err = exec.Command("ip", "link", "set", bridge.Name, "up").Run()

	if err != nil {
		return bridge, errors.New("could not set brtun interface up")
	}

	if bridgeConf.DatapathId != "" {
		err := exec.Command("ovs-vsctl", "set", "bridge", bridge.Name, fmt.Sprintf("other-config:datapath-id=%s", bridgeConf.DatapathId)).Run()
		if err != nil {
			return bridge, errors.New("could not set custom datapath id")
		}
	}

	protocolString := fmt.Sprintf("protocols=%s", bridgeConf.Protocol)
	err = exec.Command("ovs-vsctl", "set", "bridge", "brtun", protocolString).Run()

	if err != nil {
		return bridge, errors.New("could not set brtun messaging protocol to OpenFlow13")
	}

	bridge.Protocol = bridgeConf.Protocol

	err = exec.Command("ovs-vsctl", "set-controller", bridge.Name, bridgeConf.Controller).Run()

	if err != nil {
		return bridge, errors.New("could not connect to controller")
	}

	bridge.Controller = bridgeConf.Name

	return bridge, nil
}

func (bridge *Bridge) CreateVxlan(vxlan Vxlan) error {

	commandArgs := []string{
		"add-port",
		bridge.Name,
		vxlan.VxlanId,
		"--",
		"set", "interface",
		vxlan.VxlanId,
		"type=vxlan",
		"options:key=flow",
		fmt.Sprintf("options:remote_ip=%s", vxlan.RemoteIp),
		fmt.Sprintf("options:local_ip=%s", vxlan.LocalIp),
		fmt.Sprintf("options:dst_port=%s", vxlan.UdpPort),
	}
	_, err := exec.Command("ovs-vsctl", commandArgs...).Output()

	return err

}

func (bridge *Bridge) AddPort(portName string) error {

	cmd := exec.Command("ip", "link", "set", portName, "up") // i.e: ip link set veth1 up
	if err := cmd.Run(); err != nil {
		return err
	}
	exec.Command("ovs-vsctl", "add-port", bridge.Name, portName).Run() // i.e: ovs-vsctl add-port brtun veth1
	bridge.Ports = append(bridge.Ports, Port{Name: portName, Status: "UP"})
	return nil
}

func (bridge *Bridge) getPorts() error {
	// Executes the ovs-vsctl command to list ports on the bridge
	cmd := exec.Command("ovs-vsctl", "list-ports", bridge.Name)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return err
	}

	// Split the output by lines for each port name
	portNames := strings.Split(out.String(), "\n")
	for _, portName := range portNames {
		if portName == "" {
			continue
		}
		// TODO:, retrieve more details for each port; here we just set the name
		port := Port{Name: portName}

		// Retrieve status
		// cmd = exec.Command("ovs-vsctl", "get", "Interface", portName, "status")

		// Add the port to the Ports slice
		bridge.Ports = append(bridge.Ports, port)
	}

	return nil
}
