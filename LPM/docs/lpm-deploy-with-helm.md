# Helm Charts Deployment Guide

This guide will walk you through deploying a similar setup using Helm charts. We assume you have Helm installed and configured to interact with your Kubernetes cluster.

## Deployment Steps

Follow these steps to deploy your application using Helm charts:

1. [Install Prerequisites](#1-install-prerequisites)
2. [Update Values.yaml](#2-update-values.yaml)
3. [Deploy Helm Chart](#3-deploy-helm-chart)
4. [Configure Node IPs](#4-configure-node-ips)

### 1. Install Prerequisites

Ensure you have Helm installed and configured. You can find the installation instructions on the [Helm official website](https://helm.sh/docs/intro/install/).

### 2. Update Values.yaml

In the `../charts/values.yaml` file, update the following configuration to match your deployment requirements. Specify the node names and which ip addresses you want to attach to them. Any range will work as long as all ip addresses are in the same range. As these will be assigned inside the private layer 2 networks, there won't be any conflict with other ip addresses. 

```yaml
global:
  nodes:
    - name: test-l2sm-uc3m-polito-1
      ip: 10.0.0.2/24
      metrics:
        rttInterval: 10
        throughputInterval: 20
        jitterInterval: 5
    - name: test-l2sm-uc3m-polito-2
      ip: 10.0.0.3/24
      metrics:
        rttInterval: 10
        throughputInterval: 20
        jitterInterval: 5
    - name: test-l2sm-uc3m-polito-3
      ip: 10.0.0.4/24
      metrics:
        rttInterval: 10
        throughputInterval: 20
        jitterInterval: 5
  network:
    name: lpm-network
  namespace: default
```

3. Deploy Helm Chart

Navigate to the directory containing your Helm chart (LPM/charts) and deploy the chart with the following command:

```bash
helm install my-lpm-chart ./charts
```

This command installs the chart using the configurations specified in values.yaml.