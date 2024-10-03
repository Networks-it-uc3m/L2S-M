#!/bin/bash
#-------------------------------------------------------------------------------
# Copyright 2024  Universidad Carlos III de Madrid
# 
# Licensed under the Apache License, Version 2.0 (the "License"); you may not
# use this file except in compliance with the License.  You may obtain a copy
# of the License at
# 
#   http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
# License for the specific language governing permissions and limitations under
# the License.
# 
# SPDX-License-Identifier: Apache-2.0
#-------------------------------------------------------------------------------

# turn on bash's job control
set -m

# Start the onos server and put it on the background
./bin/onos-service server &

sleep 10

while true; do
    response=$(wget --spider --server-response http://localhost:8181/onos/ui 2>&1)
    status_codes=$(echo "$response" | awk '/HTTP\/1.1/{print $2}')

    if echo "$status_codes" | grep -q "200"; then
        echo "Starting the configuration"
        break
    fi

    sleep 10
done


# Start the configuration

./bin/onos-app localhost activate org.onosproject.drivers
./bin/onos-app localhost activate org.onosproject.lldpprovider
./bin/onos-app localhost activate org.onosproject.openflow-base
./bin/onos-app localhost activate org.onosproject.optical-model
./bin/onos-app localhost install! l2sm-controller-app-1.0.oar



# now we bring the server into the foreground
fg %1