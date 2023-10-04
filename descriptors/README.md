# L2S-M Ping Pong example
This section of L2S-M documentation provides an example that you can use in order to learn how to create virtual networks and attach pods to them. To do so, we are going to deploy a simple ping-pong application, where we will deploy two pods attached into a virtual network and test their connectivity.

All the neccessary descriptors can be found in the *'./L2S-M/descriptors'* directory of this repository.

### Creating our first virtual network

First of all, let's see the details of an L2S-M virtual network. This is the descriptor corresponding to the virtual network that will be used in this example: my-first-network

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: my-first-network
spec:
  config: '{
      "cniVersion": "0.3.0",
      "type": "host-device",
      "device": "l2sm-vNet"
    }'
```
As you can see, L2S-M virtual networks are a [NetworkAttachmentDefinition](https://github.com/k8snetworkplumbingwg/multus-cni/blob/master/docs/quickstart.md) from MULTUS. In order to build a new network, just changing its name in the "metadata" field will define a new network. 

**Warning**: Do not change the config section from the descriptor; the *l2sm-vNet* is an abstract interface used by the L2S-M operator to manage the virtual networks in the K8s cluster.

To create the virtual network in your cluster, use the appropriate *kubectl* command as if you were building any other K8s resource:

```bash
kubectl create -f ./L2S-M/descriptors/
```

Et voilá! You have succcesfully created your first virtual network in your K8s cluster.

### Deploying our application in the cluster

After creating our first virtual network, it is time to attach some pods to it. To do so, it is as simple as adding an annotation to your deployment/pod file, just like you would do when attaching into a multus NetworkAttachmentDefinition. 

For example, to add a deployment to my-first-network, introduce in your descriptor the following annotation in its metadata:

```yaml
annotations:
  k8s.v1.cni.cncf.io/networks: my-first-network
```

If you want to add your own Multus annotations, you are free to do so! L2S-M will not interfere with the standard Multus behaviour, so feel free to add your addittional annotations if you need them.

To assist you with the dpeloyment of your first application with L2S-M, you can use the deployments available in this repository. To deploy both "ping-pong" pods (which are simple Ubuntu alpine containers), use the following command:

```bash
kubectl create -f ./L2S-M/descriptors/deployments/
```

After a bit of time, check that both deployments were succesfully instantiated in your cluster.

### Testing the connectivity

Once we have deployed the pods, let's add some IP addresses and make sure that we can connect with one another using the overlay. To do so, use the following commands to enter into the "ping" pod and check its interfaces:

```bash
kubectl exec -it [POD_PING_NAME] -- /bin/sh
ip a s
```

From the output of the last command, you should see something similar to this:
```bash
7: net1@if6: <BROADCAST,MULTICAST,M-DOWN> mtu 1450 qdisc noop state DOWN qlen 1000link/ether 16:79:4c:0c:d2:e8 brd ff:ff:ff:ff:ff:ff
```
This is the interface that we are going to use to connect in the virtual network. Therefore, we should first leave up that interface and assign an ip address to it (for example, 192.168.12.1/30):

```bash
ip link set net1 up
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
kubectl exec -it [POD_PONG_NAME] -- /bin/sh
ip link set net1 up
ip addr add 192.168.12.2/30 dev net1
```
See if they can ping each using the ping command (e.g., in the "pong" pod):
```bash
ping 192.168.12.1
```

If you have ping betwen them, congratulations! You are now able to deploy your applications attached to the virtual network "my-fist-network" at your K8s cluster. You will notice that the *ttl* of these packets is 64: this is the case because they see each other as if they were in the same broadcast domain (i.e., in the same LAN). You can further test this fact by installing and using the *traceroute* command:

```bash
apk update
apk add traceroute
traceroute 192.168.12.1
```

One last test you can perform to see that it is using the L2S-M overlay is trying to perform the same ping through the main interface of the pod (eth0), which will not be able to reach the other pod:
```bash
ping 192.168.12.1 -I eth0
```

If you are tired of experimenting with the app, you can proceed to delete both deployments from the cluster:

```bash
kubectl delete deploy/ping-l2sm
kubectl delete deploy/pong-l2sm
```

