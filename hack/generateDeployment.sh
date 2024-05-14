#!/bin/bash

# Define the directory containing the YAML files
DIRECTORY="./deployments/custom-installation"

# Define the output file
OUTPUT_FILE="./deployments/l2sm-deployment.yaml"

# Start with an empty output file
> "$OUTPUT_FILE"

# Find all YAML files within the directory and all subdirectories,
# concatenate their contents into the output file and add '---' after each file's content
find "$DIRECTORY" -type f -name '*.yaml' | sort | while read file; do
    cat "$file" >> "$OUTPUT_FILE"
    echo "---" >> "$OUTPUT_FILE"
done

echo "All YAML files, including those within subdirectories, have been concatenated into $OUTPUT_FILE with delimiters."
