# label-svc (NestJS)

- Port: 8084
- Health: /health, /ready
- Endpoints: /v1/labels (POST), /v1/labels/{id} (GET)
- Rendering: Puppeteer (Chromium) for PDF/PNG (skeleton)

## Run locally
```
npm i
npm run dev
```

## Env
- PORT (8084)
- JWT_SECRET
- OBJECT_STORAGE_BASE_URL

