# Example: Isolating an NGINX server from a CDN with Custom L2SM networks
## Overview

This example demonstrates the isolation of traffic between pods using custom networks with L2S-M. In this scenario, two networks, v-network-1 and v-network-2, are created, and three pods (cdn-server, router, and content-server) are connected. The objective is to showcase how traffic can be isolated through a router connecting the two networks.

## Topology
This example can be seen in action [in the screencast provided](#procedure), where it's presented a Cluster scenario with three nodes, where a Pod will be deployed in each Node, as shown in the following figure:

<p align="center">
  <img src="../../assets/video-server-example.svg" width="400">
</p>

The following example doesn't really need a three Node scenario, it can be used with just a Node in the Cluster. Through the example guide, we will create the following resources:

### Networks

- [v-network-1](./v-network-1.yaml)
- [v-network-2](./v-network-2.yaml)

Two virtual L2S-M networks, without any additional configuration.

### Pods

Note: The configurations specified can be seen in each Pod YAML specification.

- **[cdn-server](./cdn-server.yaml) (CDN Server)**
  This pod will act as a CDN server, it's just an alpine image with the following pre-configuration:
  - IP: 10.0.1.2
  - Network: v-network-1

- **[router](./router.yaml) (Router)**
  This pod will act as a router, where we could launch some firewall rules if we wanted. It will have the following pre-configuration:
  - Networks: v-network-1, v-network-2
  - IP: 10.0.1.1 (net1) and 10.0.2.1 (net2)
  - Forwarding enabled

- **[content-server](./content-server.yaml) (Content Server)**
  This pod will act as a content server. The image can be found at the [./video-server directory](./video-server/). It's an NGINX image with a video file that will be served. It has the following pre-configuration: 
  - IP: 10.0.2.2
  - Network: v-network-2

## Procedure

Follow the steps below to demonstrate the isolation of traffic between pods using custom networks with L2S-M. You can watch a screencast of how this operates and how it should follow through this youtube video: 

<p align="center">
  <a href="https://www.youtube.com/watch?v=Oj2gzm-YxYE" target="_blank">
    <img src="https://img.youtube.com/vi/Oj2gzm-YxYE/maxresdefault.jpg" width="400">
  </a>
</p>

### 1. Create Virtual Networks

   - Create two virtual L2S-M networks: [v-network-1](./v-network-1.yaml) and [v-network-2](./v-network-2.yaml).

```bash
kubectl create -f ./examples/cdn/v-network-1.yaml
```
```bash
kubectl create -f ./examples/cdn/v-network-2.yaml
```

### 2. Verify Network Creation

Note: This step is optional, but it will help you understand how L2S-M internally work, if you already know a bit about SDN and network overlays. 
   - Check the logs in the `l2sm-controller` and `l2sm-operator` to ensure that the virtual networks have been successfully created.

```bash
kubectl get net-attach-def
```
```bash
kubectl logs l2sm-operator-667fc88c57-p7krv
```
```bash
kubectl logs l2sm-controller-d647b7fb5-kb2f7
```

### 3. Deploy Pods

   - Deploy the following three pods, each attached to specific networks:
     - [cdn-server](./cdn-server.yaml) (CDN Server) attached to `v-network-1`
     - [router](./router.yaml) (Router) connected to both `v-network-1` and `v-network-2`
     - [content-server](./content-server.yaml) (Content Server) attached to `v-network-2`

```bash
kubectl create -f ./examples/cdn/cdn-server.yaml
```
```bash
kubectl create -f ./examples/cdn/content-server.yaml
```
```bash
kubectl create -f ./examples/cdn/router.yaml
```
### 4. Verify Intent Creation

   - Examine the logs in the `l2sm-controller` to confirm that the intents for connecting the pods to their respective networks have been successfully created.

```bash
kubectl logs l2sm-controller-d647b7fb5-kb2f7
```
```bash
kubectl get pods
```

### 5. Inspect Content Server

   - Enter the `content-server` pod and check its IP configuration.
     
```bash
kubectl exec -it content-server /bin/bash  
```
```bash
ip a s          # Show IP addresses
```
```bash
ip r s          # Display routing table
```
   - Start the server to serve the video content.

```bash
nginx           # Start the server
```

### 6. Inspect CDN Server

   - Enter the `cdn-server` pod and add the `curl` command to initiate communication with the content server.
   - Check the IPs to ensure connectivity.

To test the connectivity from the cdn server: 
```bash
kubectl exec -it cdn-server /bin/bash   # Enter CDN-Server pod
```
In the CDN pod, execute the following commands:

```bash
apk add curl               # Install the curl cli
```
```bash
ip a s                     # Show IP addresses
```
```bash
ip r s                     # Display routing table
```

### 7. Perform Traceroute

   - Execute a traceroute to observe any intermediaries between the content server and CDN server. It should appear like theres a step between them, the router.

```bash
traceroute 10.0.2.2        # Trace route to content-server
```

### 8. Test Communication

   - Perform a `curl` from the CDN server to the content server to initiate video retrieval.
```bash
curl http://10.0.2.2/big_buck_bunny.avi --output video.avi  --limit-rate 2M # Download video
```
Note: leave this Pod running while doing the next steps.

### 9. Introduce Interruption

   - Delete the pod for the router and observe that the video communication stops.
   While the video downloads delete the router pod:

```bash
kubectl delete pod router
```

### 10. Restore Connection

   - Restart the router pod and verify the reconnection of the `content-server` and `cdn-server`.

  ```bash
  kubectl create -f router.yaml
  ```


