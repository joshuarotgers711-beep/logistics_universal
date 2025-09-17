Pyroscope agent integration (Go services)

Overview
- We integrated the grafana/pyroscope-go agent into identity-svc, shipment-svc, and tracking-svc behind an environment toggle. When enabled, each service pushes continuous profiles (CPU, alloc/inuse space & objects, goroutines) to the local Pyroscope server.
- Safe by default: when disabled, there is no startup or runtime overhead.

Environment variables (per service)
- ENABLE_PYROSCOPE: true|false (default true in docker-compose for local dev)
- PYROSCOPE_SERVER: http://pyroscope:4040 (default)
- OTEL_SERVICE_NAME: used as the application name in Pyroscope

Local usage
1) Start the stack: docker compose up -d --build pyroscope identity-svc shipment-svc tracking-svc
2) Browse Pyroscope UI: http://localhost:4040
   - You should see apps named identity-svc, shipment-svc, and tracking-svc after a minute of activity.
3) Toggle off profiling (no overhead): set ENABLE_PYROSCOPE=false for any service and restart it.

Grafana
- A Pyroscope datasource is provisioned. You can add “Flamegraph”/“Table” panels pointing at datasource “Pyroscope” to explore profiles alongside metrics and traces.

CI
- The nightly perf workflow includes a Pyroscope health check to ensure the server is up before running load. Agents are enabled by default in Compose, so profiles will be pushed during k6 runs.

Notes on dependencies
- The Go code imports github.com/grafana/pyroscope-go. During container builds, Go will fetch this dependency automatically.
- To update go.mod locally, run:
  cd services/identity-svc && go get github.com/grafana/pyroscope-go@latest && go mod tidy
  cd ../shipment-svc && go get github.com/grafana/pyroscope-go@latest && go mod tidy
  cd ../tracking-svc && go get github.com/grafana/pyroscope-go@latest && go mod tidy

Troubleshooting
- No apps in UI:
  - Check agent toggle: docker compose exec <svc> env | grep ENABLE_PYROSCOPE
  - Verify server URL: docker compose exec <svc> env | grep PYROSCOPE_SERVER
  - Inspect service logs for "pyroscope start" messages or errors
- Server down: docker compose logs pyroscope; ensure port 4040 is free
- High overhead: keep agents disabled in prod; use sampling windows in Pyroscope UI for analysis

