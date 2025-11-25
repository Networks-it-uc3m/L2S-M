#!/bin/bash

wget -q https://github.com/containernetworking/plugins/releases/download/v1.6.0/cni-plugins-linux-amd64-v1.6.0.tgz
mkdir -p plugins/bin
tar -xf cni-plugins-linux-amd64-v1.6.0.tgz -C plugins/bin
rm cni-plugins-linux-amd64-v1.6.0.tgz

# copy necessary plugins into all nodes
docker cp ./plugins/bin/. l2sm-test-control-plane:/opt/cni/bin
docker cp ./plugins/bin/. l2sm-test-worker:/opt/cni/bin
docker cp ./plugins/bin/. l2sm-test-worker2:/opt/cni/bin
docker exec -it l2sm-test-control-plane modprobe br_netfilter
docker exec -it l2sm-test-worker modprobe br_netfilter
docker exec -it l2sm-test-worker2 modprobe br_netfilter

docker exec -it l2sm-test-control-plane sysctl -p /etc/sysctl.conf
docker exec -it l2sm-test-worker sysctl -p /etc/sysctl.conf
docker exec -it l2sm-test-worker2 sysctl -p /etc/sysctl.conf


kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml
kubectl wait --for=condition=Ready pods -n kube-flannel -l app=flannel --timeout=300s

kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.1/cert-manager.yaml
kubectl apply -f https://raw.githubusercontent.com/k8snetworkplumbingwg/multus-cni/master/deployments/multus-daemonset-thick.yml
kubectl wait --for=condition=Ready pods --all -A  --timeout=300s
