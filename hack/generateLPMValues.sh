#!/bin/bash

# Define the output file path
output_file="./LPM/chart/values.yaml"

# Create or overwrite the file with the initial content
cat > "$output_file" << EOF
global:
  nodes:
EOF

# Get the list of node names
nodes=($(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'))

# Initialize IP address counter starting from 2 (as per 10.0.0.(n+1))
ip_counter=2

# Loop through each node and append the required YAML content
for node in "${nodes[@]}"; do
  cat >> "$output_file" << EOF
    - name: $node
      ip: 10.0.0.$ip_counter/24
      metrics:
        rttInterval: 10
        throughputInterval: 20
        jitterInterval: 5
EOF
  ip_counter=$((ip_counter + 1))
done

# Append the network and namespace information
cat >> "$output_file" << EOF
  network:
    name: lpm-network
  namespace: he-codeco-netma
EOF
