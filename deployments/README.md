# L2S-M Installation Guide
This guide details the necessary steps to install the L2S-M Kubernetes operator to create and manage virtual networks in your Kubernetes cluster.


# Prerequisites

1. Clone the L2S-M repository in your host. This guide will assume that all commands are executed within the L2S-M directory.

2. Install the Multus CNI Plugin in your K8s cluster. For more information on how to install Multus in your cluster, check their [official GitHub repository](https://github.com/k8snetworkplumbingwg/multus-cni).

3. The host-device CNI plugin must be able to be used in your cluster. If it is not present in your K8s distribution, you can find how to install it in your K8s cluster in their [official GitHub repository](https://github.com/containernetworking/plugins).

4. Your K8s Control-Plane node must be able to deploy K8s pods for the operator to work. Remove its master and control-plane taints using the following command:
```bash
kubectl taint nodes --all node-role.kubernetes.io/control-plane- node-role.kubernetes.io/master-
```

5. The `he-codeco-netma` namespace created. You can do so if it's not already done, by using the following kubectl command:

```bash
kubectl create namespace he-codeco-netma
```

 
## Install L2S-M

Installing L2S-M can be done by using a single command:

```bash
kubectl create -f ./deployments/l2sm-deployment.yaml -n=he-codeco-netma
```

The installation will take around a minute to finish, and to check that everyting is running properly, you may run the following command:

```bash
kubectl get pods -o wide
```

Which should give you an output like this:

```bash
NAME                               READY   STATUS    RESTARTS   AGE    IP           NODE    NOMINATED NODE   READINESS GATES
l2sm-controller-56b45487b7-nglns   1/1     Running   0          129m   10.1.72.72   l2sm2   <none>           <none>
l2sm-operator-7794c5f66d-b9nsf     2/2     Running   0          119m   10.1.14.45   l2sm1   <none>           <none>
l2sm-switch-49qpq                  1/1     Running   0          129m   10.1.14.63   l2sm1   <none>           <none>
l2sm-switch-2g696                  1/1     Running   0          129m   10.1.72.73   l2sm2   <none>           <none>
```
With the components: _l2sm-controller_, _l2sm-operator_ and one _l2sm-switch_ for **each** node in the cluster. 

After the installation, you can start using L2S-M in one Node. If your Cluster has more than one Node, additional steps must be done, to configure which Nodes are connected between themselves as can be seen in the next subsection, configuring the VxLAN tunnels.

## Configuring VxLANs

Each Node enables the creation of custom L2S-M networks, as can be seen in the [examples section.](../examples/) But for communicating pods that are in different Nodes of the cluster, additional configuration must be done, the VxLAN tunnels between them.

But don't worry! A guide on how this is configured step by step is outlined in [the vxlan configuration guide.](../deployment/vxlans.md)
