#!/bin/bash

ovsdb-server --remote=punix:/var/run/openvswitch/db.sock --remote=db:Open_vSwitch,Open_vSwitch,manager_options --pidfile=/var/run/openvswitch/ovsdb-server.pid --detach 

ovs-vsctl --db=unix:/var/run/openvswitch/db.sock --no-wait init 

ovs-vswitchd --pidfile=/var/run/openvswitch/ovs-vswitchd.pid --detach 

l2sm-br --n_vpods=$NVPODS --node_name=$NODENAME /etc/l2sm/switchConfig.json
