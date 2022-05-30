# L2S-M Installation Guide
This guide details the necessary prerequisites to install the L2S-M Kubernetes operator to create and manage virtual networks in your Kubernetes cluster!


# Prerequisites

1. Clone the L2S-M repository in your host.  This guide will assume that all commands are executed in the directory where L2S-M was downloaded.

2. In order to start with the installation of L2S-M, it is necessary to set up the IP tunnel overlay between the nodes that you want to interconnect. To do so, **it is necessary to have 10 VXLAN interfaces (named vxlan1 up to vxlan10) in the host namespace. **

This repository contains an script to generate the necessary 10 VXLANs with their respective name, although any IP tunnelling mechanism can be suitable to be used. 

To use the script, execute the following command in every of the nodes of your cluster:

```bash
sudo ./L2S-M/K8s/provision/vxlan.bash
```
If you want to create the VXLANs manually, you can use the following code instead for every VXLAN in most Linux distributions:

```bash
sudo ip link add [vxlan_Name] type vxlan id [id] dev [interface_to_use] dstport [dst_port]
```

To configure the VXLAN tunnels between neighbouring nodes, you can use the following command for every pair of interfaces you want to configure in their respective nodes:

```bash
sudo bridge fdb append to 00:00:00:00:00:00 dst [dst_IP] dev [vxlan_Name]
```

**WARNING:**  Make sure that the VXLAN id coincides between each tunnel pairs, specially when using the configure_vxlan file. You can use the following table in order to check the associated ids with each one of the vxlans.

| **VXLAN Name** |**ID**  |
|--|--|
| vxlan1 | 1961 |
| vxlan2 |  1962 |
| vxlan3 |  1963 |
| vxlan4 |  1964|
| vxlan5 |  1965 |
| vxlan6 |  1966|
| vxlan7 |  1967|
| vxlan8 |  1968|
| vxlan9 |  1969|
| vxlan10 |  1970|

3. Create the vEth virtual interfaces in every host of the cluster by using the following script
```bash
sudo ./L2S-M/K8s/provision/veth.bash
```
4. Install the Multus CNI Plugin in your K8s cluster. For more information on how to install Multus in your cluster, check their [official GitHub repository](https://github.com/k8snetworkplumbingwg/multus-cni).
5. The host-device cni plugin must be able to be used in your cluster. If it Is not present in your K8s distribution, you can find how to install it in your K8s cluster in their [official GitHub repository](https://github.com/containernetworking/plugins).
6. Your K8s Controller node must be able to deploy K8s pods for the operator to work. Remove its master and control-plane taints using the following command:
```bash
kubectl taint nodes --all node-role.kubernetes.io/control-plane- node-role.kubernetes.io/master-
```

 
## Install L2S-M

1. Create the virtual interface definitions using the following command:
 ```bash
kubectl create -f ./L2S-M/K8s/interfaces_definitions
```

**NOTE:** If you are using interfaces whose definitions are not present in the virtual interfaces definitions in the folder, you must create the corresponding virtual definition in the same fashion as the VXLANs. For example, if you want to use a VPN interface called "tun0", first write the descriptor "tun0.yaml":
 ```yaml
apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
name: tun0
spec:
config: '{
"cniVersion": "0.3.0",
"type": "host-device",
"device": "tun0"
}'
```
Afterwards, apply the new interface definitions using kubectl:
  ```bash
kubectl create -f tun0.yaml
```
2. Create the Kubernetes account Service Account and apply their configuration by applying the following command:
 ```bash
kubectl create -f ./L2S-M/operator/deploy/config/
```

3. Create the Kubernetes Persistent Volume by using the following kubectl command:
 ```bash
kubectl create -f ./L2S-M/operator/deploy/mysql/
```

4. After the previous preparation, you can deploy the operator in your cluster using the YAML deployment file:
 ```bash
kubectl create -f ./L2S-M/operator/deploy/deployOperator.yaml
```

 You can check that the deployment was successful if the pod enters the "running" state using the *kubectl get pods* command.

5. Deploy the virtual OVS Daemonset using the following .yaml:
```bash
kubectl create -f ./L2S-M/operator/daemonset
```
**NOTE:** If you have introduced new interfaces in your cluster besides the vxlans, modify the descriptor to introduce those as well. (Modify both MULTUS annotations and the commands to attach the interface to the OVS switch). 

You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the following section of the repository] (https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors)
