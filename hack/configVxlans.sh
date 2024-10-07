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
