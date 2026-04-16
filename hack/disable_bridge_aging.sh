#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="kube-system"
IMAGE="debian:bookworm-slim"
KEEP_PODS="false"
KUBECTL_ARGS=()

usage() {
  cat <<'EOF'
Usage: hack/disable_bridge_aging.sh [--kubeconfig PATH] [--namespace NAME] [--image IMAGE] [--keep-pods]

Creates one privileged host-network pod per Kubernetes node, installs bridge-utils,
and runs:

  brctl setageing br0 0
  ...
  brctl setageing br10 0

Options:
  --kubeconfig PATH   Kubeconfig to use. Defaults to kubectl's configured default.
  --namespace NAME    Namespace for helper pods. Defaults to kube-system.
  --image IMAGE       Debian-compatible image with apt. Defaults to debian:bookworm-slim.
  --keep-pods         Do not delete helper pods after completion.
  -h, --help          Show this help.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --kubeconfig)
      if [[ $# -lt 2 ]]; then
        echo "error: --kubeconfig requires a path" >&2
        exit 2
      fi
      KUBECTL_ARGS+=(--kubeconfig "$2")
      shift 2
      ;;
    --namespace|-n)
      if [[ $# -lt 2 ]]; then
        echo "error: --namespace requires a value" >&2
        exit 2
      fi
      NAMESPACE="$2"
      shift 2
      ;;
    --image)
      if [[ $# -lt 2 ]]; then
        echo "error: --image requires a value" >&2
        exit 2
      fi
      IMAGE="$2"
      shift 2
      ;;
    --keep-pods)
      KEEP_PODS="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

kubectl_cmd() {
  kubectl "${KUBECTL_ARGS[@]}" "$@"
}

sanitize_name() {
  local value="$1"
  value="$(printf '%s' "$value" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9-]+/-/g; s/^-+//; s/-+$//')"
  printf 'disable-bridge-aging-%s' "${value:0:40}"
}

mapfile -t NODES < <(kubectl_cmd get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}')

if [[ ${#NODES[@]} -eq 0 ]]; then
  echo "error: no Kubernetes nodes found" >&2
  exit 1
fi

echo "Using namespace: $NAMESPACE"
echo "Using image: $IMAGE"
echo "Found ${#NODES[@]} node(s)."

for idx in "${!NODES[@]}"; do
  node="${NODES[$idx]}"
  pod="$(sanitize_name "$node")"
  pod="${pod}-${idx}"

  echo
  echo "Configuring bridge ageing on node: $node"

  kubectl_cmd delete pod "$pod" -n "$NAMESPACE" --ignore-not-found --wait=true >/dev/null

  kubectl_cmd apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $pod
  namespace: $NAMESPACE
  labels:
    app.kubernetes.io/name: disable-bridge-aging
spec:
  restartPolicy: Never
  nodeName: "$node"
  hostNetwork: true
  hostPID: true
  hostIPC: true
  tolerations:
  - operator: Exists
  containers:
  - name: bridge-utils
    image: "$IMAGE"
    imagePullPolicy: IfNotPresent
    securityContext:
      privileged: true
      allowPrivilegeEscalation: true
      capabilities:
        add:
        - ALL
    command:
    - /bin/bash
    - -ceu
    - |
      export DEBIAN_FRONTEND=noninteractive
      apt-get update
      apt-get install -y --no-install-recommends bridge-utils

      for i in \$(seq 0 10); do
        bridge="br\${i}"
        if brctl show "\$bridge" >/dev/null 2>&1; then
          brctl setageing "\$bridge" 0
          echo "Set ageing to 0 on \$bridge"
        else
          echo "Skipping missing bridge \$bridge"
        fi
      done
EOF

  kubectl_cmd wait pod "$pod" -n "$NAMESPACE" --for=condition=Ready --timeout=180s >/dev/null || true
  if ! kubectl_cmd wait pod "$pod" -n "$NAMESPACE" --for=jsonpath='{.status.phase}'=Succeeded --timeout=300s; then
    kubectl_cmd logs "$pod" -n "$NAMESPACE" || true
    echo "error: helper pod failed on node $node" >&2
    exit 1
  fi
  kubectl_cmd logs "$pod" -n "$NAMESPACE"

  if [[ "$KEEP_PODS" != "true" ]]; then
    kubectl_cmd delete pod "$pod" -n "$NAMESPACE" --wait=false >/dev/null
  fi
done

echo
echo "Done."
