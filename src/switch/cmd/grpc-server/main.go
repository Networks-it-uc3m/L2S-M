package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	// Adjust the import path based on your module path

	"l2sm.local/ovs-switch/pkg/nedpb"
	"l2sm.local/ovs-switch/pkg/ovs"
)

// server is used to implement nedpb.VxlanServiceServer
type server struct {
	nedpb.UnimplementedVxlanServiceServer
}

// CreateVxlan implements nedpb.VxlanServiceServer
func (s *server) CreateVxlan(ctx context.Context, req *nedpb.CreateVxlanRequest) (*nedpb.CreateVxlanResponse, error) {
	ipAddress := req.GetIpAddress()

	// Implement your logic to create a VxLAN with the given IP address
	// For example, call a function from pkg/ovs/vsctl.go
	// success, message := ovs.CreateVxlan(ipAddress)

	// Placeholder implementation
	bridge := ovs.FromName("brtun")
	bridge.CreateVxlan(ovs.Vxlan{"", "", ipAddress, ""})
	message := fmt.Sprintf("VxLAN with IP %s created successfully", ipAddress)

	return &nedpb.CreateVxlanResponse{
		Success: success,
		Message: message,
	}, nil
}

// AttachInterface implements nedpb.VxlanServiceServer
func (s *server) AttachInterface(ctx context.Context, req *nedpb.AttachInterfaceRequest) (*nedpb.AttachInterfaceResponse, error) {
	interfaceName := req.GetInterfaceName()

	// Implement your logic to attach the interface to the bridge
	// For example, call a function from pkg/ovs/vsctl.go
	// openflowID, err := ovs.AttachInterface(interfaceName)

	// Placeholder implementation
	nodeName := os.Getenv("NODE_NAME")

	return &nedpb.AttachInterfaceResponse{
		OpenflowId: openflowID,
		NodeName:   nodeName,
	}, nil
}

func main() {
	// Listen on a TCP port
	lis, err := net.Listen("tcp", ":50051") // Choose your port
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Create a gRPC server
	grpcServer := grpc.NewServer()

	// Register the server
	nedpb.RegisterVxlanServiceServer(grpcServer, &server{})

	log.Printf("gRPC server listening on :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
