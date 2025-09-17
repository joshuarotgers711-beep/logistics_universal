# SLO Burn-Rate Alerts Runbook

What it means:
- Multi-window alerts trigger when short and long windows both exceed thresholds (e.g., 5m > 5% and 1h > 2%).
- Reduces false positives; indicates sustained error budget burn.

Triage steps:
1) Identify impacted service/route from alert labels (exported_job, http_route)
2) Check recent deploys; correlate with latency/error spikes
3) Inspect traces for dominant error codes and upstream dependencies
4) Review logs for exceptions; check DB/Redis health and timeouts

Immediate mitigations:
- Roll back recent change or reduce traffic to the route
- Toggle feature flags; increase resources temporarily if saturation observed

Follow-ups:
- Add tests/guards for the failure mode
- Capture assumptions as SLOs and error budget policies

