#!/usr/bin/env python3
import json, time, sys
from urllib import request, error, parse

GW = 'http://localhost:8086'
JAEGER = 'http://localhost:16686'
PROM = 'http://localhost:9090'

def http(method: str, path: str, body=None, headers=None, base=GW):
    url = base + path
    data = None
    hdrs = headers.copy() if headers else {}
    if body is not None:
        data = json.dumps(body).encode('utf-8')
        hdrs.setdefault('Content-Type', 'application/json')
    req = request.Request(url, data=data, method=method, headers=hdrs)
    try:
        with request.urlopen(req, timeout=20) as resp:
            text = resp.read().decode('utf-8', 'replace')
            return resp.status, text
    except error.HTTPError as e:
        try:
            return e.code, e.read().decode('utf-8', 'replace')
        except Exception:
            return e.code, str(e)
    except Exception as e:
        return 0, str(e)

def promq(q: str):
    qs = parse.urlencode({'query': q})
    s, b = http('GET', f'/api/v1/query?{qs}', None, None, base=PROM)
    if s != 200:
        return {'status':'error','body':b}
    try:
        return json.loads(b)
    except Exception:
        return {'status':'error','body':b}

def jaeger_traces(service='gateway-api', limit=5, lookback='1h'):
    params = {'service': service, 'limit': str(limit), 'lookback': lookback}
    qs = parse.urlencode(params)
    s, b = http('GET', f'/api/traces?{qs}', None, None, base=JAEGER)
    if s != 200:
        return {'status':'error','body':b}
    try:
        j = json.loads(b)
        return j
    except Exception:
        return {'status':'error','body':b}

def main():
    print('== mTLS E2E test (gateway -> proxies -> services) ==')

    # 0) Gateway health (sanity)
    s, b = http('GET', '/health')
    print(f"[gateway health] status={s} body={b.strip()[:120]}")

    # 1) Login
    login_body = {"email":"admin@example.com","password":"admin"}
    s, b = http('POST', '/v1/auth/login', login_body)
    print(f"[login] status={s}")
    print(b[:500])
    if s != 200:
        print('Login failed; aborting.')
        sys.exit(1)
    try:
        token = json.loads(b).get('AccessToken')
    except Exception:
        print('Could not parse AccessToken from login response; aborting.')
        sys.exit(1)
    if not token:
        print('Missing AccessToken in login response; aborting.')
        sys.exit(1)

    auth_headers = {
        'Authorization': f'Bearer {token}',
        'X-Idempotency-Key': f'test-{int(time.time())}',
        'X-Tenant-Id': 'tenant1',
    }

    # 2) Create shipment
    shipment_payload = {
        "shipperId": "shipper1",
        "serviceLevel": "GROUND",
        "from": {"line1":"123 Main St","city":"Boston","state":"MA","postalCode":"02101","country":"US"},
        "to": {"line1":"456 Oak Ave","city":"Seattle","state":"WA","postalCode":"98101","country":"US"},
        "value": 100,
        "packages": [{"weightKg": 1.5, "lengthCm": 20, "widthCm": 15, "heightCm": 10}],
    }
    s, b = http('POST', '/v1/shipments', shipment_payload, auth_headers)
    print(f"[create shipment] status={s}")
    print(b[:800])
    if s not in (200, 201):
        print('Shipment create failed; aborting.')
        sys.exit(1)
    sid = None
    try:
        resp = json.loads(b)
        for k in ('id','shipmentId','shipment_id'):
            if isinstance(resp, dict) and k in resp:
                sid = resp[k]
                break
        if not sid and isinstance(resp, dict):
            data = resp.get('data')
            if isinstance(data, dict):
                sid = data.get('id') or data.get('shipmentId')
    except Exception:
        pass
    if not sid:
        print('Could not parse shipment id; continuing with label/quote may fail.')

    # 3) Pricing quote
    quote_payload = {
        "serviceLevel": "GROUND",
        "from": shipment_payload['from'],
        "to": shipment_payload['to'],
        "packages": shipment_payload['packages'],
    }
    rate_path = f"/v1/shipments/{sid}/rate" if sid else "/v1/pricing/quote"
    s, b = http('POST', rate_path, quote_payload, auth_headers)
    print(f"[pricing quote] status={s}")
    print(b[:800])

    # 4) Create label
    label_path = f"/v1/shipments/{sid}/label" if sid else "/v1/labels"
    label_body = {"format":"PDF"}
    s, b = http('POST', label_path, label_body, auth_headers)
    print(f"[create label] status={s}")
    print(b[:800])
    # Small multi-tenant traffic burst to populate Prom metrics (client/server)
    print('\n== Traffic burst for metrics (multi-tenant) ==')
    tenants = ['tenant1', 'tenant2', 'tenant3']
    for t in tenants:
        auth_headers['X-Tenant-Id'] = t
        for i in range(5):
            auth_headers['X-Idempotency-Key'] = f"mtls-{t}-{int(time.time())}-{i}"
            http('POST', rate_path, quote_payload, auth_headers)
            http('POST', label_path, label_body, auth_headers)
            http('GET', '/health')
            # Optional: broaden coverage; ignore errors if route not present
            http('GET', '/v1/shipments', None, auth_headers)
            time.sleep(0.1)


    # 5) Observability checks
    print('\n== Observability checks ==')
    # Jaeger traces for gateway-api
    jt = jaeger_traces('gateway-api', limit=5, lookback='1h')
    tids = []
    try:
        for tr in jt.get('data', []) or []:
            tid = tr.get('traceID') or tr.get('traceId')
            if tid:
                tids.append(tid)
    except Exception:
        pass
    print(f"[jaeger] recent gateway-api traces: {tids[:5]}")

    # Prometheus metrics
    def q_and_print(name, expr):
        j = promq(expr)
        try:
            vec = j.get('data', {}).get('result', [])
            vals = [v['value'][1] for v in vec[:3]] if isinstance(vec, list) else []
            print(f"[prom] {name}: count={len(vec)} sample={vals}")
        except Exception:
            print(f"[prom] {name}: error {j}")

    q_and_print('tls_expiry', '(probe_ssl_earliest_cert_expiry - time())/86400')

    # Generic (may be empty depending on exporter naming)
    q_and_print('http_client_requests_5m', 'sum(rate(http_client_requests[5m]))')
    q_and_print('http_server_requests_5m', 'sum(rate(http_server_requests[5m]))')

    # OTEL generic histogram counts
    q_and_print('http_client_duration_count_5m', 'sum(rate(http_client_duration_count[5m]))')
    q_and_print('http_server_duration_count_5m', 'sum(rate(http_server_duration_count[5m]))')

    # Environment-specific names discovered via Prom label values
    q_and_print('http_client_requests_total_5m', 'sum(rate(http_client_requests_total[5m]))')
    q_and_print('http_server_requests_total_5m', 'sum(rate(http_server_requests_total[5m]))')
    q_and_print('http_client_req_dur_s_count_5m', 'sum(rate(http_client_request_duration_seconds_count[5m]))')
    q_and_print('http_server_dur_ms_count_5m', 'sum(rate(http_server_duration_milliseconds_count[5m]))')

    # Quantiles from histogram buckets (env-specific)
    q_and_print('http_client_p50', 'histogram_quantile(0.5, sum(rate(http_client_request_duration_seconds_bucket[5m])) by (le))')
    q_and_print('http_server_p50', 'histogram_quantile(0.5, sum(rate(http_server_duration_milliseconds_bucket[5m])) by (le))')

    print('\n== DONE ==')

if __name__ == '__main__':
    main()

