#!/bin/bash

ovsdb-server --remote=punix:/var/run/openvswitch/db.sock --remote=db:Open_vSwitch,Open_vSwitch,manager_options --pidfile=/var/run/openvswitch/ovsdb-server.pid --detach 

ovs-vsctl --db=unix:/var/run/openvswitch/db.sock --no-wait init 

ovs-vswitchd --pidfile=/var/run/openvswitch/ovs-vswitchd.pid --detach 

ned-server --node_name=$NODENAME --controller_ip=$CONTROLLERIP --provider_name=$PROVIDERNAME /etc/l2sm/topology.json