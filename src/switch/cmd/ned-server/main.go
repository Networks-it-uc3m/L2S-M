package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"

	"google.golang.org/grpc"

	// Adjust the import path based on your module path
	nedv1 "l2sm.local/ovs-switch/api/v1"

	"l2sm.local/ovs-switch/internal/inits"
	"l2sm.local/ovs-switch/pkg/nedpb"
	"l2sm.local/ovs-switch/pkg/ovs"
	"l2sm.local/ovs-switch/pkg/utils"
)

const (
	DEFAULT_CONFIG_PATH = "/etc/l2sm"
)

// server is used to implement nedpb.VxlanServiceServer
type server struct {
	nedpb.UnimplementedNedServiceServer
	Bridge   ovs.Bridge
	Settings nedv1.NedSettings
}

func main() {
	configDir, neighborsDir, err := takeArguments()

	if err != nil {
		fmt.Println("Error with the arguments provided. Error:", err)
		return
	}

	var settings nedv1.NedSettings
	err = inits.ReadFile(configDir, &settings)

	if err != nil {
		fmt.Println("Error with the provided file. Error:", err)
		return
	}

	fmt.Println("Initializing switch, connected to controller: ", settings.ControllerIP)

	nedBridgeName, _ := utils.GenerateBridgeName(settings.NedName)
	bridge, err := inits.InitializeSwitch(nedBridgeName, settings.ControllerIP)
	if err != nil {
		log.Fatalf("error initializing ned: %v", err)
	}

	var node nedv1.Node
	err = inits.ReadFile(neighborsDir, &node)

	if err != nil {
		fmt.Println("Error with the provided file. Error:", err)
		return
	}

	err = inits.ConnectToNeighbors(bridge, node)
	if err != nil {
		fmt.Println("Could not connect neighbors: ", err)
		return
	}

	// Listen on a TCP port
	lis, err := net.Listen("tcp", ":50051") // Choose your port
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create a gRPC server
	grpcServer := grpc.NewServer()

	// Register the server
	nedpb.RegisterNedServiceServer(grpcServer, &server{Bridge: bridge, Settings: settings})

	log.Printf("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// CreateVxlan implements nedpb.VxlanServiceServer
func (s *server) CreateVxlan(ctx context.Context, req *nedpb.CreateVxlanRequest) (*nedpb.CreateVxlanResponse, error) {
	ipAddress := req.GetIpAddress()

	// Implement your logic to create a VxLAN with the given IP address
	// For example, call a function from pkg/ovs/vsctl.go
	// success, message := ovs.CreateVxlan(ipAddress)

	// Placeholder implementation
	bridge := ovs.FromName("brtun")
	bridge.CreateVxlan(ovs.Vxlan{VxlanId: "", LocalIp: "", RemoteIp: ipAddress, UdpPort: ""})
	message := fmt.Sprintf("VxLAN with IP %s created successfully", ipAddress)

	return &nedpb.CreateVxlanResponse{
		Success: true,
		Message: message,
	}, nil
}

// AttachInterface implements nedpb.VxlanServiceServer
func (s *server) AttachInterface(ctx context.Context, req *nedpb.AttachInterfaceRequest) (*nedpb.AttachInterfaceResponse, error) {
	interfaceName := req.GetInterfaceName()

	// Create a new interface and attach it to the bridge
	newPort, err := AddInterfaceToBridge(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to add interface to bridge: %v", err)
	}

	interfaceNum, err := s.Bridge.GetPortNumber(newPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get port number: %v", err)
	}

	nodeName := s.Settings.NodeName
	if nodeName == "" {
		nodeName = "default-node" // Fallback if NODE_NAME is not set
	}

	return &nedpb.AttachInterfaceResponse{
		InterfaceNum: interfaceNum,
		NodeName:     nodeName,
	}, nil
}

// AddInterfaceToBridge creates a new veth pair, attaches one end to the specified bridge,
// and returns the name of the other end.
func AddInterfaceToBridge(bridgeName string) (string, error) {
	// Generate unique interface names
	timestamp := time.Now().UnixNano()
	vethName := fmt.Sprintf("veth%d", timestamp)
	peerName := fmt.Sprintf("vpeer%d", timestamp)

	// Create the veth pair
	if err := exec.Command("ip", "link", "add", vethName, "type", "veth", "peer", "name", peerName).Run(); err != nil {
		return "", fmt.Errorf("failed to create veth pair: %v", err)
	}

	// Set both interfaces up
	if err := exec.Command("ip", "link", "set", vethName, "up").Run(); err != nil {
		return "", fmt.Errorf("failed to set %s up: %v", vethName, err)
	}
	if err := exec.Command("ip", "link", "set", peerName, "up").Run(); err != nil {
		return "", fmt.Errorf("failed to set %s up: %v", peerName, err)
	}

	// Add one end to the Linux bridge
	if err := exec.Command("ip", "link", "set", vethName, "master", bridgeName).Run(); err != nil {
		return "", fmt.Errorf("failed to add %s to bridge %s: %v", vethName, bridgeName, err)
	}

	return peerName, nil
}

func takeArguments() (string, string, error) {

	configDir := flag.String("config_dir", fmt.Sprintf("%s/config.json", DEFAULT_CONFIG_PATH), "directory where the ned settings are specified. Required")
	neighborsDir := flag.String("neighbors_dir", fmt.Sprintf("%s/neighbors.json", DEFAULT_CONFIG_PATH), "directory where the ned's neighbors  are specified. Required")

	flag.Parse()

	return *configDir, *neighborsDir, nil
}
