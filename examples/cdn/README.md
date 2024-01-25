# Example: Isolating an NGINX server from a CDN with Custom L2SM networks

## Overview

This example demonstrates the isolation of traffic between pods using custom networks with L2S-M In this scenario, two networks, v-network-1 and v-network-2, are created, and three pods (cdn-server, router, and content-server) are connected. The objective is to showcase how traffic can be isolated through a router (router) connecting the two networks.

## Topology

### Networks

- v-network-1
- v-network-2

### Pods

- **podA (CDN Server)**
  - IP: 10.0.1.2
  - Network: v-network-1

- **podB (Router)**
  - Networks: v-network-1, v-network-2
  - IP: 10.0.1.1 (net1) and 10.0.2.1 (net2)

- **podC (Content Server)**
  - IP: 10.0.2.2
  - Network: v-network-2

## Procedure

1. **Show Nodes**

```bash
kubectl get nodes
```
2. **Show Pods**

```bash
kubectl get pods -o wide
```

3. **Show Networks**

```bash

kubectl get net-attach-def
```

4. **Operator Logs**
```bash

kubectl logs l2sm-operator-667fc88c57-p7krv
```

Show the creation of networks and pod attachments.

5. **Controller Logs**

```bash

kubectl logs l2sm-controller-d647b7fb5-kb2f7
```

Demonstrate the creation of networks and connections between pods.

6. **Enter CDN and Content-Server Pods**

To setup the server, enter it by doing the ``exec`` command
```bash
kubectl exec -it content-server /bin/bash   # Enter Content-Server pod
```
In the Content-Server pod, execute the following commands:

```bash
ip a s          # Show IP addresses
```

```bash
ip r s          # Display routing table
```

```bash
nginx           # Start the server
```

To test the connectivity from the cdn server: 
```bash
kubectl exec -it cdn-server /bin/bash   # Enter CDN-Server pod
```
In the CDN pod, execute the following commands:

```bash
ip a s                     # Show IP addresses
```
```bash
ip r s                     # Display routing table
```
```bash
traceroute 10.0.2.2        # Trace route to content-server
```
```bash
curl http://10.0.2.2/big_buck_bunny.avi --output video.avi  --limit-rate 2M # Download video
```

While the video downloads delete the router pod:

```bash
kubectl delete pod router
```

And watch how the traffic stops. You may continue the download by doing:
```bash
kubectl create -f router.yaml
```
Where the router pod enter the two desired networks and will start funcion again.



