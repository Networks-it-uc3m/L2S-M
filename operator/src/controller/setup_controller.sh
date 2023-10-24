#!/bin/bash

# turn on bash's job control
set -m

# Start the primary process and put it in the background
./bin/onos-service server &

sleep 90

# Start the helper process
./bin/onos-app localhost install idco-app-1.0.oar
./bin/onos-app localhost activate org.idco.app
./bin/onos-app localhost activate org.onosproject.drivers
./bin/onos-app localhost activate org.onosproject.lldpprovider
./bin/onos-app localhost activate org.onosproject.openflow-base
./bin/onos-app localhost activate org.onosproject.optical-model



    
    
    


# now we bring the primary process back into the foreground
# and leave it there
fg %1