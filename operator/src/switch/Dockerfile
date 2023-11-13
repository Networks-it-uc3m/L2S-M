FROM golang:1.20 AS build

WORKDIR /usr/src/bin

COPY ./main.go ./go.mod ./

RUN go build -v -o /usr/local/bin/l2sm-br ./... 

FROM ubuntu:latest 

WORKDIR /usr/local/bin

COPY --from=build /usr/local/bin/ .

COPY ./vswitch.ovsschema /tmp/

RUN apt-get update && \
  apt-get install -y net-tools iproute2 netcat-openbsd dnsutils curl iputils-ping iptables nmap tcpdump openvswitch-switch && \
  mkdir /var/run/openvswitch && mkdir -p /etc/openvswitch && ovsdb-tool create /etc/openvswitch/conf.db /tmp/vswitch.ovsschema 

COPY ./setup_switch.sh .

RUN chmod +x ./setup_switch.sh && \
    mkdir /etc/l2sm/



CMD [ "./setup_switch.sh" ]