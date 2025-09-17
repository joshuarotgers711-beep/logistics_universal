#!/usr/bin/env bash
set -u
ART=.artifacts/mtls
mkdir -p "$ART"

# Capture docker status
( docker compose ps || true ) > "$ART/compose_ps.txt" 2>&1
( docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' || true ) > "$ART/docker_ps.txt" 2>&1

# Gateway health
( curl -sS -w "\nHTTP %{http_code}\n" http://localhost:8086/health || true ) > "$ART/gateway_health.txt" 2>&1

# Python E2E
( python3 -u tools/e2e/mtls_e2e.py || true ) > "$ART/e2e_python.txt" 2>&1

# Node E2E (fallback)
( node tools/e2e/mtls_e2e.js || true ) > "$ART/e2e_node.txt" 2>&1

# Determine docker network for compose
NET=""
GW_CTN=$(docker ps --format '{{.Names}}' | grep -E 'gateway-api' | head -n1)
if [ -n "$GW_CTN" ]; then
  NET=$(docker inspect -f '{{range $k,$v := .NetworkSettings.Networks}}{{printf "%s\n" $k}}{{end}}' "$GW_CTN" | head -n1)
fi
if [ -z "$NET" ]; then
  NET=$(docker network ls --format '{{.Name}}' | grep _default | head -n1)
fi
echo "$NET" > "$ART/network_name.txt"

# Proxy tests
( docker run --rm --network "$NET" curlimages/curl:8.10.1 -sS -k https://identity-svc-proxy:8443/health -w "\nHTTP %{http_code}\n" || true ) > "$ART/proxy_nocert.txt" 2>&1
( docker run --rm --network "$NET" -v "$(pwd)/ops/tls/out:/certs:ro" curlimages/curl:8.10.1 -sS --cert /certs/gateway-client.crt --key /certs/gateway-client.key --cacert /certs/ca.crt https://identity-svc-proxy:8443/health -w "\nHTTP %{http_code}\n" || true ) > "$ART/proxy_withcert.txt" 2>&1

# Jaeger and Prometheus
( curl -sS 'http://localhost:16686/api/traces?service=gateway-api&limit=5&lookback=1h' || true ) > "$ART/jaeger_gateway_traces.json" 2>&1

# Prometheus TLS expiry
( curl -sS --get --data-urlencode 'query=(probe_ssl_earliest_cert_expiry - time())/86400' http://localhost:9090/api/v1/query || true ) > "$ART/prom_tls_expiry_days.json" 2>&1

# Prometheus HTTP traffic (multiple alternate metric names)
( curl -sS --get --data-urlencode 'query=sum(rate(http_client_requests[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_client_req_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_server_requests[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_server_req_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_client_duration_count[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_client_duration_count_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_server_duration_count[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_server_duration_count_5m.json" 2>&1

# Environment-specific metric names (discovered)
( curl -sS --get --data-urlencode 'query=sum(rate(http_client_requests_total[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_client_requests_total_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_server_requests_total[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_server_requests_total_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_client_request_duration_seconds_count[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_client_request_duration_seconds_count_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=sum(rate(http_server_duration_milliseconds_count[5m]))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_server_duration_milliseconds_count_5m.json" 2>&1
( curl -sS --get --data-urlencode 'query=histogram_quantile(0.5, sum(rate(http_client_request_duration_seconds_bucket[5m])) by (le))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_client_p50.json" 2>&1
( curl -sS --get --data-urlencode 'query=histogram_quantile(0.5, sum(rate(http_server_duration_milliseconds_bucket[5m])) by (le))' http://localhost:9090/api/v1/query || true ) > "$ART/prom_http_server_p50.json" 2>&1

# Optional debug endpoint snapshot (will be 404 unless enabled)
( curl -sS http://localhost:8086/__debug/mtls || true ) > "$ART/debug_mtls.json" 2>&1

# Done marker
date > "$ART/_completed.txt"

