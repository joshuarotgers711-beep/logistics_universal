import { Module } from '@nestjs/common';
import { APP_GUARD } from '@nestjs/core';
import { JwtAuthGuard } from './auth/jwt.guard';
import { RolesGuard } from './auth/roles.guard';
import { HealthController } from './health.controller';
import { AuthController } from './auth/auth.controller';
import { ShipmentsController } from './shipments.controller';
import { EventsController } from './events.controller';
import { TrackController } from './track.controller';
import { ProxyService } from './proxy/proxy.service';
import { DemoController } from './demo.controller';
import { DebugController } from './debug.controller';

@Module({
  imports: [],
  controllers: [HealthController, AuthController, ShipmentsController, EventsController, TrackController, DemoController, DebugController],
  providers: [
    ProxyService,
    { provide: APP_GUARD, useClass: JwtAuthGuard },
    { provide: APP_GUARD, useClass: RolesGuard },
  ],
})
export class AppModule {}

