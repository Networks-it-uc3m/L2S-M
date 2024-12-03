#!/usr/bin/python

"""
vxlan_mininet.py
mininet1   mininet2
--------   --------
|      |   |      |
|  s1=========s2  |
|  |   |   |   |  |
|  h1  |   |  h2  |
|      |   |      |
--------   --------
=== : cross-link
| : link
Testing enviroment (cat /etc/hosts) :
192.168.59.100 mininet1
192.168.59.101 mininet2
"""

# from mininet.examples.cluster import MininetCluster
from mininet.log import setLogLevel
from mininet.node import Controller, RemoteController
from mininet.link import Link, Intf
from mininet.util import quietRun, errRun
from mininet.log import setLogLevel
from mininet.cli import CLI
from mininet.topolib import TreeTopo
from mininet.net import Mininet



def demo():
    CONTROLLER_IP="localhost"
    CONTROLLER_PORT=6633

    # Tunneling options: ssh (default), vxlan, gre
    net = Mininet( controller=RemoteController)

    c0 = net.addController( 'c0', controller=RemoteController, ip=CONTROLLER_IP, port=CONTROLLER_PORT)

    net.addController(c0)
    # In mininet1
    s1 = net.addSwitch('s1', ip="192.168.0.2")
    h1 = net.addHost('h1', ip="10.0.0.1")
    net.addLink(s1, h1)

    # In mininet2
    s2 = net.addSwitch('s2', ip="192.168.0.2")
    h2 = net.addHost('h2', ip="10.0.0.2")
    net.addLink(s2, h2)

    net.start()
    # Cross-link between mininet1 and mininet2
    s1_endpoint_address = "192.168.59.100"
    s2_endpoint_address = "192.168.59.101"
    s1.cmdPrint("ovs-vsctl add-br brtun")

    s1.cmdPrint("ovs-vsctl add-port brtun vxlan1 -- set interface vxlan1 type=vxlan options:local_ip=" + s1_endpoint_address + " options:remote_ip=" + s2_endpoint_address + " options:key=flow")
    s2.cmdPrint("ovs-vsctl add-port s2 s2-eth2 -- set interface s2-eth2 type=vxlan options:local_ip=" + s2_endpoint_address + " options:remote_ip=" + s1_endpoint_address + " options:key=flow")


    CLI( net )
    net.stop()

if __name__ == '__main__':
    setLogLevel( 'info' )
    demo()