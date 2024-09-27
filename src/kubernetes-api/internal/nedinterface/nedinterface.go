package nedinterface

import (
	"context"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	nedpb "l2sm.local/ovs-switch/pkg/nedpb"
)

// GetConnectionInfo communicates with the NED via gRPC and returns the InterfaceNum and NodeName.
func AttachInterface(nedNetworkAttachDef string) (string, error) {
	// Get the NED address (e.g., from environment variable or configuration)
	nedAddress := os.Getenv("NED_ADDRESS")
	if nedAddress == "" {
		nedAddress = "localhost:50051" // default address
	}

	// Set up a connection to the server.
	client, err := grpc.NewClient(nedAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", fmt.Errorf("did not connect: %v", err)
	}

	defer client.Close()

	c := nedpb.NewNedServiceClient(client)

	// Set a timeout for the context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Now call AttachInterface
	attachReq := &nedpb.AttachInterfaceRequest{
		InterfaceName: nedNetworkAttachDef,
	}

	attachRes, err := c.AttachInterface(ctx, attachReq)
	if err != nil {
		return "", fmt.Errorf("could not attach interface: %v", err)
	}

	return fmt.Sprint(attachRes.GetInterfaceNum()), nil
}

// TODO
func GetNodeName(providerName string) string {
	return ""
}
