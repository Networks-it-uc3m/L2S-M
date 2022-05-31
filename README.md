# L2S-M 
Welcome to the official repository of L2S-M, a Kubernetes operator that enables virtual networking in K8s clusters.

Link-Layer Secure connectivity for Microservice platforms (L2S-M) is a K8s networking solution that complements CNI Plugin solutions in order to create and manage virtual networks in K8s clusters. These virtual networks allow microservices (pods) to have isolated link-layer connectivity with other pods deployed in a K8s cluster taht are attached to the same virtual network, regardless of their physical location. These virtual networks can be created on-demand, and its management (as well as attaching/detaching pods to these networks) is completely integrated inside the K8s environment, thanks to the L2S-M K8s operator.

The way that L2S-M achieves this operation is through the use of a programmable data-plane managed by SDN, which in turn provides a high degree of flexibility to dynamically incorporate new application and/or network configurations into a cluster that uses L2S-M. Moreover, L2S-M's design is able to flexibly accomodate various deployment options for Network Services, ranging from small clusters to those with a high number of distributed nodes. 

![alt text](https://github.com/Networks-it-uc3m/L2S-M/blob/main/v1_architecture.png?raw=true)

The main K8s interface of pods remains intact, retaining the compatibility with all the standard K8s elements (e.g., services, connectivity through the main interface). Moreover, this solution has the potential to be used for inter-cluster communications to support scenarios where network functions are spread through mutliple distributed infrastructures (work in progress).  

Further details about the architecture of L2S-M can be seen in the following [documentation](https://github.com/Networks-it-uc3m/L2S-M/tree/main/K8s).

If you want to learn how to install L2S-M in your cluster, see the [installation guide](https://github.com/Networks-it-uc3m/L2S-M/tree/main/operator) of this repository to start its installation.

If you want more information about the original idea of L2S-M and its initial design, you can check our latest publication in the [IEEE Network journal](https://ieeexplore.ieee.org/document/9740640):

- L. F. Gonzalez, I. Vidal, F. Valera and D. R. Lopez, "Link Layer Connectivity as a Service for Ad-Hoc Microservice Platforms," in IEEE Network, vol. 36, no. 1, pp. 10-17, January/February 2022, doi: 10.1109/MNET.001.2100363.

Did you already install the operator and  you cannot wait to start building your own virtual networks in your K8s cluster? Check out our [ping-pong](https://github.com/Networks-it-uc3m/L2S-M/tree/main/descriptors) example!

Did you like L2S-M and want to use it in your K8s infrastructure or project? Please, feel free to do so, and don't forget to cite us! 

### Project where L2S-M is being used:
- H2020 FISHY Project: https://fishy-project.eu (H2020-MG-2019-TwoStages-861696) 
- True5G Project: (PID2019-108713RB-C52 / AEI / 10.13039/501100011033)

### How to reach us

Do you have any doubts about L2S-M or its instalaltion? Do you want to provide feedback about the solution? Please, do not hesitate to contact us out through e-mail!
- Luis F. Gonzalez: luisfgon@it.uc3m.es (Universidad Carlos III de Madrid)
- Ivan Vidal : ividal@it.uc3m.es (Universidad Carlos III de Madrid)
- Francisco Valera: fvalera@it.uc3m.es (Universidad Carlos III de Madrid
- Diego R. Lopez: diego.r.lopez@telefonica.com (Telefónica I+D)
