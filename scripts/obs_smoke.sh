#!/usr/bin/env bash
set -euo pipefail

# Observability smoke: generate traffic, then assert metrics and logs exist
GW=${GW:-http://localhost:8086}
PROM=${PROM:-http://localhost:9090}
LOKI=${LOKI:-http://localhost:3100}

# 1) Generate traffic via existing smoke
echo "-- generating traffic via smoke.sh --"
GW="$GW" bash ./scripts/smoke.sh >/dev/null
sleep 2

fail() { echo "[OBS_SMOKE] FAIL: $*"; exit 1; }
pass() { echo "[OBS_SMOKE] PASS: $*"; }

# 2) Prometheus checks
# 2a) Route-level server metrics present for gateway-api
series=$(curl -s "${PROM}/api/v1/series?match[]="$(python3 - <<'PY'
import urllib.parse
print(urllib.parse.quote('http_server_requests_total{exported_job="gateway-api"}'))
PY
) )
[[ "$series" == *"http_route"* ]] || fail "No http_route label found for http_server_requests_total (gateway-api)"
pass "Route label present for http_server_requests_total"

# 2b) Histogram buckets for route latency present
series2=$(curl -s "${PROM}/api/v1/series?match[]="$(python3 - <<'PY'
import urllib.parse
print(urllib.parse.quote('http_server_duration_milliseconds_bucket{exported_job="gateway-api"}'))
PY
) )
[[ "$series2" == *"\"le\":"* ]] || fail "No histogram buckets (le) found for http_server_duration_milliseconds_bucket"
pass "Server latency histogram buckets present"

# 2c) Client metrics exist and have downstream label
series3=$(curl -s "${PROM}/api/v1/series?match[]="$(python3 - <<'PY'
import urllib.parse
print(urllib.parse.quote('http_client_requests_total{exported_job="gateway-api"}'))
PY
) )
[[ "$series3" == *"net_peer_name"* ]] || fail "No net_peer_name label on http_client_requests_total"
pass "Client metrics with downstream label present"

# Give Prometheus time to scrape
for i in {1..6}; do
  sleep 5
  totalReq=$(curl -sG --data-urlencode "query=sum(http_server_requests_total{exported_job=\"gateway-api\"})" "${PROM}/api/v1/query" | jq -r '.data.result[0].value[1]' || echo "0")
  if awk -v v="${totalReq:-0}" 'BEGIN{exit !(v+0>0)}'; then
    break
  fi
  echo "waiting for Prometheus scrape..."
done

# 2d) Wait for scrape, then verify server metric present (gateway-api)
totalReq=$(curl -sG --data-urlencode "query=sum(http_server_requests_total{exported_job=\"gateway-api\"})" "${PROM}/api/v1/query" | jq -r '.data.result[0].value[1]' || echo "0")
awk -v v="${totalReq:-0}" 'BEGIN{exit !(v+0>0)}' || fail "No http_server_requests_total samples for gateway-api"
pass "Server metric present (total=${totalReq})"

# 3) Loki log check: at least one recent log with trace_id
since=$(date -u +%s)
# Query logs from last 10 minutes
q="{container=~\"lmp-.*\"} |= \"trace_id=\""
resp=$(curl -sG --data-urlencode query="$q" --data-urlencode limit=50 "${LOKI}/loki/api/v1/query")
[[ "$resp" == *"trace_id="* ]] || fail "No logs with trace_id found in Loki"
pass "Loki contains logs with trace_id"

# 4) Idempotency duplicate create test via gateway
echo "-- idempotency duplicate create test --"
GW_URL=${GW:-http://localhost:8086}
TOKENS=$(curl -s -X POST "$GW_URL/v1/auth/login" -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"admin123"}')
ACCESS=$(echo "$TOKENS" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p'); if [ -z "$ACCESS" ]; then ACCESS=$(echo "$TOKENS" | sed -n 's/.*"AccessToken":"\([^"]*\)".*/\1/p'); fi
[ -n "$ACCESS" ] || fail "login failed for idempotency test"
AUTHZ=(-H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json')
IDEM="ci-idem-$(date +%s%N)"
BODY='{"shipperId":"demo","serviceLevel":"GROUND","from":{"line1":"A","city":"SF","state":"CA","postalCode":"94105","country":"US"},"to":{"line1":"B","city":"NYC","state":"NY","postalCode":"10001","country":"US"},"value":100,"packages":[{"weightKg":1,"lengthCm":10,"widthCm":10,"heightCm":10}]}'
# First create
RESP1=$(curl -s -D /tmp/idem1.hdr -o /tmp/idem1.json -w "%{http_code}" -X POST "$GW_URL/v1/shipments" -H "X-Idempotency-Key: $IDEM" "${AUTHZ[@]}" -d "$BODY")
code1=$(tail -n1 <<< "$RESP1")
[ "$code1" = "201" ] || fail "expected 201 on first create, got $code1"
ID1=$(sed -n 's/.*"id":"\([^"]*\)".*/\1/p' /tmp/idem1.json)
[ -n "$ID1" ] || fail "no id in first create response"
# Duplicate create
RESP2=$(curl -s -D /tmp/idem2.hdr -o /tmp/idem2.json -w "%{http_code}" -X POST "$GW_URL/v1/shipments" -H "X-Idempotency-Key: $IDEM" "${AUTHZ[@]}" -d "$BODY")
code2=$(tail -n1 <<< "$RESP2")
[ "$code2" = "200" ] || fail "expected 200 on duplicate create, got $code2"
ID2=$(sed -n 's/.*"id":"\([^"]*\)".*/\1/p' /tmp/idem2.json)
[ "$ID1" = "$ID2" ] || fail "duplicate create did not return same id"
grep -qi "^Idempotent-Replayed: *true" /tmp/idem2.hdr || fail "Idempotent-Replayed header not set on duplicate response"

# 5) Prometheus check for replay metric > 0 in recent window
sleep 3
q=$(python3 - <<'PY'
import urllib.parse
print(urllib.parse.quote('sum(increase(shipment_idempotency_replays_total[10m]))'))
PY
)
val=$(curl -s "${PROM}/api/v1/query?query=$q" | jq -r '.data.result[0].value[1]' || echo "0")
awk -v v="${val:-0}" 'BEGIN{exit !(v+0>0)}' || fail "no shipment_idempotency_replays_total observed"
pass "Idempotency replay metric observed"

echo "[OBS_SMOKE] OK"
