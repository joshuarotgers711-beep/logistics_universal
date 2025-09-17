import { Body, Controller, Post, Req, Res } from '@nestjs/common';
import { Response, Request } from 'express';
import { ProxyService } from '../proxy/proxy.service';
import { config } from '../config';

@Controller('/v1/auth')
export class AuthController {
  constructor(private readonly proxy: ProxyService) {}

  @Post('/login')
  async login(@Body() body: any, @Res() res: Response) {
    const p = await this.proxy.post(config.identityUrl, '/v1/auth/login', body);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('/refresh')
  async refresh(@Body() body: any, @Res() res: Response) {
    const p = await this.proxy.post(config.identityUrl, '/v1/auth/refresh', body);
    res.status(p.status).contentType('application/json').send(p.body);
  }

  @Post('/mfa/verify')
  async mfaVerify(@Body() body: any, @Res() res: Response) {
    const p = await this.proxy.post(config.identityUrl, '/v1/auth/mfa/verify', body);
    res.status(p.status).contentType('application/json').send(p.body);
  }
}

