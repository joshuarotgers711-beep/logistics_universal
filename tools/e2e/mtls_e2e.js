#!/usr/bin/env node
const http = require('http');

const GW = 'http://localhost:8086';

async function httpReq(method, path, body, headers={}) {
  const url = new URL(path, GW);
  const payload = body ? JSON.stringify(body) : null;
  const opts = {
    method,
    headers: { ...(payload ? { 'Content-Type': 'application/json' } : {}), ...headers },
  };
  return await new Promise((resolve) => {
    const req = http.request(url, opts, (res) => {
      const chunks = [];
      res.on('data', (c) => chunks.push(c));
      res.on('end', () => {
        resolve({ status: res.statusCode, body: Buffer.concat(chunks).toString('utf8') });
      });
    });
    req.on('error', (e) => resolve({ status: 0, body: String(e.message) }));
    if (payload) req.write(payload);
    req.end();
  });
}

(async () => {
  console.log('== mTLS E2E (Node) ==');
  let r = await httpReq('GET', '/health');
  console.log(`[gateway health] status=${r.status} body=${(r.body||'').slice(0,120)}`);

  r = await httpReq('POST', '/v1/auth/login', { email: 'admin@example.com', password: 'admin' });
  console.log(`[login] status=${r.status}`); console.log(r.body.slice(0,500));
  if (r.status !== 200) process.exit(1);
  const token = JSON.parse(r.body).AccessToken;
  const headers = {
    Authorization: `Bearer ${token}`,
    'X-Idempotency-Key': `test-${Date.now()}`,
    'X-Tenant-Id': 'tenant1',
  };

  const shipment = {
    shipperId: 'shipper1',
    serviceLevel: 'GROUND',
    from: { line1: '123 Main St', city: 'Boston', state: 'MA', postalCode: '02101', country: 'US' },
    to: { line1: '456 Oak Ave', city: 'Seattle', state: 'WA', postalCode: '98101', country: 'US' },
    value: 100,
    packages: [{ weightKg: 1.5, lengthCm: 20, widthCm: 15, heightCm: 10 }],
  };

  r = await httpReq('POST', '/v1/shipments', shipment, headers);
  console.log(`[create shipment] status=${r.status}`); console.log(r.body.slice(0,800));
  if (r.status !== 200 && r.status !== 201) process.exit(1);
  let sid = null;
  try {
    const j = JSON.parse(r.body);
    sid = j.id || j.shipmentId || (j.data && (j.data.id || j.data.shipmentId)) || null;
  } catch {}

  const quote = {
    serviceLevel: 'GROUND',
    from: shipment.from,
    to: shipment.to,
    packages: shipment.packages,
  };
  r = await httpReq('POST', sid ? `/v1/shipments/${sid}/rate` : '/v1/pricing/quote', quote, headers);
  console.log(`[pricing quote] status=${r.status}`); console.log(r.body.slice(0,800));

  r = await httpReq('POST', sid ? `/v1/shipments/${sid}/label` : '/v1/labels', { format: 'PDF' }, headers);
  console.log(`[create label] status=${r.status}`); console.log(r.body.slice(0,800));

  console.log('== DONE ==');
})();

