#!/usr/bin/env bash
set -euo pipefail

# Generate a dev CA and per-service certs for mTLS
# Output directory: ops/tls/out
# Note: Do NOT commit the generated artifacts.

DIR=$(cd "$(dirname "$0")" && pwd)
OUT="$DIR/out"
mkdir -p "$OUT"

# Generate CA
if [[ ! -f "$OUT/ca.key" ]]; then
  echo "Generating CA key/cert..."
  openssl genrsa -out "$OUT/ca.key" 4096
  openssl req -x509 -new -nodes -key "$OUT/ca.key" -sha256 -days 3650 -subj "/CN=Dev LMP CA" -out "$OUT/ca.crt"
fi

# Helper to generate a server cert for a service DNS name
# Args: name dns
server_cert() {
  local NAME="$1"; local DNS="$2"
  local CNF="$OUT/${NAME}.cnf"
  cat > "$CNF" <<EOF
[ req ]
distinguished_name = dn
req_extensions = v3_req
prompt = no
[ dn ]
CN = ${NAME}
[ v3_req ]
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[ alt_names ]
DNS.1 = ${DNS}
EOF
  echo "Generating server cert for ${DNS} ..."
  openssl genrsa -out "$OUT/${NAME}.key" 2048
  openssl req -new -key "$OUT/${NAME}.key" -out "$OUT/${NAME}.csr" -config "$CNF"
  openssl x509 -req -in "$OUT/${NAME}.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" -CAcreateserial \
    -out "$OUT/${NAME}.crt" -days 365 -sha256 -extensions v3_req -extfile "$CNF"
}

# Helper to generate a client cert for gateway
client_cert() {
  local NAME="$1"
  local CNF="$OUT/${NAME}.cnf"
  cat > "$CNF" <<EOF
[ req ]
distinguished_name = dn
req_extensions = v3_req
prompt = no
[ dn ]
CN = ${NAME}
[ v3_req ]
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
  echo "Generating client cert ${NAME} ..."
  openssl genrsa -out "$OUT/${NAME}.key" 2048
  openssl req -new -key "$OUT/${NAME}.key" -out "$OUT/${NAME}.csr" -config "$CNF"
  openssl x509 -req -in "$OUT/${NAME}.csr" -CA "$OUT/ca.crt" -CAkey "$OUT/ca.key" -CAcreateserial \
    -out "$OUT/${NAME}.crt" -days 365 -sha256 -extensions v3_req -extfile "$CNF"
}

# Generate gateway client cert
client_cert gateway-client

# Generate server certs for proxies
server_cert identity-svc identity-svc-proxy
server_cert shipment-svc shipment-svc-proxy
server_cert pricing-svc pricing-svc-proxy
server_cert label-svc label-svc-proxy
server_cert tracking-svc tracking-svc-proxy

echo "All certificates generated in $OUT"
