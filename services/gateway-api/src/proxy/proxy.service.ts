import { Injectable, Logger } from '@nestjs/common';
import { config } from '../config';
import { signInternalToken } from '../auth/internal-token';
import { metrics } from '@opentelemetry/api';

// Optional per-request dispatcher to guarantee mTLS usage even if global dispatcher is ignored
let mtlsDispatcher: any | undefined;
let mtlsHttpsAgent: any | undefined;
(function initMtlsClients() {
  try {
    if ((process.env.ENABLE_MTLS || '').toLowerCase() === 'true') {
      const certFile = process.env.GATEWAY_MTLS_CERT_FILE;
      const keyFile = process.env.GATEWAY_MTLS_KEY_FILE;
      const caFile = process.env.GATEWAY_MTLS_CA_FILE;
      if (certFile && keyFile && caFile) {
        // Lazy requires to avoid build-time deps
        const fs = require('fs');
        const { Agent } = require('undici');
        const https = require('https');
        const cert = fs.readFileSync(certFile);
        const key = fs.readFileSync(keyFile);
        const ca = fs.readFileSync(caFile);
        // undici Agent path (kept for environments where it works)
        mtlsDispatcher = new Agent({ connect: { tls: { cert, key, ca, rejectUnauthorized: true } } });
        // Node https.Agent fallback to resolve TLS verify edge-cases
        mtlsHttpsAgent = new https.Agent({ cert, key, ca, rejectUnauthorized: true });
      }
    }
  } catch (_) { /* noop */ }
})();

const meter = metrics.getMeter('gateway-api');
const CLIENT_DURATION = meter.createHistogram('http.client.duration', { unit: 'ms' });
const CLIENT_ERRORS = meter.createCounter('http.client.errors');
const CLIENT_REQUESTS = meter.createCounter('http.client.requests');

function attrsFor(url: string, method: string) {
  try {
    const u = new URL(url);
    const host = u.hostname;
    return { 'http.method': method, 'http.url': u.pathname, 'net.peer.name': host } as Record<string, string>;
  } catch {
    return { 'http.method': method } as Record<string, string>;
  }
}

async function httpRequestViaNode(url: string, method: string, headers: Record<string, string>, body?: string) {
  return await new Promise<{ status: number; body: string; headers: any }>((resolve, reject) => {
    const u = new URL(url);
    const isHttps = u.protocol === 'https:';
    const mod = require(isHttps ? 'https' : 'http');
    const opts: any = {
      hostname: u.hostname,
      port: u.port ? Number(u.port) : (isHttps ? 443 : 80),
      path: u.pathname + (u.search || ''),
      method,
      headers,
    };
    if (isHttps && mtlsHttpsAgent) opts.agent = mtlsHttpsAgent;
    const req = mod.request(opts, (res: any) => {
      const chunks: any[] = [];
      res.on('data', (c: any) => chunks.push(c));
      res.on('end', () => {
        const buf = Buffer.concat(chunks);
        resolve({ status: res.statusCode || 0, body: buf.toString('utf8'), headers: res.headers });
      });
    });
    req.on('error', reject);
    if (body) req.write(body);
    req.end();
  });
}

@Injectable()
export class ProxyService {
  private readonly logger = new Logger(ProxyService.name);

  private authHeader() {
    const token = signInternalToken('gateway');
    return { Authorization: `Bearer ${token}` };
  }

  async get(base: string, path: string, extraHeaders?: Record<string, string>) {
    const url = `${base}${path}`;
    const start = process.hrtime.bigint();
    const a = attrsFor(url, 'GET');
    try {
      const headers: Record<string, string> = { ...this.authHeader(), ...(extraHeaders || {}) };
      // Inject trace context + baggage into downstream request
      try { const { context, propagation } = await import('@opentelemetry/api'); propagation.inject(context.active(), headers); } catch {}
      const useNode = !!(mtlsHttpsAgent && url.startsWith('https://'));
      const res = useNode
        ? await httpRequestViaNode(url, 'GET', headers)
        : await fetch(url, { headers, ...(mtlsDispatcher ? { dispatcher: mtlsDispatcher } : {}) } as any).then(async (r) => ({ status: r.status, body: await r.text(), headers: r.headers }));
      CLIENT_REQUESTS.add(1, { ...a, 'http.status_code': String(res.status) });
      const durMs = Number(process.hrtime.bigint() - start) / 1e6;
      CLIENT_DURATION.record(Math.round(durMs), { ...a, 'http.status_code': String(res.status) });
      if (res.status >= 500) CLIENT_ERRORS.add(1, { ...a, 'http.status_code': String(res.status) });
      return res;
    } catch (err) {
      CLIENT_ERRORS.add(1, { ...a, 'error.type': 'network' });
      throw err;
    }
  }

  async post(base: string, path: string, json: any, extraHeaders?: Record<string, string>) {
    const url = `${base}${path}`;
    const start = process.hrtime.bigint();
    const a = attrsFor(url, 'POST');
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json', ...this.authHeader(), ...(extraHeaders || {}) };
      try { const { context, propagation } = await import('@opentelemetry/api'); propagation.inject(context.active(), headers); } catch {}
      const bodyStr = JSON.stringify(json ?? {});
      const useNode = !!(mtlsHttpsAgent && url.startsWith('https://'));
      const res = useNode
        ? await httpRequestViaNode(url, 'POST', headers, bodyStr)
        : await fetch(url, { method: 'POST', headers, body: bodyStr, ...(mtlsDispatcher ? { dispatcher: mtlsDispatcher } : {}) } as any).then(async (r) => ({ status: r.status, body: await r.text(), headers: r.headers }));
      CLIENT_REQUESTS.add(1, { ...a, 'http.status_code': String(res.status) });
      const durMs = Number(process.hrtime.bigint() - start) / 1e6;
      CLIENT_DURATION.record(Math.round(durMs), { ...a, 'http.status_code': String(res.status) });
      if (res.status >= 500) CLIENT_ERRORS.add(1, { ...a, 'http.status_code': String(res.status) });
      return res;
    } catch (err) {
      CLIENT_ERRORS.add(1, { ...a, 'error.type': 'network' });
      throw err;
    }
  }

  async patch(base: string, path: string, json: any, extraHeaders?: Record<string, string>) {
    const url = `${base}${path}`;
    const start = process.hrtime.bigint();
    const a = attrsFor(url, 'PATCH');
    try {
      const headers: Record<string, string> = { 'Content-Type': 'application/json', ...this.authHeader(), ...(extraHeaders || {}) };
      try { const { context, propagation } = await import('@opentelemetry/api'); propagation.inject(context.active(), headers); } catch {}
      const bodyStr = JSON.stringify(json ?? {});
      const useNode = !!(mtlsHttpsAgent && url.startsWith('https://'));
      const res = useNode
        ? await httpRequestViaNode(url, 'PATCH', headers, bodyStr)
        : await fetch(url, { method: 'PATCH', headers, body: bodyStr, ...(mtlsDispatcher ? { dispatcher: mtlsDispatcher } : {}) } as any).then(async (r) => ({ status: r.status, body: await r.text(), headers: r.headers }));
      CLIENT_REQUESTS.add(1, { ...a, 'http.status_code': String(res.status) });
      const durMs = Number(process.hrtime.bigint() - start) / 1e6;
      CLIENT_DURATION.record(Math.round(durMs), { ...a, 'http.status_code': String(res.status) });
      if (res.status >= 500) CLIENT_ERRORS.add(1, { ...a, 'http.status_code': String(res.status) });
      return res;
    } catch (err) {
      CLIENT_ERRORS.add(1, { ...a, 'error.type': 'network' });
      throw err;
    }
  }
}
