import * as jwt from 'jsonwebtoken';
import { config } from '../config';

export function signInternalToken(subject: string, extra: Record<string, any> = {}): string {
  const payload = {
    sub: subject,
    iss: 'gateway-api',
    typ: 'internal',
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 60, // 1 minute TTL
    ...extra,
  };
  return jwt.sign(payload, config.internalJwtSecret);
}

