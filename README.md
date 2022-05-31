# L2S-M 
Welcome to the official repository of L2S-M, a **Kubernetes operator** that enables virtual networking in K8s clusters.

Link-Layer Secure connectivity for Microservice platforms (L2S-M) is a K8s networking solution that complements the CNI plugin approach of K8s to create and manage virtual networks in K8s clusters. These virtual networks allow workloads (pods) to have isolated link-layer connectivity with other pods in a K8s cluster, regardless of the k8s node where they are actually deployed. L2S-M enables the creation/deletion of virtual networks on-demand, as well as attaching/detaching pods to that networks. The solution is seamlessly integrated within the K8s environment, through a K8s operator.

![alt text](https://github.com/Networks-it-uc3m/L2S-M/blob/main/v1_architecture.png?raw=true)

L2S-M provides its intended functionalities using a programmable data-plane based on Software Defined Networking (SDN), which in turn provides a high degree of flexibility to dynamically incorporate new application and/or network configurations into K8s clusters. Moreover, L2S-M has been designed to flexibly accommodate various deployment options, ranging from small K8s clusters to those with a high number of distributed nodes. 

The main K8s interface of pods remains intact (provided by a CNI plugin), retaining the compatibility with all the standard K8s elements (e.g., services, connectivity through the main interface, etc.). Moreover, the solution has the potential to be used for inter-cluster communications to support scenarios where network functions are spread through multiple distributed infrastructures (this is still a work in progress).  

The figure below outlines the design of L2S-M. See [how L2S-M works](https://github.com/Networks-it-uc3m/L2S-M/tree/main/K8s) to read further details on the L2S-M solution.

If you want to learn how to install L2S-M in your cluster, see the [installation guide](https://github.com/Networks-it-uc3m/L2S-M/tree/main/operator) of this repository to start its installation.

Did you already install the operator and  you cannot wait to start building your own virtual networks in your K8s cluster? Check out our [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors) example!

If you want more information about the original idea of L2S-M and its initial design, you can check our latest publication in the [IEEE Network journal](https://ieeexplore.ieee.org/document/9740640):

- L. F. Gonzalez, I. Vidal, F. Valera and D. R. Lopez, "Link Layer Connectivity as a Service for Ad-Hoc Microservice Platforms," in IEEE Network, vol. 36, no. 1, pp. 10-17, January/February 2022, doi: 10.1109/MNET.001.2100363.

Did you like L2S-M and want to use it in your K8s infrastructure or project? Please, feel free to do so, and don't forget to cite us! 

### Project where L2S-M is being used:
- H2020 FISHY Project: https://fishy-project.eu (H2020-MG-2019-TwoStages-861696) 
- True5G Project: (PID2019-108713RB-C52 / AEI / 10.13039/501100011033)

### How to reach us

Do you have any doubts about L2S-M or its installation? Do you want to provide feedback about the solution? Please, do not hesitate to contact us out through e-mail!
- Luis F. Gonzalez: luisfgon@it.uc3m.es (Universidad Carlos III de Madrid)
- Ivan Vidal : ividal@it.uc3m.es (Universidad Carlos III de Madrid)
- Francisco Valera: fvalera@it.uc3m.es (Universidad Carlos III de Madrid
- Diego R. Lopez: diego.r.lopez@telefonica.com (Telefónica I+D)
