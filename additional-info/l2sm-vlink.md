# L2S-M Vlink Network

## Introduction
This Markdown document provides an overview and documentation for the configuration of a virtual link (vlink) using the L2S-M (Link-Layer Secure connectivity for Microservice platforms) project.

Additionally, an example network topology of a five nodes Cluster with L2S-M installed is presented. This example will be used to discuss an example of a vlink network followed by a YAML definition of the NetworkTopology CRD, illustrating the practical application of the configuration and interoperability with the SWM project. 

## Table of Contents
- [Vlink L2S-M Configuration](#vlink-l2s-m-configuration)
  - [Overview](#overview)
  - [Sample File](#sample-file)
  - [Fields](#fields)
- [Example](#example)
  - [Vlink Sample Path](#vlink-sample-path)
  - [Network Topology](#network-topology)

## Vlink L2S-M Configuration

### Overview
L2S-M networks are implemented using the multus CRD, NetworkAttachmentDefinition. The main component of L2S-M, the L2S-M operator will manage this resource and will configure the network to reach a desired behavior.

The sample file below shows how the Vlink network is defined, in the context of the CODECO project.

The fields represent how this network is going to be implemented. The cni type is l2sm, so the operator knows which Net-Attach-Def should be managed by him. This specific L2S-M network is type 'vlink', this means it's a point to point network between two pods in the Cluster, where it's specified which Nodes should the communication pass through. This is further explained in the 'fields' subsection.

### Sample file

The Vlink network yaml file should look something like this:

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: <name>
spec:
  config: '{
  "cniVersion": "0.3.0",
  "type": "l2sm",
  "device": "l2sm-vNet",
  "kind": {
    "vlink": {
      "overlay-parameters": {
        "overlay-paths": [
          {
            "name": "<pathName1>",
            "FromEndpoint": "<endpointNodeA>",
            "ToEndpoint": "<endpointNodeB>",
            "path": ["<pathNodeA1>", "<pathNodeA2>", "...", "<pathNodeAN>"]
          },
          {
            "name": "<pathName2>",
            "fromEndpoint": "<endpointNodeB>",
            "toEndpoint": "<endpointNodeA>",
            "path": ["<pathNodeB1>", "<pathNodeB2>","...", "<pathNodeBN>"]
          }
        ]
      }
    }
  }
}'
```


### Fields

The config field is a JSON string with the following fields defined:


- `cniVersion` (string,required): "0.3.0". Current CNI plugin version.
- `type`(string,required): "l2sm". CNI type, l2sm represents an l2sm network.
- `kind`(dictionary, required): type of network. In this case, vlink, a point to point network between two pods.
- `vlink`(dictionary, required): specification of the kind field, as a vlink, has parameters that will specify the path the network should use.
- `overlay-parameters`(dictionary, required): parameters of this vlink network.
- `overlay-paths`(list,required): List of paths configured in this vlink. It's expected to be a bidirectional path, so two paths should be provided.
- `name`(string,required): Name of the path.
- `FromEndpoint`(string,required): Source endpoint.
- `ToEndpoint`(string,required): Destination endpoint.
- `path`(list,required): List of nodes representing the path.

In the context of the CODECO project, vlink networks can be mapped to the channel resource type in the SWM project through the overlay paths, where each overlay-path corresponds to a channel:

- FromEnpoint --> channelFrom.
- ToEnpoint --> channelTo.
- path --> networkPath. (the array should be mapped as network 'links', while the FromEndpoint and ToEndpoint to the 'start' and 'end' fields.)


## Example

To further understand the creation of a vlink network, and the L2S-M cluster example topology, the following figure is presented:

<p align="center">
  <img src="l2sm-f.svg" width="400">
</p>


This figure demonstrates a Cluster with 5 nodes, node-a, node-b, node-c, node-d and node-e, that are connected like shown in the image. The switches apply rules that are instructed by the L2S-M Controller, following the SDN approach. In this example there is a pod in node-a and another one in node-e that are going to be connected using an L2S-M network, of type vlink.

### Vlink sample path

In this topology, a vlink network definition between the pods in node-a and node-e looks like this:

```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: network-sample
spec:
  config: '{
  "cniVersion": "0.3.0",
  "type": "l2sm",
  "device": "l2sm-vNet",
  "kind": {
    "vlink": {
      "overlay-parameters": {
        "overlay-paths": [
          {
            "name": "first-path",
            "FromEndpoint": "node-a",
            "ToEndpoint": "node-e",
            "path": ["node-c", "node-d"]
          },
          {
            "name": "second-path",
            "fromEndpoint": "node-e",
            "toEndpoint": "node-a",
            "path": ["node-d","node-b"]
          }
        ]
      }
    }
  }
}'
```

### Network Topology

Additionally, we present using this example, how this topology could be defined using the NetworkTopology CRD, using the metrics from L2S-M. 

```yaml
apiVersion: qos-scheduler.siemens.com/v1alpha1
kind: NetworkTopology
metadata:
  name: l2sm-sample-cluster
spec:
  networkImplementation: l2sm-network
  physicalBase: logical-network
  nodes:
    - name: node-a
      type: NETWORK
    - name: node-b
      type: NETWORK
    - name: node-c
      type: NETWORK
    - name: node-d
      type: NETWORK
    - name: node-e
      type: NETWORK
  links:
    - source: node-a
      target: node-b
      capabilities:
        bandWidthBits: "1G"
        latencyNanos: "2e6"
    - source: node-a
      target: node-c
      capabilities:
        bandWidthBits: "500M"
        latencyNanos: "3e6"
    - source: node-b
      target: node-a
      capabilities:
        bandWidthBits: "1G"
        latencyNanos: "2e6"
    - source: node-b
      target: node-c
      capabilities:
        bandWidthBits: "2G"
        latencyNanos: "1e6"
    - source: node-b
      target: node-d
      capabilities:
        bandWidthBits: "1.5G"
        latencyNanos: "2.5e6"
    - source: node-c
      target: node-a
      capabilities:
        bandWidthBits: "500M"
        latencyNanos: "3e6"
    - source: node-c
      target: node-b
      capabilities:
        bandWidthBits: "2G"
        latencyNanos: "1e6"
    - source: node-c
      target: node-d
      capabilities:
        bandWidthBits: "1G"
        latencyNanos: "2e6"
    - source: node-d
      target: node-c
      capabilities:
        bandWidthBits: "1G"
        latencyNanos: "2e6"
    - source: node-d
      target: node-e
      capabilities:
        bandWidthBits: "2G"
        latencyNanos: "2.5e6"
    - source: node-e
      target: node-d
      capabilities:
        bandWidthBits: "2G"
        latencyNanos: "2.5e6"
```



