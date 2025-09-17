#!/usr/bin/env bash
set -euo pipefail
SQL=${1:-}
if [ -z "$SQL" ]; then
  echo "Usage: $0 'SELECT ...'" >&2
  exit 1
fi
exec docker compose exec -T postgres psql -U postgres -d lmp -c "EXPLAIN (ANALYZE, BUFFERS, VERBOSE) $SQL"

