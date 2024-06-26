#!/usr/bin/python

from mininet.log import setLogLevel
from mininet.node import RemoteController, Switch, OVSSwitch
from mininet.link import Link, Intf
from mininet.util import quietRun
from mininet.log import setLogLevel
from mininet.cli import CLI
from mininet.topolib import TreeTopo
from mininet.net import Mininet
import os
import pty
import re
import signal
import select
from distutils.version import StrictVersion
from re import findall
from subprocess import Popen, PIPE
from sys import exit  # pylint: disable=redefined-builtin
from time import sleep

from mininet.log import info, error, warn, debug
from mininet.util import ( quietRun, errRun, errFail, moveIntf, isShellBuiltin,
                           numCores, retry, mountCgroups, BaseString, decode,
                           encode, getincrementaldecoder, Python3, which )
from mininet.moduledeps import moduleDeps, pathCheck, TUN
from mininet.link import Link, Intf, TCIntf, OVSIntf

class CustomSwitch( Switch ):
    "Open vSwitch switch. Depends on ovs-vsctl."

    def __init__( self, name, failMode='secure', datapath='kernel',
                  inband=False, protocols=None,
                  reconnectms=1000, stp=False, batch=False, **params ):
        """name: name for switch
           failMode: controller loss behavior (secure|standalone)
           datapath: userspace or kernel mode (kernel|user)
           inband: use in-band control (False)
           protocols: use specific OpenFlow version(s) (e.g. OpenFlow13)
                      Unspecified (or old OVS version) uses OVS default
           reconnectms: max reconnect timeout in ms (0/None for default)
           stp: enable STP (False, requires failMode=standalone)
           batch: enable batch startup (False)"""
        Switch.__init__( self, name, **params )
        self.failMode = failMode
        self.datapath = datapath
        self.inband = inband
        self.protocols = protocols
        self.reconnectms = reconnectms
        self.stp = stp
        self._uuids = []  # controller UUIDs
        self.batch = batch
        self.commands = []  # saved commands for batch startup

    @classmethod
    def setup( cls ):
        "Make sure Open vSwitch is installed and working"
        pathCheck( 'ovs-vsctl',
                   moduleName='Open vSwitch (openvswitch.org)')
        # This should no longer be needed, and it breaks
        # with OVS 1.7 which has renamed the kernel module:
        #  moduleDeps( subtract=OF_KMOD, add=OVS_KMOD )
        out, err, exitcode = errRun( 'ovs-vsctl -t 1 show' )
        if exitcode:
            error( out + err +
                   'ovs-vsctl exited with code %d\n' % exitcode +
                   '*** Error connecting to ovs-db with ovs-vsctl\n'
                   'Make sure that Open vSwitch is installed, '
                   'that ovsdb-server is running, and that\n'
                   '"ovs-vsctl show" works correctly.\n'
                   'You may wish to try '
                   '"service openvswitch-switch start".\n' )
            exit( 1 )
        version = quietRun( 'ovs-vsctl --version' )
        cls.OVSVersion = findall( r'\d+\.\d+', version )[ 0 ]

    @classmethod
    def isOldOVS( cls ):
        "Is OVS ersion < 1.10?"
        return ( StrictVersion( cls.OVSVersion ) <
                 StrictVersion( '1.10' ) )

    def dpctl( self, *args ):
        "Run ovs-ofctl command"
        return self.cmd( 'ovs-ofctl', args[ 0 ], self, *args[ 1: ] )

    def vsctl( self, *args, **kwargs ):
        "Run ovs-vsctl command (or queue for later execution)"
        if self.batch:
            cmd = ' '.join( str( arg ).strip() for arg in args )
            self.commands.append( cmd )
            return None
        else:
            return self.cmd( 'ovs-vsctl', *args, **kwargs )

    @staticmethod
    def TCReapply( intf ):
        """Unfortunately OVS and Mininet are fighting
           over tc queuing disciplines. As a quick hack/
           workaround, we clear OVS's and reapply our own."""
        if isinstance( intf, TCIntf ):
            intf.config( **intf.params )

    def attach( self, intf ):
        "Connect a data port"
        self.vsctl( 'add-port', self, intf )
        self.cmd( 'ifconfig', intf, 'up' )
        self.TCReapply( intf )

    def detach( self, intf ):
        "Disconnect a data port"
        self.vsctl( 'del-port', self, intf )

    def controllerUUIDs( self, update=False ):
        """Return ovsdb UUIDs for our controllers
           update: update cached value"""
        if not self._uuids or update:
            controllers = self.cmd( 'ovs-vsctl -- get Bridge', self,
                                    'Controller' ).strip()
            if controllers.startswith( '[' ) and controllers.endswith( ']' ):
                controllers = controllers[ 1 : -1 ]
                if controllers:
                    self._uuids = [ c.strip()
                                    for c in controllers.split( ',' ) ]
        return self._uuids

    def connected( self ):
        "Are we connected to at least one of our controllers?"
        for uuid in self.controllerUUIDs():
            if 'true' in self.vsctl( '-- get Controller',
                                     uuid, 'is_connected' ):
                return True
        return self.failMode == 'standalone'

    def intfOpts( self, intf ):
        "Return OVS interface options for intf"
        opts = ''
        if not self.isOldOVS():
            # ofport_request is not supported on old OVS
            opts += ' ofport_request=%s' % self.ports[ intf ]
            # Patch ports don't work well with old OVS
            if isinstance( intf, OVSIntf ):
                intf1, intf2 = intf.link.intf1, intf.link.intf2
                peer = intf1 if intf1 != intf else intf2
                opts += ' type=patch options:peer=%s' % peer
        return '' if not opts else ' -- set Interface %s' % intf + opts

    def bridgeOpts( self ):
        "Return OVS bridge options"
        opts = ( ' other_config:datapath-id=%s' % self.dpid +
                 ' fail_mode=%s' % self.failMode )
        if not self.inband:
            opts += ' other-config:disable-in-band=true'
        if self.datapath == 'user':
            opts += ' datapath_type=netdev'
        if self.protocols and not self.isOldOVS():
            opts += ' protocols=%s' % self.protocols
        if self.stp and self.failMode == 'standalone':
            opts += ' stp_enable=true'
        opts += ' other-config:dp-desc=%s' % self.name
        return opts

    def start( self, controllers ):
        "Start up a new OVS OpenFlow switch using ovs-vsctl"
        if self.inNamespace:
            raise Exception(
                'OVS kernel switch does not work in a namespace' )
        int( self.dpid, 16 )  # DPID must be a hex string
        # Command to add interfaces
        intfs = ''.join( ' -- add-port %s %s' % ( self, intf ) +
                         self.intfOpts( intf )
                         for intf in self.intfList()
                         if self.ports[ intf ] and not intf.IP() )
        # Command to create controller entries
        clist = [ ( self.name + c.name, '%s:%s:%d' %
                  ( c.protocol, c.IP(), c.port ) )
                  for c in controllers ]
        if self.listenPort:
            clist.append( ( self.name + '-listen',
                            'ptcp:%s' % self.listenPort ) )
        ccmd = '-- --id=@%s create Controller target=\\"%s\\"'
        if self.reconnectms:
            ccmd += ' max_backoff=%d' % self.reconnectms
        cargs = ' '.join( ccmd % ( name, target )
                          for name, target in clist )
        # Controller ID list
        cids = ','.join( '@%s' % name for name, _target in clist )
        # Try to delete any existing bridges with the same name
        if not self.isOldOVS():
            cargs += ' -- --if-exists del-br %s' % self
        # One ovs-vsctl command to rule them all!
        
        # If necessary, restore TC config overwritten by OVS
        if not self.batch:
            for intf in self.intfList():
                self.TCReapply( intf )

    # This should be ~ int( quietRun( 'getconf ARG_MAX' ) ),
    # but the real limit seems to be much lower
    argmax = 128000

    @classmethod
    def batchStartup( cls, switches, run=errRun ):
        """Batch startup for OVS
           switches: switches to start up
           run: function to run commands (errRun)"""
        info( '...' )
        cmds = 'ovs-vsctl'
        for switch in switches:
            if switch.isOldOVS():
                # Ideally we'd optimize this also
                run( 'ovs-vsctl del-br %s' % switch )
            for cmd in switch.commands:
                cmd = cmd.strip()
                # Don't exceed ARG_MAX
                if len( cmds ) + len( cmd ) >= cls.argmax:
                    run( cmds, shell=True )
                    cmds = 'ovs-vsctl'
                cmds += ' ' + cmd
                switch.cmds = []
                switch.batch = False
        if cmds:
            run( cmds, shell=True )
        # Reapply link config if necessary...
        for switch in switches:
            for intf in switch.intfs.values():
                if isinstance( intf, TCIntf ):
                    intf.config( **intf.params )
        return switches

    def stop( self, deleteIntfs=True ):
        """Terminate OVS switch.
           deleteIntfs: delete interfaces? (True)"""
        self.cmd( 'ovs-vsctl del-br', self )
        if self.datapath == 'user':
            self.cmd( 'ip link del', self )
        super( OVSSwitch, self ).stop( deleteIntfs )

    @classmethod
    def batchShutdown( cls, switches, run=errRun ):
        "Shut down a list of OVS switches"
        delcmd = 'del-br %s'
        if switches and not switches[ 0 ].isOldOVS():
            delcmd = '--if-exists ' + delcmd
        # First, delete them all from ovsdb
        run( 'ovs-vsctl ' +
             ' -- '.join( delcmd % s for s in switches ) )
        # Next, shut down all of the processes
        pids = ' '.join( str( switch.pid ) for switch in switches )
        run( 'kill -HUP ' + pids )
        for switch in switches:
            switch.terminate()
        return switches

def configure_switch(switch, controller_ip, controller_port, neighbors, switch_ips):

    # Add a bridge
    switch.cmdPrint("ovs-vsctl add-br brtun-" +switch.name)

    switch.cmdPrint("ip link set brtun-" + switch.name + " up")
    switch.cmdPrint("ovs-vsctl set bridge brtun-" + switch.name + " protocols=OpenFlow13")

    target = "tcp:{}:{}".format(controller_ip, controller_port)
    switch.cmdPrint("ovs-vsctl set-controller brtun-" + switch.name + " " + target)
    # Attach Ethernet interface


    # Set the controller
   

    for neighbor in neighbors:
        switch.cmdPrint("ovs-vsctl add-port brtun-" + switch.name + " vxlan{}-{} -- set interface vxlan{}-{} type=vxlan "
                        "options:local_ip={} options:remote_ip={} options:key=flow".format(
                        switch.name, neighbor, switch.name, neighbor, switch_ips[switch.name], switch_ips[neighbor]))

def demo2():
    CONTROLLER_IP = "localhost"
    CONTROLLER_PORT = 6633

    net = Mininet(controller=RemoteController)

    c0 = net.addController('c0', controller=RemoteController, ip=CONTROLLER_IP, port=CONTROLLER_PORT)

    s1 = net.addSwitch('s1',cls=OVSSwitch)
    s2 = net.addSwitch('s2',cls=OVSSwitch)
    h1 = net.addHost('h1', ip="10.0.0.1")

    net.addLink(s1, h1)

    
    
    h2 = net.addHost('h2', ip="10.0.0.2")
    net.addLink(s2, h2)
    net.addLink(s1,s2)
    net.start()


    CLI(net)
    net.stop()

def demo():
    CONTROLLER_IP = "localhost"
    CONTROLLER_PORT = 6633

    net = Mininet(controller=RemoteController)

    c0 = net.addController('c0', controller=RemoteController, ip=CONTROLLER_IP, port=CONTROLLER_PORT)

    s1 = net.addSwitch('s1',cls=CustomSwitch)
    s2 = net.addSwitch('s2',cls=CustomSwitch)
    s3 = net.addSwitch('s3',cls=CustomSwitch)
    s4 = net.addSwitch('s4',cls=CustomSwitch)
    s5 = net.addSwitch('s5',cls=CustomSwitch)

    h1 = net.addHost('h1', ip="10.0.0.1")
    net.addLink(s1, h1)

    
    h2 = net.addHost('h2', ip="10.0.0.2")
    net.addLink(s5, h2)

    switch_ips = {
        "s1": "192.168.0.2",
        "s2": "192.168.0.3",
        "s3": "192.168.0.4",
        "s4": "192.168.0.5",
        "s5": "192.168.0.6"
    }
    vxlan_config = {
        "s1": ["s2", "s3"],
        "s2": ["s1", "s3","s4"],
        "s3": ["s1", "s2","s4"],
        "s4": ["s2","s3","s5"],
        "s5": ["s4"]
    }
    # Add links based on VXLAN configuration
    for switch, neighbors in vxlan_config.items():
        for neighbor in neighbors:
            net.addLink(
                net.get(switch),
                net.get(neighbor),
                params1={'ip': switch_ips[switch] + "/24"},
                params2={'ip': switch_ips[neighbor] + "/24"}
            )
    net.start()

   
  

    # Configure switches
    configure_switch(s1, CONTROLLER_IP, CONTROLLER_PORT, vxlan_config.get("s1", []), switch_ips)
    configure_switch(s2, CONTROLLER_IP, CONTROLLER_PORT, vxlan_config.get("s2", []), switch_ips)
    configure_switch(s3, CONTROLLER_IP, CONTROLLER_PORT, vxlan_config.get("s3", []), switch_ips)
    configure_switch(s4, CONTROLLER_IP, CONTROLLER_PORT, vxlan_config.get("s4", []), switch_ips)
    configure_switch(s5, CONTROLLER_IP, CONTROLLER_PORT, vxlan_config.get("s5", []), switch_ips)
    s1.cmdPrint("ovs-vsctl add-port brtun-s1 s1-eth1")
    s5.cmdPrint("ovs-vsctl add-port brtun-s5 s5-eth1")

    CLI(net)
    net.stop()

if __name__ == '__main__':
    setLogLevel('info')
    demo()


