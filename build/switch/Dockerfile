FROM golang:1.21 AS build

WORKDIR /usr/src/l2sm-switch

COPY ./src/switch/ ./build/switch/build-go.sh ./

RUN chmod +x ./build-go.sh  && ./build-go.sh

FROM ubuntu:jammy 

WORKDIR /usr/local/bin

COPY ./src/switch/vswitch.ovsschema /tmp/

COPY --from=build /usr/local/bin/ .

RUN apt-get update && \
  apt-get install -y net-tools iproute2 netcat-openbsd dnsutils curl iputils-ping iptables nmap tcpdump openvswitch-switch && \
  mkdir /var/run/openvswitch && mkdir -p /etc/openvswitch && ovsdb-tool create /etc/openvswitch/conf.db /tmp/vswitch.ovsschema 

COPY ./src/switch/setup_switch.sh .

RUN chmod +x ./setup_switch.sh && \
    mkdir /etc/l2sm/

CMD [ "./setup_switch.sh" ]