import 'reflect-metadata';
import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { LoggingMiddleware } from './logging.middleware';
import './tracing';

async function bootstrap() {
  const app = await NestFactory.create(AppModule, { logger: ['log', 'error', 'warn'] });
  app.use(new LoggingMiddleware().use);
  const port = process.env.PORT || 8084;
  await app.listen(port);
  console.log(`label-svc listening on ${port}`);
}
bootstrap();

