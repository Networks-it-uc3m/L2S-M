/*******************************************************************************
 * Copyright 2024  Charles III University of Madrid
 * 
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.  You may obtain a copy
 * of the License at
 * 
 *   http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 * License for the specific language governing permissions and limitations under
 * the License.
 * 
 * SPDX-License-Identifier: Apache-2.0
 ******************************************************************************/
package sdnclient

import (
	"encoding/json"
	"fmt"
	"net/http"

	l2smv1 "l2sm.k8s.local/l2smnetwork/api/v1"
)

// ExternalClient is part of the Client interface, and implements the SessionClient, which is a wrapper of the http function
// this type of client is for the specific idco onos app, which manages inter cluster networks.
type ExternalClient struct {
	Session *SessionClient
}

func (c *ExternalClient) beginSessionController() bool {
	//TODO: implement healthcheck in idco onos app
	resp, err := c.Session.Get("/idco/mscs/status")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check if the status code indicates success (HTTP 200 OK).
	return resp.StatusCode == http.StatusOK
}

// CreateNetwork creates a new network in the SDN controller
func (c *ExternalClient) CreateNetwork(networkType l2smv1.NetworkType, config interface{}) error {

	jsonData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	response, err := c.Session.Post("/idco/mscs", jsonData)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to create network, status code: %d", response.StatusCode)
	}

	return nil
}

// CheckNetworkExists checks if the specified network exists in the SDN controller
func (c *ExternalClient) CheckNetworkExists(networkType l2smv1.NetworkType, networkID string) (bool, error) {
	response, err := c.Session.Get(fmt.Sprintf("/idco/mscs/%s", networkID))
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	return response.StatusCode == http.StatusOK, nil
}

// DeleteNetwork deletes an existing network from the SDN controller
func (c *ExternalClient) DeleteNetwork(networkType l2smv1.NetworkType, networkID string) error {
	response, err := c.Session.Delete(fmt.Sprintf("/idco/mscs/%s", networkID))
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("SDN controller responded with status code: %d", response.StatusCode)
	}

	return nil
}
