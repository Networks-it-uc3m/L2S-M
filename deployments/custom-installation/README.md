# L2S-M Installation Guide (Custom Installation)

This guide provides detailed steps for installing the L2S-M Kubernetes operator, enabling you to create and manage virtual networks within your Kubernetes cluster. This custom installation is intended for debugging or understanding the L2S-M components and their functionality.

## Introduction

The L2S-M custom installation is designed for debugging purposes and gaining a deeper understanding of the L2S-M components. Follow the steps below to install the L2S-M Kubernetes operator and configure virtual networks.

## Prerequisites

Before proceeding, ensure that you meet the prerequisites outlined in the [Prerequisites section](./deployment/README.md). Refer to the [./deployment/README.md](./deployment/README.md) file for detailed instructions on meeting these requirements.


## Custom Installation Steps

Follow the steps below to perform the custom installation of L2S-M:


1. Create the virtual interface definitions using the following command:
 ```bash
kubectl create -f ./deployments/custom-installation/interfaces_definitions
```

2. Create the Kubernetes account Service Account and apply their configuration by applying the following command:
 ```bash
kubectl create -f ./deployments/config/
```

3. Create the Kubernetes Persistent Volume by using the following kubectl command:
 ```bash
kubectl create -f ./deployments/custom-installation/mysql/
```

4. Before deploying the L2S-M operator, it is neccessary to label your master node as the "master" of the cluster. To do so, get the names of your Kubernetes nodes, select the master and apply the "master" label with the following command:

 ```bash
kubectl get nodes
kubectl label nodes [your-master-node] dedicated=master
```
5. Deploy the L2S-M Controller by using the following command: 

```bash
kubectl create -f ./deployments/custom-installation/deployController.yaml
```
 You can check that the deployment was successful if the pod enters the "running" state using the *kubectl get pods* command.

6. After the previous preparation, (make sure the controller is running) you can deploy the operator in your cluster using the YAML deployment file:
 ```bash
kubectl create -f ./deployments/custom-installation/deployOperator.yaml
```

Once these two pods are in running state, you can finally deploy the virtual switches

7. This is done by:

**First deploying the virtual OVS Daemonset:**
```bash
kubectl create -f ./deployments/custom-installation/deploySwitch.yaml
```

And check there is a pod running in each node, with ```kubectl get pods -o wide```

## Configuring Vxlans

Each node enables the creation of custom L2S-M networks, as can be seen in the [examples section](../../examples/) section. But for communicating pods that are in different Nodes of the cluster, additional configuration must be done, of configuring the Vxlan tunnels between them.

You can proceed to configure Vxlans by following the steps outlined in [the vxlan configuration guide.](../deployment/vxlans.md)

