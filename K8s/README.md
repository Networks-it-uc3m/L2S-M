# How does L2S-M work?

L2S-M takes a different approach to K8s networking in comparison with other solutions available, which mostly implement CNI plugins to be used as the main connectivity basis for a cluster. L2S-M is deployed as a complementary solution to these CNI Plugins, since it allows the creation and management of virtual networks in a K8s cluster in order to provide workloads with one (or several) interface(s) to communicate with other pods attached to the same network(s) at the link-layer level. The main CNI Plugin interface in these pods remains intact, allowing the standard K8s functionalities to still available for the pods (services, communications using the main iterface, etc...).

In the following figure, you can see an example of a K8s cluster with L2S-M installed and running. In principle, L2S-M builds a programmable data plane between different programmable switches deployed (or present) over a K8s infrastructure. These switches can be either physical (like they can be found in a classic datacentre infrastructure) or virtual (deployed by the L2S-M operator). 

![alt text](https://github.com/Networks-it-uc3m/L2S-M/blob/main/v1_architecture.png?raw=true)

**NOTE**: The current version of L2S-M supports the deployment of virtual switches based on [Open Virtual Switch (OVS)](http://www.openvswitch.org). Physical switch infrastructures will be supported in future versions.

Each one of these switches are interconnected with selected peers by taking advantage of IP tunneling mechanisms (VXLAN) to build an overlay of programmable switches. Aftearwards, an SDN Application is used to inject the corresponding traffic rules to ensure that traffic between virtual network are isolated between each other (i.e., instrucing the switches which ports should be used to forwards and/or block incoming-outgoing traffic).

Specifically for K8s clusters, the element in charge of managing the creation, deletion and management of virtual networks is the L2S-M operator. This operator treats virtual networks as Multus CRDs, using the K8s events to detect the instances where a pod wants to attach/detach from a virtual network. In the former case, the operator will select one of the available interfaces in the switch, and associate it with the virtual network that wants to be used. This interface will appear in the pod as a secondary interface that can be used to communicate with other pods attached to the network, which will be seen as if they were deployed in the same Local Area Network (LAN). The CNI interface remains intact.

To provide the isolation mechanisms between virtual networks, an SDN controller is deployed in the cluster as part of the L2S-M solution. The operator will interact with this compotent to communicate which ports are associated with each virtual network, updating its status everytime a pod is deployed/deleted. Using this information, the SDN Controller injects the corresponding rules in the switches, forwarding and/or blocking traffic according to the virtual networks being used at each moment.

**NOTE**: An initial implementation of an SDN controller based on [RYU](https://ryu.readthedocs.io/en/latest/) will be available in a few weeks. Afterwards, an SDN application that support link-layer isolation between virtual networks will be released in subsequent releases. 

More information on how to deploy virtualise workloads attached to virtual networks can be seen in the [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors) example.
