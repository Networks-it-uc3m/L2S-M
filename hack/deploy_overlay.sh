#!/usr/bin/env bash
set -euo pipefail

TEMPLATE=./examples/crd-templates/monitored-overlay.yaml

# Get node names (one per line)
mapfile -t NODES < <(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')

mkdir -p ./local/

# Create temp files for insertion blocks
NODES_FILE="$(mktemp)"
LINKS_FILE="$(mktemp)"
trap 'rm -f "$NODES_FILE" "$LINKS_FILE"' EXIT

# Build nodes YAML block (indented for topology.nodes list items)
printf '%s\n' "${NODES[@]}" | sed 's/^/      - /' > "$NODES_FILE"

# Build links YAML block (indented for topology.links list items)
: > "$LINKS_FILE"
for ((i=0; i<${#NODES[@]}; i++)); do
  for ((j=i+1; j<${#NODES[@]}; j++)); do
    {
      printf '      - endpointA: %s\n' "${NODES[i]}"
      printf '        endpointB: %s\n' "${NODES[j]}"
    } >> "$LINKS_FILE"
  done
done

# Replace placeholder lines by reading from files, then deleting the placeholder line
sed \
  -e "/^[[:space:]]*#__NODES__$/{
        r $NODES_FILE
        d
      }" \
  -e "/^[[:space:]]*#__LINKS__$/{
        r $LINKS_FILE
        d
      }" \
  "$TEMPLATE" > ./local/overlay.yaml

echo "Wrote: ./local/overlay.yaml"
echo "Nodes: ${#NODES[@]}"
echo "Links: $(( (${#NODES[@]} * (${#NODES[@]} - 1)) / 2 ))"

kubectl apply -f ./local/overlay.yaml
