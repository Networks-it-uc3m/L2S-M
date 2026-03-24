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

package monitoringnetwork

import (
	"fmt"

	l2smv1 "github.com/Networks-it-uc3m/L2S-M/api/v1"
	"github.com/Networks-it-uc3m/L2S-M/internal/env"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

type ClientFactory interface {
	Internal() (sdnclient.Client, error)
	ForProvider(provider *l2smv1.ProviderSpec) (sdnclient.Client, error)
}

type DefaultClientFactory struct{}

func (DefaultClientFactory) Internal() (sdnclient.Client, error) {
	clientConfig := sdnclient.ClientConfig{
		BaseURL:  fmt.Sprintf("http://%s:%s/onos", env.GetControllerIP(), env.GetControllerPort()),
		Username: "karaf",
		Password: "karaf",
	}

	return sdnclient.NewClient(sdnclient.InternalType, clientConfig)
}

func (DefaultClientFactory) ForProvider(provider *l2smv1.ProviderSpec) (sdnclient.Client, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider is nil")
	}
	if len(provider.Domain) == 0 || provider.Domain[0] == "" {
		return nil, fmt.Errorf("provider %q has no domain configured", provider.Name)
	}

	clientConfig := sdnclient.ClientConfig{
		BaseURL:  fmt.Sprintf("http://%s:%s/onos", provider.Domain[0], utils.DefaultIfEmpty(provider.SDNPort, "30808")),
		Username: "karaf",
		Password: "karaf",
	}

	// Monitoring networks still rely on the vnets API plus port attachment.
	// Keep provider selection isolated here so the concrete client type can be
	// switched once the external provider exposes equivalent capabilities.
	return sdnclient.NewClient(sdnclient.InternalType, clientConfig)
}
