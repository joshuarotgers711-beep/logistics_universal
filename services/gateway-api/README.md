# gateway-api (NestJS)

- Port: 8080
- Health: /health, /ready
- Swagger: /docs
- Auth: JWT Bearer (Authorization: Bearer <token>)
- RBAC: @Roles('SHIPPER'|'DRIVER'|'MANAGER'|'ADMIN') via RolesGuard

## Run locally
```
npm i
npm run dev
```

## Env
- PORT (default 8080)
- JWT_SECRET (default devsecret)
- *-URL service endpoints for proxying

