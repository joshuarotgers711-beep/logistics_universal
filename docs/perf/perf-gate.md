# Performance Gate (CI)

Workflow: .github/workflows/perf.yml
- Triggers: nightly + manual dispatch
- Stack: docker compose up (core services + pyroscope)
- Test: k6 baseline (6m, thresholds aligned to SLO budgets)
- Artifacts: k6 summary JSON, pprof snapshots, pg_stat_statements excerpt

Pass/Fail:
- k6 thresholds enforce: error rate ≤ 1%, P95 ≤ 600ms, P99 ≤ 1000ms
- Non-zero exit if thresholds fail (k6 behavior)

PR smoke (optional):
- You can run smoke locally: k6 gateway_funnel_smoke.js (2–3m)

Next improvements:
- Add language agents to push continuous profiles to Pyroscope
- Track trend/baselines over time and gate on percent regression

