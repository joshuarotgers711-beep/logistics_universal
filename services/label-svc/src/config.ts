import * as fs from 'fs';

function readSecret(fileEnv: string | undefined, env: string | undefined, fallback: string): string {
  if (fileEnv && fs.existsSync(fileEnv)) {
    try { return fs.readFileSync(fileEnv, 'utf8').trim(); } catch {}
  }
  return env || fallback;
}

export const config = {
  shipmentUrl: process.env.SHIPMENT_URL || 'http://shipment-svc:8082',
  internalJwtSecret: readSecret(process.env.INTERNAL_JWT_SECRET_FILE, process.env.INTERNAL_JWT_SECRET || process.env.JWT_SECRET, 'devsecret'),
};

