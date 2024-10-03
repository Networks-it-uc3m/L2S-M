# Multi domain L2S-M
Welcome to the official repository of L2S-M, a **Kubernetes operator** that enables virtual networking in K8s clusters.

Link-Layer Secure connectivity for Microservice platforms (L2S-M) is a K8s networking solution that complements the CNI plugin approach of K8s to create and manage virtual networks in K8s clusters. These virtual networks allow workloads (pods) to have isolated link-layer connectivity with other pods in a K8s cluster, regardless of the k8s node where they are actually deployed. L2S-M enables the creation/deletion of virtual networks on-demand, as well as attaching/detaching pods to that networks. The solution is seamlessly integrated within the K8s environment, through a K8s operator:

![alt text](./assets/v1_architecture.png?raw=true)

L2S-M provides its intended functionalities using a programmable data-plane based on Software Defined Networking (SDN), which in turn provides a high degree of flexibility to dynamically incorporate new application and/or network configurations into K8s clusters. Moreover, L2S-M has been designed to flexibly accommodate various deployment options, ranging from small K8s clusters to those with a high number of distributed nodes. 

The main K8s interface of pods remains intact (provided by a CNI plugin), retaining the compatibility with all the standard K8s elements (e.g., services, connectivity through the main interface, etc.). Moreover, the solution has the potential to be used for inter-cluster communications to support scenarios where network functions are spread through multiple distributed infrastructures (this is still a work in progress).  

The figure outlines the design of L2S-M. See [how L2S-M works](./additional-info/) to read further details on the L2S-M solution.

If you want to learn how to install L2S-M in your cluster, see the [installation guide](./deployments) of this repository to start its installation.


Did you already install the operator and  you cannot wait to start building your own virtual networks in your K8s cluster? Check out our [general usage guide](./additional-info/general-use.md)! If you're more interested in seeing a simple working example, you can start out with the [ping pong example](./examples/ping-pong/).

If you want more information about the original idea of L2S-M and its initial design, you can check our latest publication in the [IEEE Network journal](https://ieeexplore.ieee.org/document/9740640):

- L. F. Gonzalez, I. Vidal, F. Valera and D. R. Lopez, "Link Layer Connectivity as a Service for Ad-Hoc Microservice Platforms," in IEEE Network, vol. 36, no. 1, pp. 10-17, January/February 2022, doi: 10.1109/MNET.001.2100363.

Did you like L2S-M and want to use it in your K8s infrastructure or project? Please, feel free to do so, and don't forget to cite us! 

### Demo video

This [video](https://youtube.com/watch?v=Oj2gzm-YxYE&si=bV9eN77wTlXQZY3Y) exemplifies the process to create virtual networks in Kubernetes using the L2S-M open-source software. More concretely, it shows how L2S is used to create a simple content distribution network on a Kubernetes cluster.

### Inter-cluster communications

One of the most interesting features L2S-M has is that it enables communications among workloads deployed on differente Kubernetes clusters. You can perform the creation and deletion of virtual link-layer networks to connect application workloads running in different virtualization domains. This way, it supports inter-domain link-layer communications among remote workloads.  

The solution can work jointly with L2S-M or be used standalone through the [Multus CNI](https://github.com/k8snetworkplumbingwg/multus-cni). Details can be checked [here](./additional-info/inter-cluster.md). Even though the inter-cluster solution is meant to be used via [the multi-domain client](http://github.com/Networks-it-uc3m/l2sm-md), we provide examples of how can you manually set up an entire inter-cluster virtual overlay network in the [inter cluster setup guide](./examples/inter-cluster-setup/).  If you have your infrastructure ready, you can go ahead to the [inter cluster networks example](./examples/inter-cluster-network)!

### Additional information about L2S-M
In the [following section](./additional-info) of the repository, you can find a series of documents and slides that provide additional information about L2S-M, including presentations where our solution has been showcased to the public in various events.

L2S-M has been presented in the following events:

* ETSI Network Operator Council Meeting #195 (June 2023)

* [ETSI Open Source MANO (OSM) Proof-of-Concept: OSM PoC 14 Leveraging OSM virtual networking in Kubernetes clusters](https://osm.etsi.org/wikipub/index.php/OSM_PoC_14_Leveraging_OSM_virtual_networking_in_Kubernetes_clusters)

* Demo in the [ETSI OSM MR14 Ecosystem Day](https://osm.etsi.org/wikipub/index.php/OSM-MR14_Ecosystem_Day) (March 2023)

* [FIHY Summer Camp (20/04/2023)](https://drcn2023.upc.edu/FISHYSummerCamp.html). In this summer camp, we described the utilization of L2S-M in next-generation secured communication scenarios, which are covered in the H2020 FIHSY and Labyrinth projects (March 2023).

* [Open Source Mano (OSM) #13 plenary meeting (June 2022)](https://github.com/Networks-it-uc3m/L2S-M/blob/main/additional%20info/OSM%2313%20Plenary%20Meeting.pdf): In this meeting, L2S-M was presented as a solution to enable virtual networking to deploy Cloud Network Functions (CNFs) in K8s clusters. Moreover, the potential use of L2S-M to become the basis for a feature to be introduced in OSM's code was discussed as well.

### Use cases where L2S-M has been used:
The following publications and references showcase various use cases where L2S-M was used as the basis for providing secure communications within one, or multiple, Kubernetes clusters:

* **Fishy Reference Framework:** Complex experimentation platform developed to support the H2020 FISHY project use cases & its main functionalities. L2S-M was utilised as a component that provided secured communications between all FISHY functionalities and elements, either virtualised or physical. 
    - I. Vidal et al., 'A Multi-domain Testbed for Collaborative Research on the IoT-Edge-Cloud Continuum,' in 2023 20th Annual IEEE International Conference on Sensing, Communication, and Networking (SECON), 2023, pp. 394--395. *DOI: 10.1109/SECON58729.2023.10287436*.

* **Smart Campus use case:** This use case was centred around the deployment of a "Content Delivery Network" (CDN) for distributing audiovisual content in a Univeristy environment composed of multiple edge and cloud K8s clusters. L2S-M provided inter-cluster and intra-cluster networking to securely connect all its elements.
    - L. F. Gonzalez, I. Vidal, F. Valera, R. Martin, and D. Artalejo, 'A Link-Layer Virtual Networking Solution for Cloud-Native Network Function Virtualisation Ecosystems: L2S-M,' Future Internet, vol. 15, no. 8, 2023.* *DOI: 10.3390/fi15080274*. [Online]. Available [here.](https://www.mdpi.com/1999-5903/15/8/274)

* **Secure Federated Learning use case:** This use case, done in collaboration with the University of Bristol, was centred around the implementation of a secured federated learning infrastructure. L2S-M enabled the isolated and secured communication between the FL clients and the server.
    - J. M. Parra-Ullauri, L. F. Gonzalez, A. Bravalheri, R. Hussain, X. Vasilakos, I. Vidal, F. Valera, R. Nejabati, and D. Simeonidou, 'Privacy Preservation in Kubernetes-based Federated Learning: A Networking Approach,' in IEEE INFOCOM2023 - IEEE Conference on Computer CommunicationsWorkshops (INFOCOMWKSHPS), 2023, pp. 1--7. *DOI: 10.1109/INFOCOMWKSHPS57453.2023.10225925*.


### How to reach us

Do you have any doubts about L2S-M or its installation? Do you want to provide feedback about the solution? Please, do not hesitate to contact us out through e-mail!

- Alex T. de Cock Buning: 100383348@alumnos.uc3m.es (Universidad Carlos III de Madrid)
- Luis F. Gonzalez: luisfgon@it.uc3m.es (Universidad Carlos III de Madrid)
- Ivan Vidal : ividal@it.uc3m.es (Universidad Carlos III de Madrid)
- Francisco Valera: fvalera@it.uc3m.es (Universidad Carlos III de Madrid)
- Diego R. Lopez: diego.r.lopez@telefonica.com (Telefónica I+D)


### Acknowledgement
The work in this open-source project has partially been supported by the European Horizon NEMO project (grant agreement 101070118), the European Horizon CODECO project (grant agreement 101092696), and by the national 6GINSPIRE project (PID2022-137329OB-C429). 

#### Other projects where L2S-M has been used
- H2020 FISHY Project: https://fishy-project.eu (grant agreement 952644) 
- True5G Project: (PID2019-108713RB-C52 / AEI / 10.13039/501100011033)
- H2020 Labyrinth project: https://labyrinth2020.eu/ (grant agreement 861696).
