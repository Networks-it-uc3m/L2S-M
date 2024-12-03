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

#!/bin/bash

# Unset variables to prevent overlapping issues
unset configFilePath
unset podInfo
unset pods
unset entry
unset podName
unset nodeName
unset pod

# Path to the configuration file
configFilePath="./configs/switchConfig.json"

# Use kubectl to get pods and grep for those starting with 'l2sm-switch'
# Then, use awk to extract the pod name and node name by matching an IP pattern and taking the next column as the node
mapfile -t podInfo < <(kubectl get pods -n he-codeco-netma -o wide | grep 'l2sm-switch' | awk '{for (i=1; i<=NF; i++) { if ($i ~ /^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$/) { print $1 ":" $(i+1); break; } }}')

# Declare an associative array to store pod names and their respective nodes
declare -A pods

# Fill the associative array with pod names as keys and node names as values
for entry in "${podInfo[@]}"
do
  podName="${entry%%:*}"
  nodeName="${entry##*:}"
  pods["$podName"]=$nodeName
done

# Loop through the array
for pod in "${!pods[@]}"
do
  nodeName=${pods[$pod]}
  echo $pod

  # Copy the configuration file to the pod
  echo "Copying config file to $pod..."
  kubectl cp -n he-codeco-netma "$configFilePath" "$pod:/etc/l2sm/switchConfig.json"

  # Execute the script inside the pod
  echo "Executing configuration script on $pod..."
  kubectl exec -it "$pod" -n he-codeco-netma -- /bin/bash -c "l2sm-vxlans --node_name=$nodeName /etc/l2sm/switchConfig.json"
done

echo "Configuration deployment completed."

#!/bin/bash

# Define file paths
helm_chart_path="./LPM/chart/"
non_deployments_file="non_deployments.yaml"

# Remove previous files if they exist
rm -f $non_deployments_file
rm -f deployments_*.yaml

# Run helm template and process the output
helm template $helm_chart_path | awk '
  BEGIN { in_deployment = 0; doc = "" }
  /^---$/ {
    if (doc != "") {
      if (in_deployment == 0) {
        print doc >> "'$non_deployments_file'"
        print "---" >> "'$non_deployments_file'"
      } else {
        # Increment deployment counter
        deployment_count += 1
        file_name = sprintf("deployments_%02d.yaml", deployment_count)
        print doc >> file_name
        print "---" >> file_name
      }
    }
    doc = ""; in_deployment = 0
  }
  {
    doc = doc $0 "\n"
    if ($1 == "kind:" && $2 == "Deployment") {
      in_deployment = 1
    }
  }
  END {
    if (doc != "") {
      if (in_deployment == 0) {
        print doc >> "'$non_deployments_file'"
      } else {
        deployment_count += 1
        file_name = sprintf("deployments_%02d.yaml", deployment_count)
        print doc >> file_name
      }
    }
  }
'

# Apply non-deployment resources
echo "Applying non-deployment resources..."
kubectl apply -f $non_deployments_file

# Apply deployments one by one with a 5-second delay
echo "Applying deployments individually with a 5-second delay..."
for deployment_file in deployments_*.yaml; do
  echo "Applying $deployment_file..."
  kubectl apply -f "$deployment_file"
  sleep 5
done

echo "Deployment process completed."
