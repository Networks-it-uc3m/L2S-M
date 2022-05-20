#!/bin/bash
ip link add vxlan1 type vxlan id 1969 dev enp1s0 dstport 4789
ip link set vxlan1 up
#bridge fdb append to 00:00:00:00:00:00 dst 163.117.140.237 dev vxlan1
ip link add dev vpod1 mtu 1450 type veth peer name vhost1 mtu 1450
ip link add dev vpod2 mtu 1450 type veth peer name vhost2 mtu 1450
ip link add dev vpod3 mtu 1450 type veth peer name vhost3 mtu 1450
ip link add dev vpod4 mtu 1450 type veth peer name vhost4 mtu 1450
