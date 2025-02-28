



# L2S-M Inter-Cluster Configuration Guide

The L2S-M platform now supports advanced inter-cluster scenarios that extend the basic overlay networking to enable secure, dynamic communication between multiple Kubernetes clusters. This guide explains how to deploy the necessary components—a centralized Network Controller and distributed Network Edge Devices (NEDs)—and how to configure inter-cluster networks using the updated L2S-M API.

> **Note:** For more advanced features (such as integrated DNS, gRPC, and CLI tools), refer to the [l2sm-md repository](https://github.com/Networks-it-uc3m/l2sm-md). 



## Architecture Overview

In an inter-cluster setup, the following components work together:

- **Network Controller:**  
  Manages inter-cluster connectivity by orchestrating the SDN and interfacing with all NEDs. The controller is also used by the L2Network definitions via the provider field.

- **Network Edge Devices (NEDs):**  
  Deployed on each cluster node, these devices bridge local networks with external clusters. Each NED is configured with its node’s details and the connection parameters of neighboring clusters.

- **Inter-Cluster L2Network:**  
  An L2Network CRD that, through its provider field, instructs the L2S-M controller to create secure, overlay connections between pods in different clusters.

<p align="center">
  <img src="../assets/inter-cluster-arch.svg" width="600" alt="Inter-cluster architecture">
</p>

<!-- <p align="center">
  <img src="../assets/inter-cluster-diagram.svg" width="600" alt="Inter-cluster sequence diagram">
</p> -->

---

## Step 1: Deploying the Network Controller

The network controller is the central component for managing inter-cluster connectivity. It communicates with NEDs and L2Network CRDs via the provider API. A reference implementation is available in [l2sm-md](https://github.com/Networks-it-uc3m/l2sm-md), which bundles a DNS service, a gRPC server, and a CLI tool.

Deploy the controller using Docker:

```bash
sudo docker run -d \
  --name idco-controller \
  -p 6633:6633 \
  -p 8181:8181 \
  alexdecb/l2sm-controller:2.4
```

Ensure that the controller’s IP (e.g., `192.168.122.60`) and API port (e.g., `8181`) are correctly referenced in all subsequent configurations.

### Optional: deploy the DNS service
If you're not planning to use DNS, you can skip this part.

The DNS Service can be found at (https://github.com/Networks-it-uc3m/l2sm-dns). You can launch it with docker or kubernetes, just make sure to use the same IP Address as the IDCO Controller, and set available ports from the managed clusters.


---

## Step 2: Deploying the Network Edge Device (NED)

NEDs extend your network beyond a single cluster by acting as gateways between clusters. They are deployed using the updated **NetworkEdgeDevice** CRD. Key configuration elements include:

- **provider:**  
  Specifies the SDN controller managing this NED. Set the ports to the ones setted earlier in step 1.

- **nodeConfig:**  
  Defines the Kubernetes node where the NED is deployed. It includes:
  - **nodeName:** The name of the node (obtainable via `kubectl get nodes`).
  - **ipAddress:** A reachable IP address of the node, used for inter-cluster communication.

- **neighbors:**  
  Lists the other clusters (or nodes) that should be connected. Each neighbor requires:
  - **node:** A unique identifier for the neighboring cluster.
  - **domain:** The IP address (or hostname) where the neighbor’s NED can be reached.

- **switchTemplate:**  
  Defines container image and parameters for the switch configuration. The default template is recommended.

### Example NED Configuration

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: NetworkEdgeDevice
metadata:
  name: example-ned
  labels:
    app: l2sm
spec:
  provider:
    name: idco-controller
    domain: 192.168.122.60  # Controller IP address
  nodeConfig:
    nodeName: <node-name>
    ipAddress: <node-ip-address>
  neighbors:
    - node: cluster-b
      domain: <cluster-b-ip-address>
    - node: cluster-c
      domain: <cluster-c-ip-address>
  switchTemplate:
    spec:
      hostNetwork: true
      containers:
        - name: l2sm-ned
          image: alexdecb/l2sm-switch:1.2.9
          command: ["./setup_ned.sh"]
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
```

Deploy the NED by applying the configuration:

```bash
kubectl create -f path/to/your/ned-configuration.yaml
```

For modifications, update the YAML file and reapply using `kubectl apply -f`. (This functionalty is still being tested, for flexible topologies and configurations)

---

## Step 3: Creating an Inter-Cluster L2Network

Once the controller and NEDs are in place, define an inter-cluster network using the **L2Network** CRD. The provider field in this CRD must mirror the network controller configuration to ensure that the SDN orchestrates connectivity between clusters.

### Example Inter-Cluster L2Network Configuration

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: L2Network
metadata:
  name: ping-network
spec:
  type: vnet 
  provider: # Must be the same as the NED
    name: idco-controller 
    domain: "192.168.122.60" 
  # Optional: Specify additional parameters like networkCIDR or podAddressRange if required
```

Deploy the L2Network configuration:

```bash
kubectl create -f path/to/your/l2network.yaml
```

Make sure the same L2Network definition is created in each cluster that participates in the inter-cluster setup.

---

## Step 4: Attaching Pods to the Inter-Cluster Network

Pods are dynamically attached to an inter-cluster network by using annotations. The L2S-M controller handles the addition of a new network interface and IP address management (if configured).

### Example Pod Configuration

```yaml
apiVersion: v1
kind: Pod
labels:
  l2sm: true
metadata:
  name: mypod
  annotations:
    l2sm/networks: ping-network
spec:
  containers:
    - name: ping
      image: busybox
```

Once deployed, the pod will have an additional network interface corresponding to the `ping-network` and will participate in inter-cluster communication.

---

## Provider Field Details

The `provider` field is a critical element used across multiple CRDs (L2Network and NetworkEdgeDevice). It establishes the connection to the SDN controller and must be consistent across your configurations.

- **Name:**  
  The identifier for the controller (e.g., `idco-controller`).

- **Domain:**  
  The address where the controller service is reachable (e.g., `192.168.122.60`).

- **Optional Ports:**  
  Depending on your network requirements, you may also specify additional ports such as:
  - **SDNPort:** For SDN API communications.
  - **DNSPort:** For DNS services.
  - **DNSGrpcPort:** For gRPC-based DNS entry creation.
  - **OFPort:** For OpenFlow communication.

---


For further details on CRDs and network attachment procedures, refer to the [General Use of L2S-M Application](./general-use.md) document and the [examples section](../examples/).

Happy networking!
