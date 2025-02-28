# L2S-M VxLAN configuration guide

In order to connect the nodes between themselves, an additional configuration must be done. A configuration CR specifying which nodes we want to connect and and which virtual switches will be made.

Create a file anywhere or use [overlay-sample.yaml](./overlay-sample.yaml) as a reference. 
In this file, you will specify, using the template shown in the reference file, the name of the nodes in the cluster. 

These are the configurable fields:
    - **Topology**: Specifies how nodes should be interconnected. Is a list of nodes (name of the k8s nodes) and links. Links have two endpoints, corresponding to the nodes defined earlier. Links are bidirectional, meaning just one link between two nodes is enough.
    - **Switch Template**: Defines the type of switches used within the overlay. By default, set the one defined by us, using the image `alexdecb/l2sm-switch:TAG`. 
    - **Provider**: Identifies the SDN controller responsible for managing the topology. Meant to be set by default as the one that comes with the installation.
    - **InterfaceNumber**: (Optional) The number of interfaces per node for the switches, by default is 10. These can't be added dynamically so 
For example:
```bash
$ kubectl get nodes
NAME                            STATUS   ROLES    AGE   VERSION
l2sm-test-control-plane         Ready    <none>   64d   v1.30.5
l2sm-test-worker                Ready    <none>   63d   v1.30.5
l2sm-test-worker2               Ready    <none>   63d   v1.30.5
```
In this example we have two nodes: l2sm1 and l2sm2, that we want to interconnect.
  
We want to connect them directly, so we modify the reference CR:
```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: Overlay
metadata:
  name: overlay-sample
spec:
  provider:
    name: l2sm-test-sdn
    domain: "l2sm-controller-service.l2sm-system.svc"
  numberInterfaces: 30
  topology:
    nodes:
      - [<NODE_1>]
      - [<NODE_2>]
      - ...
      - [<NODE_N>]
    links:
      - endpointA: [<NODE_1>]
        endpointB: [<NODE_3>]
      - endpointA: [<NODE_2>]
        endpointB: [<NODE_3>]
      - ...
  switchTemplate:
    spec:
      containers:
        - name: l2sm-switch
          image: alexdecb/l2sm-switch:1.2.9
          imagePullPolicy: Always
          resources: {}
          env:
          - name: NODENAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
```

> [!NOTE]
> This used switch template is required by the l2sm-switch api, and it's not yet completly supported the use of another switch image.  How this switch works can be seen in the [l2sm-switch repository](https://github.com/Networks-it-uc3m/l2sm-switch). If you are interested in developing or using your custom virtual switch, you can contact the developers for more info.

Example of how it looks:

```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: Overlay
metadata:
  name: overlay-sample
spec:
  provider:
    name: l2sm-test-sdn
    domain: "l2sm-controller-service.l2sm-system.svc"
  topology:
    nodes:
      - l2sm-test-control-plane
      - l2sm-test-worker
      - l2sm-test-worker2
    links:
      - endpointA: l2sm-test-worker
        endpointB: l2sm-test-worker2
      - endpointA: l2sm-test-worker
        endpointB: l2sm-test-control-plane
      - endpointA: l2sm-test-control-plane
        endpointB: l2sm-test-worker2
  switchTemplate:
    spec:
      containers:
        - name: l2sm-switch
          image: alexdecb/l2sm-switch:1.2.9
          imagePullPolicy: Always
          resources: {}
          env:
          - name: NODENAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
```

Once this file is created, we create the overlay topology using the kubectl create command:

```bash
kubectl create -f ./examples/overlay-setup/overlay-sample.yaml
```

If you want to modify the topology, just modify the contents of the file and use `kubectl apply -f ...` with the new file.

You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the ping-pong example](../ping-pong/)!


If you want to extend your network overlay to make it accesible from other clusters, check the [inter cluster setup guide](../inter-cluster-setup/)
