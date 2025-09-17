import * as jwt from 'jsonwebtoken';
import { config } from '../config';

export function signInternalToken(subject: string): string {
  const payload = {
    sub: subject,
    iss: 'label-svc',
    typ: 'internal',
    exp: Math.floor(Date.now() / 1000) + 60,
  } as any;
  return jwt.sign(payload, config.internalJwtSecret);
}

