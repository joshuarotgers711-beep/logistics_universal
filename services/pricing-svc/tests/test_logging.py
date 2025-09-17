import base64
import hashlib
import hmac
import json
import time
from fastapi.testclient import TestClient

from app.main import app


def _b64url(x: bytes) -> str:
    return base64.urlsafe_b64encode(x).decode().rstrip("=")


def make_token(secret: str = "devsecret", ttl: int = 60) -> str:
    header = _b64url(json.dumps({"alg": "HS256", "typ": "JWT"}).encode())
    payload = _b64url(json.dumps({"exp": int(time.time()) + ttl}).encode())
    signing_input = f"{header}.{payload}".encode()
    sig = hmac.new(secret.encode(), signing_input, hashlib.sha256).digest()
    return f"{header}.{payload}.{_b64url(sig)}"


def test_request_logs_trace_and_request_id_success(caplog):
    caplog.set_level("INFO")
    client = TestClient(app)
    token = make_token()
    payload = {
        "serviceLevel": "GROUND",
        "from": {"line1": "1 a", "city": "x", "state": "CA", "postalCode": "90001", "country": "US"},
        "to": {"line1": "2 b", "city": "y", "state": "CA", "postalCode": "90002", "country": "US"},
        "packages": [{"weightKg": 1.2, "lengthCm": 10, "widthCm": 10, "heightCm": 10}],
    }
    r = client.post("/v1/pricing/quote", json=payload, headers={"Authorization": f"Bearer {token}", "x-tenant-id": "t1"})
    assert r.status_code == 200
    text = caplog.text
    assert "service=pricing-svc" in text
    assert "request_id=" in text
    assert "trace_id=" in text and "span_id=" in text


def test_request_logs_on_unauthorized(caplog):
    caplog.set_level("INFO")
    client = TestClient(app)
    r = client.post("/v1/pricing/quote", json={})
    assert r.status_code in (401, 422)
    text = caplog.text
    assert "service=pricing-svc" in text
    assert "request_id=" in text
    assert "trace_id=" in text  # present even on errors

