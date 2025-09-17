import { flowOnce, headers, GW } from './helpers.js';

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '2m',
  thresholds: {
    http_req_failed: ['rate<=0.01'],
    http_req_duration: ['p(95)<=600', 'p(99)<=1000'],
  },
};

export default function () {
  flowOnce();
}

