#!/bin/bash
ip link add dev vpod1 mtu 1450 type veth peer name vhost1 mtu 1450
ip link add dev vpod2 mtu 1450 type veth peer name vhost2 mtu 1450
ip link add dev vpod3 mtu 1450 type veth peer name vhost3 mtu 1450
ip link add dev vpod4 mtu 1450 type veth peer name vhost4 mtu 1450
ip link add dev vpod5 mtu 1450 type veth peer name vhost5 mtu 1450
ip link add dev vpod6 mtu 1450 type veth peer name vhost6 mtu 1450
ip link add dev vpod7 mtu 1450 type veth peer name vhost7 mtu 1450
ip link add dev vpod8 mtu 1450 type veth peer name vhost8 mtu 1450
ip link add dev vpod9 mtu 1450 type veth peer name vhost9 mtu 1450
ip link add dev vpod10 mtu 1450 type veth peer name vhost10 mtu 1450
