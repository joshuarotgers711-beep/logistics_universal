import 'reflect-metadata';
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger';
import { LoggingMiddleware } from './logging.middleware';
import './tracing';

// Optional global mTLS client for fetch via undici
(function configureMtlsClient() {
  try {
    if ((process.env.ENABLE_MTLS || '').toLowerCase() === 'true') {
      const certFile = process.env.GATEWAY_MTLS_CERT_FILE;
      const keyFile = process.env.GATEWAY_MTLS_KEY_FILE;
      const caFile = process.env.GATEWAY_MTLS_CA_FILE;
      if (certFile && keyFile && caFile) {
        const fs = require('fs');
        const { Agent, setGlobalDispatcher } = require('undici');
        const cert = fs.readFileSync(certFile);
        const key = fs.readFileSync(keyFile);
        const ca = fs.readFileSync(caFile);
        const agent = new Agent({ connect: { tls: { cert, key, ca, rejectUnauthorized: true } } });
        setGlobalDispatcher(agent);
        // eslint-disable-next-line no-console
        console.log('mTLS enabled for outbound requests');
      }
    }
  } catch (e: any) {
    // eslint-disable-next-line no-console
    console.warn('mTLS client setup skipped:', e?.message || e);
  }
})();

async function bootstrap() {
  const app = await NestFactory.create(AppModule, { logger: ['log', 'error', 'warn'] });
  app.use(new LoggingMiddleware().use);
  app.setGlobalPrefix('');

  const config = new DocumentBuilder()
    .setTitle('Gateway API')
    .setVersion('1.0.0')
    .addBearerAuth()
    .build();
  const document = SwaggerModule.createDocument(app, config);
  SwaggerModule.setup('/docs', app, document);

  const port = process.env.PORT || 8080;
  await app.listen(port);
  // eslint-disable-next-line no-console
  console.log(`gateway-api listening on ${port}`);
}
bootstrap();

