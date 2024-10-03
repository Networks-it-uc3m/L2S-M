# L2S-M Ping Pong example
This section of L2S-M documentation provides an example that you can use in order to learn how to create virtual networks and attach pods to them. To do so, we are going to deploy a simple ping-pong application, where we will deploy two pods attached to a virtual network and test their connectivity.

All the necessary descriptors can be found in the *'./examples/ping-pong/'* directory of this repository.

This guide will assume that all commands are executed within the L2S-M directory.

## Pre-requisites

In order to get this example moving, it's required to have L2S-M installed alongside an overlay topology deployed. You can learn how to do so in [the overlay example section](../overlay-setup).

### Creating our first virtual network

First of all, let's see the details of an L2S-M virtual network. This is the descriptor corresponding to the virtual network that will be used in this example: ping-network

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: L2Network
metadata:
  name: ping-network
spec:
  type: vnet
```
As you can see, L2S-M virtual networks are custom CRDs, the L2Networks. In order to build a new network, just changing its name in the "metadata" field will define a new network. 


To create the virtual network in your cluster, use the appropriate *kubectl* command as if you were building any other K8s resource:

```bash
kubectl create -f ./examples/ping-pong/network.yaml
```

Et voilá! You have successfully created your first virtual network in your K8s cluster.

### Deploying our application in the cluster

After creating our first virtual network, it is time to attach some pods to it. To do so, it is as simple as adding an annotation to your deployment/pod file, just like you would do when attaching into a multus NetworkAttachmentDefinition. 

For example, to add one deployment to ping-network, enter the following annotation in your descriptor in its metadata:

```yaml
labels:
  l2sm: "true"
annotations:
  l2sm/networks: ping-network
```


To assist you with the deployment of your first application with L2S-M, you can use the pod definitions available in this repository. To deploy both "ping-pong" pods (which are simple Ubuntu alpine containers), use the following commands:

```bash
kubectl create -f ./examples/ping-pong/ping.yaml
```

And then:
``` bash
kubectl create -f ./examples/ping-pong/pong.yaml
```

After a bit of time, check that both pods were successfully instantiated in your cluster.

### Testing the connectivity

Once we have deployed the pods, let's add some IP addresses and make sure that we can connect with one another using the overlay. To do so, use the following commands to enter into the "ping" pod and check its interfaces:

```bash
kubectl exec -it ping -- /bin/sh
ip a s
```

From the output of the last command, you should see something similar to this:
```bash
7: net1@if6: <BROADCAST,MULTICAST,M-DOWN> mtu 1450 qdisc noop state DOWN qlen 1000link/ether 16:79:4c:0c:d2:e8 brd ff:ff:ff:ff:ff:ff
```
This is the interface that we are going to use to connect in the virtual network. Therefore, we should first leave up that interface and assign an ip address to it (for example, 192.168.12.1/30):

```bash
ip addr add 192.168.12.1/30 dev net1
```

**WARNING:**  You must have the "[NET_ADMIN]" capabilities enabled for your pods to allow the modification of interfaces status and/or ip addresses. If not, do so by adding the following code to the *securityContext* of your pod in the descriptor:
```yaml
securityContext:
  capabilities:
    add: ["NET_ADMIN"]
```

Do the same action for your "pong" pod (with a different IP address, 192.168.12.2/30):

```bash
kubectl exec -it pong -- /bin/sh
ip link set net1 up
ip addr add 192.168.12.2/30 dev net1
```
See if they can ping each using the ping command (e.g., in the "pong" pod):
```bash
ping 192.168.12.1
```

If you have ping between them, congratulations! You are now able to deploy your applications attached to the virtual network "my-fist-network" at your K8s cluster. You will notice that the *ttl* of these packets is 64: this is the case because they see each other as if they were in the same broadcast domain (i.e., in the same LAN). You can further test this fact by installing and using the *traceroute* command:

```bash
apk update
apk add traceroute
traceroute 192.168.12.1
```

One last test you can perform to see that it is using the L2S-M overlay is trying to perform the same ping through the main interface of the pod (eth0), which will not be able to reach the other pod:
```bash
ping 192.168.12.1 -I eth0
```

If you are tired of experimenting with the app, you can proceed to delete both pods from the cluster:

```bash
kubectl delete pod ping pong
```

