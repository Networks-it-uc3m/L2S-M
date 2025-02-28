# L2S-M Installation Guide
This guide details the necessary steps to install the L2S-M Kubernetes operator to create and manage virtual networks in your Kubernetes cluster.


# Prerequisites

1. Clone the L2S-M repository in your host. This guide will assume that all commands are executed within the L2S-M directory.

2. Install the Multus CNI Plugin in your K8s cluster. For more information on how to install Multus in your cluster, check their [official GitHub repository](https://github.com/k8snetworkplumbingwg/multus-cni).

3. Install the Cert-Manager in your K8s cluster. For more information on how to install Cert-Manager in your cluster, check their official installation guide.

4. The host-device CNI plugin must be able to be used in your cluster. If it is not present in your K8s distribution, you can find how to install it in your K8s cluster in their [official GitHub repository](https://github.com/containernetworking/plugins). 

5. Make sure that packages are forwarded by default: `sudo  iptables -P FORWARD ACCEPT`

6. You need at least one amd64 node in your cluster for the sdn controller to launch. 

## Install L2S-M

Installing L2S-M can be done by using a single command:

```bash
kubectl create -f ./deployments/l2sm-deployment.yaml
```

The installation will take around a minute to finish, and to check that everyting is running properly, you may run the following command:

```bash
kubectl get pods -o wide -n l2sm-system
```

Which should give you an output like this:

```bash
NAME                                        READY   STATUS    RESTARTS   AGE    IP           NODE    NOMINATED NODE   READINESS GATES
l2sm-controller-56b45487b7-nglns             1/1     Running   0          129m   10.1.72.72   l2sm2   <none>           <none>
l2sm-controller-manager-7794c5f66d-b9nsf     2/2     Running   0          119m   10.1.14.45   l2sm1   <none>           <none>
```


After the installation, you can start using L2S-M. The first thing you want to do is to create an overlay topology, that will be the basis of the virtual network creations, but don't worry! Check out the [overlay setup guide](../examples/overlay-setup/) for more information.