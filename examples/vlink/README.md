# Vlink 'ping-pong' example

## Introduction

This document provides a guide for L2S-M users. It focuses on creating a virtual link (`vlink`) network and managing traffic flows between pods across different nodes using L2S-M components.

This guide is meant for developers who want to dig further into how L2S-M works. If you want to check a more simple example of L2S-M, it's recommended to check first [the ping pong guide.](../ping-pong/). Here it will be shown not only how the Vlinks can be used but how it works under the hood.

## Prerequisites
- A Kubernetes cluster
- Multus CNI installed
- L2S-M and a network overlay installed. You can check how to [in the overlay setup guide.](../overlay-setup/)


## Creating a Vlink Network

The first step involves creating a `vlink` network, named "vlink-sample", using our L2SMNetwork CRD. This network facilitates direct, isolated communication between pods across different nodes, through custom paths.


```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: L2Network
metadata:
  name: vlink-sample
spec:
  type: vlink
  config: |
    {
      "overlay-parameters": {
        "path": {
          "name": "first-path",
          "FromEndpoint": "node-a",
          "ToEndpoint": "node-e",
          "links": ["link-ac","link-cd","link-de"],
          "capabilities": {
            "bandwidthBits": "20M",
            "latencyNanos": "8e5"
          }
        },
        "reverse-path": {
          "name": "second-path",
          "fromEndpoint": "node-e",
          "toEndpoint": "node-a",
          "links": ["link-ed","link-db","link-ba"]
        }
      }
    }
```

### Process Overview

1. **Vlink Creation**: Deploy the `vlink-sample` YAML configuration to define the vlink network.
2. **L2SM Operator Activation**: Upon recognizing the new network configuration, the L2SM operator initiates, contacting the L2SM controller. This process includes saving the network path information for future use.
3. **L2SM Controller**: The controller is informed about the new network but does not initiate traffic flow immediately. It waits for pods to be connected to the network.

## Deploying Pods with Network Annotations

Deployment involves creating pods with specific annotations to connect them to the `vlink-sample` network. This section explains how PodA and PodB are deployed and managed within the network.

### Deploying pod 'ping'

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: ping
  labels:
    app: ping-pong
    l2sm: "true"
  annotations:
    l2sm/networks:  '[
            { "name": "vlink-sample",
              "ips": ["192.168.1.2/24"]
            }]'
spec:
  containers:
  - name: router
    command: ["/bin/ash", "-c", "trap : TERM INT; sleep infinity & wait"]
    image: alpine:latest
    securityContext:
      capabilities:
        add: ["NET_ADMIN"]
    nodeName: NodeA
```

- **Pod Configuration**: Pod 'ping' is defined with the `vlink-sample` annotation and an "ips" argument specifying its IP address. If no IP address is specified, the connection defaults to layer 2.
- **Connection to L2SM-Switch**: Pod 'ping' is attached via Multus to an L2S-M component known as the l2sm-switch, controlled by the L2S-M controller. This grants 'ping' two network interfaces: the default (provided by Flannel or Calico) and the new vlink interface.


### Deploying PodB

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pong
  labels:
    app: ping-pong
    l2sm: "true"
  annotations:
    l2sm/networks:  '[
            { "name": "vlink-sample",
              "ips": ["192.168.1.3/24"]
            }]'
spec:
  containers:
  - name: router
    command: ["/bin/ash", "-c", "trap : TERM INT; sleep infinity & wait"]
    image: alpine:latest
    securityContext:
      capabilities:
        add: ["NET_ADMIN"]
    nodeName: NodeE
```

- **Node Placement**: Pod 'pong' is created on NodeE with the `vlink-sample` network annotation but uses a different IP address than pod 'ping'.
- **Network Connectivity**: The L2SM controller then establishes the necessary intents and flows, ensuring traffic between 'ping' and 'pong' traverses the predefined nodes. This setup guarantees direct, isolated connectivity between the two pods.


