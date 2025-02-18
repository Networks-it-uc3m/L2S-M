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

package dnsinterface

import (
	"context"
	"fmt"
	"time"

	"github.com/Networks-it-uc3m/l2sm-dns/api/v1/dns"
	dnspb "github.com/Networks-it-uc3m/l2sm-dns/api/v1/dns"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type DNSClient struct {
	ServerAddress string
	Scope         string
}

func (client *DNSClient) AddDNSEntry(podName, networkName, ipAddress string) error {

	// Create a gRPC connection.
	conn, err := grpc.NewClient(client.ServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %v", client.ServerAddress, err)
	}
	defer conn.Close()

	// Create a DNS service client.
	dnsClient := dnspb.NewDnsServiceClient(conn)

	req := &dns.AddEntryRequest{
		Entry: &dnspb.DNSEntry{
			PodName:   podName,
			IpAddress: ipAddress,
			Scope:     client.Scope,
			Network:   networkName,
		},
	}
	// Wrap the call in a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = dnsClient.AddEntry(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add DNS entry: %v", err)
	}
	return nil
}
