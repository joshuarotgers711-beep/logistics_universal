import { Controller, Get, Res } from '@nestjs/common';
import { Response } from 'express';
import * as fs from 'fs';
import { config } from './config';

@Controller('/__debug')
export class DebugController {
  @Get('/mtls')
  mtls(@Res() res: Response) {
    // Dev-only: require explicit enable to avoid exposing internals by default
    const enabledFlag = (process.env.DEBUG_ENDPOINTS || '').toLowerCase() === 'true';
    if (!enabledFlag) {
      return res.status(404).json({ message: 'not found' });
    }

    const certFile = process.env.GATEWAY_MTLS_CERT_FILE || '';
    const keyFile = process.env.GATEWAY_MTLS_KEY_FILE || '';
    const caFile = process.env.GATEWAY_MTLS_CA_FILE || '';
    const certOk = !!certFile && fs.existsSync(certFile);
    const keyOk = !!keyFile && fs.existsSync(keyFile);
    const caOk = !!caFile && fs.existsSync(caFile);

    const payload = {
      enable_mtls: config.ENABLE_MTLS,
      targets: {
        identityUrl: config.identityUrl,
        shipmentUrl: config.shipmentUrl,
        pricingUrl: config.pricingUrl,
        labelUrl: config.labelUrl,
        trackingUrl: config.trackingUrl,
      },
      client: {
        certPath: certFile,
        keyPath: keyFile,
        caPath: caFile,
        filesPresent: certOk && keyOk && caOk,
      },
      note: 'This endpoint is for development only. Do not enable in production.',
    };
    return res.status(200).json(payload);
  }
}

