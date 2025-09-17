# pricing-svc (FastAPI)

- Port: 8083
- Health: /health, /ready
- Endpoint: /v1/pricing/quote
- Cache: Redis for hot pricing rules (placeholder)

## Run locally
```
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8083
```

## Env
- PORT (8083)
- REDIS_URL (redis://)

