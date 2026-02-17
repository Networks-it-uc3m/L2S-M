// Copyright 2024 Universidad Carlos III de Madrid
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package talpainterface

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/client"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	talpav1 "github.com/Networks-it-uc3m/l2sm-switch/api/v1"
	dp "github.com/Networks-it-uc3m/l2sm-switch/pkg/datapath"
	nedpb "github.com/Networks-it-uc3m/l2sm-switch/pkg/nedpb"
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
		if ned.Spec.Provider.Name == providerName {
			// Return the first matching device
			return ned, nil
		}
	}

	// Return a clearer message indicating the provider was not found.
	return l2smv1.NetworkEdgeDevice{}, fmt.Errorf("no NetworkEdgeDevice found for provider: %s", providerName)
}

func ProbeInterface(node, provider string) string {

	ifid := dp.New(dp.GetSwitchName(dp.DatapathParams{NodeName: node, ProviderName: provider}))
	return ifid.Probe(talpav1.RESERVED_PROBE_ID)
}
