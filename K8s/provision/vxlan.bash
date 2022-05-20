#!/bin/bash
ip link add vxlan1 type vxlan id 1961 dev $1 dstport 4789
ip link add vxlan2 type vxlan id 1962 dev $1 dstport 4789
ip link add vxlan3 type vxlan id 1963 dev $1 dstport 4789
ip link add vxlan4 type vxlan id 1964 dev $1 dstport 4789
ip link add vxlan5 type vxlan id 1965 dev $1 dstport 4789
ip link add vxlan6 type vxlan id 1966 dev $1 dstport 4789
ip link add vxlan7 type vxlan id 1967 dev $1 dstport 4789
ip link add vxlan8 type vxlan id 1968 dev $1 dstport 4789
ip link add vxlan9 type vxlan id 1969 dev $1 dstport 4789
ip link add vxlan10 type vxlan id 1970 dev $1 dstport 4789
