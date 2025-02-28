# L2S-M examples

This section of L2S-M documentation provides examples that you can use in order to learn how to create virtual networks and attach pods to them. 

You can use as an example the 3 node cluster presented in [the quickstart section](./quickstart/).

Feel free to make use of this tool in any scenario that it could be used in. 

Right now an [overlay setup](./overlay-setup/) and an [interdomain setup](./inter-cluster-setup/) example are presented, using as a base the quickstart example with an addition of three main examples of l2networks:

Firstly, there's [the ping-pong example](./ping-pong/). This is the most basic example, a virtual network that connects two pods through a L2S-M virtual network, and checking the connectivity using the ping command.

Secondly, there's the [cdn example](./cdn). In this example, there are two networks that isolate a content-server, storing a video, from the rest of the Cluster. It will only accessible by a cdn-server, using a router pod between these two other pods. This way, if the Cluster or cdn-server are under any safety risks, or custom firewall restrictions are applied through a Pod, there's more control in accessing the Pod. Additionally, this section has an L2S-M live demo showcasing this scenario.

Finally, we have a [vlink example](./vlink/). Showcasing the SDN potetial to define specific paths for the traffic.
