# GeoCore Next — Master Generation Prompt

You are a senior software architect rebuilding a legacy PHP classifieds platform in modern Go + Next.js.

You have access to **two repositories**:
1. **`hossam-create/geocore-community`** — the original PHP platform (source of truth for features)
2. **`hossam-create/geocore-next`** — the new Go rebuild already started (your base to work from)

Read both before writing anything.

---

## STEP 1 — Read the Original PHP Project (Feature Reference)

Fetch these from `https://raw.githubusercontent.com/hossam-create/geocore-community/main/`:

```
README.md
composer.json
docker-compose.yml
src/config.example.php
```

Then browse directory structure at:
```
https://github.com/hossam-create/geocore-community/tree/main/src
https://github.com/hossam-create/geocore-community/tree/main/src/classes
```

Try reading key PHP class files like:
```
src/classes/listings/listings.class.php
src/classes/auctions/auctions.class.php
src/classes/users/users.class.php
src/classes/geo/geo.class.php
src/admin/listings.php
src/admin/auctions.php
```

From this, extract:
- Every feature the PHP app has
- Database table structure (from PHP models or config)
- Business rules (bid validation, expiry logic, anti-sniping, etc.)
- Admin panel capabilities
- Payment providers, multi-language support, geo features

---

## STEP 2 — Read the Go Rebuild Already Started

**Repo:** `https://github.com/hossam-create/geocore-next`

Fetch every file below using `web_fetch` on the raw URL
(`https://raw.githubusercontent.com/hossam-create/geocore-next/main/`):

```
# Config
backend/go.mod
docker-compose.yml
backend/.env.example

# Entry point
backend/cmd/api/main.go

# Infrastructure
backend/pkg/database/database.go
backend/pkg/middleware/auth.go
backend/pkg/response/response.go
backend/pkg/redis/redis.go

# Auth
backend/internal/auth/handler.go
backend/internal/auth/routes.go

# Listings
backend/internal/listings/model.go
backend/internal/listings/handler.go
backend/internal/listings/routes.go
backend/internal/listings/seed.go

# Auctions
backend/internal/auctions/model.go
backend/internal/auctions/handler.go
backend/internal/auctions/routes.go
backend/internal/auctions/websocket.go

# Chat
backend/internal/chat/model.go
backend/internal/chat/handler.go
backend/internal/chat/routes.go
backend/internal/chat/websocket.go

# Users
backend/internal/users/model.go
backend/internal/users/handler.go
backend/internal/users/routes.go

# Payments
backend/internal/payments/handler.go
backend/internal/payments/routes.go
```

After reading both repos, output a short **"What I Found"** summary:
- Which PHP features are NOT yet in the Go rebuild
- Which Go files exist but need to be completed
- Any patterns or inconsistencies in the Go code to fix
- Current state of each domain (auth / listings / auctions / chat / payments)

---

## STEP 3 — Generate 3 Documents

Everything below must be grounded in what you actually read.

---

## DOCUMENT 1 — PRD.md

### 1. Executive Summary
- Vision (2 sentences)
- Problem: what's wrong with the PHP version
- Solution: the Go rebuild
- 5 KPIs with target numbers

### 2. User Personas
3 personas from the real use cases (classifieds + auctions).
Each: name, age, country, occupation, goals ×3, pain points ×3, tech level, one quote.

### 3. Feature Requirements

Map every PHP feature to a Go requirement. Format:

```
#### [Feature Name]
**Priority:** P0 | P1 | P2
**Parity:** Full parity | Improved | New
**PHP Status:** Exists in PHP | Missing from PHP
**Go Status:** Done | Partial | Not started
**User Story:** As a [role], I want [action] so that [benefit]
**Acceptance Criteria:**
- [ ] testable item
**Edge Cases:**
- specific edge case
```

**P0 — MVP (PHP feature parity):**
All core features found in the PHP app

**P1 — Improvements over PHP:**
Things PHP does badly that Go should do better (real-time, performance, mobile API, etc.)

**P2 — New features PHP never had:**
- AI auto-categorization (GPT-4o)
- AI price suggestion
- Semantic vector search (pgvector)
- WebSocket real-time (if PHP used polling)
- React Native mobile app

### 4. Non-Functional Requirements
- Performance targets (compare to PHP baseline)
- Scalability, security (fix PHP vulnerabilities), availability, i18n

### 5. API Contract
Full request + response JSON for:
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/listings` (all query params documented)
- `POST /api/v1/listings`
- `POST /api/v1/auctions/:id/bid`
- WebSocket: auction bid broadcast `{type, current_bid, bid_count, ends_at}`
- WebSocket: chat message `{type, content, sender_id, created_at}`

### 6. UI/UX Requirements
Homepage, Listing Detail, Create Listing (multi-step), Auction Detail (live), Chat, User Profile, Admin Panel.
For each: layout, key components, what's better than PHP version.

### 7. PHP → Go Migration Strategy
- MySQL → PostgreSQL schema mapping
- Data migration script approach
- Zero-downtime cutover plan

---

## DOCUMENT 2 — CLAUDE.md

Complete instructions for Claude Code working on `hossam-create/geocore-next`.
Reference the **actual code you read** — exact function names, file paths, patterns.

### Project Overview
- What this rebuilds (link both repos)
- Stack
- Exact repo tree (from what you read, not assumed)
- `docker-compose up -d` → API on :8080

### Architecture Rules (Non-Negotiable)

Base these rules on the **actual patterns** you saw in the Go code:

1. **Layer law** — what goes where (model / handler / routes). Quote an example from the actual code.
2. **Error handling** — exact pattern used. Show a real example from the code you read.
3. **Response format** — show the actual `response.go` functions and when to use each.
4. **Auth context** — `user_id` is a `string` in gin context. Show the correct cast.
5. **GORM patterns** — soft delete, transactions, preloads — with examples from the actual models.
6. **WebSocket Hub** — reference `auctions/websocket.go` and `chat/websocket.go`. Explain the Hub pattern so a new dev can add a WS feature safely.

### Adding a New Feature — Step-by-Step Template
Walk through adding "Saved Searches with email alerts" as a complete example.
Show every file to create/modify, with code sketches.

### Go Conventions
Based on the naming you saw in the actual files.

### Frontend Conventions
RSC vs client components, Zustand, React Query, Zod, api.ts.

### Environment Variables Table
| Variable | Description | Required | Default | Example |
|---|---|---|---|---|
| APP_ENV | Runtime environment | yes | development | production |
| PORT | HTTP port | no | 8080 | 8080 |
| DB_HOST | PostgreSQL host | yes | localhost | postgres |
| DB_PORT | PostgreSQL port | no | 5432 | 5432 |
| DB_USER | DB username | yes | — | geocore |
| DB_PASSWORD | DB password | yes | — | secret |
| DB_NAME | Database name | yes | — | geocore_dev |
| DB_SSLMODE | SSL mode | no | disable | require |
| REDIS_HOST | Redis host | yes | localhost | redis |
| REDIS_PORT | Redis port | no | 6379 | 6379 |
| JWT_SECRET | Signing key (min 32 chars) | yes | — | random_32_chars |
| FRONTEND_URL | CORS origin | yes | http://localhost:3000 | https://geocore.com |
| STRIPE_SECRET_KEY | Stripe secret | yes | — | sk_live_... |
| STRIPE_PUBLISHABLE_KEY | Stripe publishable | yes | — | pk_live_... |
| CLOUDINARY_URL | Cloudinary DSN | yes | — | cloudinary://k:s@cloud |
| MEILI_HOST | Meilisearch URL | no | http://localhost:7700 | http://meilisearch:7700 |
| MEILI_MASTER_KEY | Meilisearch key | yes | — | random_key |

### Common Commands
```bash
# Docker
docker-compose up -d
docker-compose logs -f api
docker-compose exec postgres psql -U geocore geocore_dev

# Go
cd backend
go run ./cmd/api/main.go
go test ./...
go build -o bin/api ./cmd/api/main.go

# Frontend
cd frontend
npm run dev
npm run build
npm run lint
```

### Top 10 Pitfalls
Based on the actual code you read — specific gotchas for this exact codebase.

---

## DOCUMENT 3 — TASKS.md

98 tasks. Every task fully written. No abbreviations. No "similar to above".

Format:
```markdown
## TASK-XXX: [Clear Action Title]
**Epic:** [name]
**Type:** feature | chore | bug | refactor
**Priority:** P0 | P1 | P2
**Estimate:** Xh
**Depends on:** TASK-YYY (or "none")
**Go Status:** Done | Partial | Not started

### Description
What to build and why.

### Technical Notes
Specific Go/TS guidance. Which existing file to use as pattern. Gotcha to avoid.

### Acceptance Criteria
- [ ] concrete, testable

### Files
- `backend/internal/xxx/yyy.go` — what to create or change
- `frontend/src/xxx/yyy.tsx` — what to create or change
```

**Note:** For tasks marked `Go Status: Done`, the acceptance criteria should describe what **testing and verification** is needed, not reimplementation.

**EPIC 1 — Foundation** (TASK-001 → TASK-010)
- TASK-001: Verify `go build ./...` passes with zero errors — fix any import issues
- TASK-002: PostgreSQL connection with retry on startup (5 attempts, 2s backoff)
- TASK-003: Redis connection with ping health check on startup
- TASK-004: JWT middleware — verify token, set user_id (string) + user_email in gin context
- TASK-005: Rate limiting — Redis INCR per IP, 100 req/min, skip OPTIONS preflight
- TASK-006: Standard response package — OK/Created/BadRequest/Unauthorized/Forbidden/NotFound/InternalError
- TASK-007: Docker Compose — all 5 services healthy (postgres, redis, meilisearch, adminer, api)
- TASK-008: GitHub Actions CI — go vet + go test + go build on every push
- TASK-009: Next.js 15 init — App Router + Tailwind + shadcn/ui + TypeScript strict
- TASK-010: Frontend API client — axios, base URL from NEXT_PUBLIC_API_URL, auto-refresh token on 401

**EPIC 2 — Auth & Users** (TASK-011 → TASK-020)
- TASK-011: POST /auth/register — bcrypt cost 12, return access + refresh tokens
- TASK-012: POST /auth/login — verify password, tokens with correct expiry (15min / 30d)
- TASK-013: POST /auth/refresh — validate refresh token, rotate both, invalidate old in Redis
- TASK-014: POST /auth/google — OAuth 2.0, upsert user on first login
- TASK-015: GET /users/me — full user object excluding password_hash
- TASK-016: PUT /users/me — name, bio, location, language, currency
- TASK-017: GET /users/:id/profile — public profile (no PII)
- TASK-018: POST /users/me/avatar — Cloudinary upload, update avatar_url
- TASK-019: Frontend login page — Zod validation, Zustand auth store, redirect on success
- TASK-020: Frontend register page + auth store + protected route HOC

**EPIC 3 — Categories & Listings** (TASK-021 → TASK-035)
- TASK-021: Seed 10 categories with EN + AR names on startup (idempotent)
- TASK-022: GET /categories — tree with children, Redis cache 5min
- TASK-023: GET /listings — q, category_id, country, city, min_price, max_price, condition, type, sort, page, per_page
- TASK-024: GET /listings/:id — preload Images + Category, increment view_count async
- TASK-025: POST /listings — validate, expires_at = now+60d, status = active
- TASK-026: PUT /listings/:id — owner-only, whitelist updatable fields
- TASK-027: DELETE /listings/:id — soft delete, block if active auction exists
- TASK-028: POST /listings/:id/images — up to 10 files, Cloudinary, first = is_cover
- TASK-029: POST /listings/:id/favorite — toggle, update favorite_count atomically
- TASK-030: GET /listings/me — current user's listings filtered by status
- TASK-031: Cron — midnight, expire listings where expires_at < NOW() AND status = active
- TASK-032: Frontend homepage — category bar, search, featured grid, auction countdowns
- TASK-033: Frontend listing detail — gallery, info, Leaflet map, contact seller CTA, similar row
- TASK-034: Frontend create listing — 3-step wizard with Zod per step
- TASK-035: Frontend search results — filters sidebar, grid, sort, pagination

**EPIC 4 — Auctions** (TASK-036 → TASK-047)
- TASK-036: POST /auctions — linked to listing, validate time range, max 30 days
- TASK-037: GET /auctions — active, sorted by ends_at ASC, time_remaining field
- TASK-038: GET /auctions/:id — top 20 bids, is_reserve_met, time_remaining
- TASK-039: POST /auctions/:id/bid — SELECT FOR UPDATE TX, validate > current_bid, block self-bid
- TASK-040: Auto-bid proxy — after new bid, check others' max_amount and outbid incrementally
- TASK-041: Anti-sniping — if time_remaining < 5min, extend ends_at += 5min
- TASK-042: Auction WS Hub — broadcast {type, current_bid, bid_count, ends_at} on every bid
- TASK-043: GET /ws/auctions/:id — WebSocket upgrade, join auction room
- TASK-044: Auction end cron — every minute, mark ended auctions, set winner_id, notify
- TASK-045: Frontend auctions list — cards with live countdown timers
- TASK-046: Frontend auction detail — live bid ticker, countdown, bid history, place bid modal
- TASK-047: Frontend useAuctionWebSocket — connect, update state on bid_update, reconnect

**EPIC 5 — Chat** (TASK-048 → TASK-059)
- TASK-048: POST /chat/conversations — create or return existing for (user_a, user_b, listing_id)
- TASK-049: GET /chat/conversations — ordered by last_msg_at DESC, with unread_count
- TASK-050: GET /chat/conversations/:id/messages — paginated, verify membership
- TASK-051: POST /chat/conversations/:id/messages — save, update last_msg_at, increment unread
- TASK-052: Chat WS Hub — rooms by conversation_id, broadcast to all members
- TASK-053: GET /chat/conversations/:id/ws — WS upgrade, verify JWT from query param
- TASK-054: Read receipts — GET messages resets unread_count for requesting user
- TASK-055: GET /chat/unread — total unread across all conversations
- TASK-056: Frontend chat list — avatar, name, last message preview, unread badge
- TASK-057: Frontend message thread — bubbles, timestamps, ✓✓ read ticks, auto-scroll
- TASK-058: Frontend useChatWebSocket — send/receive, update message list, reconnect
- TASK-059: Frontend contact seller button — POST /conversations, redirect to /chat/:id

**EPIC 6 — Payments** (TASK-060 → TASK-067)
- TASK-060: Stripe init + webhook with signature verification
- TASK-061: POST /payments/intent — create PaymentIntent with metadata
- TASK-062: GET /payments/key — return STRIPE_PUBLISHABLE_KEY
- TASK-063: Featured listing — 7d (500¢) / 30d (1500¢) tiers, set is_featured on webhook
- TASK-064: Auction deposit — optional hold for high-value auctions (reserve > $1000)
- TASK-065: Webhook handler — payment_intent.succeeded → update payment status + trigger action
- TASK-066: Frontend featured modal — tier picker, Stripe Elements card form
- TASK-067: Frontend Stripe Elements — @stripe/stripe-js, confirm payment, success/error states

**EPIC 7 — Reviews & Trust** (TASK-068 → TASK-074)
- TASK-068: POST /users/:id/reviews — rating 1-5, verify prior transaction, one per pair per listing
- TASK-069: GET /users/:id/reviews — paginated with reviewer name + avatar
- TASK-070: Rating aggregation — update users.rating + review_count after new review
- TASK-071: POST /listings/:id/report — reason enum + description
- TASK-072: POST /users/:id/report — same structure, type=user
- TASK-073: Frontend review form — star picker, textarea, submit
- TASK-074: Frontend star rating component — fractional display, count

**EPIC 8 — Admin Panel** (TASK-075 → TASK-082)
- TASK-075: Admin middleware — check role = "admin" from JWT, 403 otherwise
- TASK-076: GET /admin/listings — all statuses, search, pagination
- TASK-077: POST /admin/listings/:id/approve and /reject with reason
- TASK-078: GET /admin/users — search by name/email, is_blocked filter
- TASK-079: POST /admin/users/:id/block and /unblock
- TASK-080: GET /admin/reports — queue filtered by type + status
- TASK-081: GET /admin/stats — users, listings, auctions, revenue totals, new today
- TASK-082: Frontend admin — Next.js /admin/* layout, sidebar, data tables, action buttons

**EPIC 9 — Performance & DevOps** (TASK-083 → TASK-090)
- TASK-083: Meilisearch sync — index on create/update, remove on delete, use for text search
- TASK-084: Redis cache — categories (5min), featured listings (2min), user profiles (10min)
- TASK-085: Query audit — EXPLAIN ANALYZE listings search + auction bids + messages, add indexes
- TASK-086: Gzip middleware — compress responses > 1KB
- TASK-087: K8s manifests — Deployment (3 replicas), Service, Ingress (TLS), HPA (CPU 70%)
- TASK-088: Prometheus — /metrics, track requests, latency, WS connections, bids placed
- TASK-089: Structured logging — correlation_id per request, log method/path/status/latency/user_id
- TASK-090: Sentry — backend gin middleware + frontend @sentry/nextjs, capture 5xx

**EPIC 10 — AI Features** (TASK-091 → TASK-098)
- TASK-091: pgvector — install extension, add embedding vector(1536) to listings, ivfflat index
- TASK-092: Auto-embed — goroutine after listing create, call OpenAI embeddings, store vector
- TASK-093: POST /ai/search — embed query, cosine similarity `<=>`, return top 20
- TASK-094: POST /ai/categorize — GPT-4o with category list, return top 3 with confidence scores
- TASK-095: GET /ai/price-suggest — 20 similar by category+condition+location, return min/max/suggested
- TASK-096: POST /ai/moderate — GPT-4o moderation before listing goes active
- TASK-097: Frontend AI search bar — debounce 500ms, calls /ai/search, shows semantic results
- TASK-098: Frontend price tooltip — after title+category+condition filled, fetch /ai/price-suggest

---

## Output Format

```
---
# 📋 DOCUMENT 1: PRD.md
---
[full PRD]

---
# 🤖 DOCUMENT 2: CLAUDE.md
---
[full CLAUDE.md — references actual code from geocore-next]

---
# ✅ DOCUMENT 3: TASKS.md
---
[all 98 tasks, every one fully written with Go Status field]
```

**Hard requirements:**
- Read BOTH repos before writing anything
- The "What I Found" summary must come first
- CLAUDE.md must quote real function/file names from geocore-next
- All 98 tasks fully written — no skipping
- Minimum 5000 words total
- Zero placeholders
