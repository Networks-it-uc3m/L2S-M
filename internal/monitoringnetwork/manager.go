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
	"github.com/Networks-it-uc3m/L2S-M/internal/lpminterface"
	"github.com/Networks-it-uc3m/L2S-M/internal/sdnclient"
	"github.com/Networks-it-uc3m/L2S-M/internal/utils"
)

// Manager owns monitoring-network lifecycle operations independently of any
// specific reconciler or LPM resource-building logic.
type Manager struct {
	Client sdnclient.Client
}

func (m Manager) Ensure(name, providerName string, nodes []string) error {
	if m.Client == nil {
		return fmt.Errorf("monitoring network client is nil")
	}

	lpmNetName := utils.GenerateLPMNetworkName(name)
	if err := m.Client.CreateNetwork(l2smv1.NetworkTypeVnet, sdnclient.VnetPayload{NetworkId: lpmNetName}); err != nil {
		return fmt.Errorf("create monitoring network %q: %w", lpmNetName, err)
	}

	lpmPorts := lpminterface.GenerateLPMPorts(nodes, providerName)
	if len(lpmPorts) == 0 {
		return nil
	}

	if err := m.Client.AttachPodToNetwork(
		l2smv1.NetworkTypeVnet,
		sdnclient.VnetPayload{NetworkId: lpmNetName, Port: lpmPorts},
	); err != nil {
		return fmt.Errorf("attach monitoring ports to network %q: %w", lpmNetName, err)
	}

	return nil
}

func (m Manager) Delete(name string) error {
	if m.Client == nil {
		return fmt.Errorf("monitoring network client is nil")
	}

	lpmNetName := utils.GenerateLPMNetworkName(name)
	if err := m.Client.DeleteNetwork(l2smv1.NetworkTypeVnet, lpmNetName); err != nil {
		return fmt.Errorf("delete monitoring network %q: %w", lpmNetName, err)
	}

	return nil
}
