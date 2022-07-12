# L2S-M Installation Guide
This guide details the necessary steps to install the L2S-M Kubernetes operator to create and manage virtual networks in your Kubernetes cluster.


# Prerequisites

1. Clone the L2S-M repository in your host. This guide will assume that all commands are executed in the directory where L2S-M was downloaded.

2. As a prerequisite to start with the installation of L2S-M, it is necessary to set up an IP tunnel overlay among the nodes of your k8s cluster (see  [how L2S works](https://github.com/Networks-it-uc3m/L2S-M/tree/main/K8s). To do so, **the installation needs 10 VXLAN interfaces (named vxlan1 up to vxlan10) in the host namespace, as well as 10 vEth pairs in order to support the attachment of pods to virtual networks.**

    This repository contains a script to generate all the necessary interfaces with their respective names. (this is the **recommended option**).

    To use this script, firstly it is neccessary to write the destinations that each IP tunnel will have for every VXLAN interface in each host. To perfom this action, **open the vxlans.txt file (located in ./L2S-M/K8s/provision/vxlan.txt) with your prefered text editor and assign each vxlan interface an IP address of the neighboring K8s node used in each VXLAN tunnel**.

    **WARNING:**  Make sure that the VXLAN network identifier (VNI) is the same at every pair of k8s nodes terminating an IP tunnel. In case that you use the script for automatic VXLAN configuration, the VXLAN interface names and their corresponding VNIs are indicated in the table below. 

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

    After modifying the vxlan.txt file, you can create all the interfaces (VXLAN and vEth) with the provided script using the following command:

    ```bash
    sudo sh ./L2S-M/K8s/provision/set-interfaces.sh [interface_to_use_for_vxlan_tunnel] ./L2S-M/K8s/provision/vxlans.txt
    ```

    **IMPORTANT** In order to keep the configuration after the host has been rebooted, a cron job should be written in order to use this script to create and configure the virtual interfaces. To enable its use, open (or create) a new crontab in the host:

    ```bash
    sudo crontab -e 
    ```

    Once opened, append the following line at the end of the file:

    ```bash
    @reboot sh [directory-where-L2S-M-directory-is-located]/L2S-M/K8s/provision/set-interfaces.sh [interface_to_use] [directory-where-L2S-M-directory-is-located]/L2S-M/K8s/provision/vxlans.txt
    ```

* You may want to manually create the VXLANs instead. **Note: We highly suggest to use the reccomended option described in 2. to keep the configuration across reboots**. To that purpose, you can use the following command for every VXLAN in most Linux distributions:

    ```bash
    sudo ip link add [vxlan_Name] type vxlan id [id] dev [interface_to_use] dstport [dst_port]
    ```

    **WARNING:** Make sure that the VXLAN network identifier (VNI) is the same at every pair of k8s nodes terminating an IP tunnel, if you are manually creating the interfaces. In case that you use the script mentioned above for automatic VXLAN configuration, the VXLAN interface names and their corresponding VNIs are indicated in the table above.

    To finish the configuration of a VXLAN tunnel between two neighboring k8s nodes, you can execute the following command at each K8s nodes:

    ```bash
    sudo bridge fdb append to 00:00:00:00:00:00 dst [dst_IP] dev [vxlan_Name]
    ```
    where *dst_IP* must be replaced by the IP address of the neighboring K8s node in the VXLAN tunnel.

    To create the set of vEth virtual interfaces in every host of the K8s cluster, you can use the following script:

    ```bash
    sudo ./L2S-M/K8s/provision/veth.bash
    ```

3. Install the Multus CNI Plugin in your K8s cluster. For more information on how to install Multus in your cluster, check their [official GitHub repository](https://github.com/k8snetworkplumbingwg/multus-cni).

4. The host-device CNI plugin must be able to be used in your cluster. If it is not present in your K8s distribution, you can find how to install it in your K8s cluster in their [official GitHub repository](https://github.com/containernetworking/plugins).

5. Your K8s Controller node must be able to deploy K8s pods for the operator to work. Remove its master and control-plane taints using the following command:
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

4. Before deploying the L2S-M operator, it is neccessary to label your master node as the "master" of the cluster. To do so, get the names of your Kubernetes nodes, select the master and apply the "master" label with the following command:

 ```bash
kubectl get nodes
kubectl label nodes [your-master-node] dedicated=master
```

5. After the previous preparation, you can deploy the operator in your cluster using the YAML deployment file:
 ```bash
kubectl create -f ./L2S-M/operator/deploy/deployOperator.yaml
```

 You can check that the deployment was successful if the pod enters the "running" state using the *kubectl get pods* command.

6. Deploy the virtual OVS Daemonset using the following .yaml:
```bash
kubectl create -f ./L2S-M/operator/daemonset
```
**NOTE:** If you have introduced new interfaces in your cluster besides the VXLANs, you will need to modify the descriptor to introduce those as well (modify both MULTUS annotations and the commands to attach the interface to the OVS switch). 

You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the following section of the repository](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors)
