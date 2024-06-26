for mininet. In this dir:
sudo mn --controller remote,ip=localhost:6633 --custom topo.py --switch ovs,protocols=OpenFlow13  --topo mytopo

## for onos. in ~/onos: 

bazel run onos-local -- clean debug

## onos cli, in ./onos:

tools/test/bin/onos localhost -l karaf

## mininet 2:

python3 mininet_setup.py

