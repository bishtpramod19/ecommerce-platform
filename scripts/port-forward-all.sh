#!/bin/bash
# Starts every port-forward needed for local development.
# Run it in its own terminal and leave it open. Ctrl+C stops everything.
#
# If a forward drops (for example, ArgoCD replaced the pod after a
# deployment), the script reconnects to the new pod automatically.

PIDS=()

# label | namespace | service | local_port:remote_port
FORWARDS=(
  "user-service REST|ecommerce-dev|user-service|8001:8001"
  "user-service gRPC|ecommerce-dev|user-service|50051:50051"
  "product-service REST|ecommerce-dev|product-service|8002:8002"
  "product-service gRPC|ecommerce-dev|product-service|50052:50052"
  "inventory gRPC|ecommerce-dev|inventory-service|50053:50053"
  "inventory metrics|ecommerce-dev|inventory-service|9090:9090"
  "order-service REST|ecommerce-dev|order-service|8004:8004"
  "postgres|ecommerce-dev|postgres-service|5433:5432"
  "mongodb|ecommerce-dev|mongodb-service|27018:27017"
  "prometheus|ecommerce-dev|prometheus|9091:9090"
  "grafana|ecommerce-dev|grafana|3000:3000"
  "argocd|argocd|argocd-server|8080:443"
)

port_in_use() {
  ss -ltn 2>/dev/null | awk '{print $4}' | grep -qE "[:.]$1$"
}

# Runs one port-forward in a background loop, restarting it if it exits.
forward() {
  local label=$1 ns=$2 svc=$3 ports=$4
  (
    child=""
    trap '[ -n "$child" ] && kill "$child" 2>/dev/null; exit 0' TERM
    while true; do
      kubectl port-forward -n "$ns" "svc/$svc" "$ports" >/dev/null 2>&1 &
      child=$!
      wait "$child"
      echo "[$(date +%T)] $label ($ports) disconnected, reconnecting in 2s..."
      sleep 2
    done
  ) &
  PIDS+=($!)
}

cleanup() {
  echo
  echo "Stopping all port-forwards..."
  kill -TERM "${PIDS[@]}" 2>/dev/null
  wait 2>/dev/null
  exit 0
}
trap cleanup INT TERM

echo "Starting port-forwards..."
echo

for entry in "${FORWARDS[@]}"; do
  IFS='|' read -r label ns svc ports <<< "$entry"
  local_port=${ports%%:*}

  if port_in_use "$local_port"; then
    echo "  ⚠  skipping $label: local port $local_port is already in use"
    continue
  fi

  forward "$label" "$ns" "$svc" "$ports"
  printf "  %-22s localhost:%-6s → %s/%s\n" "$label" "$local_port" "$ns" "$svc"
done

echo
echo "Ready. Press Ctrl+C to stop all port-forwards."
wait