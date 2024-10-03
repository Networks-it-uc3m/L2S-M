# General Use of L2S-M Application

This document will guide you through the usage of L2S-M, a tool designed to manage L2 networks, overlays, and Network Edge Devices (NEDs) within a Kubernetes cluster environment. L2S-M uses Custom Resource Definitions (CRDs) to enable flexible network management and pod attachment within Kubernetes.

For more specific examples, you can go to the [examples section](../examples/), this document is meant to be a general use guide.
## Custom Resource Definitions (CRDs)

L2S-M introduces three core CRDs that allow users to configure networks, overlays, and edge devices dynamically:

### 1. **L2Network CRD**
   - **Purpose**: Defines a Layer 2 virtual network inside the Kubernetes environment.
   - **Configurable Fields**:
     - **Network Type**: Specifies the type of network.
     - **Connectivity**: Controls the connection with the Software-Defined Networking (SDN) controller.
     - **Connected Pods**: Lists the pods connected to this network.
   - **Usage**: Once a network is defined, pods can be connected to it. The L2Network CRD provides specifications through the `spec` field, where the user defines the network attributes, while the `status` field reports the current state of the network, including the pods connected to it.
   - An example of this CR can be found [here](../config/samples/l2sm_v1_l2network.yaml)

### 2. **Overlay CRD**
   - **Purpose**: Defines the logical connections between nodes in the cluster, creating the overlay network.
   - **Configurable Fields**:
     - **Topology**: Specifies how nodes should be interconnected.
     - **Switch Template**: Defines the type of switches used within the overlay.
     - **Network Controller**: Identifies the SDN controller responsible for managing the topology.
   - **Usage**: Administrators can use the Overlay CRD to define the connections between nodes based on their resource capacities or geographic location, creating custom topologies suited to specific needs.
    - An example of this CR can be found [here](../config/samples/l2sm_v1_overlay.yaml)


### 3. **NetworkEdgeDevice (NED) CRD**
   - **Purpose**: Extends the network beyond the cluster, enabling communication with external networks or other clusters.
   - **Configurable Fields**:
     - **Device Type**: Defines the hardware or software that forms the edge device.
     - **Connections**: Specifies the external networks or clusters this NED should connect to.
   - **Usage**: The NED CRD facilitates inter-cluster communication by connecting Kubernetes clusters or other platforms like OpenStack. Each NED is controlled by an SDN controller for dynamic flow control and traffic management.
    - An example of this CR can be found [here](../config/samples/l2sm_v1_networkedgedevice.yaml)


## Attaching Pods to Networks

Pods can be dynamically attached to L2 networks defined by the L2Network CRD. This process involves the following steps:

1. **Defining the L2Network**: Use the L2Network CRD to create a network in Kubernetes. The network will be managed by the L2S-M controller, which communicates with the SDN controller to configure the necessary networking parameters.
   
2. **Scheduling Pods**: When a pod is deployed in the cluster, it can be attached to the L2Network by specifying the network during the pod creation process. The L2S-M controller will automatically configure the required network interfaces and assign IP addresses via the integrated IP Address Management (IPAM) system.
   
3. **Monitoring Connectivity**: Once attached, the status of the pod’s network connectivity can be checked via the L2Network CRD’s `status` field, which will list all connected pods and report any changes in the connectivity state.


