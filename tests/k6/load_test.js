import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '10s', target: 2 },
    { duration: '20s', target: 5 },
    { duration: '5s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  const url = __ENV.TARGET_URL || 'http://app:8080';
  
  let res = http.get(`${url}/stats`);
  
  check(res, {
    'status 200': (r) => r.status === 200,
  });
  
  sleep(1);
}
