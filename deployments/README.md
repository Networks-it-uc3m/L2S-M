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

 
## Install L2S-M

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

**Lastly, we configure the Vxlans:**

In order to connect the switches between themselves, an additional configuarion must be done. A configuration file specifying which nodes we want to connect and which IP addresses their switches have will be made, and then a script will be run in each l2sm switch, using this configuration file. 

  a. Create a file anywhere or use the reference in ./configs/sampleFile.json. In this installation, this file will be used as a reference.
  b. In this file, you will specify, using the template shown in the reference file, the name of the nodes in the cluster and the IP addresses of **the switches** running on them. For example:
  ```bash
  $ kubectl get pods -o wide
  >NAME                                               READY   STATUS    RESTARTS   AGE     IP            NODE    NOMINATED NODE   READINESS GATES
  >l2sm-controller-d647b7fb5-lpp2h                    1/1     Running   0          30m     10.1.14.55    l2sm1   <none>           <none>
  >l2sm-operator-7d487d8468-lhgkx                     2/2     Running   0          2m11s   10.1.14.56    l2sm1   <none>           <none>
  >l2sm-switch-8p5td                                  1/1     Running   0          71s     10.1.14.58    l2sm1   <none>           <none>
  >l2sm-switch-xdkvz                                  1/1     Running   0          71s     10.1.72.111   l2sm2   <none>           <none>

  ```
  In this example we have two nodes: l2sm1 and l2sm2, with two switches, with IP addresses 10.1.14.58 and 10.1.72.111.
  
  We want to connect them directly, so we modify the reference file, ./configs/sampleFile.json:
```json
[
    {
        "name": "<NODE_SWITCH_1>",
        "nodeIP": "<IP_SWITCH_1>",
        "neighborNodes": ["<NODE_SWITCH_2>"]
    },
    {
        "name": "<NODE_SWITCH_2>",
        "nodeIP": "<IP_SWITCH_2>",
        "neighborNodes": ["<NODE_SWITCH_1>"]
    }
]

```
Note: The parameters to be changed are shown in the NODE and IP columns of the table above.

Example of how it looks:
```json
[
    {
        "name": "l2sm1",
        "nodeIP": "10.1.14.58",
        "neighborNodes": ["l2sm2"]
    },
    {
        "name": "l2sm2",
        "nodeIP": "10.1.72.111",
        "neighborNodes": ["l2sm1"]
    }
]

```
Note: Any number of nodes can be configured, as long as the entry is in this file. The desired connections are under the neighborNodes field, in an array, such as this other example, where we add a neighbor to l2sm2: ["l2sm1","l2sm3"]

Once this file is created, we inject it to each node using the kubectl cp command:

```bash
kubectl cp ./configs/sampleFile.json <pod-name>:/etc/l2sm/switchConfig.json 
```
And then executing the script in the switch-pod:
```bash
kubectl exec -it <switch-pod-name> -- setup_switch.sh
```

This must be done in each switch-pod. In the provided example, using two nodes, l2sm1 and l2sm2, we have to do it twice, in l2-ps-8p5td and l2-ps-xdkvz.
When the exec command is done, we should see an output like this:

```bash
$ kubectl exec -it l2-ps-xdkvz -- setup_switch.sh
2023-10-30T10:22:18Z|00001|ovs_numa|INFO|Discovered 1 CPU cores on NUMA node 0
2023-10-30T10:22:18Z|00002|ovs_numa|INFO|Discovered 1 NUMA nodes and 1 CPU cores
2023-10-30T10:22:18Z|00003|reconnect|INFO|unix:/var/run/openvswitch/db.sock: connecting...
2023-10-30T10:22:18Z|00004|netlink_socket|INFO|netlink: could not enable listening to all nsid (Operation not permitted)
2023-10-30T10:22:18Z|00005|reconnect|INFO|unix:/var/run/openvswitch/db.sock: connected
initializing switch, connected to controller:  10.1.14.8
Switch initialized and connected to the controller.
Created vxlan between node l2sm2 and node l2sm1.
```


You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the following section of the repository](https://github.com/Networks-it-uc3m/L2S-M/tree/release-2.0/examples/)
