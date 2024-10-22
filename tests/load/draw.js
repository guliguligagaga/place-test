import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const BASE_URL = 'https://guliguli.work';
// Replace with a valid JWT token
const TOKEN = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjc2NDE5MDgsImlzcyI6Imdvb2dsZSIsInN1YiI6IjEwNzQxMzE1NTkwNDk2OTIzNzkzMyJ9.WkGwGR6LAjTBPYvjBL4BpS0CGnFOjdPeWmKrUsMW_U8';

export const options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp up to 50 users over 30 seconds
    { duration: '1m', target: 100 },   // Ramp up to 100 users over 1 minute
    { duration: '30s', target: 0 },     // Ramp down to 0 users over 30 seconds
  ],
};

export default function () {
  const url = `${BASE_URL}/api/draw`;

  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${TOKEN}`,
  };

  const payload = JSON.stringify({
    x: randomIntBetween(0, 99),
    y: randomIntBetween(0, 99),
    color: randomIntBetween(0, 15),
  });

  const response = http.post(url, payload, { headers: headers });

  check(response, {
    'status is 200': (r) => r.status === 200,
    'response body is valid': (r) => r.body.includes('status'),
  });
}