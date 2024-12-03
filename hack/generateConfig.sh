#!/bin/bash

# Filename for the JSON config
jsonConfigPath="./configs/switchConfig.json"

# Use kubectl to get pods, grep for those starting with 'l2sm-switch', and parse with awk
mapfile -t podInfo < <(
  kubectl get pods -n he-codeco-netma -o wide |
  grep 'l2sm-switch' |
  awk '{for (i=1; i<=NF; i++) { 
    if ($i ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$/) { 
      print $(i+1) ":" $i; 
      break; 
    } 
  }}'
)

# Associative array to store node names and their respective IPs
declare -A nodeIPs

# Fill the associative array with node names as keys and IPs as values
for entry in "${podInfo[@]}"
do
  nodeName="${entry%%:*}"
  nodeIP="${entry##*:}"
  nodeIPs["$nodeName"]="$nodeIP"
done

# Prepare Nodes array
nodesArray=()
for nodeName in "${!nodeIPs[@]}"
do
  nodesArray+=("{\"name\": \"$nodeName\", \"nodeIP\": \"${nodeIPs[$nodeName]}\"}")
done

# Prepare Links array
linksArray=()
declare -A linkPairs
for srcNode in "${!nodeIPs[@]}"
do
  for dstNode in "${!nodeIPs[@]}"
  do
    if [[ "$srcNode" != "$dstNode" && -z "${linkPairs["$srcNode|$dstNode"]}" && -z "${linkPairs["$dstNode|$srcNode"]}" ]]; then
      linksArray+=("{\"endpointA\": \"$srcNode\", \"endpointB\": \"$dstNode\"}")
      linkPairs["$srcNode|$dstNode"]=1
    fi
  done
done

# Generate JSON output
echo "{
  \"Nodes\": [
    $(IFS=,; echo "${nodesArray[*]}")
  ],
  \"Links\": [
    $(IFS=,; echo "${linksArray[*]}")
  ]
}" > "$jsonConfigPath"

echo "Network topology configuration has been generated at $jsonConfigPath."
