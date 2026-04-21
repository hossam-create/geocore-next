import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// ── Custom metrics ──────────────────────────────────────────────────────────
const errorRate = new Rate('error_rate');
const listingLatency = new Trend('listing_latency', true);
const searchLatency = new Trend('search_latency', true);

// ── Configuration ───────────────────────────────────────────────────────────
// Run with: k6 run --vus 100 --duration 30s tests/load/geocore-smoke.js
// Or use the stages below for a ramp-up test:
export const options = {
  stages: [
    { duration: '10s', target: 50 },   // warm up
    { duration: '20s', target: 200 },   // ramp to 200 VUs
    { duration: '30s', target: 500 },   // peak load
    { duration: '10s', target: 0 },     // cool down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<2000'],
    error_rate: ['rate<0.05'],          // <5% error rate
    listing_latency: ['p(95)<300'],
    search_latency: ['p(95)<500'],
  },
};

const BASE = __ENV.API_BASE || 'http://localhost:8080';

// ── Scenarios ───────────────────────────────────────────────────────────────

export default function () {
  // Rotate through different endpoints to simulate realistic traffic mix
  const scenario = (__ITER % 5);

  switch (scenario) {
    case 0: listListings(); break;
    case 1: searchListings(); break;
    case 2: getCategories(); break;
    case 3: healthCheck(); break;
    case 4: getListingDetail(); break;
  }

  sleep(Math.random() * 0.5); // 0–500ms think time
}

function listListings() {
  const page = Math.floor(Math.random() * 5) + 1;
  const res = http.get(`${BASE}/api/v1/listings?page=${page}&per_page=20`, {
    tags: { endpoint: 'listings' },
  });
  listingLatency.add(res.timings.duration);
  errorRate.add(res.status >= 400);
  check(res, {
    'listings status 200': (r) => r.status === 200,
    'listings has data': (r) => JSON.parse(r.body).success === true,
  });
}

function searchListings() {
  const queries = ['iPhone', 'laptop', 'car', 'furniture', 'watch', 'camera'];
  const q = queries[Math.floor(Math.random() * queries.length)];
  const res = http.get(`${BASE}/api/v1/listings/search?q=${q}`, {
    tags: { endpoint: 'search' },
  });
  searchLatency.add(res.timings.duration);
  errorRate.add(res.status >= 400);
  check(res, {
    'search status 200': (r) => r.status === 200 || r.status === 429, // 429 = rate limit OK
  });
}

function getCategories() {
  const res = http.get(`${BASE}/api/v1/categories`, {
    tags: { endpoint: 'categories' },
  });
  errorRate.add(res.status >= 400 && res.status !== 429);
  check(res, {
    'categories status 200': (r) => r.status === 200,
  });
}

function healthCheck() {
  const res = http.get(`${BASE}/health/ready`, {
    tags: { endpoint: 'health' },
  });
  check(res, {
    'health status 200': (r) => r.status === 200,
  });
}

function getListingDetail() {
  // Hit a random listing ID — most will 404 which is fine for load testing
  const id = Math.floor(Math.random() * 100) + 1;
  const res = http.get(`${BASE}/api/v1/listings/${id}`, {
    tags: { endpoint: 'listing_detail' },
  });
  errorRate.add(res.status >= 500); // only count 5xx as errors
}
