FROM ubuntu:20.04

COPY ./vswitch.ovsschema /tmp/

RUN apt-get update && \
  apt-get install -y net-tools iproute2 netcat dnsutils curl iputils-ping iptables nmap tcpdump openvswitch-switch && \
  mkdir /var/run/openvswitch && mkdir -p /etc/openvswitch && ovsdb-tool create /etc/openvswitch/conf.db /tmp/vswitch.ovsschema 



  