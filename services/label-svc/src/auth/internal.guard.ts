import { CanActivate, ExecutionContext, Injectable } from '@nestjs/common';
import * as jwt from 'jsonwebtoken';
import * as fs from 'fs';

function secret(): string {
  const fileEnv = process.env.INTERNAL_JWT_SECRET_FILE;
  if (fileEnv && fs.existsSync(fileEnv)) {
    try { return fs.readFileSync(fileEnv, 'utf8').trim(); } catch {}
  }
  return process.env.INTERNAL_JWT_SECRET || process.env.JWT_SECRET || 'devsecret';
}

function isOpenPath(path: string): boolean {
  return path === '/health' || path === '/ready';
}

@Injectable()
export class InternalJwtGuard implements CanActivate {
  canActivate(context: ExecutionContext): boolean {
    const req = context.switchToHttp().getRequest();
    const path: string = req.path || req.url || '';
    if (isOpenPath(path)) return true;
    const auth: string | undefined = req.headers['authorization'] || req.headers['Authorization'];
    if (!auth || typeof auth !== 'string' || !auth.startsWith('Bearer ')) return false;
    const token = auth.substring('Bearer '.length);
    try {
      const decoded = jwt.verify(token, secret());
      return !!decoded;
    } catch {
      return false;
    }
  }
}

