#!/usr/bin/env bash
set -euo pipefail
ART_DIR=${1:-.artifacts/pprof}
mkdir -p "$ART_DIR"

services=(identity-svc shipment-svc tracking-svc)
for s in "${services[@]}"; do
  echo "Capturing pprof from $s"
  docker compose exec -T "$s" wget -qO- http://localhost:6060/debug/pprof/heap > "$ART_DIR/${s}_heap.pb" || echo "heap failed for $s"
  docker compose exec -T "$s" wget -qO- http://localhost:6060/debug/pprof/profile?seconds=20 > "$ART_DIR/${s}_cpu.pb" || echo "cpu failed for $s"
  docker compose exec -T "$s" wget -qO- http://localhost:6060/debug/pprof/goroutine?debug=1 > "$ART_DIR/${s}_goroutine.txt" || echo "goroutine failed for $s"
Done
