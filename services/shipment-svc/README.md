# shipment-svc (Go)

- Port: 8082
- Health: /health, /ready
- Endpoints: /v1/shipments (POST, GET, PATCH)
- DB: PostgreSQL
- MQ: RabbitMQ (outbox placeholder)

## Run locally
```
go run ./cmd/server
```

## Env
- PORT (8082)
- DATABASE_URL (postgres connection)
- RABBITMQ_URL (amqp connection)

