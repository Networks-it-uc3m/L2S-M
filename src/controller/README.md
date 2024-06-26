# L2S-M Controller ONOS Applications

This repository contains the source code for two ONOS applications, vlink and vnet, which together constitute the L2S-M Controller. The L2S-M Controller is a critical component in the Link-Layer Secure connectivity for Microservice platforms (L2S-M) ecosystem. The applications are designed to be compiled with `mvn clean install` and then added to a running instance of ONOS using `onos-app install vnet-1-0.oar` or `onos-app install vlink-1-0.oar`. 

## Overview

L2S-M is a K8s networking solution designed to complement the Container Network Interface (CNI) plugin approach of K8s. Its primary goal is to create and manage virtual networks in K8s clusters, providing isolated link-layer connectivity for workloads (pods) regardless of the K8s node they are deployed on.

### Features

- **On-Demand Virtual Networks (vnet)**: Allows the creation and deletion of virtual networks on-demand, specifying the pods that will be connected.

- **Virtual Link Creation (vlink)**: Creates a virtual link between two pods, specifying a path for configuration.

- **Isolated Connectivity**: Workloads (pods) within a K8s cluster can have isolated link-layer connectivity with other pods.

- **Dynamic Attachment/Detachment**: Pods can be dynamically attached or detached from virtual networks.

- **Seamless K8s Integration**: L2S-M seamlessly integrates with the K8s environment through a K8s operator.

## L2S-M Repository

The core functionality of L2S-M, including its architecture and implementation details, is hosted in a separate repository. Find more information about L2S-M [here](https://github.com/Networks-it-uc3m/L2S-M).

## Getting Started

To set up the L2S-M Controller with ONOS, follow the steps below:

1. **Clone Repository:**
```bash
git clone https://github.com/Networks-it-uc3m/l2sm-controller
cd l2sm-controller
```

2. **Compile and Install Applications:**
```bash
mvn clean install
```

3. **Install vnets and vlinks on ONOS:**

This component is meant to be run in a k8s Cluster as its shown in the [L2S-M custom installation](https://github.com/Networks-it-uc3m/L2S-M/tree/main/deployments/custom-installation). So by updating the images used to refer to the applications built in the previous step, you can install your own version of vnets and vlinks.

When using these apps locally you may run an ONOS instance and install this applications by doing the following commands:

```bash
onos-app <onos-ip> install! vnet-1-0.oar
onos-app <onos-ip> install! vlink-1-0.oar
```


## Developing for L2S-M Controller

An alternative branch has been added where developing tools and guidelines have been introduced, to ease the development of new features. This can be accesed through the [development branch](https://github.com/Networks-it-uc3m/l2sm-controller/tree/development)

<!-- ## License

This project is licensed under the [Apache 2.0 License](LICENSE.md). -->

## Support and Contact

For support or inquiries, please feel free to contact me [100383348@alumnos.uc3m.es].
