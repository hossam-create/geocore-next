# 📋 DOCUMENT 1: PRD.md — GeoCore Next Product Requirements

---

## What I Found — Gap Analysis Summary

### PHP Platform (geocore-community) Features Identified
| Domain | PHP Feature | Status |
|---|---|---|
| Auth | Session-based login, registration, password reset | Exists |
| Listings | Full CRUD, categories, search, browse by tag/seller/featured, RSS feed | Exists |
| Auctions | Standard, Dutch, Reverse auctions, auto-bidding proxy, anti-sniping extension, buy-now, reserve price, delayed start | Exists |
| Bidding | Blacklist/invited list for bidders, subscription gating, verified account requirement, bid quantity (Dutch) | Exists |
| Users | Full user management (profile, favorites, current/expired ads, bids, balance, communications), seller stores | Exists |
| Payments | Stripe, PaymentExpress, order items system, balance/transactions | Exists |
| Admin | Full admin panel (listings, auctions, users, reports, settings) | Exists |
| Search | Multi-field search with geo boundaries | Exists |
| Cron | Recurring processes, listing expiry, auction end | Exists |
| i18n | Multi-language via message codes | Exists |
| Geo | Geo-based browsing and filtering | Exists |
| Affiliate | Affiliate system | Exists |
| Feedback | Auction feedback/reviews | Exists |
| Voting | Listing voting system | Exists |
| Notify | Email notifications to friend/seller | Exists |

### Go Rebuild (geocore-next) Current State
| Domain | Go Status | Notes |
|---|---|---|
| Auth | **Partial** | Register + Login work, bcrypt cost 12. **Missing**: refresh tokens, Google OAuth, password reset |
| Listings | **Partial** | Full CRUD with filters, categories seeded (EN+AR), favorites toggle, view count. **Missing**: image upload (accepts URLs only), expiry cron, Meilisearch sync |
| Auctions | **Partial** | Create + PlaceBid with TX + List + Get with top bids. **Missing**: auto-bid proxy, anti-sniping, auction end cron, SELECT FOR UPDATE, Dutch/Reverse types |
| Chat | **Partial** | Conversations CRUD, messages with unread count, WebSocket hub. **Missing**: WS doesn't save to DB, no typing indicators, pagination |
| Users | **Partial** | GetMe + UpdateMe + public profile with ToPublic(). **Missing**: avatar upload, phone verification |
| Payments | **Partial** | Stripe PaymentIntent + publishable key. **Missing**: webhook, featured listing flow, deposit, payment model |
| Reviews | **Not started** | No model, no endpoints |
| Admin | **Not started** | AdminOnly middleware exists but no endpoints, no role in JWT |
| Reports | **Not started** | No model, no endpoints |
| Cron | **Not started** | No cron/scheduler setup |
| Search | **Partial** | ILIKE text search. **Missing**: Meilisearch, vector/semantic search |
| Frontend | **Not started** | No Next.js app yet |
| DevOps | **Partial** | Docker Compose with 5 services. **Missing**: CI/CD, K8s, monitoring, structured logging |

### Patterns & Inconsistencies in Go Code
1. **Duplicate `getenv`** — defined in both `main.go` and `database.go`
2. **Duplicate `defaultStr`** — defined in both `listings/handler.go` and `chat/handler.go`
3. **No connection retry** — DB and Redis connect once, no retry logic
4. **No rate limiting** — publicly accessible endpoints have no throttling
5. **Auth single token** — 30-day JWT with no access/refresh split
6. **Redis pkg unused** — `pkg/redis/redis.go` defines helpers but `main.go` creates its own client
7. **AdminOnly dead code** — reads `user_role` from context but auth middleware only sets `user_id` and `user_email`
8. **PlaceBid no SELECT FOR UPDATE** — race condition on concurrent bids
9. **Chat WS route param mismatch** — `ServeWS` reads `c.Param("conversationId")` but route defines `/:id`
10. **Listing `GET /me`** route registered AFTER `GET /:id` — Gin will match `/me` as an `:id` parameter

---

## 1. Executive Summary

### Vision
GeoCore Next is a modern, high-performance classifieds and auctions marketplace platform that replaces the legacy PHP-based GeoCore Community with a Go backend and Next.js frontend, delivering real-time auctions, instant messaging, AI-powered search, and mobile-first experience.

### Problem
The PHP platform (GeoCore Community) suffers from an aging PHP 7.4 codebase with MariaDB, Smarty templates, no real API layer, session-based auth, polling-based updates, no mobile support, SQL injection risks in raw query construction, and cron-dependent auction/listing lifecycle that scales poorly.

### Solution
The Go rebuild provides a REST+WebSocket API on Gin with PostgreSQL (PostGIS), Redis pub/sub for real-time features, JWT auth, Stripe integration, Meilisearch full-text + pgvector semantic search, and a Next.js 15 App Router frontend with shadcn/ui.

### KPIs
| KPI | Target |
|---|---|
| API p95 latency | < 50ms (vs ~800ms PHP) |
| Concurrent WebSocket connections | 10,000+ per pod |
| Listing search response time | < 100ms (Meilisearch) |
| Time to first meaningful paint | < 1.5s (Next.js SSR) |
| Uptime SLA | 99.9% |

---

## 2. User Personas

### Persona 1: Ahmed (Seller)
- **Age:** 34 | **Country:** UAE | **Occupation:** Used car dealer
- **Goals:** List vehicles quickly with photos, reach buyers across GCC, accept offers via chat
- **Pain Points:** PHP admin panel is slow, no mobile app, no way to promote listings, image upload breaks often
- **Tech Level:** Intermediate
- **Quote:** *"I need my listings to show up first and get messages from serious buyers instantly."*

### Persona 2: Sarah (Buyer & Bidder)
- **Age:** 28 | **Country:** Egypt | **Occupation:** Interior designer
- **Goals:** Find unique furniture at auction, set auto-bid limits, get notified when outbid
- **Pain Points:** Auction page doesn't update in real-time (must refresh), sniped at last second with no protection, no bid history clarity
- **Tech Level:** Advanced
- **Quote:** *"I lost 3 auctions because someone bid in the last 2 seconds — I need anti-sniping and live updates."*

### Persona 3: Khalid (Administrator)
- **Age:** 42 | **Country:** Saudi Arabia | **Occupation:** Platform moderator
- **Goals:** Review flagged listings, block scam users, see revenue dashboards, approve high-value auctions
- **Pain Points:** PHP admin panel has no search, no bulk actions, can't see revenue at a glance
- **Tech Level:** Basic
- **Quote:** *"I spend 2 hours a day reviewing reported ads — I need AI moderation and a faster admin panel."*

---

## 3. Feature Requirements

### P0 — MVP (PHP Feature Parity)

#### User Registration & Authentication
**Priority:** P0 | **Parity:** Improved | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a user, I want to register and login securely so that I can access my account.
**Acceptance Criteria:**
- [ ] POST /auth/register — bcrypt cost 12, returns access (15min) + refresh (30d) tokens
- [ ] POST /auth/login — returns same token pair
- [ ] POST /auth/refresh — validates refresh, rotates both, invalidates old in Redis
- [ ] Passwords must be 8+ chars
**Edge Cases:**
- Duplicate email → 400 with clear message
- Expired refresh token → 401, force re-login
- Concurrent refresh → only first succeeds (Redis atomic check)

#### Category Browsing
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Done
**User Story:** As a buyer, I want to browse categories so that I can find relevant listings.
**Acceptance Criteria:**
- [ ] GET /categories returns tree with children
- [ ] 10 categories seeded with EN + AR names
- [ ] Categories cached in Redis for 5 minutes
**Edge Cases:**
- Empty categories should not appear in tree (optional filter)

#### Listing CRUD
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a seller, I want to create, edit, and delete listings so that I can manage my inventory.
**Acceptance Criteria:**
- [ ] POST /listings — validate all fields, expires_at = now+60d, status = active
- [ ] PUT /listings/:id — owner-only, whitelist updatable fields
- [ ] DELETE /listings/:id — soft delete, block if active auction exists
- [ ] POST /listings/:id/images — up to 10 files via Cloudinary, first = is_cover
- [ ] GET /listings/:id — preload Images + Category, increment view_count async
- [ ] GET /listings — q, category_id, country, city, min_price, max_price, condition, type, sort, page, per_page
- [ ] GET /listings/me — current user's listings filtered by status
**Edge Cases:**
- Cannot delete listing with active auction → 409
- Image count exceeds 10 → 400

#### Listing Search & Filtering
**Priority:** P0 | **Parity:** Improved | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a buyer, I want to search and filter listings so that I can find what I'm looking for.
**Acceptance Criteria:**
- [ ] Full-text search via Meilisearch (fallback to ILIKE)
- [ ] Filter by category, country, city, price range, condition, type
- [ ] Sort by newest, price_asc, price_desc, popular
- [ ] Paginated with total count and page metadata
**Edge Cases:**
- Empty search results → return empty array, not 404
- Extremely long query strings → truncate to 200 chars

#### Auction Lifecycle
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a seller, I want to create an auction for my listing so that buyers can bid on it.
**Acceptance Criteria:**
- [ ] POST /auctions — linked to listing, validate time range, max 30 days
- [ ] GET /auctions — active only, sorted by ends_at ASC, computed time_remaining field
- [ ] GET /auctions/:id — top 20 bids, is_reserve_met boolean, time_remaining
- [ ] POST /auctions/:id/bid — SELECT FOR UPDATE TX, validate > current_bid, block self-bid
- [ ] Auto-bid proxy — after new bid, check other bidders' max_amount and outbid incrementally
- [ ] Anti-sniping — if time_remaining < 5min, extend ends_at += 5min
- [ ] Auction end cron — every minute, mark ended auctions, set winner_id, notify participants
**Edge Cases:**
- Anti-sniping can extend max 3 times (15min total)
- Self-bid → 400 "Cannot bid on your own auction"
- Bid exactly at end time → honor if within 1s tolerance
- Reserve not met → auction ends without winner

#### Real-Time Auction Updates
**Priority:** P0 | **Parity:** New (PHP used polling) | **PHP Status:** Missing | **Go Status:** Partial
**User Story:** As a bidder, I want to see bids update in real-time so that I can react quickly.
**Acceptance Criteria:**
- [ ] WebSocket at /ws/auctions/:id
- [ ] Broadcasts `{type: "bid_update", current_bid, bid_count, ends_at}` on every bid
- [ ] Broadcasts `{type: "auction_ended", winner_id}` when auction closes
**Edge Cases:**
- Client disconnect → clean up from Hub, attempt reconnect from frontend

#### In-App Chat
**Priority:** P0 | **Parity:** Improved | **PHP Status:** Exists (email-based) | **Go Status:** Partial
**User Story:** As a buyer, I want to message a seller in real-time so that I can ask questions about a listing.
**Acceptance Criteria:**
- [ ] POST /chat/conversations — create or return existing for (user_a, user_b, listing_id)
- [ ] GET /chat/conversations — ordered by last_msg_at DESC, with unread_count
- [ ] GET /chat/conversations/:id/messages — paginated, verify membership
- [ ] POST /chat/conversations/:id/messages — save, update last_msg_at, increment unread
- [ ] WebSocket at /chat/conversations/:id/ws — real-time message delivery
- [ ] Read receipts — GET messages resets unread_count
**Edge Cases:**
- User blocked by other → prevent conversation creation
- Very long messages → truncate to 2000 chars

#### User Profile Management
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a user, I want to manage my profile so other users can learn about me.
**Acceptance Criteria:**
- [ ] GET /users/me — full user object excluding password_hash
- [ ] PUT /users/me — name, bio, location, language, currency
- [ ] GET /users/:id/profile — public profile (no PII)
- [ ] POST /users/me/avatar — Cloudinary upload, update avatar_url
**Edge Cases:**
- Avatar file > 5MB → 400

#### Payments
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Partial
**User Story:** As a seller, I want to promote my listing as featured so that it gets more visibility.
**Acceptance Criteria:**
- [ ] Stripe init + webhook with signature verification
- [ ] POST /payments/intent — create PaymentIntent with listing metadata
- [ ] GET /payments/key — return STRIPE_PUBLISHABLE_KEY
- [ ] Featured listing — 7d ($5) / 30d ($15) tiers
- [ ] Webhook handler — payment_intent.succeeded → update payment status + set is_featured
**Edge Cases:**
- Duplicate webhook delivery → idempotent handler
- Payment fails → listing stays non-featured

#### Favorites
**Priority:** P0 | **Parity:** Full parity | **PHP Status:** Exists | **Go Status:** Done
**User Story:** As a buyer, I want to save listings I like so I can return to them later.
**Acceptance Criteria:**
- [ ] POST /listings/:id/favorite — toggle on/off
- [ ] Update favorite_count atomically
**Edge Cases:**
- Favoriting a deleted listing → 404

### P1 — Improvements Over PHP

#### Google OAuth
**Priority:** P1 | **Parity:** New | **PHP Status:** Missing | **Go Status:** Not started
**User Story:** As a user, I want to sign in with Google so that I don't need another password.
**Acceptance Criteria:**
- [ ] POST /auth/google — OAuth 2.0 flow, upsert user on first login
- [ ] Link Google account to existing email account
**Edge Cases:**
- Google account email already registered → link accounts

#### Reviews & Trust System
**Priority:** P1 | **Parity:** Improved | **PHP Status:** Exists (feedback) | **Go Status:** Not started
**User Story:** As a buyer, I want to see seller reviews so that I can trust them before transacting.
**Acceptance Criteria:**
- [ ] POST /users/:id/reviews — rating 1-5, verify prior transaction, one per pair per listing
- [ ] GET /users/:id/reviews — paginated with reviewer name + avatar
- [ ] Rating aggregation — update users.rating + review_count after new review
- [ ] POST /listings/:id/report — reason enum + description
- [ ] POST /users/:id/report — same structure, type=user
**Edge Cases:**
- Self-review → 400
- Review without transaction → 403

#### Admin Panel
**Priority:** P1 | **Parity:** Improved | **PHP Status:** Exists | **Go Status:** Not started
**User Story:** As an admin, I want to manage users, listings, and reports from a dashboard.
**Acceptance Criteria:**
- [ ] Admin middleware — check role = "admin" from JWT, 403 otherwise
- [ ] GET /admin/listings — all statuses, search, pagination
- [ ] POST /admin/listings/:id/approve and /reject with reason
- [ ] GET /admin/users — search by name/email, is_blocked filter
- [ ] POST /admin/users/:id/block and /unblock
- [ ] GET /admin/reports — queue filtered by type + status
- [ ] GET /admin/stats — users, listings, auctions, revenue totals, new today
**Edge Cases:**
- Blocking a user with active auctions → end all auctions, refund deposits

#### Real-Time Chat Improvements
**Priority:** P1 | **Parity:** Improved | **PHP Status:** Polling | **Go Status:** Partial
**User Story:** As a user, I want live chat with read receipts and typing indicators.
**Acceptance Criteria:**
- [ ] Typing indicators via WebSocket
- [ ] Online/offline status via last_seen_at
- [ ] Unread badge count across all conversations
- [ ] Image/offer message types

### P2 — New Features PHP Never Had

#### AI Auto-Categorization
**Priority:** P2 | **Parity:** New | **PHP Status:** Missing | **Go Status:** Not started
**User Story:** As a seller, I want the platform to suggest categories for my listing so that I don't have to browse through the tree.
**Acceptance Criteria:**
- [ ] POST /ai/categorize — GPT-4o with category list, return top 3 with confidence scores
**Edge Cases:**
- Ambiguous listings → return multiple suggestions with lower confidence

#### AI Price Suggestion
**Priority:** P2 | **Parity:** New | **PHP Status:** Missing | **Go Status:** Not started
**User Story:** As a seller, I want price suggestions based on similar items so that I price competitively.
**Acceptance Criteria:**
- [ ] GET /ai/price-suggest — 20 similar by category+condition+location, return min/max/suggested

#### Semantic Vector Search
**Priority:** P2 | **Parity:** New | **PHP Status:** Missing | **Go Status:** Not started
**User Story:** As a buyer, I want natural language search ("comfortable chair for reading") instead of exact keyword matching.
**Acceptance Criteria:**
- [ ] pgvector extension with embedding vector(1536) on listings
- [ ] POST /ai/search — embed query, cosine similarity, return top 20

#### AI Content Moderation
**Priority:** P2 | **Parity:** New | **PHP Status:** Missing | **Go Status:** Not started
**User Story:** As an admin, I want AI to pre-screen listings before they go live.
**Acceptance Criteria:**
- [ ] POST /ai/moderate — GPT-4o moderation before listing goes active
- [ ] Auto-flag listings with inappropriate content

---

## 4. Non-Functional Requirements

### Performance
| Metric | PHP Baseline | Go Target |
|---|---|---|
| API p95 latency | ~800ms | < 50ms |
| Listings search | ~2s | < 100ms |
| Concurrent users | ~500 | 50,000+ |
| WS connections | N/A (polling) | 10,000+ per pod |
| Cold start | ~5s | < 500ms |

### Scalability
- Horizontal scaling via Kubernetes HPA (target CPU 70%)
- 3 replicas minimum in production
- Redis cluster for pub/sub fan-out across pods
- Connection pooling: 10 idle, 100 max open DB connections

### Security
- bcrypt cost 12 for password hashing (fixed from PHP's weaker hashing)
- JWT with short-lived access tokens (15min) + refresh tokens
- Stripe webhook signature verification
- Input validation on all endpoints via Gin bindings
- CORS restricted to FRONTEND_URL
- SQL injection prevention via GORM parameterized queries (vs PHP's raw SQL)
- Rate limiting: 100 req/min per IP

### Availability
- 99.9% uptime SLA
- Graceful shutdown with 30s drain period
- Health check endpoint at GET /health
- PostgreSQL health checks in Docker

### Internationalization
- Category names in EN + AR
- User-selectable language and currency preferences
- RTL layout support in Next.js frontend

---

## 5. API Contract

### POST /api/v1/auth/register
**Request:**
```json
{
  "name": "Ahmed Hassan",
  "email": "ahmed@example.com",
  "password": "SecurePass123!"
}
```
**Response (201):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Ahmed Hassan",
      "email": "ahmed@example.com",
      "role": "user",
      "language": "en",
      "currency": "USD",
      "created_at": "2026-03-05T04:00:00Z"
    }
  }
}
```

### POST /api/v1/auth/login
**Request:**
```json
{
  "email": "ahmed@example.com",
  "password": "SecurePass123!"
}
```
**Response (200):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "user": { "id": "...", "name": "Ahmed Hassan", "email": "ahmed@example.com" }
  }
}
```

### GET /api/v1/listings?q=car&category_id=xxx&country=UAE&city=Dubai&min_price=1000&max_price=50000&condition=used&type=sell&sort=price_asc&page=1&per_page=20
**Response (200):**
```json
{
  "success": true,
  "data": [
    {
      "id": "...",
      "title": "Toyota Camry 2023",
      "description": "Low mileage, single owner...",
      "price": 25000.00,
      "currency": "USD",
      "price_type": "negotiable",
      "condition": "used",
      "type": "sell",
      "country": "UAE",
      "city": "Dubai",
      "view_count": 142,
      "favorite_count": 23,
      "is_featured": true,
      "expires_at": "2026-05-05T04:00:00Z",
      "created_at": "2026-03-05T04:00:00Z",
      "images": [
        { "id": "...", "url": "https://res.cloudinary.com/...", "is_cover": true }
      ],
      "category": { "id": "...", "name_en": "Vehicles", "name_ar": "السيارات", "slug": "vehicles" }
    }
  ],
  "meta": { "total": 150, "page": 1, "per_page": 20, "pages": 8 }
}
```

### POST /api/v1/listings
**Request:**
```json
{
  "category_id": "550e8400-...",
  "title": "iPhone 15 Pro Max",
  "description": "Brand new, sealed box, 256GB",
  "price": 1200.00,
  "currency": "USD",
  "price_type": "fixed",
  "condition": "new",
  "type": "sell",
  "country": "UAE",
  "city": "Abu Dhabi",
  "latitude": 24.4539,
  "longitude": 54.3773,
  "image_urls": ["https://res.cloudinary.com/..."]
}
```
**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "...", "title": "iPhone 15 Pro Max", "status": "active",
    "expires_at": "2026-05-05T04:00:00Z"
  }
}
```

### POST /api/v1/auctions/:id/bid
**Request:**
```json
{
  "amount": 26000.00,
  "is_auto": true,
  "max_amount": 30000.00
}
```
**Response (201):**
```json
{
  "success": true,
  "data": {
    "id": "...", "auction_id": "...", "user_id": "...",
    "amount": 26000.00, "is_auto": true, "max_amount": 30000.00,
    "placed_at": "2026-03-05T04:30:00Z"
  }
}
```

### WebSocket: Auction Bid Broadcast
```json
{
  "type": "bid_update",
  "current_bid": 26000.00,
  "bid_count": 15,
  "ends_at": "2026-03-10T18:00:00Z",
  "bidder_name": "Sarah M."
}
```

### WebSocket: Chat Message
```json
{
  "type": "new_message",
  "content": "Is this still available?",
  "sender_id": "550e8400-...",
  "sender_name": "Sarah M.",
  "created_at": "2026-03-05T04:30:00Z"
}
```

---

## 6. UI/UX Requirements

### Homepage
- **Layout:** Full-width hero with search bar, horizontal scrolling category bar with emoji icons, featured listings grid (3 columns), active auctions with live countdown timers, recently added section
- **Key Components:** SearchBar, CategoryBar, ListingCard, AuctionCountdown, FeaturedBadge
- **Improvement over PHP:** Modern responsive grid vs. PHP's table-based layout; live auction countdowns vs. static timestamps

### Listing Detail
- **Layout:** Image gallery (swipeable), info panel (price, condition, seller, location), Leaflet/Mapbox map, "Contact Seller" CTA, similar listings row
- **Key Components:** ImageGallery, PriceDisplay, SellerCard, MapEmbed, SimilarListings
- **Improvement over PHP:** Full-screen image gallery vs. thumbnail grid; instant chat vs. email form

### Create Listing (Multi-Step Wizard)
- **Layout:** 3-step wizard: (1) Category + Title + Description, (2) Price + Condition + Location, (3) Photos + Review
- **Key Components:** StepIndicator, CategoryPicker, PriceInput, LocationPicker, ImageUploader, ReviewSummary
- **Improvement over PHP:** Guided multi-step vs. single long form; drag-and-drop image upload

### Auction Detail (Live)
- **Layout:** Hero with countdown timer, current bid display, bid history table, place bid modal, related auctions
- **Key Components:** CountdownTimer, BidTicker (live), BidHistory, PlaceBidModal, AutoBidConfig
- **Improvement over PHP:** Real-time bid updates via WebSocket; visual anti-sniping indicator

### Chat
- **Layout:** Split view — conversation list (left), message thread (right), compose area bottom
- **Key Components:** ConversationList, MessageBubble, ComposeBar, UnreadBadge, ReadReceipt
- **Improvement over PHP:** Real-time WebSocket messaging vs. page-refresh polling

### User Profile
- **Layout:** Avatar + name header, rating with stars, tabs (listings, reviews, about)
- **Key Components:** AvatarUpload, StarRating, ListingTab, ReviewTab
- **Improvement over PHP:** Modern card-based design vs. table layout

### Admin Panel
- **Layout:** Sidebar navigation, top stats cards, data tables with search/filter, action buttons
- **Key Components:** StatsCard, DataTable, ApproveRejectButtons, UserBlockToggle, ReportQueue
- **Improvement over PHP:** React-based SPA with real-time stats vs. PHP page reloads

---

## 7. PHP → Go Migration Strategy

### MySQL → PostgreSQL Schema Mapping
| PHP (MariaDB) | Go (PostgreSQL) | Notes |
|---|---|---|
| `INT AUTO_INCREMENT` | `UUID DEFAULT uuid_generate_v4()` | UUIDs for all PKs |
| `DATETIME` | `TIMESTAMPTZ` | Timezone-aware |
| `VARCHAR(255)` | `TEXT` / `VARCHAR` with GORM tags | Flexible lengths |
| `TINYINT(1)` | `BOOLEAN` | Native booleans |
| `FLOAT` | `NUMERIC(12,2)` / `FLOAT8` | Precision for prices |
| `deleted_at IS NULL` check | `gorm.DeletedAt` (soft delete) | Built-in GORM pattern |
| Raw SQL via ADOdb | GORM ORM | Parameterized queries |
| Manual indexes | GORM tags + manual SQL | `gorm:"index"` |

### Data Migration Script Approach
1. **Export** — Dump MariaDB tables to CSV via `SELECT INTO OUTFILE` or mysqldump
2. **Transform** — Python/Go script to:
   - Generate UUIDs for each row (store old_id→new_uuid mapping)
   - Update foreign keys using mapping table
   - Convert DATETIME to TIMESTAMPTZ
   - Hash existing plain-text passwords with bcrypt (if any)
   - Convert image paths to Cloudinary URLs (batch upload)
3. **Load** — Use PostgreSQL `COPY FROM` for bulk insert
4. **Verify** — Row count comparison, spot-check 100 random records per table

### Zero-Downtime Cutover Plan
1. **Phase 1 (Week −4):** Run Go API in shadow mode, mirror all PHP writes to PostgreSQL
2. **Phase 2 (Week −2):** Enable read traffic split (10% Go, 90% PHP), compare responses
3. **Phase 3 (Week −1):** Full data sync, final consistency check, DNS TTL lowered to 60s
4. **Phase 4 (D-Day):** DNS swap to Go API, PHP enters read-only mode for 48h as rollback safety
5. **Phase 5 (D+2):** Decommission PHP if no issues; archive MariaDB backup
