import * as fs from 'fs';

function readSecret(fileEnv: string | undefined, env: string | undefined, fallback: string): string {
  if (fileEnv && fs.existsSync(fileEnv)) {
    try { return fs.readFileSync(fileEnv, 'utf8').trim(); } catch {}
  }
  return env || fallback;
}

const ENABLE_MTLS = (process.env.ENABLE_MTLS || '').toLowerCase() === 'true';

function pickUrl(stdEnv: string | undefined, stdDefault: string, mtlsEnv: string | undefined, mtlsDefault: string): string {
  if (ENABLE_MTLS) {
    return mtlsEnv || mtlsDefault;
  }
  return stdEnv || stdDefault;
}

export const config = {
  identityUrl: pickUrl(process.env.IDENTITY_URL, 'http://identity-svc:8081', process.env.IDENTITY_URL_MTLS, 'https://identity-svc-proxy:8443'),
  shipmentUrl: pickUrl(process.env.SHIPMENT_URL, 'http://shipment-svc:8082', process.env.SHIPMENT_URL_MTLS, 'https://shipment-svc-proxy:8443'),
  pricingUrl: pickUrl(process.env.PRICING_URL, 'http://pricing-svc:8083', process.env.PRICING_URL_MTLS, 'https://pricing-svc-proxy:8443'),
  labelUrl: pickUrl(process.env.LABEL_URL, 'http://label-svc:8084', process.env.LABEL_URL_MTLS, 'https://label-svc-proxy:8443'),
  trackingUrl: pickUrl(process.env.TRACKING_URL, 'http://tracking-svc:8085', process.env.TRACKING_URL_MTLS, 'https://tracking-svc-proxy:8443'),
  internalJwtSecret: readSecret(process.env.INTERNAL_JWT_SECRET_FILE, process.env.INTERNAL_JWT_SECRET || process.env.JWT_SECRET, 'devsecret'),
  ENABLE_MTLS,
};

