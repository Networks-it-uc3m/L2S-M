# L2S-M 
Welcome to the official repository of L2S-M, a Kubernetes operator that enables virtual networking in K8s clusters.

In the following figure, you can see an example of a K8s cluster with L2S-M installed and running. In principle, L2S-M builds a programmable data plane between different programmable switches deployed (or present) over a K8s infrastructure. These switches can be either physical (like they can be found in a classic datacentre infrastructure) or virtual (deployed by the L2S-M operator). 

![alt text](https://github.com/Networks-it-uc3m/L2S-M/blob/main/v1_architecture.png?raw=true)

**NOTE**: The current verison of L2S-M only supports the deployment of virtual switches based on [Open Virtual Switch (OVS)](http://www.openvswitch.org).

Each one of these switches are interconnected with selected peers by taking advantage of IP tunneling mechanisms (VXLAN) to build an overlay of programmable switches. Aftearwards, an SDN Application is used to inject the corresponding traffic rules to ensure that traffic between virtual network are isolated between each other (i.e., instrucing the switches which ports should be used to forwards and/or block incoming-outgoing traffic).

Specifically for K8s clusters, the element in charge of managing the creation, deletion and management of virtual networks is the L2S-M operator. This operator treats virtual networks as Multus CRDs, using the K8s events to detect the instances where a pod wants to attach/detach from a virtual network. In the former case, the operator will select one of the available interfaces in the switch, and associate it with the virtual network that wants to be used. This interface will appear in the pod as a secondary interface that can be used to communicate with other pods attached to the network, which will be seen as if they were deployed in the same Local Area Network (LAN). The CNI interface remains intact.

To provide the isolation mechanisms between virtual networks, an SDN controller is deployed in the cluster as part of the L2S-M solution. The operator will interact with this compotent to communicate which ports are associated with each virtual network, updating its status everytime a pod is deployed/deleted. Using this information, the SDN Controller injects the corresponding rules in the switches, forwarding and/or blocking traffic according to the virtual networks being used at each moment.

**NOTE**: The current version of L2S-M does not implement an SDN controller yet: the first iteration of this component is expected to be added in the near future. 

More information on how to deploy virtualise workloads attached to virtual networks can be seen in the [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors) example.

If you want to learn how to install L2S-M in your cluster, see the [installation guide](https://github.com/Networks-it-uc3m/L2S-M/tree/main/operator) of this repository to start its installation.

If you want more information about the original idea of L2S-M and its initial design, you can check our latest publication in the [IEEE Network journal](https://ieeexplore.ieee.org/document/9740640).

Did you already install the operator and  you cannot wait to start building your own virtual networks in your K8s cluster? Check out our [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors) example!

### Project where L2S-M is being used:
- H2020 FISHY Project: https://fishy-project.eu (H2020-MG-2019-TwoStages-861696) 
- True5G Project: (PID2019-108713RB-C52 / AEI / 10.13039/501100011033)

### How to reach us

If you have any doubts about L2S-M or its instalaltion, please do not hesitate to contact us out through our e-mail!
- Luis F. Gonzalez: luisfgon@it.uc3m.es (Universidad Carlos III de Madrid)
- Ivan Vidal : ividal@it.uc3m.es (Universidad Carlos III de Madrid)
- Francisco Valera: fvalera@it.uc3m.es (Universidad Carlos III de Madrid
- Diego R. Lopez: diego.r.lopez@telefonica.com (Telefónica I+D)
