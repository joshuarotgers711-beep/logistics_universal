# Infra Saturation Runbook

Symptoms:
- Node CPU > 80/90%, memory < 10% available, or filesystem > 90%
- Container CPU/memory alerts

Checks:
1) Grafana: Infra USE & Service RED dashboard
2) Prometheus: node/cadvisor metrics for affected instance/container
3) Application: request rate/latency changes; GC/alloc stats (pprof if enabled)

Actions:
- Scale up/out services; adjust limits/requests
- Optimize hotspots; enable caching; DB index review
- Prune logs/images if filesystem pressure

Prevention:
- Capacity planning based on RED metrics and burn-rate alerts
- Periodic load testing; alert tuning

