import { Injectable, NestMiddleware } from '@nestjs/common';
import { context, trace, metrics } from '@opentelemetry/api';
import { randomUUID } from 'crypto';

// Metrics instruments (exported via OTLP once NodeSDK is configured with a MetricReader)
const meter = metrics.getMeter(process.env.OTEL_SERVICE_NAME || 'gateway-api');
const REQ_DURATION = meter.createHistogram('http.server.duration', { unit: 'ms' });
const REQ_ACTIVE = meter.createUpDownCounter('http.server.active_requests');
const REQ_ERRORS = meter.createCounter('http.server.errors');
const REQ_TOTAL = meter.createCounter('http.server.requests');

@Injectable()
export class LoggingMiddleware implements NestMiddleware {
  use(req: any, res: any, next: () => void) {
    const start = process.hrtime.bigint();

    let requestId: string = req.headers['x-request-id'] || req.headers['x-correlation-id'];
    if (!requestId) requestId = randomUUID().replace(/-/g, '');
    res.setHeader('X-Request-Id', requestId);
    res.setHeader('X-Correlation-Id', requestId);

    const ip = (req.headers['x-forwarded-for']?.toString().split(',')[0].trim()) || req.socket?.remoteAddress || '';

    const activeSpan = trace.getSpan(context.active());

    // Domain attributes
    const tenantId = (req.headers['x-tenant-id'] || '').toString();
    const debugTrace = (req.headers['x-debug-trace'] || '').toString();

    // Extract shipmentId from common routes like /v1/shipments/:id/...
    let shipmentId = '';
    const m = (req.originalUrl || req.url || '').match(/\/v1\/shipments\/([^/]+)/);
    if (m && m[1]) shipmentId = m[1];

    if (activeSpan) {
      if (shipmentId) activeSpan.setAttribute('app.shipment_id', shipmentId);
      if (tenantId) activeSpan.setAttribute('tenant.id', tenantId);
      activeSpan.setAttribute('http.route', req.route?.path || '');
      if (debugTrace) activeSpan.setAttribute('debug', 'true');
    }

    // Metrics: increment active requests
    try {
      const attrs: Record<string, string> = {
        'http.method': req.method,
        'http.route': req.route?.path || '',
      };
      if (tenantId) attrs['tenant.id'] = tenantId;
      if (shipmentId) attrs['app.shipment_id'] = shipmentId;
      REQ_ACTIVE.add(1, attrs);
    } catch {}

    const sc = activeSpan?.spanContext();
    const traceId = sc?.traceId || '';
    const spanId = sc?.spanId || '';

    res.on('finish', () => {
      const end = process.hrtime.bigint();
      const durationMs = Number(end - start) / 1e6;
      const contentLen = res.getHeader('content-length');
      let responseBytes = 0;
      if (typeof contentLen === 'string') responseBytes = parseInt(contentLen, 10) || 0;
      else if (typeof contentLen === 'number') responseBytes = contentLen;
      else if (Array.isArray(contentLen) && contentLen.length > 0) responseBytes = parseInt(String(contentLen[0]), 10) || 0;

      const route = req.route?.path || '';
      const service = process.env.OTEL_SERVICE_NAME || process.env.npm_package_name || 'gateway-api';
      const msg = `request service=${service} request_id=${requestId} trace_id=${traceId} span_id=${spanId} method=${req.method} path=${req.originalUrl || req.url} status=${res.statusCode} duration_ms=${Math.round(durationMs)} response_bytes=${responseBytes} client_ip=${ip}`;
      console.log(msg);

      // Metrics: record duration, request count, errors, and decrement active
      try {
        const attrs: Record<string, string> = {
          'http.method': req.method,
          'http.route': route,
          'http.status_code': String(res.statusCode),
        };
        if (tenantId) attrs['tenant.id'] = tenantId;
        if (shipmentId) attrs['app.shipment_id'] = shipmentId;
        REQ_TOTAL.add(1, attrs);
        REQ_DURATION.record(Math.round(durationMs), attrs);
        if (res.statusCode >= 500) REQ_ERRORS.add(1, attrs);
        REQ_ACTIVE.add(-1, { 'http.method': req.method, 'http.route': route });
      } catch {}
    });

    next();
  }
}

