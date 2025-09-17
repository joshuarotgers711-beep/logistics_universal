# Local Development

## Prerequisites
- Docker + Docker Compose
- Node 20 (optional for local runs), Go 1.22, Python 3.11

## Start dependencies and services

1) Build images (first time):
- gateway-api: `docker build -t gateway-api:local services/gateway-api`
- identity-svc: `docker build -t identity-svc:local services/identity-svc`
- shipment-svc: `docker build -t shipment-svc:local services/shipment-svc`
- pricing-svc: `docker build -t pricing-svc:local services/pricing-svc`
- label-svc: `docker build -t label-svc:local services/label-svc`
- tracking-svc: `docker build -t tracking-svc:local services/tracking-svc`

2) Start infra + services:
- `docker-compose up -d postgres redis rabbitmq`
- Run services individually (either via docker-compose build/run or locally):
  - `docker-compose up -d gateway-api identity-svc shipment-svc pricing-svc label-svc tracking-svc`

## Smoke test (via gateway)
- Ensure services are up, then run: `bash scripts/smoke.sh`
- This will: login → create shipment → quote → label → pickup → deliver → track

## Health checks
- gateway-api: http://localhost:8080/health
- identity-svc: http://localhost:8081/health
- shipment-svc: http://localhost:8082/health
- pricing-svc: http://localhost:8083/health
- label-svc: http://localhost:8084/health
- tracking-svc: http://localhost:8085/health

## Environment
Each service has a `.env.example` with defaults for local compose.

## Notes
- Endpoints currently return placeholder responses (except health/ready). Implement business logic per docs and OpenAPI specs in apis/.
- Helm charts are provided for k8s deployments; for local k8s, use manifests under k8s/local/ and set image tags to `*:local`.

