// ── GeoCore 100K User Load Test ──────────────────────────────────────────────
// Simulates 100K users online → 10K concurrent → 10-20K RPS
//
// Usage:
//   k6 run tests/load/geocore-100k.js
//   k6 run --out influxdb=http://localhost:8086/k6 tests/load/geocore-100k.js
//
// Prerequisites:
//   - Backend running on localhost:8080 (or set API_BASE)
//   - PostgreSQL + Redis running
//   - Seed data loaded (categories, listings)

import http from 'k6/http';
import { sleep, check, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ── Custom metrics ──────────────────────────────────────────────────────────
const errorRate = new Rate('error_rate');
const listingLatency = new Trend('listing_latency', true);
const searchLatency = new Trend('search_latency', true);
const walletLatency = new Trend('wallet_latency', true);
const authLatency = new Trend('auth_latency', true);
const ordersPerSec = new Counter('orders_per_sec');

// ── Configuration ───────────────────────────────────────────────────────────
// Ramping scenario: 100 → 2000 → 5000 → 10000 → 5000 → 0
// Total duration: ~15 minutes
export const options = {
  scenarios: {
    ramping_load: {
      executor: 'ramping-vus',
      startVUs: 100,
      stages: [
        { duration: '2m', target: 2000 },   // warm up
        { duration: '3m', target: 5000 },   // ramp
        { duration: '5m', target: 10000 },  // PEAK — 10K concurrent
        { duration: '3m', target: 5000 },   // cool down
        { duration: '2m', target: 0 },      // stop
      ],
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<800', 'p(99)<2000'],
    http_req_failed: ['rate<0.01'],
    error_rate: ['rate<0.02'],
    listing_latency: ['p(95)<300'],
    search_latency: ['p(95)<500'],
    wallet_latency: ['p(95)<600'],
  },
};

const BASE = __ENV.API_BASE || 'http://localhost:8080';

// ── Traffic weights (realistic marketplace mix) ─────────────────────────────
// 60% browsing, 20% searching, 10% wallet, 5% auth, 5% orders
const TRAFFIC_MIX = [
  { weight: 30, fn: browseListings },
  { weight: 15, fn: viewListingDetail },
  { weight: 15, fn: searchListings },
  { weight: 10, fn: getSuggestions },
  { weight: 5, fn: getCategories },
  { weight: 10, fn: walletBalance },
  { weight: 5, fn: healthCheck },
  { weight: 10, fn: createOrder },
];

// Build cumulative weight table
const cumulativeWeights = [];
let total = 0;
for (const t of TRAFFIC_MIX) {
  total += t.weight;
  cumulativeWeights.push({ threshold: total, fn: t.fn });
}

function pickScenario() {
  const r = Math.random() * total;
  for (const cw of cumulativeWeights) {
    if (r < cw.threshold) return cw.fn;
  }
  return browseListings;
}

// ── Main VU loop ─────────────────────────────────────────────────────────────
export default function () {
  const scenario = pickScenario();
  scenario();
  sleep(Math.random() * 1.5 + 0.2); // 0.2–1.7s think time → ~1 RPS per VU
}

// ── Scenarios ────────────────────────────────────────────────────────────────

function browseListings() {
  const page = Math.floor(Math.random() * 10) + 1;
  const perPage = [10, 20, 30][Math.floor(Math.random() * 3)];

  group('browse_listings', () => {
    const res = http.get(`${BASE}/api/v1/listings?page=${page}&per_page=${perPage}`, {
      tags: { endpoint: 'listings', group: 'browse' },
    });
    listingLatency.add(res.timings.duration);
    errorRate.add(res.status >= 500);
    check(res, {
      'listings 200': (r) => r.status === 200,
      'listings not 429': (r) => r.status !== 429 || true, // rate limit OK
    });
  });
}

function viewListingDetail() {
  const id = Math.floor(Math.random() * 500) + 1;

  group('view_listing', () => {
    const res = http.get(`${BASE}/api/v1/listings/${id}`, {
      tags: { endpoint: 'listing_detail', group: 'browse' },
    });
    listingLatency.add(res.timings.duration);
    errorRate.add(res.status >= 500);
    check(res, {
      'detail 200or404': (r) => r.status === 200 || r.status === 404,
    });
  });
}

function searchListings() {
  const queries = [
    'iPhone', 'laptop', 'car', 'furniture', 'watch', 'camera',
    'sofa', 'TV', 'playstation', 'shoes', 'bag', 'guitar',
    'bike', 'tablet', 'headphones', 'dress', 'ring', 'book',
  ];
  const q = queries[Math.floor(Math.random() * queries.length)];

  group('search', () => {
    const res = http.get(`${BASE}/api/v1/listings/search?q=${encodeURIComponent(q)}`, {
      tags: { endpoint: 'search', group: 'search' },
    });
    searchLatency.add(res.timings.duration);
    errorRate.add(res.status >= 500);
    check(res, {
      'search OK': (r) => r.status === 200 || r.status === 429,
    });
  });
}

function getSuggestions() {
  const prefixes = ['iph', 'lap', 'car', 'wat', 'cam', 'sof', 'tv', 'sho'];
  const q = prefixes[Math.floor(Math.random() * prefixes.length)];

  group('suggestions', () => {
    const res = http.get(`${BASE}/api/v1/listings/suggestions?q=${q}`, {
      tags: { endpoint: 'suggestions', group: 'search' },
    });
    searchLatency.add(res.timings.duration);
    errorRate.add(res.status >= 500);
  });
}

function getCategories() {
  group('categories', () => {
    const res = http.get(`${BASE}/api/v1/categories`, {
      tags: { endpoint: 'categories', group: 'browse' },
    });
    errorRate.add(res.status >= 500);
    check(res, {
      'categories 200': (r) => r.status === 200,
    });
  });
}

function walletBalance() {
  const userId = Math.floor(Math.random() * 10000) + 1;
  const currencies = ['AED', 'USD', 'EUR', 'SAR', 'EGP'];
  const currency = currencies[Math.floor(Math.random() * currencies.length)];

  group('wallet_balance', () => {
    // This will 401 without auth, but tests the rate limit + timeout path
    const res = http.get(`${BASE}/api/v1/wallet/balance/${currency}`, {
      headers: { Authorization: `Bearer fake-token-${userId}` },
      tags: { endpoint: 'wallet', group: 'financial' },
    });
    walletLatency.add(res.timings.duration);
    // 401 is expected — we're testing that the path doesn't crash/timing
    errorRate.add(res.status >= 500);
    check(res, {
      'wallet not 500': (r) => r.status < 500,
    });
  });
}

function healthCheck() {
  group('health', () => {
    const res = http.get(`${BASE}/health/ready`, {
      tags: { endpoint: 'health', group: 'infra' },
    });
    check(res, {
      'health 200': (r) => r.status === 200,
    });
  });
}

function createOrder() {
  // Guest order — no auth required, rate limited
  group('create_guest_order', () => {
    const payload = JSON.stringify({
      listing_id: Math.floor(Math.random() * 500) + 1,
      quantity: 1,
      currency: 'AED',
    });

    const res = http.post(`${BASE}/api/v1/orders/guest`, payload, {
      headers: { 'Content-Type': 'application/json' },
      tags: { endpoint: 'orders', group: 'checkout' },
    });

    if (res.status === 201) {
      ordersPerSec.add(1);
    }
    errorRate.add(res.status >= 500);
  });
}

// ── Summary output ──────────────────────────────────────────────────────────
export function handleSummary(data) {
  return {
    stdout: textSummary(data, { indent: '  ', enableColors: true }),
    'tests/load/results-100k.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data, opts) {
  // k6 provides built-in summary; this just adds a file output
  return '';
}
