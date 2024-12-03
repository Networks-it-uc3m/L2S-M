#!/bin/bash

# Check if deployment files exist
if ls deployments_*.yaml 1> /dev/null 2>&1; then
  echo "Deleting deployments individually with a 5-second delay..."
  # Delete deployments one by one
  for deployment_file in deployments_*.yaml; do
    echo "Deleting resources in $deployment_file..."
    kubectl delete -f "$deployment_file"
    sleep 5
  done
else
  echo "No deployment files found. Skipping deployment deletion."
fi

# Check if non-deployment file exists
if [ -f non_deployments.yaml ]; then
  echo "Deleting non-deployment resources..."
  kubectl delete -f non_deployments.yaml
else
  echo "Non-deployment resources file not found. Skipping."
fi

# Optionally, clean up the generated YAML files
echo "Cleaning up generated YAML files..."
rm -f non_deployments.yaml deployments_*.yaml

echo "Undeployment process completed."
