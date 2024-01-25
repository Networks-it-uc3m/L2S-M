# L2S-M VxLAN configuration guide

In order to connect the switches between themselves, an additional configuration must be done. A configuration file specifying which nodes we want to connect and which IP addresses their switches have will be made, and then a script will be run in each **l2sm-switch**, using this configuration file. 

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
kubectl exec -it <switch-pod-name> -- /bin/bash -c 'l2sm-vxlans --node_name=$NODENAME /etc/l2sm/switchConfig.json'
```

This must be done in each switch-pod. In the provided example, using two nodes, l2sm1 and l2sm2, we have to do it twice, in l2-ps-8p5td and l2-ps-xdkvz.
When the exec command is done, we should see an output like this:

```bash
kubectl exec -it l2sm-switch-8p5td -- /bin/bash -c 'l2sm-vxlans --node_name=$NODENAME /etc/l2sm/switchConfig.json'
Defaulted container "l2sm-switch" out of: l2sm-switch, wait-for-l2sm-controller (init)
Created vxlan between node l2sm1 and node l2sm2.
```

You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the following section of the repository](https://github.com/Networks-it-uc3m/L2S-M/tree/release-2.0/examples/)
