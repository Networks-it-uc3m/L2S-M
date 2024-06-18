#!/bin/bash

file=./chart/values.yaml

# Get nodes information from Kubernetes
node_data=$(kubectl get nodes -o json)

# Initialize the starting IP address
ip_base="10.0.0."
ip_counter=2  # Start from .2

# Write the beginning of the values.yaml file
echo "global:" > $file
echo "  nodes:" >> $file

# Loop through each node, incrementing IP for each
echo "$node_data" | jq -c '.items[]' | while read -r node; do
    echo "    - name: $(echo $node | jq -r '.metadata.name')" >> $file
    echo "      ip: ${ip_base}${ip_counter}/24" >> $file
    echo "      metrics:" >> $file
    echo "        rttInterval: 10" >> $file
    echo "        throughputInterval: 20" >> $file
    echo "        jitterInterval: 5" >> $file
    ((ip_counter++))
done

# Finish the values.yaml file
echo "  network:" >> $file
echo "    name: lpm-network" >> $file
echo "  namespace: default" >> $file
