#!/bin/bash

ovsdb-server --remote=punix:/var/run/openvswitch/db.sock --remote=db:Open_vSwitch,Open_vSwitch,manager_options --pidfile=/var/run/openvswitch/ovsdb-server.pid --detach 

ovs-vsctl --db=unix:/var/run/openvswitch/db.sock --no-wait init 

ovs-vswitchd --pidfile=/var/run/openvswitch/ovs-vswitchd.pid --detach 

l2sm-init --n_veths=$NVETHS --controller_ip=$CONTROLLERIP --switch_name=$NODENAME

sleep 20

l2sm-vxlans --node_name=$NODENAME /etc/l2sm/topology.json

sleep infinity
