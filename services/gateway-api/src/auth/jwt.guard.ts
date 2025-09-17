import { Injectable, CanActivate, ExecutionContext } from '@nestjs/common';
import * as jwt from 'jsonwebtoken';

@Injectable()
export class JwtAuthGuard implements CanActivate {
  canActivate(context: ExecutionContext): boolean {
    const req = context.switchToHttp().getRequest();
    const openPaths = [
      '/health',
      '/ready',
      '/v1/auth/login',
      '/v1/auth/refresh',
      '/v1/auth/mfa/verify',
      /^\/v1\/track\/.*/,
      // Demo alert exercise endpoints are guarded separately by DEMO_ALERTS and X-Demo-Token
      '/__demo/5xx',
      '/__demo/slow',
    ];
    const path = req.path as string;
    if (openPaths.some((p) => (p instanceof RegExp ? p.test(path) : p === path))) return true;

    const auth = req.headers['authorization'];
    if (!auth || !auth.startsWith('Bearer ')) return false;
    const token = auth.substring(7);
    try {
      const decoded = jwt.verify(token, process.env.JWT_SECRET || 'devsecret');
      (req as any).user = decoded;
      return true;
    } catch (e) {
      return false;
    }
  }
}

