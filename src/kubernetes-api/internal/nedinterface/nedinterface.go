package nedinterface

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/client"

	l2smv1 "l2sm.k8s.local/controllermanager/api/v1"
	nedpb "l2sm.local/ovs-switch/pkg/nedpb"
)

// GetConnectionInfo communicates with the NED via gRPC and returns the InterfaceNum and NodeName.
func AttachInterface(nedAddress, nedNetworkAttachDef string) (string, error) {

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

func GetNetworkEdgeDevice(ctx context.Context, c client.Client, providerName string) (l2smv1.NetworkEdgeDevice, error) {
	neds := &l2smv1.NetworkEdgeDeviceList{}

	if err := c.List(ctx, neds); err != nil {
		return l2smv1.NetworkEdgeDevice{}, fmt.Errorf("failed to list NetworkEdgeDevices: %w", err)
	}

	for _, ned := range neds.Items {
		if ned.Spec.NetworkController.Name == providerName {
			// Return the first matching device
			return ned, nil
		}
	}

	// Return a clearer message indicating the provider was not found.
	return l2smv1.NetworkEdgeDevice{}, fmt.Errorf("no NetworkEdgeDevice found for provider: %s", providerName)
}
