<!---
 Copyright 2024  Charles III University of Madrid
 
 Licensed under the Apache License, Version 2.0 (the "License"); you may not
 use this file except in compliance with the License.  You may obtain a copy
 of the License at
 
   http://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
 License for the specific language governing permissions and limitations under
 the License.
 
 SPDX-License-Identifier: Apache-2.0
-->

# L2S-M examples

This section of L2S-M documentation provides examples that you can use in order to learn how to create virtual networks and attach pods to them. 

Feel free to make use of this tool in any scenario that it could be used in. Right now two examples are shown.

Firstly, there's [the ping-pong example](./ping-pong/). This is the most basic example, a virtual network that connects two pods through a L2S-M virtual network, and checking the connectivity using the ping command.

Secondly, there's the [cdn example](./cdn). In this example, there are two networks that isolate a content-server, storing a video, from the rest of the Cluster. It will only accessible by a cdn-server, using a router pod between these two other pods. This way, if the Cluster or cdn-server are under any safety risks, or custom firewall restrictions are applied through a Pod, there's more control in accessing the Pod. Additionally, this section has an L2S-M live demo showcasing this scenario.
