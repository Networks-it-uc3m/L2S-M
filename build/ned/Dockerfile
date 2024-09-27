FROM golang:1.21 AS build

WORKDIR /usr/src/l2sm-ned

COPY ./src/switch/  ./

RUN go build -v -o /usr/local/bin/ned-server ./cmd/ned-server

FROM ubuntu:latest 

WORKDIR /usr/local/bin

COPY ./src/switch/vswitch.ovsschema /tmp/

COPY --from=build /usr/local/bin/ .

RUN apt-get update && \
  apt-get install -y net-tools iproute2 netcat-openbsd dnsutils curl iputils-ping iptables nmap tcpdump openvswitch-switch && \
  mkdir /var/run/openvswitch && mkdir -p /etc/openvswitch && ovsdb-tool create /etc/openvswitch/conf.db /tmp/vswitch.ovsschema 

COPY ./src/switch/setup_ned.sh .

RUN chmod +x ./setup_ned.sh && \
    mkdir /etc/l2sm/

CMD [ "./setup_ned.sh" ]