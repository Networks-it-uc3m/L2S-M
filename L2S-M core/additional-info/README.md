<!---
 Copyright 2024  Universidad Carlos III de Madrid
 
 Licensed under the Apache License, Version 2.0 (the "License"); you may not
 use this file except in compliance with the License.  You may obtain a copy
 of the License at
 
   http://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 License for the specific language governing permissions and limitations under
 the License.
 
 SPDX-License-Identifier: Apache-2.0
-->

# How does L2S-M work?

L2S-M takes a different approach to K8s networking in comparison with other solutions available, which mostly implement CNI plugins to be used as the main connectivity basis for a cluster. L2S-M is deployed as a complementary solution to these CNI Plugins, since it allows the creation and management of virtual networks in a K8s cluster in order to provide workloads with one (or several) interface(s) to communicate with other workloads attached to the same network(s) at the link-layer. The main CNI Plugin interface in these pods remains intact, allowing the standard K8s functionalities to still be available for the pods (services, communications using the main interface, etc.).

The following figure outlines a high-level overview of L2S-M, with an illustrative example of a K8s cluster with L2S-M installed and running. L2S-M builds a programmable data plane using SDN switches over a K8s infrastructure. These switches can be either virtual (deployed by the L2S-M operator) or physical (such as those that can be found in a classic datacentre infrastructure). 

![alt text](../assets/v1_architecture.png?raw=true)

**NOTE**: The current version of L2S-M utilizes an infrastructure of virtual switches based on [Open Virtual Switch (OVS)](http://www.openvswitch.org). The integration of physical switches is currently ongoing.

In L2S-M, a k8s node deploys a virtual SDN switch or is connected to a physical SDN switch. Virtual switches are interconnected through point-to-point links. These links are established using IP tunnels (based on VXLAN technologies). This way, SDN switches build an overlay network that interconnects all the K8s nodes. L2S-M uses an SDN controller to install forwarding rules on the virtual/physical switches. This way, data traffic among workloads is appropriately distributed through isolated virtual networks (i.e., the SDN controller instructs the switches which ports should be used to forward and/or block incoming/outgoing traffic).

Specifically for K8s clusters, the element in charge of managing the creation, deletion and management of virtual networks is the L2S-M operator. This operator treats virtual networks as Multus CRDs, using the K8s events to detect the instances where a pod wants to attach/detach from a virtual network. In the former case, the operator will select one of the available interfaces in the SDN switch, and associate it with the virtual network that wants to be used. This interface will appear in the pod as a secondary interface that can be used to communicate with other pods attached to the network, which will be seen as if they were deployed in the same Local Area Network (LAN). The CNI interface remains intact.

To provide isolation among virtual networks, the operator interacts with the SDN controller component to communicate which ports are associated with each virtual network, updating its status every time a pod is deployed/deleted. Using this information, the SDN controller injects the corresponding rules in the switches, forwarding and/or blocking traffic according to the virtual networks being used at each moment.

**NOTE**: The current version of L2S-M utilizes a simple-switch implementation based on the [RYU](https://ryu.readthedocs.io/en/latest/) SDN controller. An SDN application to specifically support virtual network isolation is currently under implementation.

More information on how to deploy virtualise workloads attached to virtual networks can be seen in the [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/examples/ping-pong) example.
