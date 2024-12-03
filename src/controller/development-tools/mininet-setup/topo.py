"""Custom topology example

Two directly connected switches plus a host for each switch:

   host --- switch --- switch --- host

Adding the 'topos' dict with a key/value pair to generate our newly defined
topology enables one to pass in '--topo=mytopo' from the command line.
"""

from mininet.topo import Topo
from mininet.node import OVSSwitch
class MyTopo( Topo ):
    "Simple topology example."

    def build( self ):
      "Create custom topo."

      # Add hosts and switches
      leftHost = self.addHost('h1')
      rightHost = self.addHost('h2')

      # Use OVSSwitch instead of default switch
      firstSwitch = self.addSwitch('s1')
      secondSwitch = self.addSwitch('s2')
      thirdSwitch = self.addSwitch('s3')
      fourthSwitch = self.addSwitch('s4')
      fifthSwitch = self.addSwitch('s5')

      # Add links with VXLAN tunnels
      self.addLink(leftHost, firstSwitch)
      self.addLink(fifthSwitch, rightHost)

      
      s1_endpoint_address = "192.168.59.100"
      s2_endpoint_address = "192.168.59.101"
      firstSwitch.cmdPrint("ovs-vsctl add-port s1 s1-eth2 -- set interface s1-eth2 type=vxlan options:local_ip=" + s1_endpoint_address + " options:remote_ip=" + s2_endpoint_address + " options:key=flow")
      secondSwitch.cmdPrint("ovs-vsctl add-port s2 s2-eth2 -- set interface s2-eth2 type=vxlan options:local_ip=" + s2_endpoint_address + " options:remote_ip=" + s1_endpoint_address + " options:key=flow")

      
      # self.addLink(fifthSwitch, rightHost)

      # self.addLink(firstSwitch, secondSwitch, intfName1='s1-eth2', intfName2='s2-eth2')
      # self.addLink(firstSwitch, thirdSwitch, intfName1='s1-eth3', intfName2='s3-eth2')
      # self.addLink(secondSwitch, thirdSwitch, intfName1='s2-eth3', intfName2='s3-eth3')
      # self.addLink(secondSwitch, fourthSwitch, intfName1='s2-eth4', intfName2='s4-eth2')
      # self.addLink(thirdSwitch, fourthSwitch, intfName1='s3-eth4', intfName2='s4-eth3')
      # self.addLink(fourthSwitch, fifthSwitch, intfName1='s4-eth4', intfName2='s5-eth2')


topos = { 'mytopo': ( lambda: MyTopo() ) }
