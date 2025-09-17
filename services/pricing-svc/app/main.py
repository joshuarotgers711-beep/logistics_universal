from fastapi import FastAPI, Depends, HTTPException, status, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
import os
import hmac
import hashlib
import base64
import json
import time
import uuid
import logging
from typing import Callable

import psycopg2
import redis

from .tracing import init_tracing
from opentelemetry.trace import get_current_span
from opentelemetry import baggage
from opentelemetry import context as context_api
from opentelemetry import metrics as ot_metrics

# structured logger for request lines
logger = logging.getLogger("app.request")
if not logger.handlers:
    h = logging.StreamHandler()
    h.setFormatter(logging.Formatter('%(message)s'))
    logger.addHandler(h)
logger.setLevel(logging.INFO)

app = FastAPI(title="pricing-svc")
init_tracing(app)

# Metrics instruments (exported via OTLP by tracing.init_tracing)
meter = ot_metrics.get_meter("pricing-svc")
REQ_DURATION = meter.create_histogram("http.server.duration", unit="ms")
REQ_ACTIVE = meter.create_up_down_counter("http.server.active_requests")
REQ_ERRORS = meter.create_counter("http.server.errors")
REQ_TOTAL = meter.create_counter("http.server.requests")
KPI_QUOTES = meter.create_counter("pricing_quotes_total")
HEALTH_TOTAL = meter.create_counter("pricing_health_check_total")
HEALTH_LATENCY = meter.create_histogram("pricing_health_check_latency_ms", unit="ms")

# --- Request logging with trace correlation, baggage, and metrics ---
@app.middleware("http")
async def log_requests(request: Request, call_next: Callable):
    start = time.time()
    rid = request.headers.get("x-request-id") or uuid.uuid4().hex
    # ensure correlation header is set
    request.state.request_id = rid

    path = request.url.path
    method = request.method

    # Baggage propagation: tenant id + debug flag (optional)
    tenant_id = request.headers.get("x-tenant-id")
    debug_trace = request.headers.get("x-debug-trace")
    # set baggage entries into current context for downstream propagation
    ctx = None
    if tenant_id:
        ctx = baggage.set_baggage("tenant.id", tenant_id)
    if debug_trace:
        ctx = baggage.set_baggage("debug", "true", context=ctx)
    if ctx is not None:
        try:
            context_api.attach(ctx)
        except Exception:
            pass

    # Capture current span early so we keep context
    pre_span = get_current_span()

    # Active requests +1
    try:
        REQ_ACTIVE.add(1, {"http.method": method, "http.route": path})
    except Exception:
        pass

    response = None
    try:
        response = await call_next(request)
        return response
    finally:
        dur_ms = int((time.time() - start) * 1000)
        # Prefer end-of-request span; fall back to pre_span
        span = get_current_span() or pre_span
        sc = span.get_span_context() if span else None
        trace_id = f"{sc.trace_id:032x}" if sc and sc.trace_id else ""
        span_id = f"{sc.span_id:016x}" if sc and sc.span_id else ""
        # domain attributes
        if span is not None:
            if tenant_id:
                span.set_attribute("tenant.id", tenant_id)
            span.set_attribute("http.route", path)
            if debug_trace:
                span.set_attribute("debug", "true")
        # add ids to response
        if response is not None:
            response.headers["X-Request-Id"] = rid
            response.headers["X-Correlation-Id"] = rid
        client_ip = request.headers.get("x-forwarded-for") or (request.client.host if request.client else "")
        status_code = getattr(response, 'status_code', 0) if response is not None else 0
        resp_len = response.headers.get('content-length') if response is not None else ''
        logger.info(
            f"request service=pricing-svc request_id={rid} trace_id={trace_id} span_id={span_id} method={method} path={path} status={status_code} duration_ms={dur_ms} response_bytes={resp_len} client_ip={client_ip}"
        )
        # Metrics
        try:
            attrs = {"http.method": method, "http.route": path, "http.status_code": str(status_code)}
            REQ_TOTAL.add(1, attrs)
            REQ_DURATION.record(dur_ms, attrs)
            if status_code >= 500:
                REQ_ERRORS.add(1, attrs)
        except Exception:
            pass
        # Active requests -1
        try:
            REQ_ACTIVE.add(-1, {"http.method": method, "http.route": path})
        except Exception:
            pass


# --- Internal JWT verification (HS256 minimal) ---

def _secret():
    f = os.getenv("INTERNAL_JWT_SECRET_FILE")
    if f and os.path.exists(f):
        try:
            with open(f, 'r') as fh:
                return fh.read().strip()
        except Exception:
            pass
    return os.getenv("INTERNAL_JWT_SECRET") or os.getenv("JWT_SECRET") or "devsecret"

def _b64url_decode(s: str) -> bytes:
    padding = '=' * (-len(s) % 4)
    return base64.urlsafe_b64decode(s + padding)

def verify_jwt(token: str) -> bool:
    try:
        header_b64, payload_b64, sig_b64 = token.split('.')
        signing_input = f"{header_b64}.{payload_b64}".encode()
        expected = hmac.new(_secret().encode(), signing_input, hashlib.sha256).digest()
        sig = _b64url_decode(sig_b64)
        if not hmac.compare_digest(expected, sig):
            return False
        payload = json.loads(_b64url_decode(payload_b64))
        exp = payload.get('exp')
        if exp is not None and int(exp) < int(time.time()):
            return False
        return True
    except Exception:
        return False

def require_internal_auth(request: Request):
    auth = request.headers.get('authorization') or request.headers.get('Authorization')
    if not auth or not auth.startswith('Bearer '):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail='unauthorized')
    token = auth.split(' ', 1)[1]
    if not verify_jwt(token):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail='unauthorized')

class Address(BaseModel):
    line1: str
    city: str
    state: str
    postalCode: str
    country: str

class Package(BaseModel):
    weightKg: float = Field(gt=0)
    lengthCm: float
    widthCm: float
    heightCm: float

class QuoteRequest(BaseModel):
    serviceLevel: str
    from_: Address = Field(alias="from")
    to: Address
    packages: list[Package]

class QuoteResponse(BaseModel):
    currency: str = "USD"
    base: float
    distanceKm: float
    surcharges: list[dict] = []
    total: float

@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/ready")
def ready():
    return {"status": "ready"}

@app.get("/healthz/deep")
def healthz_deep(request: Request):
    overall = "healthy"
    components = {}

    # DB check
    db_url = None
    f = os.getenv("DATABASE_URL_FILE")
    if f and os.path.exists(f):
        try:
            with open(f, 'r') as fh:
                s = fh.read().strip()
                if s:
                    db_url = s
        except Exception:
            pass
    if not db_url:
        db_url = os.getenv("DATABASE_URL")
    db_status = "unhealthy"
    db_latency = 0
    db_error = None
    t0 = time.monotonic()
    try:
        if not db_url:
            raise RuntimeError("DATABASE_URL not set")
        conn = psycopg2.connect(db_url, connect_timeout=2)
        try:
            cur = conn.cursor()
            cur.execute("SELECT 1")
            cur.fetchone()
            db_status = "healthy"
        finally:
            conn.close()
    except Exception as e:
        db_error = str(e)
        overall = "unhealthy"
    finally:
        db_latency = int((time.monotonic() - t0) * 1000)
        try:
            HEALTH_TOTAL.add(1, {"component": "db", "status": db_status})
            HEALTH_LATENCY.record(db_latency, {"component": "db"})
        except Exception:
            pass
        span = get_current_span()
        if span is not None:
            span.set_attribute("health.db.status", db_status)
            span.set_attribute("health.db.latency_ms", db_latency)

    # Redis check
    redis_url = os.getenv("REDIS_URL") or "redis://redis:6379/0"
    redis_status = "unhealthy"
    redis_latency = 0
    redis_error = None
    t1 = time.monotonic()
    try:
        r = redis.Redis.from_url(redis_url, socket_connect_timeout=1, socket_timeout=1)
        r.ping()
        redis_status = "healthy"
    except Exception as e:
        redis_error = str(e)
        overall = "unhealthy"
    finally:
        redis_latency = int((time.monotonic() - t1) * 1000)
        try:
            HEALTH_TOTAL.add(1, {"component": "redis", "status": redis_status})
            HEALTH_LATENCY.record(redis_latency, {"component": "redis"})
        except Exception:
            pass
        span = get_current_span()
        if span is not None:
            span.set_attribute("health.redis.status", redis_status)
            span.set_attribute("health.redis.latency_ms", redis_latency)

    body = {
        "overall": {"status": overall},
        "components": {
            "db": {"status": db_status, "latency_ms": db_latency, **({"error": db_error} if db_error else {})},
            "redis": {"status": redis_status, "latency_ms": redis_latency, **({"error": redis_error} if redis_error else {})},
        },
    }
    status_code = 200 if overall == "healthy" else 503
    return JSONResponse(content=body, status_code=status_code)

@app.post("/v1/pricing/quote", response_model=QuoteResponse)
def quote(req: QuoteRequest, request: Request, _: None = Depends(require_internal_auth)):
    # Placeholder pricing: base on weight sum; fake distance
    total_weight = sum(p.weightKg for p in req.packages)
    base = 5.0 + 1.0 * total_weight
    distance_km = 1000.0
    fuel = round(0.1 * base, 2)
    total = round(base + fuel, 2)

    # KPI metric: pricing_quotes_total (tenant from baggage, fallback to header)
    tenant = baggage.get_baggage("tenant.id")
    if not tenant:
        tenant = request.headers.get("x-tenant-id")
    attrs = {"tenant_id": tenant} if tenant else None
    KPI_QUOTES.add(1, attrs)

    return QuoteResponse(
        base=base,
        distanceKm=distance_km,
        surcharges=[{"type": "fuel", "amount": fuel}],
        total=total,
    )

