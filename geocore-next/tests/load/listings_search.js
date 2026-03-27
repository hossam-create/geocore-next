import http from 'k6/http';
  import { check, sleep } from 'k6';
  import { Rate, Trend } from 'k6/metrics';

  // ── Custom metrics ───────────────────────────────────────────────────────────
  const errorRate   = new Rate('errors');
  const searchTrend = new Trend('search_duration_ms', true);

  // ── Load profile ─────────────────────────────────────────────────────────────
  // Target: 100 concurrent VUs, p95 < 500ms, error rate < 1%
  export const options = {
    stages: [
      { duration: '30s', target: 20 },   // ramp up
      { duration: '1m',  target: 100 },  // peak load
      { duration: '30s', target: 0 },    // ramp down
    ],
    thresholds: {
      http_req_duration: ['p(95)<500'],
      errors: ['rate<0.01'],
    },
  };

  const BASE = __ENV.API_BASE || 'https://geo-core-next.replit.app/api/v1';

  const QUERIES = ['iPhone', 'Toyota', 'villa Dubai', 'laptop', 'watch'];
  const CITIES  = ['Dubai', 'Riyadh', 'Doha', 'Kuwait City', 'Abu Dhabi'];

  export default function () {
    const q    = QUERIES[Math.floor(Math.random() * QUERIES.length)];
    const city = CITIES[Math.floor(Math.random() * CITIES.length)];

    // Search
    const t0   = Date.now();
    const res  = http.get(`${BASE}/listings/search?q=${encodeURIComponent(q)}&city=${city}&per_page=20`);
    searchTrend.add(Date.now() - t0);
    errorRate.add(res.status >= 400);

    check(res, {
      'search status 200': (r) => r.status === 200,
      'has data array':    (r) => { try { return Array.isArray(JSON.parse(r.body).data); } catch { return false; } },
    });

    // Autocomplete
    const acRes = http.get(`${BASE}/listings/suggestions?q=${encodeURIComponent(q.split(' ')[0])}`);
    errorRate.add(acRes.status >= 400);

    // Health check
    const hRes = http.get(`${BASE.replace('/api/v1', '')}/health`);
    check(hRes, { 'health ok': (r) => r.status === 200 });

    sleep(1);
  }
  