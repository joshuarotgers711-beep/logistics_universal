# identity-svc (Go)

- Port: 8081
- Health: /health, /ready
- Endpoints: /v1/auth/login, /v1/auth/refresh, /v1/auth/mfa/verify, /v1/users
- JWT: HS256 with JWT_SECRET

## Run locally
```
go run ./cmd/server
```

## Env
- PORT (default 8081)
- JWT_SECRET (default devsecret)
- REDIS_URL (optional)
- DATABASE_URL (optional)

