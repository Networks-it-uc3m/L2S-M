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
