# L2S-M Ping Pong Example with Static IP Addresses

This guide is a continuation of the [L2S-M Ping Pong example](../ping-pong/) and demonstrates how to deploy the same ping-pong application with static IP address assignment for a faster, streamlined setup. In this version, pod annotations and network definitions are modified so that pods automatically receive fixed IP addresses without manual intervention.

All necessary descriptors for this example can be found in the `./examples/adding-ips/` directory.

---

## Pre-requisites

Before you begin, ensure that L2S-M is installed and an overlay topology is deployed. For more details on setting up the overlay, refer to [the overlay example section](../overlay-setup).

---

## Creating a Virtual Network with Static IP Assignment

In this example, the virtual network is defined using a new CRD called `ping-network-intra-l3`. Unlike the default virtual network, this one supports static IP configuration via pod annotations.

**Network Descriptor (`network.yaml`):**

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: L2Network
metadata:
  name: ping-network-intra-l3
spec:
  type: vnet
```

Create the virtual network with:

```bash
kubectl create -f ./examples/adding-ips/network.yaml
```

---

## Deploying the Application with Static IPs

The main difference in this static IP example is the pod annotation. Instead of simply listing the network name, the annotation now includes an `ips` field with the desired IP address (and subnet mask) for the pod.

**Pod Annotation Example:**

```yaml
labels:
  l2sm: "true"
annotations:
  l2sm/networks: '[{"name": "ping-network-intra-l3", "ips": ["10.0.0.1/24"]}]'
```

Deploy the two pods (ping and pong) using:

```bash
kubectl create -f ./examples/adding-ips/ping.yaml
kubectl create -f ./examples/adding-ips/pong.yaml
```

Since static IPs are pre-assigned via annotations, there is no need for manual IP configuration within the pod shell.

---

## Testing Connectivity

With the static IP configuration in place, simply verify connectivity by pinging the assigned IP address of the peer pod:

```bash
kubectl exec -it ping -- ping 10.0.0.2
```

If the ping is successful, your pods are correctly connected via the L2S-M virtual network using static IP addresses.

---

## Cleanup

To remove all resources from this example, run:

```bash
kubectl delete -f ./examples/adding-ips/
```

