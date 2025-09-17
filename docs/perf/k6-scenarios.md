# k6 Scenarios

Files:
- tests/perf/k6/helpers.js
- tests/perf/k6/gateway_funnel_smoke.js (2–3 min)
- tests/perf/k6/gateway_funnel_baseline.js (5–8 min)

Run locally:
- docker compose up -d
- docker compose run --rm --profile perf -e GW_URL=http://gateway-api:8080 k6 run /scripts/gateway_funnel_smoke.js --summary-export /out/k6/smoke.json

Budgets (thresholds):
- Error rate ≤ 1%
- P95 ≤ 600ms, P99 ≤ 1000ms

Env:
- GW_URL (default http://gateway-api:8080)
- TENANT_ID (default demo)
- VUS, DURATION (override defaults)

