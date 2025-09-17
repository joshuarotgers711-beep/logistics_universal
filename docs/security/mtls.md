# mTLS between Gateway and Backend Services

This document describes how we enable mutual TLS (mTLS) between the gateway-api and backend services in local/dev and CI.

Goals:
- Require and validate gateway client certificates at each backend entrypoint
- Maintain backward compatibility (toggleable via `ENABLE_MTLS`)
- Keep application code unchanged in backends by using sidecar TLS proxies
- Provide easy certificate generation and rotation for dev/CI
- Add certificate expiry monitoring and alerting

## Architecture

- Each backend service runs as-is (HTTP) on its internal port.
- A sidecar TLS proxy (Caddy) listens on `:8443` per backend and enforces client certificate auth against a local dev CA.
- The gateway uses a client certificate to call the proxies over HTTPS.
- Toggle behavior:
  - `ENABLE_MTLS=false` (default): gateway calls plain HTTP `*-svc` directly (backward compatible)
  - `ENABLE_MTLS=true`: gateway calls HTTPS `*-svc-proxy:8443` using client certs and the CA bundle

## Files and scripts

- `ops/tls/gen-certs.sh`: generates a dev CA, a gateway client cert, and per-service server certs
  - Output directory: `ops/tls/out` (gitignored)
- `ops/mtls/Caddyfile.<service>`: Caddy v2 configs that:
  - terminate TLS with server certs
  - require and verify client certs signed by the dev CA
  - reverse-proxy to the local backend HTTP port

## How to enable mTLS locally

1. Generate certificates (creates `ops/tls/out/*`):

   ```bash
   bash ops/tls/gen-certs.sh
   ```

2. Start the stack with mTLS enabled:

   ```bash
   ENABLE_MTLS=true docker compose up -d --build \
     identity-svc identity-svc-proxy \
     shipment-svc shipment-svc-proxy \
     pricing-svc pricing-svc-proxy \
     label-svc label-svc-proxy \
     tracking-svc tracking-svc-proxy \
     gateway-api otel-collector jaeger prometheus alertmanager grafana loki promtail pyroscope blackbox
   ```

   Notes:
   - Gateway mounts `/run/mtls` from `ops/tls/out` and sets `GATEWAY_MTLS_*` envs.
   - Gateway loads a global TLS client from these files and talks to `https://*-svc-proxy:8443` by default when `ENABLE_MTLS=true`.

3. Validate:
   - Gateway routes should work end-to-end.
   - Direct calls without a valid client cert to `https://identity-svc-proxy:8443/health` should be rejected (401/403/TLS error), confirming client cert enforcement.

## CI usage

- Use `ops/tls/gen-certs.sh` in a CI step to create ephemeral CA and certs.
- Start the stack with `ENABLE_MTLS=true` and include the `*-svc-proxy` services.
- The current perf workflow can be extended to turn on `ENABLE_MTLS` if desired.

Example (snippet to insert into a job step):

```bash
bash ops/tls/gen-certs.sh
export ENABLE_MTLS=true
# docker compose up -d ... (include *-svc-proxy services)
```

## Certificate rotation (dev/CI)

1. Generate a new CA + certs (or reuse CA and rotate leaf certs only):
   - Run `bash ops/tls/gen-certs.sh` to refresh artifacts
2. Rolling update:
   - Restart proxy containers to pick up the new server certs
   - Redeploy/restart gateway to pick up the new client cert
3. Validation:
   - Confirm gateway calls succeed
   - Check Prometheus/Grafana for TLS expiry metrics and alerts (see below)

## Expiry monitoring and alerting

- Compose now runs `prom/blackbox-exporter` and Prometheus scrapes HTTPS endpoints of the proxies.
- Alerts in `ops/alerts/tls.yml` fire when time to expiry is <14d (warning) and <3d (critical).
- Tune thresholds as needed.

- Note: The blackbox exporter uses an mTLS-enabled module and mounts the dev CA and gateway client cert/key to probe TLS endpoints that require client authentication, enabling certificate expiry metrics.

## Troubleshooting

- Gateway returns 502/UNABLE_TO_VERIFY_LEAF_SIGNATURE:
  - Ensure `GATEWAY_MTLS_CA_FILE` points to `ca.crt` and is mounted
  - Ensure `ENABLE_MTLS=true`
- Gateway TLS handshake failure:
  - Verify client cert/key files exist and match (`gateway-client.crt`/`.key`)
  - Check proxy logs (Caddy) for client auth errors
- Proxies deny request:
  - Ensure the gateway uses the client certificate and the CA used to sign it matches the proxy trust store

## Security notes

- The dev CA and certs are for local/dev and CI only; rotate frequently and never use in production.
- For production, use your organization’s PKI or a secrets manager for key/cert distribution.
- The application code in backends remains unchanged; TLS enforcement occurs in sidecars, simplifying operational consistency.

