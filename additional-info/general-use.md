# General Use of L2S-M Application

This document will guide you through the usage of L2S-M, a tool designed to manage L2 networks, overlays, and Network Edge Devices (NEDs) within a Kubernetes cluster environment. L2S-M uses Custom Resource Definitions (CRDs) to enable flexible network management and pod attachment within Kubernetes.

For more specific examples, you can go to the [examples section](../examples/). This document is meant to be a general use guide.
## Custom Resource Definitions (CRDs)

L2S-M introduces three core CRDs that allow users to configure networks, overlays, and edge devices dynamically:

### 1. **L2Network CRD**
  - **Purpose**: Defines a Layer 2 virtual network inside the Kubernetes environment.
  - **Configurable Fields**:
     - **Network Type**: Specifies the type of network. It can be either "vnet" or "vlink". The default option is vnet, which connects multiple pods, vlinks are meant for specific scenarios between two pods where the network administrator desires to use a specific path of nodes in the communication.
     - **Provider**: Note: this field is [inter domain only](./inter-cluster.md). Below there's more info on how to set up this field. 
     - **NetworkCIDR**: Overall NetworkCIDR is used for routing and addressing pods. If this configuration is set, pods will automatically have an IP address assigned in this pool.
     - **PodAddressRange**: Complementary to Network CIDR, this field is meant to be used alongside the prior only in specific scenarios where you want to assign a specific address range but with the routing mask from NetworkCIDR (For instance, NetworkCIDR can be 10.34.0.0/16 and PodAddressRange 10.34.20.0/24. Pods will have set in their network devices ips from 10.34.20.1/16 to 10.34.20.255/16).
     - **Config**: Field meant to be used for additional configuration parameters, such as [vlinks paths](../examples/vlink/README.md).
  - **Status Fields**: This field reports the current state of the network, giving this information:
      - **Connected Pod Count**: Number of pods connected to this network.
      - **LastAssignedIP**: When using NetworkCIDR this field is for keeping track of the assigned IP addresses. Please be careful when modifying it as it can lead to errors in ip assignment.
      - **Assigned IPs**: Map of already assigned IP addresses, that helps avoid giving the same IP address to two pods in the same network, and can be useful for quick lookups.
      - **InternalConnectivity**: Gives the status of the network in the local sdn controller. It can be available, unavailable, or unknown.

   - **Usage**: Once a network is defined, pods can be connected to it. The L2Network CRD provides specifications through the `spec` field, where the user defines the network attributes, while the `status` field reports the current state of the network, including the pods connected to it.
   - An example of this CR can be found [here](../examples/ping-pong/network.yaml)
### 2. **Overlay CRD**
   - **Purpose**: Defines the logical connections between nodes in the cluster, creating the overlay network.
   - **Configurable Fields**:
     - **Topology**: Specifies how nodes should be interconnected. Is a list of nodes (name of the k8s nodes) and links. Links have two endpoints, corresponding to the nodes defined earlier. Links are bidirectional, meaning just one link between two nodes is enough.
     - **Switch Template**: Defines the type of switches used within the overlay. By default, set the one defined by us, using the image `alexdecb/l2sm-switch:TAG`. An implementation of a driver for any kind of virtualization switches is not pending in the make unless required.
     - **Provider**: Identifies the SDN controller responsible for managing the topology. Meant to be set by default as the one that comes with the installation.
     - **InterfaceNumber**: The number of interfaces per node for the switches, by default is 10. These can't be added dynamically so they must be specified at the beginning of the creation.

   - **Usage**: Administrators can use the Overlay CRD to define the connections between nodes based on their resource capacities or geographic location, creating custom topologies suited to specific needs. 
   - An example of this CR can be found [here](../examples/overlay-setup/overlay-sample.yaml)


### 3. **NetworkEdgeDevice (NED) CRD**
   - **Purpose**: Extends the network beyond the cluster, enabling communication with external networks or other clusters. [Head here for more in-depth info on the inter cluster](./inter-cluster.md)
   - **Configurable Fields**:
     - **Provider**: Identifies the SDN controller responsible for managing this NED.
     - **Node Config**: Specifies the desired node in the cluster where the NED should be deployed. It contains the node name, (from the k8s perspective) and the IP address. This IP Address can be any of the ones that the Node has, for instance, the node can be connected to the cluster through one interface, but connect to another domain through another IP Address. The important part is that when setting up multiple NEDs, these have to be connected and reachable directly via IP.
     - **Neighbors**: List of NEDs you want this NED to be connected to. For each neighbor, you have to specify a name and an IP Address. On the other side, a NED must be existing or pending to exist in these addresses for these NEDs to connect. 
     - **Switch Template**: Just as for the overlays, we recommend using the default provided template 
   - **Usage**: The NED CRD facilitates inter-cluster communication by connecting Kubernetes clusters or other platforms like OpenStack. Each NED is controlled by an SDN controller for dynamic flow control and traffic management.
    - An example of this CR can be found [here](../examples/inter-cluster-setup/example-ned.yaml)


### 4. **Provider Field**

This field is used by every CRD in L2S-M, in the NEDs and L2Networks for inter-domain controllers, and in the overlay for internal ones.

It controls the connection with the Software-Defined Networking (SDN) controller, in which case you specify:
  - **Name**: The name of the provider, must be the same in an L2Network and a NED. For the Overlay, it's best to leave it in the default (the one that comes with the default l2sm installation)
  - **Domain**: where the service is located, again the same for L2Networks and NEDs. For the overlays, the default is "l2sm-controller-service.l2sm-system.svc", unless you change the namespace from l2sm-system to another one. (Check services with `kubectl get svc -n *l2sm namespace*` to verify it exists).
  - **SDNPort**: Http port with an API to create networks
  - **DNSPort**: DNS port for the DNS server if using a DNS configuration.
  - **DNSGrpcPort**: grpc port for creating dns entries with l2sm-dns microservice. 
  - **OFPort**: Port where the Openflow communication is happening.

## Attaching Pods to Networks

Pods can be dynamically attached to L2 networks defined by the L2Network CRD. This API is meant to be used with labels and annotations:
- **Labels:**
  - *l2sm: true* -> Put this in a K8s workload, such as a  Deployment, ReplicaSet, DaemonSet, Job, etc. So the pods created in the workload will be managed by l2sm. Alternatively, you can specify this directly in the Pod.
  - *l2sm/app: name* -> (Optional) Put this in a k8s workload, and this workload, when receiving a DNS entry (if this functionality has been deployed), will use the name as a reference. For instance, we have a workload that we call "nginx-server" and is in the inter-domain network "cdn-network", we can call any of these servers created by the deployment, from another workload by accessing `nginx-server.cdn-network.inter.l2sm`. Our inter-domain DNS has basic load balancing, similar to the CoreDNS provided in the classic Kubernetes. If this field is not set, but DNS is being used in the network, the pods will still be accessible, by using the assigned pod names. `nginx-server-xpzvh.cdn-network.inter.l2sm`
- **Annotations:**
  - *l2sm/networks:* Specify the networks to be used by the pod. These networks must be created and available for this to work. Multiple networks can be used simultaneously. Below are some example inputs: 
    - `[{"name": "v-network-1", "ips": ["10.0.1.1/24"]}, {"name": "v-network-2", "ips": ["10.0.2.1/24"]}]`: Two networks, and statically assign IP addresses.
    - `v-network-1, v-network-2`: Two networks without static IP addresses. Pod will receive addresses automatically in the new interface if the network has a NetworkCIDR, if not, it will be L2 by default.
    - `[{"name": "ping-network"}]` or `ping-network`: Just use one network.

An additional network interface will be added to the pod for each assigned network.  


So the process involves the following steps:

1. **Defining the L2Network**: Use the L2Network CRD to create a network in Kubernetes. The network will be managed by the L2S-M controller, which communicates with the SDN controller to configure the necessary networking parameters.
   
2. **Scheduling Pods**: When a pod is deployed in the cluster, it can be attached to the L2Network by specifying the network during the pod creation process. The L2S-M controller will automatically configure the required network interfaces and assign IP addresses via the integrated IP Address Management (IPAM) system. 
   
3. **Monitoring Connectivity**: Once attached, the status of the pod’s network connectivity can be checked via the L2Network CRD’s `status` field, which will list all connected pods and report any changes in the connectivity state.


