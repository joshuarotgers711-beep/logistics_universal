import http from 'k6/http';
import { check, sleep } from 'k6';

export const GW = __ENV.GW_URL || 'http://gateway-api:8080';
export const TENANT = __ENV.TENANT_ID || 'demo';

export function headers(extra = {}) {
  return Object.assign({ 'content-type': 'application/json', 'x-tenant-id': TENANT }, extra);
}

export function flowOnce() {
  const h = headers();
  const q = http.get(`${GW}/v1/pricing/quote?weight=1`, { headers: h });
  check(q, { 'quote 200': (r) => r.status === 200 });

  const lbl = http.post(`${GW}/v1/labels`, JSON.stringify({ quoteId: 'q1' }), { headers: h });
  check(lbl, { 'label 201/200': (r) => r.status === 201 || r.status === 200 });

  const idem = Math.random().toString(36).slice(2);
  const sh = http.post(`${GW}/v1/shipments`, JSON.stringify({}), { headers: headers({ 'X-Idempotency-Key': idem }) });
  check(sh, { 'shipment 201/200': (r) => r.status === 201 || r.status === 200 });

  sleep(0.5);
}

