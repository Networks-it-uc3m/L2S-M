FROM golang:1.20

COPY ./vswitch.ovsschema /tmp/

RUN apt-get update && \
  apt-get install -y net-tools iproute2 netcat-openbsd dnsutils curl iputils-ping iptables nmap tcpdump openvswitch-switch && \
  mkdir /var/run/openvswitch && mkdir -p /etc/openvswitch && ovsdb-tool create /etc/openvswitch/conf.db /tmp/vswitch.ovsschema 

COPY ./setup_switch.sh /usr/local/bin/

WORKDIR /usr/src/bin

COPY ./main.go ./go.mod ./

RUN go build -v -o /usr/local/bin/l2sm-br ./... && \
    chmod +x /usr/local/bin/setup_switch.sh && \
    mkdir /etc/l2sm/

WORKDIR /usr/local/bin


CMD [ "./setup_switch.sh" ]

  