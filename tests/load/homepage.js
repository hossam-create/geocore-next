// k6 Load Test — Mnbarh Platform
// Run: k6 run tests/load/homepage.js
// Install: https://k6.io/docs/getting-started/installation/

import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate } from 'k6/metrics'

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const errorRate = new Rate('errors')

export const options = {
  stages: [
    { duration: '2m',  target: 100  }, // Ramp up to 100 users
    { duration: '5m',  target: 1000 }, // Sustained load — 1000 concurrent
    { duration: '2m',  target: 5000 }, // Peak load
    { duration: '1m',  target: 0    }, // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests < 500ms
    http_req_failed:   ['rate<0.01'],  // Error rate < 1%
    errors:            ['rate<0.05'],
  },
}

export default function () {
  // ── Test 1: Homepage listings ──────────────────────────────────────────
  let res = http.get(`${BASE_URL}/api/v1/listings?page=1&per_page=20`)
  check(res, {
    'listings: status 200':       (r) => r.status === 200,
    'listings: response < 500ms': (r) => r.timings.duration < 500,
    'listings: has data':         (r) => r.json().data !== undefined,
  })
  errorRate.add(res.status !== 200)
  sleep(1)

  // ── Test 2: Search ─────────────────────────────────────────────────────
  res = http.get(`${BASE_URL}/api/v1/listings?q=iphone&page=1`)
  check(res, {
    'search: status 200': (r) => r.status === 200,
    'search: < 500ms':    (r) => r.timings.duration < 500,
  })
  errorRate.add(res.status !== 200)
  sleep(1)

  // ── Test 3: Categories ─────────────────────────────────────────────────
  res = http.get(`${BASE_URL}/api/v1/categories`)
  check(res, {
    'categories: status 200': (r) => r.status === 200,
    'categories: < 200ms':    (r) => r.timings.duration < 200,
  })
  errorRate.add(res.status !== 200)
  sleep(1)

  // ── Test 4: Public config ──────────────────────────────────────────────
  res = http.get(`${BASE_URL}/api/v1/config/public`)
  check(res, {
    'config: status 200': (r) => r.status === 200,
    'config: < 100ms':    (r) => r.timings.duration < 100,
  })
  errorRate.add(res.status !== 200)
  sleep(2)
}

// ── Auth flow test (separate scenario) ──────────────────────────────────
export function authFlow() {
  const payload = JSON.stringify({
    email: `loadtest-${__VU}@test.com`,
    password: 'wrongpassword123',
  })
  const params = { headers: { 'Content-Type': 'application/json' } }

  let res = http.post(`${BASE_URL}/api/v1/auth/login`, payload, params)
  check(res, {
    'login: responds': (r) => r.status === 401 || r.status === 200 || r.status === 429,
  })
  sleep(3)
}
