# Observability Runbook

This runbook explains how to verify and troubleshoot metrics, logs, and traces for the Logistics Management Platform (LMP).

## Stack Overview
- OTEL SDKs in services (Node.js, Python)
- OTEL Collector (contrib) exposing Prometheus at :8889 and exporting traces to Jaeger via OTLP gRPC
- Prometheus (scrapes Collector)
- Grafana (dashboards, datasources provisioned)
- Loki + Promtail (Docker logs scraping)
- Jaeger (trace UI)

## Endpoints
- Grafana: http://localhost:3000
- Prometheus: http://localhost:9090
- Alertmanager: http://localhost:9093
- Jaeger: http://localhost:16686
- Loki API: http://localhost:3100

## Dashboards
- Service Overview (ops/grafana/dashboards/service-overview.json)
  - Request rate, Latency percentiles, Error rate, Dependency latency, Logs panel
- Route-level SLOs (ops/grafana/dashboards/route-slos.json)
  - Variables: Service (exported_job), Route (http_route)
  - Panels: req/s, P50/P90/P95/P99 latency, error-rate, dependency P95 by downstream

Notes on labels:
- HTTP server metrics: exported_job (service), http_route, http_method, http_status_code
- HTTP client metrics: exported_job, net_peer_name (downstream), http_url, http_method, http_status_code

## Quick Checks
1) Services running
```
docker compose ps
```
2) Collector metrics endpoint is scraped by Prometheus
- Prometheus -> Status -> Targets should list otel-collector:8889
3) Prometheus has http_server_* and http_client_* metrics
- Example queries:
```
sum(http_server_requests_total)
sum by (exported_job, http_route) (increase(http_server_requests_total[15m]))
histogram_quantile(0.95, sum by (exported_job, le) (rate(http_server_duration_milliseconds_bucket[5m])))
histogram_quantile(0.95, sum by (net_peer_name, le) (rate(http_client_duration_milliseconds_bucket[5m])))
```
4) Logs correlated by trace_id in Grafana Explore (Loki)
- Query: `{container=~"lmp-.*"} |= "trace_id="`
5) Traces visible in Jaeger by service (gateway-api, label-svc, pricing-svc, etc.)

## Automated Observability Smoke
Generate traffic and validate observability with a single command:
```
GW=http://localhost:8086 bash ./scripts/obs_smoke.sh
```
What it validates:
- http_server_requests_total has http_route label (route-level visibility)
- http_server_duration_milliseconds_bucket has histogram buckets
- http_client_requests_total has net_peer_name (downstream) label
- http_server_requests_total for gateway-api has samples (post-scrape)
- Loki contains logs with trace_id

If it fails:
- Ensure stack is up: `docker compose up -d otel-collector prometheus grafana loki promtail jaeger`
- Run again after ~30s to allow scrapes

## Alerts
- Prometheus rules: ops/alerts/app-rules.yml
  - HighErrorRate (>2% 5xx over 10m) [warning]
  - CriticalHighErrorRate (>10% 5xx over 10m) [critical]
  - HighLatencyP99 (>500ms over 10m)
  - DependencyErrors (>5 in 5m)
  - NoServerMetrics (no http.server metrics in 10m)
- Alertmanager config: ops/alertmanager.yml
  - Default receiver `dev-null` (no external notifications)

To enable notifications:
1) Edit ops/alertmanager.yml and add a receiver, e.g. Slack or email
2) Restart: `docker compose up -d alertmanager prometheus`

## Common Troubleshooting
- No metrics in Prometheus:
  - Check Collector container logs
  - Verify service env vars: OTEL_EXPORTER_OTLP_ENDPOINT set to http://otel-collector:4317
  - Confirm Prometheus target up (Status -> Targets)
- No http_route labels:
  - Ensure gateway routes use controller decorators and instrumentation is enabled
  - Verify Node services started with tracing.ts SDK init
- No logs with trace_id:
  - Ensure services log trace/span IDs; check Loki labels and promtail config
  - Confirm trace context propagation from gateway to downstream
- No traces in Jaeger:
  - Verify Collector `traces` pipeline exporters: [otlp], and Jaeger is healthy
  - Check sampling config (env OTEL_TRACES_SAMPLER/_ARG)

## Extending Dashboards & Alerts
- Add service/route specific SLO thresholds by cloning panels and pinning variables
- Add downstream-specific dependency panels using net_peer_name
- Create alerts for route-specific P99 latency or 5xx rate using http_route label

## How to add a new service
1) Initialize OTEL SDK with OTLP gRPC exporter (4317)
2) Ensure http server instrumentation enabled (auto-instrumentation or middleware metrics)
3) Emit http.client metrics for outbound calls; include net_peer_name if possible
4) Add service environment: OTEL_SERVICE_NAME, OTEL_EXPORTER_OTLP_ENDPOINT
5) Rebuild and start the service via docker compose
6) Validate with the Observability Smoke script

