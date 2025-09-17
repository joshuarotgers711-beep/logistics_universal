import { Controller, Get, Param, Res, Req } from '@nestjs/common';
import { Request, Response } from 'express';
import { ProxyService } from './proxy/proxy.service';
import { config } from './config';

@Controller('/v1/track')
export class TrackController {
  constructor(private readonly proxy: ProxyService) {}

  @Get('/:trackingNumber')
  async get(@Param('trackingNumber') trackingNumber: string, @Res() res: Response, @Req() req: Request) {
    const eh: Record<string, string> = {};
    const t = req.header('x-tenant-id'); if (t) eh['x-tenant-id'] = t;
    const d = req.header('x-debug-trace'); if (d) eh['x-debug-trace'] = d;
    const p = await this.proxy.get(config.trackingUrl, `/v1/track/${trackingNumber}`, eh);
    res.status(p.status).contentType('application/json').send(p.body);
  }
}

