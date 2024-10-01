package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"l2sm.local/ovs-switch/pkg/nedpb"

	"google.golang.org/grpc"
)

func main() {
	// Set up a connection to the gRPC server.
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure()) // Update with the actual server address and credentials if needed
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	// Create a new client for the NedService.
	client := nedpb.NewNedServiceClient(conn)

	// Create a context with a timeout to ensure that the request doesn't hang.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Prepare the request with the interface name "br10".
	req := &nedpb.AttachInterfaceRequest{
		InterfaceName: "br10",
	}

	// Call the AttachInterface method.
	resp, err := client.AttachInterface(ctx, req)
	if err != nil {
		log.Fatalf("Error calling AttachInterface: %v", err)
	}

	// Handle and display the response.
	fmt.Printf("Interface attached successfully:\n")
	fmt.Printf("Interface Number: %d\n", resp.GetInterfaceNum())
	fmt.Printf("Node Name: %s\n", resp.GetNodeName())
}
