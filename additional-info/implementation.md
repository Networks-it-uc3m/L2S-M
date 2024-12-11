# L2S-M Development

This component is essentially a set of Custom Resource Definitions (CRDs) accompanied by a controller and a manager. It's designed to manage the overlays and virtual networks that L2S-M uses between pods within a K8s cluster. 

## Getting Started

### Prerequisites
- go version v1.21.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`, inside the Makefile:**

```sh
make docker-build docker-push
```

**NOTE:** The image ought to be published in the personal registry you specified. 
And it is required to have access to pull the image from the working environment. 
Make sure you have the proper permission to the registry if the above commands don’t work.

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy 
```



**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```


> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin 
privileges or be logged in as admin.


### To Run locally the solution and make your custom changes

If you are interested in running the solution locally, feel free to make your own branch and start developing! Any feedback is welcome as well.

We provide the following commands to run the application locally

1. **Create a test cluster**

Firstly you can create a cluster with all the dependencies installed already, using Kind:

```sh
make create-cluster
```
2. **Then you can launch the custom dev tools:**
```sh
make deploy-dev
```

With this command you will install all CRDs, SDN Controller and Webhook, with a local configuration. Please modify the DEV_IP env variable to your local one (localhost won't work), to run there the Webhook Server. Localhost won't work because it must be accesible from the docker containers created by kind.
3. **Run the application with the changes:**
Once the SDN Controller is running, which may take from 1 to 5 mins (you can verify using kubectl get pods and checking that controller is in state running), you can run the application like this:
```sh
make run
```

And once you've finished:
**Delete the resources:**

```sh
make undeploy-dev
```
**Or delete the cluster entirely:**
```sh
make delete-cluster
```


## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the deployment directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/L2S-M/<tag or branch>/deployments/l2sm-deployment.yaml
```