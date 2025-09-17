import { Body, Controller, Param, Post, Res, Req } from '@nestjs/common';
import { Request, Response } from 'express';
import { ProxyService } from './proxy/proxy.service';
import { config } from './config';

@Controller('/v1/shipments/:id/events')
export class EventsController {
  constructor(private readonly proxy: ProxyService) {}

  @Post('pickup')
  async pickup(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const payload = { shipmentId: id, ...body };
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.post(config.trackingUrl, '/v1/events/pickup', payload, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('transit')
  async transit(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const payload = { shipmentId: id, ...body };
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.post(config.trackingUrl, '/v1/events/transit', payload, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('deliver')
  async deliver(@Param('id') id: string, @Body() body: any, @Res() res: Response, @Req() req: Request) {
    const payload = { shipmentId: id, ...body };
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.post(config.trackingUrl, '/v1/events/deliver', payload, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }
}

