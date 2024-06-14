# L2S-M Development Guide

Welcome to the L2S-M development guide. This README provides detailed instructions for setting up and developing L2S-M, which consists of four main components: `l2sm-controller`, `l2sm-operator`, `l2sm-switch`, and `l2sm-ned`. Follow the steps below to set up your development environment and deploy each component.

## Table of Contents

- [Repository Structure](#repository-structure)
- [Prerequisites](#prerequisites)
- [Component Development and Deployment](#component-development-and-deployment)
  - [L2SM-Controller](#l2sm-controller)
  - [L2SM-Operator](#l2sm-operator)
  - [L2SM-Switch](#l2sm-switch)

## Repository Structure

Below is a brief overview of the repository's structure and the purpose of major directories:

```bash
L2S-M
├── .vscode
│   └── launch.json
├── LICENSE
├── README.md
├── deployment
|   └── custom-installation
├── build
... [shortened for brevity] ...
└── src
    ├── controller
    ├── operator
    └── switch
```
In the L2S-M/src directory you will find the source code of each component, which is used to build the images in L2S-M/build. 
In L2S-M/build/README.md there is a guide of how to use the Dockerfiles and build the docker images referencing the code in L2S-M/src. A script has been made to ease this task.
In L2S-M/deployment/custom-installation there is a guide on how to install each component in kubernetes, that will enable the developing of specific parts of L2S-M 

## Prerequisites

Before you begin, ensure you have met the following requirements:

- A kubernetes cluster you have complete access to.
- Multus installed in the cluster. 
- For each component you're gonna develop, you may need specific tools and software.
- [L2S-M custom installation](../deployments/custom-installation/). Install L2S-M up to the component you want to modify/debug/develop, and come back here to check how to proceed with the installation.

## Component Development and Deployment

### L2SM-Controller

1. **Custom installation**: The source code for `l2sm-controller` is hosted in a separate repository. Refer to it to see how this component works and how to change it and deploy it manually.

2. **Configuration**: Specify the IP address the `l2sm-controller` is working on in the `deployOperator.yaml` and `deploySwitch.yaml` files, in the [custom-installation](../deployments/custom-installation/) directory.

3. **Custom Installation**: Follow the custom installation instructions exactly as described in the [custom-installation](../deployments/custom-installation/) directory.

### L2SM-Operator

>**Note:** you need python3 and the requirements specified in the [L2S-M/src/operator/requirements.txt](../src/operator/requirements.txt) to run it.

1. **Database Setup**: Run the MySQL development database using `mysql-development.yaml`.

2. **Configuration**: Update `launch.json` with the `l2sm-controller` service IP Address and the database IP Address. This file has been made to help launching the application locally.

3. **Debugging**: In Visual Studio Code, run the debug Kopf application. It will launch the app in a terminal, but it doesn't allow actual debugging tools such as custom breakpoints, as it's not a feature in kopf applications.

### L2SM-Switch

1. **Deployment**: Deploy `l2sm-switch` normally, ensuring to comment out `initContainers` in the YAML file. Remove the initial configuration script by using as input spec.container.args: ["sleep infinity"]

2. **Debugging**: For debugging, remove the initial configuration script by and use `exec -it` on the pods to achieve the desired configuration. Since it doesn’t run any background process, no specific image is needed, the current one implements custom commands that enable the current configuration, you can check in the script [L2S-M/src/switch/setup_switch.sh](../src/switch/setup_switch.sh) how the configuration is made.

