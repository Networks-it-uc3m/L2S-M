# Detailed operational example

In this section a demonstration of a deployment is presented. The demonstration scenario is the same as the previously described, with three nodes interconnected with L2S-M:

![alt text](../resources/metrics_module_3nodes.svg "Module's architecture")

In summary this deployment will be done following these steps:


0. [Install requisites](#requisites)
1. [Deploy LPM-C](#1-deploy-lpm-c)
2. [Virtual networks creation](#2-virtual-networks-creation)
3. [Configure each node LPM](#3-configure-each-node-lpm)
4. [Deploy LPM](#4-deploy-the-lpm-instances)


## Requisites

As an L2S-M module, LPM requires a running installation of L2S-M in the cluster, prior to start running. 

The Cluster should have two or more Nodes to run the module, as the measurements are going to be on each end. These nodes must be interconnected according to the L2S-M [installation guide](https://github.com/Networks-it-uc3m/L2S-M/tree/main/operator), with VXLANs tunnels. 

In this example, an installation has been done of L2S-M with three nodes: NodeA, NodeB and NodeC, so each will have its own Programmable L2S-M Switch. NodeA will act as the master node.

## 1. Deploy LPM Collector

LPM Collector is deployed within a Pod and implemented with Prometheus. This Prometheus instance gathers and stores metrics from the network links across the cluster, accessing the LPM instances in each node (this is configured later in the installation). Follow these steps:

1. Configure the targets in `./lpm-c/prometheus-config.yaml` to specify the lpm endpoints from which metrics should be collected. The instances aren't created yet, thus, the endpoints are yet to exist, but can still be configured, by pointing to the local DNS service. For instance:

   ```yaml
      apiVersion: v1
      kind: ConfigMap
      metadata:
      name: prometheus-config
      namespace: prometheus
      data:
      prometheus.yml: |-
         global:
            scrape_interval: 15s
            evaluation_interval: 15s

         scrape_configs:
            - job_name: 'prometheus'
            static_configs:
            - targets:
               - node-a-lpm:8090
               - node-b-lpm:8090
               - node-c-lpm:8090
      ```
   In this configuration, the scraping interval is set to 15 seconds, and three LPM instances are going to be scraped: NodeA, NodeB, NodeC. 

   The target name, (node-a-lpm) is going to be later introduced when the services are created, in [step 4.](#4-deploy-the-lpm-instances)

2. Deploy the configuration map:

   ```
   kubectl create -f ./lpm-c/prometheus-config.yaml
   ```

3. Launch the LPM-C deployment with:

   ```
   kubectl create -f ./lpm-c/operator.yaml
   ```

   It's recommended to have only one LPM-C instance per cluster to centralize metric storage.



## 2. Virtual Networks creation

A single NetworkAttachmentDefinition is going to connect all instances, and each PLS will route traffic through the optimal path.

This is defined in `lpm/network.yaml`. And created with the following command:

   ```
   kubectl create -f ./lpm/network.yaml
   ```

Note that if this resource's name is changed, the lpm instances' spec, deployed in [step 4,](#4-deploy-the-lpm-instances) must be changed as well to make them work properly.


## 3. Configure each node LPM

Prior to creating the LPM instances, multiple configmaps must be created, one for each connected node in the cluster. This configuration is described with a JSON format. Here's an example that will help explain the available parameters:

```json
   {
      "Nodename": "NodeA",
      "MetricsNeighbourNodes": [
         {
               "name": "NodeB",
               "ip": "10.0.2.4",
               "jitterInterval": 10,
               "throughputInterval": 20,
               "rttInterval": 3
         },
         {
               "name": "NodeC",
               "ip": "10.0.2.6",
               "rttInterval": 20,
               "throughputInterval": 20,
         }
      ]
   }
```

This configuration is suitable for NodeA. Firstly the name of the Node is introduced. It will help identify the metrics measured from this Node. Then an array is introduced with multiple instances of the nodes this Node is going to measure to.

NodeA is going to measure its overlay network direct links with NodeB and NodeC, so two instances are introduced. In each of these, the following fields are introduced:

- **Name:** For identifying purposes.
- **IP:** The Node's LPM instance IP address, which will be used as the measurement endpoint. This IP address must be the one that's configured for the `net1` interface, in the LPM pods. As this IP is configured during the deployment, any local IP address can be used,  as long as its taken into account in [step 4,](#4-deploy-the-lpm-instances) when this configuration occurs.  For example, NodeB is going to be configured with the IP address `10.0.2.4`.
- **Intervals:** these configure two things: 
   
   1. Which specific metrics are to be measured between the node using this configuration and the target node specified in the configuration. For instance, you can specify that you want to measure Round-Trip Time (RTT) from 'NodeA' to 'NodeB.'

   2. The interval (in minutes) between each measurement.

   So, if the interval is not specified, the measurement will not take place. In the provided example, NodeA won't measure the amount of Jitter with NodeC, and will run a Jitter measurement with NodeB every 10 minutes.

This JSON configuration is introduced in a config-map. The following file, `lpm/nodeA-config.yaml`, may be used as a reference:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: node-a-config
data:
  config.json: |
    {
      "Nodename": "NodeA",
      "MetricsNeighbourNodes": [
         {
               "name": "NodeB",
               "ip": "10.0.2.4",
               "jitterInterval": 10,
               "throughputInterval": 20,
               "rttInterval": 3
         },
         {
               "name": "NodeC",
               "ip": "10.0.2.6",
               "rttInterval": 20,
               "throughputInterval": 20,
         }
      ]
   }
```

Any name for the config-map can be used, but it should be noted when creating, so later, during the deployment, it can be mounted without any issues. 

Once all the yaml files are written, we can create the config-maps with:

   ```
   kubectl create -f ./lpm/config/node-a-config.yaml
   ```

   ```
   kubectl create -f ./lpm/config/node-b-config.yaml
   ```

   ```
   kubectl create -f ./lpm/config/node-c-config.yaml
   ```

## 4. Deploy the LPM instances

The core of the module resides in all of the LPM instances. Details on how it operates can be found in the [overview section](#overview), and in the [source code implementation](./src/core/doc.go). To deploy these components, follow these steps:

1. Write a yaml file for each node that will run the metrics. The following file may be used as a reference:
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
   name: node-a-lpm
   spec:
   replicas: 1
   selector:
      matchLabels:
         app: node-a-lpm
   template:
      metadata:
         labels:
         app: node-a-lpm
      spec:
         containers:
         - name: lpm-container
         image: alexdecb/net_exporter:latest
         workingDir: /usr/src/app
         securityContext:
            capabilities:
               add: ["NET_ADMIN"]
         ports:
         - containerPort: 8090  
         volumeMounts:
         - name: config-volume
            mountPath: /usr/src/app/config.json
            subPath: config.json
         volumes:
         - name: config-volume
         configMap:
            name: node-a-config
   ---
   apiVersion: v1
   kind: Service
   metadata:
   name: node-a-lpm
   spec:
   selector:
      app: node-a-lpm
   ports:
      - protocol: TCP
         port: 8090
         targetPort: 8090
   ```

   When configuring this file be aware of the following points:

   - The used image is made from the Dockerfile in the source code, you can build your own image or use the maintained public image, `alexdecb/net_exporter:latest`.

   - The container, by default, will listen on the target port `8090`. 

   - The attached config-map's name must correspond with the one created in [step 3](#3-configure-each-node-lpm).

   - The service name and port must correspond with the defined endpoint in the LPM-C configuration ([step 1](#1-deploy-lpm-c)).

   -  NET_ADMIN must be used in the `spec.securityContext.capabilities` of the container, so later an IP address can be assigned to the L2S-M virtual interface.

   
 2. After entering the parameters, launch the LPM instances and their associated services using:

      ```
      kubectl create -f ./lpm/deploy/node-a-deploy.yaml
      ```
      ```
      kubectl create -f ./lpm/deploy/node-b-deploy.yaml
      ```
      ```
      kubectl create -f ./lpm/deploy/node-c-deploy.yaml
      ```

3. After each individual instance is deployed, the IP addresses must be assigned through the interfaces. These should be configured according to the modules defined configuration in [step 3.](#3-configure-each-node-lpm) In the provided use-case scenario:
   - IP Address of NodeA: `10.0.2.2`
   - IP Address of NodeB: `10.0.2.4`
   - IP Address of NodeC: `10.0.2.6`

   This configuration can be done by entering each container and adding it manually, for example, for the LPM in NodeA:

   ```
   kubectl exec -it [node-a-pod-name] -- /bin/bash
   ```
   ```
   ip link set net1 up
   ```
   ```
   ip addr add 10.0.2.2/28 dev net1
   ```

Once all these instances are up and running you may access the LPM-C HTTP server, through the `http://prometheus:9090/metrics` endpoint. An additional deployment of the Grafana tool has been added to the repo, if it may be of any use to visualize the data, in `ĺpm-c/grafana.yaml`.

