# GeoCore Next — Production Launch Plan

> **Status:** Repository is **NOT greenfield** — ~85% built across 26+ sprints.
> This plan documents what exists, identifies real gaps, and provides a finite
> roadmap to Go-Live. It intentionally does **not** re-plan work already done.

---

## Phase 0 — Grounded Analysis

### Hard constraints — status

| Constraint | Status | Evidence |
|---|---|---|
| **No Flutter/Dart** | ✅ Clean | `find *.dart` → 0 results; no `pubspec.yaml` |
| **Mobile = React Native** | ✅ Done | `mobile/` is Expo 54 + RN 0.81.5 + expo-router 6 |
| **Scalable backend** | ✅ Go | `backend/internal/` — 80+ packages |
| **Production-grade code** | ⚠️ Partial | Code is modular; **test coverage unverified** |

### Actual architecture

```
geocore-next/                    # pnpm monorepo
├── backend/                     Go 1.21+ · Gin · GORM · PostgreSQL · Redis · Kafka · WebSocket
│   ├── cmd/api/                 Entry point (main.go — wires 60+ route groups)
│   ├── internal/                80+ domain packages (see inventory below)
│   └── pkg/                     Shared middleware, database, observability
│
├── frontend/                    Next.js 15 · React 19 · Tailwind · Zustand
│   ├── app/                     104 route files (App Router)
│   └── components/              110 UI components
│
├── frontend-admin/              Next.js admin (40 items)
├── admin/                       Legacy admin (106 items — audit: likely dead)
│
├── mobile/                      Expo 54 · RN 0.81.5 · expo-router 6 · Zustand
│   └── app/                     login/register/wallet/bids/chat/listings/favorites/settings
│
├── services/                    Sidecar microservices (361 items)
├── ai-service/                  ML endpoints (pricing/matching)
│
├── infra/                       Terraform + modules + envs + deploy.sh
├── k8s/                         22 manifests (HPA, canary, ArgoCD, Kafka, Redis,
│                                observability-stack, prometheus-alerts, PDB, network-policy)
├── nginx/                       Reverse-proxy config
├── monitoring/                  Prometheus + Grafana dashboards
├── .github/workflows/           ci.yml + deploy.yml
├── docker-compose.yml           dev stack
├── docker-compose.monitoring.yml
├── render.yaml                  Render.com config
└── Makefile                     build/test/deploy targets
```

### Backend package inventory (condensed)

| Domain | Packages | Status |
|---|---|---|
| **Auth & Users** | auth, users, authz, kyc | ✅ Done |
| **Commerce** | listings, cart, order, stores, watchlist, deals, reviews, reputation | ✅ Done |
| **Auctions** | auctions (English + Dutch + Buy-Now), reverseauctions, livestream | ✅ Done |
| **Exchange / P2P** | exchange, p2p, crypto, bnpl | ✅ Done |
| **Payments** | payments, wallet, fees, settlement, forex, subscriptions, billing, referral | ✅ Done |
| **Trust & Safety** | fraud, security, securityops, moderation, disputes, redteam | ✅ Done |
| **Realtime** | chat, notifications | ✅ Done |
| **Search / Discovery** | search, recommendations, matching, pricing | ✅ Done |
| **Logistics** | crowdshipping, geoscore, region | ✅ Done |
| **Growth** | invite, waitlist, loyalty, growth, gameday | ✅ Done |
| **Ops / Admin** | admin, controltower, controlplane, analytics, reports, ops, backup, tenant | ✅ Done |
| **Advanced** | aichat, aiops, arpreview, blockchain, plugins, policy, saga, slo, stress, chaos, reslab, warroom | ✅ Done |
| **Compliance** | compliance (GDPR/hash-chain) | ✅ Done (Sprint 26) |

### Real gaps — verified unknowns

🔴 **Must verify before launch** (blocks go-live if broken)

1. **Payment providers actually wired** — `STRIPE_SECRET_KEY` env exists but is handler-complete and webhook-verified?
2. **Email service sending** — SMTP/Resend integration or no-op stubs?
3. **File uploads to R2** — `images/` package wired to Cloudflare R2 credentials?
4. **Mobile ↔ Backend API contract** — does the Expo app consume current endpoint shapes?
5. **Test coverage** — any `*_test.go` in `backend/internal/`? Any frontend tests?
6. **DB migrations** — currently `GORM AutoMigrate`; production needs versioned migrations.
7. **Twilio / LiveKit / Firebase** — configured in env but implementation state unknown.

🟡 **Likely missing** (soft-blockers)

1. **Push notifications pipeline** — Expo Push not visible in `mobile/package.json`.
2. **User-facing legal pages** — TOS, Privacy Policy HTML (backend has `/meta/disclaimer` only).
3. **App Store / Play Store submission assets** — screenshots, store copy, privacy manifest.
4. **Analytics client instrumentation** — PostHog env exists, but frontend/mobile events fired?
5. **Error tracking** — Sentry DSN exists, but initialized in every surface (backend/frontend/mobile)?

---

## Phase 1 — System Design (already in place)

Keep the existing architecture. Do not refactor.

### Authentication
- JWT access + refresh (`backend/internal/auth/`)
- OAuth social (`GoogleID`, `AppleID`, `FacebookID` columns on `users`)
- Email verification tokens (`VerificationToken` + `EmailVerified`)
- Sprint 24 GlobalGuard (fraud-based pre-action gating)

### File storage
- **Cloudflare R2** (S3-compatible) — env vars already defined.
- `backend/internal/images/` handles multipart upload + presigned URLs.
- **TODO:** verify R2 creds are populated + bucket exists.

### Realtime
- WebSocket hubs: `chat.Hub`, `auctions.Hub`, `livestream`
- Redis pub/sub for horizontal scaling (`SubscribeRedis`)
- LiveKit for A/V streaming (livestream auctions)

### Database
- PostgreSQL 16 + GORM (ORM)
- ~100 tables across all domains
- **Gap:** `golang-migrate` not integrated → ⚠️ Sprint MIGR below

---

## Phase 2 — Remaining Sprints (only real gaps)

### Sprint V — Verification (2-3 days)

Goal: Convert all 🔴 unknowns above into verified pass/fail with file paths.

**V1 — Payment integration audit**
- Description: Open every payment handler, trace actual HTTP calls to Stripe/Coinbase/Tabby. Confirm webhook signature verification on `/webhooks/stripe`.
- Files: `backend/internal/payments/`, `wallet/`, `crypto/`, `bnpl/`
- Dependencies: None
- Estimated: 4h
- Acceptance: Written report `docs/PAYMENT_AUDIT.md` — one row per provider with status (wired / stub / broken).

**V2 — Email delivery test**
- Description: Send a real verification email via configured SMTP to a test account. Confirm template rendering + link round-trip.
- Files: `backend/internal/auth/`, `notifications/`
- Dependencies: `SMTP_*` env set
- Estimated: 2h
- Acceptance: Screenshot/log of delivered email; verification click → `EmailVerified = true`.

**V3 — R2 upload round-trip**
- Description: Upload an image via `/api/v1/images/presign` → PUT to R2 → confirm `R2_PUBLIC_URL` serves it.
- Files: `backend/internal/images/`
- Estimated: 2h
- Acceptance: Uploaded `jpg` is fetchable from public URL + row in `images` table.

**V4 — Mobile-backend contract sweep**
- Description: Grep `mobile/app/` for every `axios` call, match against `backend/cmd/api/main.go` route table. List mismatches.
- Files: `mobile/app/`, `mobile/utils/`
- Estimated: 3h
- Acceptance: `docs/MOBILE_API_DIFF.md` listing each endpoint used by mobile with backend status.

**V5 — Test coverage report**
- Description: Run `go test ./... -coverprofile=cover.out` and `pnpm test` in frontend. Produce coverage summary.
- Estimated: 2h
- Acceptance: Coverage report + list of packages <40% coverage.

### Sprint MIGR — Migrations System (1 day)

**M1 — Install golang-migrate CLI + structure**
- Create `backend/migrations/` with `0001_baseline.up.sql` / `0001_baseline.down.sql`.
- Baseline dumped from current GORM-migrated prod schema via `pg_dump --schema-only`.
- Add Makefile targets: `migrate-up`, `migrate-down`, `migrate-new`.
- Acceptance: `make migrate-up` runs clean on empty DB; matches GORM AutoMigrate output.

**M2 — Wire main.go to run migrations before AutoMigrate**
- On boot: run `migrate.Up()` first, then keep AutoMigrate as a sanity check.
- Fail fast if migrations are behind code expectations.
- Acceptance: Boot logs show "migrations applied: N"; mismatched version blocks startup.

### Sprint DEPLOY — Deployment Hardening (2 days)

**D1 — Verify existing `infra/deploy.sh` + `k8s/` manifests**
- Read `infra/deploy.sh`, `k8s/backend-deployment.yaml`, `ingress.yaml`.
- Document exact deploy command + rollback command.
- Files: `docs/DEPLOY.md` (new)
- Acceptance: A fresh engineer can deploy + rollback from doc alone.

**D2 — GitHub Actions end-to-end dry-run**
- Read `.github/workflows/deploy.yml`; trigger on test branch.
- Confirm: build → test → push image → deploy → health check → rollback on fail.
- Acceptance: One successful dry-run to staging cluster.

**D3 — Secrets audit**
- Compare `.env.example` vs `k8s/secrets-template.yaml` vs prod secret store.
- Flag any secret missing in prod.
- Acceptance: `docs/SECRETS.md` mapping env var → source of truth.

**D4 — Blue/Green or canary verification**
- `k8s/api-canary.yaml` exists — test the canary flip with a known-bad image.
- Acceptance: Bad canary auto-rolls back within SLO.

### Sprint LOAD — Load Testing (1 day)

**L1 — k6 scripts for hot paths**
- Create `loadtest/k6/`:
  - `auction-bid-flood.js` (1000 concurrent bids to single auction)
  - `exchange-match.js` (match burst)
  - `withdraw-flood.js` (concurrent withdraw requests — tests fraud GlobalGuard)
  - `signup-spam.js` (tests IDS auto-block)
- Acceptance: Each script runs against staging; results saved to `loadtest/results/`.

**L2 — k6 CI integration**
- Add `.github/workflows/loadtest.yml` — runs nightly against staging.
- Alert to Slack on regression (>20% p95 latency).
- Acceptance: One successful nightly run.

### Sprint MOBILE — Mobile Completion (3-5 days)

**MB1 — Push notifications**
- Install `expo-notifications` + `expo-device`.
- Register device token → `POST /api/v1/notifications/register-token`.
- Backend already has `notifications` package — confirm token handler exists.
- Acceptance: Test push received on physical device.

**MB2 — Payment flow**
- Wire Stripe React Native SDK (or WebView fallback).
- Deep-link back to app after 3DS challenge.
- Acceptance: Real $1 test charge succeeds + wallet balance updated.

**MB3 — Store submission prep**
- `app.json`: icon, splash, bundleIdentifier (iOS), package (Android).
- Privacy manifest (iOS 17+ requirement): declare network domains + tracking.
- Screenshots: 5 per device size.
- Acceptance: EAS build succeeds for both platforms; TestFlight + internal testing track live.

### Sprint LEGAL — Legal Pages (1 day)

**LG1 — TOS + Privacy Policy pages**
- `frontend/app/legal/terms/page.tsx`
- `frontend/app/legal/privacy/page.tsx`
- Content: non-custodial disclaimer + GDPR rights + payment terms. Have counsel review.
- Acceptance: Pages render with correct copy; linked from footer + signup flow.

**LG2 — Consent popup on first visit**
- `frontend/components/consent-banner.tsx` — calls `POST /user/consent` (already built).
- Acceptance: Cookie banner shows once, consent row persisted.

### Sprint ANALYTICS — Instrumentation (1 day)

**AN1 — PostHog client everywhere**
- `frontend/lib/posthog.ts` + mobile equivalent.
- Core events: `signup`, `listing_created`, `bid_placed`, `purchase`, `withdraw_requested`.
- Acceptance: Events visible in PostHog dashboard within 1 min.

**AN2 — Sentry on all three surfaces**
- Backend: already wired (`SENTRY_DSN` env).
- Frontend: `@sentry/nextjs` in `next.config.ts`.
- Mobile: `sentry-expo`.
- Acceptance: Intentional error in each surface appears in Sentry within 5 min.

---

## Phase 3 — Task Breakdown Format

Format used per task:

```
<ID> — <Task Name>
  Description: <what>
  Files: <paths>
  Dependencies: <other tasks / env>
  Estimated: <hours>
  Acceptance: <verification>
```

All Sprint V/MIGR/DEPLOY/LOAD/MOBILE/LEGAL/ANALYTICS tasks above follow this format.

---

## Testing Requirements

### Per-feature minimum
- **Happy path** — one integration test.
- **Auth edge cases** — expired token, wrong role, missing header.
- **Concurrency** — for money-moving endpoints, a 10-worker race test.
- **Fraud/throttle** — verify GlobalGuard + IDS trigger as expected.

### Coverage targets
- Backend financial packages (`payments`, `wallet`, `exchange`, `fees`): **≥ 80%**
- Backend auth + fraud: **≥ 70%**
- Frontend critical flows (auth, checkout, bid): **≥ 60%**
- Mobile (smoke tests via Detox): **happy path only**

---

## .env Template

See `.env.example` in repo root (126 lines, groups for App, DB, Redis, Auth, AI, Analytics, R2, Payments, Comms, Realtime, Firebase, Frontend).

**Required-for-launch minimum:**
```bash
APP_ENV=production
DATABASE_URL=postgres://...?sslmode=require
REDIS_URL=rediss://...
JWT_SECRET=<32+ random bytes>
JWT_REFRESH_SECRET=<32+ random bytes>
R2_ACCOUNT_ID=...
R2_ACCESS_KEY_ID=...
R2_SECRET_ACCESS_KEY=...
R2_BUCKET=geocore-uploads-prod
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
SMTP_HOST=smtp.resend.com
SMTP_USER=resend
SMTP_PASS=...
SENTRY_DSN=...
FIREBASE_SERVICE_ACCOUNT_JSON=<base64>
ENABLE_SECURITY_MONITORING=true
ENABLE_AUTO_FREEZE=true
ENABLE_REDTEAM=false
```

---

## Deployment Steps (use existing infra/)

1. `cd infra/ && terraform init && terraform apply -var-file=envs/prod.tfvars`
2. Populate k8s secrets from `k8s/secrets-template.yaml`.
3. `kubectl apply -k k8s/` (applies all 22 manifests).
4. ArgoCD (`k8s/argocd-app.yaml`) takes over sync loop.
5. First deploy: `bash infra/deploy.sh` to seed baseline.
6. Confirm health: `curl https://api.example.com/healthz` → 200.
7. Canary flip: increment weight in `k8s/api-canary.yaml`.

Detailed runbook to be written in Sprint DEPLOY → D1.

---

## Revenue Strategy

The backend already supports **five** parallel revenue streams. No new code needed — just marketing activation:

| Stream | Mechanism | Package | Target |
|---|---|---|---|
| **Listing fees** | Price Plans — Free / Premium / Pro tiers | `subscriptions`, `billing` | 5-15% of sellers |
| **Transaction commission** | Configurable % on each completed sale | `fees`, `settlement` | 3-7% of GMV |
| **Featured listings** | Paid placement at top of search | `listings`, `admin` | $5-20 per listing |
| **Live auction commission** | House take on winning bid | `livestream`, `auctions` | 10% typical |
| **BNPL referral** | Revenue share from Tabby/Tamara | `bnpl` | 1-3% of BNPL GMV |

Future additions (already scaffolded):
- **Crypto swap spread** (`crypto`, `forex`)
- **Loyalty-program monetization** (`loyalty`)
- **Affiliate / referral commissions** (`referral`, `invite`)

**Recommended launch mix:** Free listings + 5% transaction fee + $10 featured-listing boost. Low-friction onboarding, revenue scales with GMV.

---

## Go-Live Checklist

### Week -2 (T-14 days)
- [ ] Sprint V complete — all 🔴 unknowns resolved
- [ ] Sprint MIGR complete — baseline migration deployed to staging
- [ ] Staging environment running full stack for 72h without incident
- [ ] Sprint LOAD — baseline k6 results captured

### Week -1 (T-7 days)
- [ ] Sprint DEPLOY — one successful production dry-run to staging
- [ ] Sprint LEGAL — TOS + Privacy Policy reviewed by counsel
- [ ] Sprint ANALYTICS — events flowing to PostHog
- [ ] Payment providers: Stripe live keys + webhook receiving events
- [ ] DNS + SSL + Cloudflare configured for production domain
- [ ] Incident-response runbook written (`docs/INCIDENT.md`)
- [ ] On-call rotation set up in PagerDuty/OpsGenie

### T-0 (Launch Day)
- [ ] Feature flags: `ENABLE_EMERGENCY_MODE=false`, `ENABLE_AUTO_FREEZE=true`, `ENABLE_REDTEAM=false`
- [ ] Fraud thresholds set conservatively (block at 80, limit at 60)
- [ ] Rate limits: 100 req/min per IP default, higher for verified users
- [ ] Admin alerts wired to Slack `#launch` channel
- [ ] Sentry + PagerDuty integration confirmed
- [ ] Mobile apps: iOS on TestFlight external testing, Android on internal track
- [ ] 5 beta users complete full buy/sell/bid/withdraw flow
- [ ] One full compliance audit chain verification: `GET /admin/compliance/audit/verify` → `valid: true`

### T+1 (First 24h)
- [ ] Monitor p95 latency on hot paths (< 300ms target)
- [ ] Monitor error rate (< 0.5%)
- [ ] Review first 100 sign-ups for fraud false-positives
- [ ] First `GET /admin/redteam/run` with `spam` scenario passes

### T+7 (First week)
- [ ] GDPR export requested + fulfilled for at least one user
- [ ] Refund / dispute flow exercised end-to-end
- [ ] CDN cache hit rate > 80%
- [ ] DB connection pool utilization < 60%

---

## Post-Launch Scaling Plan

### Traffic tiers

| Tier | MAU | Scaling action |
|---|---|---|
| **0-10k** | Current setup | Single Hetzner CX32 + managed Postgres |
| **10k-100k** | 10x | Move DB to managed PG HA (Supabase/Neon), add 2× backend replicas, enable Redis Sentinel |
| **100k-1M** | 100x | k8s HPA (`k8s/hpa.yaml` already configured), PG read replicas, Cloudflare R2 multi-region |
| **1M+** | 1000x | Shard by region (`tenant/` + `region/` packages already support multi-tenant), Kafka for event fan-out (`k8s/kafka.yaml` ready), ArgoCD canary deploys (`api-canary.yaml`) |

### Cost progression

| Tier | ~Cost/mo | Biggest line item |
|---|---|---|
| Pre-launch | €10 | Hetzner CX22 |
| 10k MAU | ~€80 | DB + egress |
| 100k MAU | ~€800 | Managed PG + 3× VPS + R2 egress |
| 1M MAU | ~€8,000 | Kafka cluster + multi-region PG |

### Key scaling levers already built

1. **Sprint 24 fraud Predictor** — 60s Redis cache reduces DB hits under load.
2. **GlobalGuard** — blocks abusive traffic at the edge before hitting handlers.
3. **Kafka event bus** (`k8s/kafka.yaml`) — async fan-out for analytics, notifications, search indexing.
4. **Canary deploys** — safe rollouts without downtime.
5. **HPA** — auto-scales on CPU/memory/custom metrics.
6. **Observability stack** (`k8s/observability-stack.yaml`) — Prometheus + Grafana + Alertmanager pre-configured.

---

## Execution Roadmap (finite)

```
Week 1: Sprint V (verification)         → all unknowns resolved
Week 2: Sprint MIGR + Sprint DEPLOY     → migrations + deploy runbook
Week 3: Sprint LOAD + Sprint ANALYTICS  → load tests + instrumentation
Week 4: Sprint MOBILE                   → push + payments + store prep
Week 5: Sprint LEGAL + soft beta        → legal pages + 50 beta users
Week 6: Launch 🚀
```

**Total: ~6 weeks from today to launch** — assuming Sprint V doesn't surface showstoppers.

---

## What this plan does NOT include

- ❌ New feature development (you have everything for v1).
- ❌ Refactoring existing packages (they work; don't touch).
- ❌ Choosing a new tech stack (it's chosen).
- ❌ Re-writing the admin dashboard (exists).
- ❌ Flutter removal (never had any).

If you find yourself tempted to open any of the 80 backend packages to "improve" them — don't. The path to launch is **verify → migrate → deploy → test → launch**. Nothing else.
