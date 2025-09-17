import { Controller, Get, Header, Headers, Query, Res } from '@nestjs/common';
import { Response } from 'express';
import { context, propagation, trace } from '@opentelemetry/api';

function enabled(): boolean {
  return (process.env.DEMO_ALERTS || '').toLowerCase() === 'true';
}

function verifyToken(headers: Record<string, string | string[] | undefined>): boolean {
  const required = process.env.DEMO_ALERTS_TOKEN || '';
  if (!required) return false;
  const token = (headers['x-demo-token'] || headers['X-Demo-Token']) as string | undefined;
  return !!token && token === required;
}

function setTenantBaggage(tenantId?: string) {
  if (!tenantId) return;
  const bag = propagation.createBaggage({ 'tenant.id': { value: String(tenantId) } });
  const ctx = propagation.setBaggage(context.active(), bag);
  // Attach for the remainder of the handler
  (context as any).attach?.(ctx);
  const span = trace.getSpan(ctx);
  span?.setAttribute('tenant.id', String(tenantId));
}

@Controller()
export class DemoController {
  @Get('/__demo/5xx')
  @Header('Cache-Control', 'no-store')
  async fivexx(
    @Query('status') statusStr = '500',
    @Query('duration_s') durationStr = '130',
    @Query('tenant_id') tenantId?: string,
    @Headers() headers?: Record<string, string>,
    @Res() res?: Response,
  ) {
    if (!enabled() || !verifyToken(headers || {})) {
      return res?.status(404).send({ error: 'not found' });
    }

    const status = Math.max(500, Math.min(599, parseInt(String(statusStr), 10) || 500));
    const durationSec = Math.max(1, parseInt(String(durationStr), 10) || 130);

    setTenantBaggage(tenantId);

    // Stamp route and debug if header present
    const span = trace.getSpan(context.active());
    span?.setAttribute('http.route', '/__demo/5xx');
    const debug = (headers?.['x-debug-trace'] || headers?.['X-Debug-Trace']) as string | undefined;
    if (debug) span?.setAttribute('debug', 'true');

    // Keep behavior simple: always return configured 5xx; CI will loop for duration
    res?.status(status).send({ error: 'demo 5xx', status, duration_s: durationSec });
  }

  @Get('/__demo/slow')
  @Header('Cache-Control', 'no-store')
  async slow(
    @Query('delay_ms') delayMsStr = '800',
    @Query('duration_s') durationStr = '130',
    @Query('tenant_id') tenantId?: string,
    @Headers() headers?: Record<string, string>,
    @Res() res?: Response,
  ) {
    if (!enabled() || !verifyToken(headers || {})) {
      return res?.status(404).send({ error: 'not found' });
    }

    const delayMs = Math.max(1, parseInt(String(delayMsStr), 10) || 800);
    const durationSec = Math.max(1, parseInt(String(durationStr), 10) || 130);

    setTenantBaggage(tenantId);

    // Stamp route and debug
    const span = trace.getSpan(context.active());
    span?.setAttribute('http.route', '/__demo/slow');
    const debug = (headers?.['x-debug-trace'] || headers?.['X-Debug-Trace']) as string | undefined;
    if (debug) span?.setAttribute('debug', 'true');

    // Optional minimal rate limit
    const limit = parseInt(process.env.DEMO_RPS_LIMIT || '10', 10);
    // (Not implementing shared counter to keep lightweight; CI calls every 2s.)

    await new Promise((r) => setTimeout(r, delayMs));
    res?.status(200).send({ ok: true, delay_ms: delayMs, duration_s: durationSec });
  }
}

