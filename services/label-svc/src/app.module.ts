import { Module } from '@nestjs/common';
import { APP_GUARD } from '@nestjs/core';
import { HealthController } from './health.controller';
import { LabelController } from './label.controller';
import { InternalJwtGuard } from './auth/internal.guard';

@Module({
  imports: [],
  controllers: [HealthController, LabelController],
  providers: [
    { provide: APP_GUARD, useClass: InternalJwtGuard },
  ],
})
export class AppModule {}

