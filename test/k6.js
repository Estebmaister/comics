// @ts-nocheck
// k6 run --iterations 10 test/k6.js
import { check } from 'k6'; // sleep
import http from 'k6/http';
import { getCurrentStageIndex } from 'https://jslib.k6.io/k6-utils/1.3.0/index.js';

const apiUrl = 'http://localhost:8081/login';
const testPass = '1234';

export const options = {
  // define thresholds
  thresholds: {
    http_req_failed: ['rate<0.01'], // http errors should be less than 1%
    http_req_duration: ['p(99)<500'], // 99% of requests should be below 500ms
  },
  // define scenarios
  scenarios: {
    // arbitrary name of scenario
    average_load: {
      executor: 'ramping-vus',
      stages: [
        { duration: '10s', target: 20 }, // Ramp up to 20 users over 10 seconds
        { duration: '30s', target: 50 }, // Stay at 50 users for 30 seconds
        { duration: '10s', target: 30 }, // Ramp down to 30 users over 10 second
      ],
    },
  },
};

// eslint-disable-next-line import/no-anonymous-default-export
export default function () {
  if (getCurrentStageIndex() === 1) {
    console.log('Running the second stage where the expected target is 50');
  }

  const payload = JSON.stringify({
    username: 'test_case',
    password: testPass,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const res = http.post(apiUrl, payload, params);

  check(res, {
    'status is 200': r => r.status === 200,
    'transaction time is OK': r => r.timings.duration < 500,
  });

  // sleep(1); // Add a short sleep to control the request rate
}