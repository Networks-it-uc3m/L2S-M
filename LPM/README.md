# L2S-M Performance Measurements (LPM)

Welcome to **L2S-M Performance Measurements (LPM)**, an L2S-M module designed to flexibly and automatically collect performance metrics of the connectivity provided by L2S-M within a single Kubernetes cluster. This module carries out a comprehensive network performance profiling of the overlay network that L2S-M deploys in a cluster to offer link-layer connectivity. For this purpose, LPM considers different network metrics (e.g., available bandwidth, end-to-end delay, etc), and it is also designed to easily and agilely include new and tailored metrics.


> **NOTE**:
>  All the information about L2S-M is available in its official repository: http://l2sm.io


## Table of Contents

- [Overview of the LPM design](#overview-of-the-lpm-design)
- [How it works?](#how-it-works)
- [TBD: Adding tailored metrics](#adding-tailored-metrics)


## Overview of the LPM design

As previously commented, LPM is able to characterize the layer-2 links of the overlay network managed by L2S-M, conducting different performance measurements over it. In addition, LPM supports the exposure of the collected metrics over a unique HTTP endpoint in the cluster, so that they can be visualized through any third-party application such as Grafana. The following figure showcases the components comprising LPM, taking as a reference a K8s cluster that consists of a node that provides the K8s control-plane, and two additional nodes implementing respective K8s workers.



![alt text](./resources/metrics_module_3nodes.svg "Module's architecture")

Next, we briefly introduce the functionality of each of the components encompassed within this module:

**LPM**

This component is in charge of realizing the performance measurements at different time-intervals, as well as storing the results obtained for each measurement. It is deployed in each node of the cluster, and each instance executes simultaneously several processes:


- *Endpoint Measurements*: this process leverages common network utilities such as _iperf_ and _ping_ to periodically measure the network performance between different LPM instances. The results of these measurements, referred to as 'metrics', are stored locally. The structure in which these metrics are stored follows a key-value pair format for each measurement. 

- *Metric Exposure*: metrics are accessible over HTTP and within the LPM component itself. Therefore, a service is set up to enable this endpoint to expose metrics information and thus, allowing _LPM Collector_ to request the  and store the metrics in a database.

- *Constant Server*: ensures that the necessary servers for measurements are continuously operational. While LPM actively measures the network capabilities in one direction, it also allows other LPM instances on the opposite ends of network links to initiate measurements from their end. 

**LPM Collector** 

The LPM Collector component is responsible of accessing to every LPM instance (_i.e._, to the `/metrics` endpoint offered at each LPM), and retrieve the performance data ,easured. The LPM Collector performs this scraping every 15 seconds (this is the interval configure by default, but can be modified depending on the preferences) with every LPM instance within the cluster. For this purpose, it uses the standard K8s communication interface configured in K8s pods (for instance, the communications interfaces implemented with Flannel).

This component is based on [Prometheus](https://github.com/prometheus/prometheus), which allows to store the information as entries of key-value pairs in a Non-SQL time series database. Therefore, this information can be later obtained (using HTTP requests) and use any visualization tool, such as Grafana, to represent it.

**Programmable L2 Switch (PLS)**

The PLS components are the programable virtual switches deployed by L2S-M, which enables the link-layer connectivity within a K8s cluster. As further explained in [L2S-M](http://l2sm.io), these virtual switches, and the links among them (established through tunneling technologies such as VxLAN), constitute the layer-2 overlay network over which L2S-M is able to realize the lifecycle management of virtual networks. Thus, enabling isolated and link-layer communications among pods deployed in the cluster itself.   

Then, the LPM instances leverages the connectivity links implemented by L2S-M and these virtual switches to carry out a performance analysis of the overlay network. 


## How it works

This section presents a conceptual (step-by-step) workflow to better understand how LPM works. This workflow considers the scenario presented in the figure above, where we have a cluster consisting of three nodes: Node A (K8s control-plane), Node B and Node C (workers). Each node will have an LPM instance with a specific configuration, and they will be connected through a PLS. Node A, will host the LPM Collector which, as commented before, will store the collected performance metrics.

1. **Configuration setup**:
   as a previous step before deploying the module and initiating measurements, it is crucial to plan in advance the configuration of each LPM instance according to the specific metrics to be measured. For instance, it might be required to measure the Round-Trip Time (RTT) from Node A to Node B,  but not from Node B to Node A. To address these requirements, it is possible to independently configure each LPM instance (in a human-friendly manner by using YAML files), specifying the metrics to be measuredand with which LPM instance, and the periodicity (time intervals) at which they will be measured .

2. **Deployment and measurement**:
   the LPM instances and the LPM Collector are deployed as any other application deployed on K8s (_i.e._, using a K8s deployment). At this precise moment, it is assumed that there is an overlay network controlled by L2S-M among the nodes of our cluster, and that L2S-M provides a virtual network over which the LPM components are able to conduct the performance measurements. Once the components are instantiated, the different performance measurements (configured in the previous step) are initiated independently, and at the preset time-instant. As part of the process, each LPM instance deploys a performance server for each measurement type (_e.g._, jitter or throughput) in background mode.


3. **Performance measurements collection**:
   Throughout the measurement process, each LPM instance is accessible (via HTTP) by the LPM Collector to obtain and store all the metrics from the last measurement. As commented above, this is done by periodically scraping the `/metrics` endpoint offered at each LPM instance every 15 seconds.

   So in this scenario, if a metric being measured between Node A and Node B occurs every 30 minutes, the LPM Collector will store 120 entries (60 seconds/minute / 15 seconds * 30 minutes). This is done this way to exploit the characteristics of the time-series database, where each entry has a timestamp assigned to it, and can be compared in time with other metrics from other links.


In the [documentation folder](docs/lpm-operational-example.md), we have included an LPM operational where we detail the steps involved workflow from a technical perspective. In particular, we specify the commands and configuration files so that the deployment of the LPM module can be reproduced and its functionality tested in an environment similar to the one presented in our reference scenario. 

## Adding tailored metrics
TBD
