# L2S-M VxLAN configuration guide

In order to connect the nodes between themselves, an additional configuration must be done. A configuration CR specifying which nodes we want to connect and and which virtual switches will be made.

Create a file anywhere or use [overlay-sample.yaml](./overlay-sample.yaml) as a reference. 
In this file, you will specify, using the template shown in the reference file, the name of the nodes in the cluster. For example:
```bash
$ kubectl get nodes
NAME          STATUS   ROLES    AGE   VERSION
l2sm1         Ready    <none>   64d   v1.30.5
l2sm2         Ready    <none>   63d   v1.30.5
```
In this example we have two nodes: l2sm1 and l2sm2, that we want to interconnect.
  
We want to connect them directly, so we modify the reference CR:
```yaml
apiVersion: l2sm.l2sm.k8s.local/v1
kind: Overlay
metadata:
  name: overlay-sample
spec:
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
          image: alexdecb/l2sm-switch:2.7
          resources: {}
          env:
          - name: NODENAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          - name: NVETHS
            value: "10"  
          - name: CONTROLLERIP
            value: "l2sm-controller-service"
          - name: PODNAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          imagePullPolicy: Always
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
          ports:
            - containerPort: 80
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
  topology:
    nodes:
      - l2sm1
      - l2sm2
    links:
      - endpointA: l2sm1
        endpointB: l2sm2
  switchTemplate:
    spec:
      containers:
        - name: l2sm-switch
          image: alexdecb/l2sm-switch:2.7
          resources: {}
          env:
          - name: NODENAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
          - name: NVETHS
            value: "10"  
          - name: CONTROLLERIP
            value: "l2sm-controller-service"
          - name: PODNAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          imagePullPolicy: Always
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
          ports:
            - containerPort: 80
```


Once this file is created, we create the overlay topology using the kubectl create command:

```bash
kubectl create -f ./examples/overlay-setup/overlay-sample.yaml
```

If you want to modify the topology, just modify the contents of the file and use `kubectl apply -f ...` with the new file.

You are all set! If you want to learn how to create virtual networks and use them in your applications, [check the ping-pong example](../ping-pong/)!


If you want to extend your network overlay to make it accesible from other clusters, check the [inter cluster setup guide](../inter-cluster-setup/)
