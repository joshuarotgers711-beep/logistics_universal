# tracking-svc (Go)

- Port: 8085
- Health: /health, /ready
- Endpoints: /v1/events/*, /v1/track/{trackingNumber}
- DB: TimescaleDB (Postgres) for telemetry (skeleton)
- Geofencing: Haversine calc utility

## Run locally
```
go run ./cmd/server
```

## Env
- PORT (8085)
- DATABASE_URL

