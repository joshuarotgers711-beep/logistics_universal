#!/usr/bin/env python3
import json, time, sys, os
from urllib import request, error

GW = 'http://localhost:8086'
TENANT = os.getenv('TENANT_ID', 'tenant1')

def http_json(method, url, body=None, headers=None):
    data = None
    hdrs = headers.copy() if headers else {}
    if body is not None:
        data = json.dumps(body).encode('utf-8')
        hdrs.setdefault('Content-Type', 'application/json')
    req = request.Request(url, data=data, method=method, headers=hdrs)
    with request.urlopen(req, timeout=15) as resp:
        return resp.status, resp.read().decode('utf-8', 'replace')

def login_token():
    s, b = http_json('POST', f'{GW}/v1/auth/login', {"email":"admin@example.com","password":"admin"})
    if s != 200:
        raise RuntimeError(f'login failed: {s} {b[:120]}')
    j = json.loads(b)
    return j['AccessToken']

def create_shipment(token):
    payload = {
        "shipperId": "shipper1",
        "serviceLevel": "GROUND",
        "from": {"line1":"1","city":"A","state":"MA","postalCode":"02101","country":"US"},
        "to": {"line1":"2","city":"B","state":"WA","postalCode":"98101","country":"US"},
        "value": 100,
        "packages": [{"weightKg": 1}]
    }
    hdrs = {
        'Authorization': f'Bearer {token}',
        'X-Idempotency-Key': f'mtls-burn-ship-{int(time.time()*1000)}',
        'X-Tenant-Id': TENANT
    }
    s, b = http_json('POST', f'{GW}/v1/shipments', payload, hdrs)
    try:
        return json.loads(b).get('id')
    except Exception:
        return None

def post_rate(token, sid):
    payload = {
        "serviceLevel": "GROUND",
        "from": {"line1":"1","city":"A","state":"MA","postalCode":"02101","country":"US"},
        "to": {"line1":"2","city":"B","state":"WA","postalCode":"98101","country":"US"},
        "packages": [{"weightKg": 1}]
    }
    hdrs = {
        'Authorization': f'Bearer {token}',
        'X-Tenant-Id': TENANT,
        'X-Idempotency-Key': f'mtls-burn-{int(time.time()*1000)}'
    }
    try:
        http_json('POST', f'{GW}/v1/shipments/{sid}/rate', payload, hdrs)
    except Exception:
        pass

if __name__ == '__main__':
    minutes = 13
    if len(sys.argv) > 1:
        try:
            minutes = int(sys.argv[1])
        except Exception:
            pass
    token = login_token()
    sid = create_shipment(token)
    if not sid:
        print('failed to create shipment; exiting', file=sys.stderr)
        sys.exit(1)
    t0 = time.time()
    t_refresh = t0
    end = t0 + minutes*60
    while time.time() < end:
        # ~10 req/sec
        for _ in range(10):
            post_rate(token, sid)
        time.sleep(1)
        now = time.time()
        if now - t_refresh > 300:
            try:
                token = login_token()
                t_refresh = now
            except Exception:
                pass
    print('mtls_burn complete')

