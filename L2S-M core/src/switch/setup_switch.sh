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

ovsdb-server --remote=punix:/var/run/openvswitch/db.sock --remote=db:Open_vSwitch,Open_vSwitch,manager_options --pidfile=/var/run/openvswitch/ovsdb-server.pid --detach 

ovs-vsctl --db=unix:/var/run/openvswitch/db.sock --no-wait init 

ovs-vswitchd --pidfile=/var/run/openvswitch/ovs-vswitchd.pid --detach 

l2sm-init --n_veths=$NVETHS --controller_ip=$CONTROLLERIP 

#l2sm-vxlans --node_name=$NODENAME /etc/l2sm/switchConfig.json
sleep infinity
