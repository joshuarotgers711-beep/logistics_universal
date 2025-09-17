import { Body, Controller, Get, Param, Patch, Post, Res, Req } from '@nestjs/common';
import { Request, Response } from 'express';
import { ProxyService } from './proxy/proxy.service';
import { config } from './config';

@Controller('/v1/shipments')
export class ShipmentsController {
  constructor(private readonly proxy: ProxyService) {}

  @Post()
  async create(@Body() body: any, @Res() res: Response, @Req() req: Request) {
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const idem = req.header('x-idempotency-key'); if (idem) eh['x-idempotency-key'] = idem;
    const p = await this.proxy.post(config.shipmentUrl, '/v1/shipments', body, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Get('/:id')
  async get(@Param('id') id: string, @Res() res: Response, @Req() req: Request) {
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.get(config.shipmentUrl, `/v1/shipments/${id}`, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Patch('/:id')
  async patch(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.patch(config.shipmentUrl, `/v1/shipments/${id}`, body, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('/:id/rate')
  async rate(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.post(config.pricingUrl, '/v1/pricing/quote', body, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('/:id/label')
  async label(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const payload = { shipmentId: id, format: body?.format ?? 'PDF' };
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.post(config.labelUrl, '/v1/labels', payload, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }
}

