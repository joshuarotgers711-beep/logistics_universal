#!/usr/bin/env bash
set -euo pipefail

# Simple E2E smoke through gateway
GW=${GW:-http://localhost:8080}

# 1) Login
TOKENS=$(curl -s -X POST "$GW/v1/auth/login" -H 'Content-Type: application/json' -d '{"email":"admin@example.com","password":"admin123"}')
# Support both camelCase and PascalCase keys from identity-svc
ACCESS=$(echo "$TOKENS" | sed -n 's/.*"accessToken":"\([^"]*\)".*/\1/p')
if [ -z "$ACCESS" ]; then
  ACCESS=$(echo "$TOKENS" | sed -n 's/.*"AccessToken":"\([^"]*\)".*/\1/p')
fi
[ -n "$ACCESS" ] || { echo "Login failed: $TOKENS"; exit 1; }
echo "Access token acquired"

AUTHZ=( -H "Authorization: Bearer $ACCESS" -H 'Content-Type: application/json' )

# 2) Create shipment (require idempotency key)
IDEM_KEY="smoke-$(date +%s%N)"
CREATE_BODY='{"shipperId":"demo","serviceLevel":"GROUND","from":{"line1":"A","city":"SF","state":"CA","postalCode":"94105","country":"US"},"to":{"line1":"B","city":"NYC","state":"NY","postalCode":"10001","country":"US"},"value":100,"packages":[{"weightKg":1,"lengthCm":10,"widthCm":10,"heightCm":10}]}'
SHIP=$(curl -s -X POST "$GW/v1/shipments" -H "X-Idempotency-Key: $IDEM_KEY" "${AUTHZ[@]}" -d "$CREATE_BODY" )
SID=$(echo "$SHIP" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
[ -n "$SID" ] || { echo "Create shipment failed: $SHIP"; exit 1; }
echo "Shipment created: $SID"

# 3) Quote pricing
QUOTE_BODY='{"serviceLevel":"GROUND","from":{"line1":"A","city":"SF","state":"CA","postalCode":"94105","country":"US"},"to":{"line1":"B","city":"NYC","state":"NY","postalCode":"10001","country":"US"},"packages":[{"weightKg":1,"lengthCm":10,"widthCm":10,"heightCm":10}]}'
QUOTE=$(curl -s -X POST "$GW/v1/shipments/$SID/rate" "${AUTHZ[@]}" -d "$QUOTE_BODY")
echo "Quote: $QUOTE"

# 4) Generate label
LABEL=$(curl -s -X POST "$GW/v1/shipments/$SID/label" "${AUTHZ[@]}" -d '{"format":"PDF"}')
echo "Label: $LABEL"

# 4b) Verify label metadata persisted on shipment
LID=$(echo "$LABEL" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')
SHIPGET=$(curl -s "$GW/v1/shipments/$SID" "${AUTHZ[@]}")
echo "$SHIPGET" | grep -q "$LID" || { echo "Label not persisted: $SHIPGET"; exit 1; }


# 5) Events pickup and deliver (fail if non-2xx)
PICK=$(curl -s -f -X POST "$GW/v1/shipments/$SID/events/pickup" "${AUTHZ[@]}" -d '{"lat":0,"lon":0}')
echo "Pickup: $PICK"
DEL=$(curl -s -f -X POST "$GW/v1/shipments/$SID/events/deliver" "${AUTHZ[@]}" -d '{"lat":0,"lon":0}')
echo "Deliver: $DEL"

# 6) Public tracking (fail if non-2xx)
TRK=$(curl -s -f "$GW/v1/track/$SID")
echo "Track: $TRK"

echo "Smoke OK"

