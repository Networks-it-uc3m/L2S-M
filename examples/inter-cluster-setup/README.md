# L2S-M Inter-Cluster Configuration Guide

To connect multiple clusters across different nodes, we need to extend the basic overlay configuration to inter-cluster communication. This guide explains how to deploy a network controller and network edge devices (NEDs) to create an inter-cluster network. 

There is a solution that can be useful to manage multiple clusters together, it [can be found here](https://github.com/Networks-it-uc3m/l2sm-md). There you can specify which clusters to connect, and the component will use the L2S-M API to reach this desired state. But below is a more in depth explanation on how to do this, for more complex scenarios.

## Step 1: Deploying the Network Controller

The first step is to deploy a network controller, which will manage the communication between clusters. You can deploy the controller using Docker, in a machine reachable to the clusters. Run the following command:

```bash
sudo docker run -d \
  --name idco-controller \
  -p 6633:6633 \
  -p 8181:8181 \
  alexdecb/l2sm-controller:2.4
```


## Step 2: Deploying the Network Edge Device (NED)

Once the controller is running, we can deploy the NED in each cluster. The NED acts as a bridge between clusters, ensuring proper VxLAN communication.

The NED configuration includes the IP address of the network controller and the node configuration where it is deployed.

### NED Example Configuration

Here’s an example of how the NED's configuration should look like:

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: NetworkEdgeDevice
metadata:
  name: example-ned
  labels:
    app: l2sm
spec:
  provider:
    name: <controller-name>
    domain: <controller-domain>
    sdnPort: "8181"
    ofPort: "6633"
  nodeConfig:
    nodeName: <node-name>
    ipAddress: <node-ip-address>
  neighbors:
    - node: <cluster-name>
      domain: <neighb-cluster-reachable-ip-address>
  switchTemplate:
    spec:
      hostNetwork: true
      containers:
        - name: l2sm-ned
          image: alexdecb/l2sm-ned:2.7.1
          resources: {}
          command: ["./setup_ned.sh"]
          ports:
            - containerPort: 80
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
```

- 8181 and 6633 are the ports previously configured in the docker launch of the controller.
### Important Fields in NED Configuration:

1. **networkController**: This defines the network controller to which the NED will connect.
    - **name**: The name of the network controller.
    - **domain**: The IP address of the controller container (get this from the Docker container running the controller).

2. **nodeConfig**: This defines the specific node where the NED is deployed.
    - **nodeName**: The name of the node (can be found using `kubectl get nodes`).
    - **ipAddress**: The IP address of the node where you are deploying the NED (can be checked using `kubectl get nodes -o wide`).

3. **neighbors**: This is where you list the other clusters and their corresponding IP addresses to establish communication.
    - **node**: The name of the neighboring cluster.
    - **domain**: The IP address of the neighboring cluster's node.


## Step 3: Deploying the NED

After configuring the NED for each node, apply the configuration using `kubectl`:

```bash
kubectl create -f ./examples/ned-setup/ned-sample.yaml
```

If you need to modify the NED configuration, update the YAML file and apply the changes using:

```bash
kubectl apply -f ./examples/ned-setup/ned-sample.yaml
```


## Example NED Configuration for Multiple Clusters

Here's an example of how to configure the NED to connect multiple clusters:

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
    domain: 192.168.122.60  # Network controller IP
    sdnPort: "8181" # Port where you have the sdn http port (specified in the docker container)
    ofPort: "6633" # Port where the openflow 
  nodeConfig:
    nodeName: ant-machine
    ipAddress: 192.168.122.60
  neighbors:
    - node: tucci
      domain: 192.168.122.244  # IP of tucci node
    - node: l2sm3
      domain: 192.168.123.100  # IP of another cluster node
```

## Deploying an inter-cluster network

Once you've got the inter cluster topology, you can connect pods that are in both clusters by creating inter-cluster networks. This example is the same as the one shown in [the ping pong guide](../ping-pong/), with the peculiarity that when the L2Network is deployed, a provider is specified. L2S-M checks the provider field and on top of the ned, will create this network that enables this secure connection between the pods in both clusters.

This is an inter cluster L2Network:

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: L2Network
metadata:
  name: ping-network
spec:
  type: vnet
  provider:
    name: idco-controller
    domain: "192.168.122.60:8181"
    sdnPort: "8181"
```
> Notice that the provider name is the same one as the one specified in [the NED](#example-ned-configuration-for-multiple-clusters).


This same L2Network must be created in both clusters. Afterwards the pods can be deployed just like in every other L2S-M example, as shown in the ping pong files.

## Conclusion

By following this guide, you can deploy a network controller and configure network edge devices (NEDs) to connect multiple clusters in an inter-cluster VxLAN network. The key is to accurately configure the controller and NEDs and ensure proper communication between clusters through the `neighbors` section.
