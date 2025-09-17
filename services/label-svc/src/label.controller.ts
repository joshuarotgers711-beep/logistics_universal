import { Body, Controller, Get, Headers, Param, Post } from '@nestjs/common';
import { randomUUID } from 'crypto';
import { signInternalToken } from './auth/internal-token';
import { config } from './config';
import { metrics, context, propagation } from '@opentelemetry/api';

const meter = metrics.getMeter(process.env.OTEL_SERVICE_NAME || 'label-svc');
const LABELS_GENERATED = meter.createCounter('labels_generated_total', { description: 'Labels generated' });

@Controller()
export class LabelController {
  @Post('/v1/labels')
  async create(@Body() body: any, @Headers() headers: Record<string, string>) {
    const shipmentId: string = body?.shipmentId;
    const format: string = body?.format || 'PDF';
    if (!shipmentId) {
      throw new Error('shipmentId required');
    }

    // Generate label artifact (placeholder)
    const id = randomUUID();
    const url = `https://example.com/labels/${id}.${format.toLowerCase()}`;

    // Persist metadata in shipment-svc
    const token = signInternalToken('label-svc');
    const res = await fetch(`${config.shipmentUrl}/v1/shipments/${shipmentId}/labels`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ id, format, storageUrl: url }),
    });
    if (!res.ok) {
      const t = await res.text().catch(() => '');
      throw new Error(`persist label failed: ${res.status} ${t}`);
    }

    // KPI metric: labels_generated_total (tenant from baggage, fallback to header)
    try {
      const bag = propagation.getBaggage(context.active());
      let tenant = bag?.getEntry('tenant.id')?.value as string | undefined;
      if (!tenant) {
        tenant = (headers['x-tenant-id'] || headers['X-Tenant-Id'] || '') as string;
      }
      const attrs: Record<string, string> = {};
      if (tenant) attrs['tenant_id'] = String(tenant);
      LABELS_GENERATED.add(1, attrs);
    } catch {}

    return { id, format, url };
  }

  @Get('/v1/labels/:id')
  get(@Param('id') id: string) {
    return { id, format: 'PDF', url: 'https://example.com/label.pdf' };
  }
}

