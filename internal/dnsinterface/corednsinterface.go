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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/l2sm-dns/pkg/configmapmanager"
)

func AddServerToLocalCoreDNS(c client.Client, networkName, remoteServerDomain, remoteServerPort string) error {
	// Create a new DNSManager using the factory.
	// Since we already have a controller-runtime client,
	// pass nil for the rest.Config parameter.
	dnsManager, err := configmapmanager.NewDNSManager(
		env.GetIntraConfigmapNamespace(),
		env.GetIntraConfigmapName(),
		nil, // no k8sConfig needed if crClient is provided
		c,
	)
	if err != nil {
		return fmt.Errorf("could not create DNS manager: %v", err)
	}

	domainName := fmt.Sprintf("%s.%s.l2sm", networkName, "inter")
	return dnsManager.AddServerToConfigMap(context.TODO(), domainName, remoteServerDomain, remoteServerPort)
}
