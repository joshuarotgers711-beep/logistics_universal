# pprof Usage

Enable:
- docker-compose sets ENABLE_PPROF=true for Go services (identity, shipment, tracking)
- Services expose :6060 /debug/pprof endpoints (heap, profile, goroutine)

Collect:
- scripts/perf/capture_pprof.sh .artifacts/pprof
- or manual:
  - docker compose exec -T shipment-svc wget -qO- http://localhost:6060/debug/pprof/profile?seconds=20 > /tmp/shipment_cpu.pb
  - docker compose exec -T shipment-svc wget -qO- http://localhost:6060/debug/pprof/heap > /tmp/shipment_heap.pb

View:
- go tool pprof -http=:0 /path/to/profile.pb

Pyroscope:
- Pyroscope server runs at http://localhost:4040 and is configured as a Grafana datasource
- Agents are not yet pushing profiles continuously; use ad-hoc pprof captures for now
- Option: integrate language agents (pyroscope-go, pyroscope-js) to push to http://pyroscope:4040

