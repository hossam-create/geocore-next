# GeoCore Next — Implementation Task Guide
  > **GCC Marketplace Platform**: Classifieds · Live Auctions · Messaging · Wallet · Notifications  
  > **Stack**: Go (Gin + GORM + Redis) · Expo React Native · React + Vite · PostgreSQL  
  > **Colors**: Primary `#0071CE` · Secondary `#FFC220` · Sidebar `#1B2559`  
  > **API Base**: `https://geo-core-next.replit.app/api/v1`  
  > **Repo**: `github.com/hossam-create/geocore-next`

  ---

  ## 📊 Progress Overview

  | Phase | Focus | Status | Week |
  |-------|-------|--------|------|
  | [Phase 1](#phase-1-security--authentication) | Security & Authentication | ✅ Complete | 1–2 |
  | [Phase 2](#phase-2-payment-system) | Payment System (Stripe) | ✅ Complete | 3–4 |
  | [Phase 3](#phase-3-image-management) | Image Upload (Cloudflare R2) | ✅ Complete | 5 |
  | [Phase 4](#phase-4-advanced-search) | Advanced Search & Filters | ✅ Complete | 6 |
  | [Phase 5](#phase-5-notification-system) | Push/Email/In-App Notifications | ✅ Complete | 7 |
  | [Phase 6](#phase-6-admin-dashboard) | Admin Dashboard | ✅ Complete | 8–9 |
  | [Phase 7](#phase-7-testing--optimization) | Testing Suite + Load Tests | ✅ Complete | 10 |
  | [Phase 8](#phase-8-production-deployment) | Docker + K8s + CI/CD | ✅ Complete | 11 |

  ### Task Completion Status

  | Task | Description | Status |
  |------|-------------|--------|
  | 1.1 | Email Verification System | ✅ Done |
  | 1.2 | Password Reset System | ✅ Done |
  | 1.3 | Social Login (Google / Apple / Facebook) | ✅ Done |
  | 1.4 | Rate Limiting Middleware | ✅ Done |
  | 1.5 | Input Validation & Sanitization | ✅ Done |
  | 2.1 | Stripe Integration – Payment Intents + Escrow | ✅ Done |
  | 2.2 | Stripe Webhooks + Event Queue | ✅ Done |
  | 3.1 | Image Upload → Cloudflare R2 + Processing | ✅ Done |
  | 4.1 | Advanced Search + Haversine + Facets + Redis Cache | ✅ Done |
  | 5.1 | Full Notification System (in-app, email, push, WebSocket) | ✅ Done |
  | 6.1 | Admin API + React Dashboard UI (Vite + shadcn) | ✅ Done |
  | 7.1 | Unit / Integration / E2E / Load Tests | ⬜ Pending |
  | 8.1 | Docker + Kubernetes + Monitoring | ✅ Done |

  ---

  ## 🔑 Completed Work (Reference)

  ### ✅ 1.1 Email Verification
  - Endpoint: `POST /api/v1/auth/verify-email`
  - Endpoint: `POST /api/v1/auth/resend-verification`
  - 64-char token via `crypto/rand`, 24-hour TTL
  - Redis rate-limit on resend (5/hour per user)
  - `email_verified` / `verification_token` / `verification_token_expires_at` columns on `users`

  ### ✅ 1.2 Password Reset
  - `POST /api/v1/auth/forgot-password` — rate-limited, email-enumeration-safe
  - `POST /api/v1/auth/validate-reset-token` — UI guard (check before showing form)
  - `POST /api/v1/auth/reset-password` — validates strength, revokes all prior JWTs
  - Redis key `revoke-before:{userID}` read by `middleware.Auth()` for JWT revocation
  - Security emails: reset-request + change-confirmation
  - DB fields: `password_reset_token`, `password_reset_expires_at`, `password_changed_at`

  ### ✅ 1.3 Social Login
  - `POST /api/v1/auth/social` — accepts `{provider, token}`
  - Providers: Google (verifies with tokeninfo API) · Apple (JWT verify) · Facebook (graph API)
  - Creates new user or links to existing account by email
  - Mobile: `SocialAuthButtons.tsx` with Google / Apple (iOS only) / Facebook
  - DB fields: `google_id`, `apple_id`, `facebook_id`, `auth_provider`

  ---

  ## Phase 1: Security & Authentication

  ### Task 1.4 — Rate Limiting Middleware
  **File:** `backend/pkg/middleware/ratelimit.go`

  **What to build:**
  Redis sliding-window rate limiter as a reusable Gin middleware.

  **Rate limit rules:**

  | Endpoint | Limit | Window | Key |
  |----------|-------|--------|-----|
  | Global (per IP) | 100 req | 15 min | `ratelimit:ip:{ip}:{endpoint}` |
  | Global (authenticated) | 200 req | 15 min | `ratelimit:user:{uid}:{endpoint}` |
  | `POST /auth/register` | 5 req | 1 hour | per IP |
  | `POST /auth/login` | 10 req | 15 min | per IP |
  | `POST /auth/forgot-password` | 3 req | 1 hour | per IP |
  | `POST /auth/resend-verification` | 5 req | 1 hour | per user |
  | `POST /listings` (create) | 20 req | 1 day | per user |
  | `POST /auctions/:id/bid` | 50 req | 1 hour | per user |

  **Middleware signature:**
  ```go
  func RateLimitMiddleware(limit int, window time.Duration, keyPrefix string) gin.HandlerFunc
  ```

  **Response headers to add:**
  - `X-RateLimit-Limit`
  - `X-RateLimit-Remaining`
  - `X-RateLimit-Reset` (Unix timestamp)

  **429 Response body:**
  ```json
  {
    "error": "rate_limit_exceeded",
    "message": "Too many requests. Please try again in X minutes.",
    "retry_after": 300
  }
  ```

  **Extra features:**
  - Whitelist IPs (admin IPs from `RATE_LIMIT_WHITELIST` env var)
  - Bypass `/health` and `/ready` endpoints
  - Log rate-limit violations with IP + endpoint
  - Alert on abuse patterns (>10x limit in 1 minute)

  ---

  ### Task 1.5 — Input Validation & Sanitization
  **File:** `backend/pkg/validator/validator.go`

  **Dependencies:**
  - `github.com/go-playground/validator/v10`
  - `github.com/microcosm-cc/bluemonday` (HTML sanitization)

  **Key structs:**
  ```go
  type RegisterRequest struct {
      Email    string `json:"email"    validate:"required,email,max=255"`
      Password string `json:"password" validate:"required,min=8,max=128,password"`
      Name     string `json:"name"     validate:"required,min=2,max=100,alphanumunicode"`
      Phone    string `json:"phone"    validate:"omitempty,e164"`
  }

  type CreateListingRequest struct {
      Title       string   `json:"title"       validate:"required,min=5,max=200"`
      Description string   `json:"description" validate:"required,min=20,max=5000"`
      Price       float64  `json:"price"       validate:"required,min=0,max=999999999"`
      CategoryID  int64    `json:"category_id" validate:"required,gt=0"`
      Location    Location `json:"location"    validate:"required"`
      Images      []string `json:"images"      validate:"omitempty,max=10,dive,url"`
  }

  type Location struct {
      Latitude  float64 `json:"latitude"  validate:"required,latitude"`
      Longitude float64 `json:"longitude" validate:"required,longitude"`
      Address   string  `json:"address"   validate:"required,max=500"`
  }

  type PlaceBidRequest struct {
      AuctionID int64   `json:"auction_id" validate:"required,gt=0"`
      Amount    float64 `json:"amount"     validate:"required,gt=0"`
  }
  ```

  **Custom validators:**
  - `password`: min 8 chars + uppercase + lowercase + digit
  - `e164`: E.164 phone format (+971501234567)
  - `latitude`: -90 to 90
  - `longitude`: -180 to 180

  **Sanitization pipeline:**
  1. Trim whitespace
  2. Strip HTML tags (bluemonday strict policy)
  3. Remove control characters
  4. GORM parameterized queries (already protects SQL injection)

  **Error response format:**
  ```json
  {
    "error": "validation_failed",
    "message": "Validation errors occurred",
    "details": [
      {"field": "email", "message": "must be a valid email address"},
      {"field": "password", "message": "must be at least 8 characters with uppercase, lowercase, and number"}
    ]
  }
  ```

  **Middleware:**
  ```go
  func ValidateRequest(v interface{}) gin.HandlerFunc
  ```

  Support EN/AR error messages via `Accept-Language` header.

  ---

  ## Phase 2: Payment System

  ### Task 2.1 — Stripe Integration
  **Dependency:** `github.com/stripe/stripe-go/v76`  
  **Env vars:** `STRIPE_SECRET_KEY`, `STRIPE_PUBLISHABLE_KEY`, `STRIPE_WEBHOOK_SECRET`

  **Database schema:**
  ```sql
  CREATE TABLE payments (
      id                       BIGSERIAL PRIMARY KEY,
      user_id                  BIGINT NOT NULL REFERENCES users(id),
      listing_id               BIGINT REFERENCES listings(id),
      auction_id               BIGINT REFERENCES auctions(id),
      stripe_payment_intent_id VARCHAR(255) UNIQUE,
      amount                   DECIMAL(10,2) NOT NULL,
      currency                 VARCHAR(3) DEFAULT 'AED',
      status                   VARCHAR(50),   -- pending, succeeded, failed, refunded
      payment_method           VARCHAR(50),   -- card, wallet
      description              TEXT,
      metadata                 JSONB,
      created_at               TIMESTAMP DEFAULT NOW(),
      updated_at               TIMESTAMP DEFAULT NOW()
  );

  CREATE TABLE escrow_accounts (
      id          BIGSERIAL PRIMARY KEY,
      payment_id  BIGINT REFERENCES payments(id),
      seller_id   BIGINT REFERENCES users(id),
      buyer_id    BIGINT REFERENCES users(id),
      amount      DECIMAL(10,2),
      status      VARCHAR(50),   -- held, released, refunded
      released_at TIMESTAMP,
      created_at  TIMESTAMP DEFAULT NOW()
  );

  ALTER TABLE users ADD COLUMN stripe_customer_id VARCHAR(255) UNIQUE;
  ```

  **Endpoints:**

  | Method | Path | Auth | Description |
  |--------|------|------|-------------|
  | POST | `/payments/create-payment-intent` | ✅ | Create Stripe PaymentIntent |
  | POST | `/payments/confirm` | ✅ | Confirm and move to escrow |
  | POST | `/payments/release-escrow` | ✅ | Buyer confirms receipt → release |
  | POST | `/payments/request-refund` | ✅ | Request refund |
  | GET | `/payments/payment-methods` | ✅ | List saved cards |
  | POST | `/payments/add-payment-method` | ✅ | Save new card |
  | DELETE | `/payments/payment-methods/:id` | ✅ | Remove card |

  **Escrow rules:**
  - Payment → escrow immediately on `payment_intent.succeeded`
  - Auto-release after 7 days if buyer doesn't dispute
  - Manual release via `POST /payments/release-escrow`

  **Error cases to handle:**
  - Card declined (`card_declined`)
  - Insufficient funds
  - 3D Secure required (`authentication_required`)
  - Network errors
  - All `stripe.Error` types

  ---

  ### Task 2.2 — Stripe Webhooks
  **Endpoint:** `POST /api/v1/webhooks/stripe` (public, no auth)

  **Signature verification:**
  ```go
  func VerifyStripeSignature(payload []byte, sig string, secret string) (*stripe.Event, error)
  ```

  **Events to handle:**

  | Event | Action |
  |-------|--------|
  | `payment_intent.succeeded` | Update payment status, create escrow, send emails |
  | `payment_intent.payment_failed` | Update status, notify user, log reason |
  | `charge.refunded` | Update payment + escrow, notify buyer & seller |
  | `customer.subscription.created` | Unlock premium features |
  | `customer.subscription.deleted` | Downgrade user |

  **Idempotency table:**
  ```sql
  CREATE TABLE webhook_events (
      id              BIGSERIAL PRIMARY KEY,
      stripe_event_id VARCHAR(255) UNIQUE,
      event_type      VARCHAR(100),
      payload         JSONB,
      processed       BOOLEAN DEFAULT FALSE,
      processed_at    TIMESTAMP,
      retry_count     INT DEFAULT 0,
      error           TEXT,
      created_at      TIMESTAMP DEFAULT NOW()
  );
  ```

  **Processing rules:**
  - Return `200` immediately, process async (Redis queue + worker)
  - Idempotency: check `stripe_event_id` before processing
  - Retry failed events (max 3 attempts, exponential backoff)
  - Dead letter queue after max retries

  ---

  ## Phase 3: Image Management

  ### Task 3.1 — Image Upload (Cloudflare R2)
  **Dependencies:** `github.com/aws/aws-sdk-go-v2/service/s3`, `github.com/disintegration/imaging`

  **Env vars:**
  ```
  R2_ACCOUNT_ID=xxx
  R2_ACCESS_KEY_ID=xxx
  R2_SECRET_ACCESS_KEY=xxx
  R2_BUCKET_NAME=geocore-images
  R2_PUBLIC_URL=https://images.geocore.com
  ```

  **Database:**
  ```sql
  CREATE TABLE images (
      id                BIGSERIAL PRIMARY KEY,
      user_id           BIGINT REFERENCES users(id),
      listing_id        BIGINT REFERENCES listings(id),
      filename          VARCHAR(255),
      original_filename VARCHAR(255),
      url               TEXT,
      thumbnail_url     TEXT,
      size_bytes        BIGINT,
      width             INT,
      height            INT,
      format            VARCHAR(10),   -- jpg, png, webp
      is_primary        BOOLEAN DEFAULT FALSE,
      created_at        TIMESTAMP DEFAULT NOW()
  );

  CREATE INDEX idx_images_listing ON images(listing_id);
  ```

  **Endpoints:**

  | Method | Path | Description |
  |--------|------|-------------|
  | POST | `/images/upload` | Upload 1–10 images (multipart/form-data) |
  | DELETE | `/images/:id` | Delete image (verify ownership first) |

  **Upload constraints:**
  - Max 10 images per request
  - Max 5 MB per image
  - Allowed formats: JPEG, PNG, WebP (check magic bytes, not extension)
  - Max dimensions: 4000 × 4000 px

  **Processing pipeline:**
  1. Validate (mime type + size + dimensions)
  2. Convert to WebP
  3. Resize to 3 variants:
     - Original (capped at 1920 × 1920)
     - Medium (800 × 800)
     - Thumbnail (400 × 400, center-crop)
  4. Upload to R2 at path `{year}/{month}/{uuid}-{size}.webp`
  5. Save metadata to DB, return URLs

  **Helper functions:**
  ```go
  func ValidateImage(file multipart.File) error
  func ProcessImage(file multipart.File) ([]ProcessedImage, error)
  func UploadToR2(filename string, data []byte) (string, error)
  func DeleteFromR2(filename string) error
  func GenerateThumbnail(img image.Image) image.Image
  ```

  ---

  ## Phase 4: Advanced Search

  ### Task 4.1 — Advanced Search & Filters
  **Dependencies:** PostgreSQL 16 + PostGIS extension

  **Database indexes:**
  ```sql
  -- Full-text search
  ALTER TABLE listings ADD COLUMN search_vector tsvector;
  CREATE INDEX idx_listings_search ON listings USING GIN(search_vector);
  CREATE TRIGGER tsvector_update BEFORE INSERT OR UPDATE ON listings
      FOR EACH ROW EXECUTE FUNCTION
      tsvector_update_trigger(search_vector, 'pg_catalog.english', title, description);

  -- PostGIS location index
  CREATE INDEX idx_listings_location ON listings
      USING GIST(ST_MakePoint(longitude, latitude));

  -- Supporting indexes
  CREATE INDEX idx_listings_category ON listings(category_id);
  CREATE INDEX idx_listings_price    ON listings(price);
  CREATE INDEX idx_listings_created  ON listings(created_at DESC);
  CREATE INDEX idx_listings_status   ON listings(status);
  ```

  **Search endpoint:** `GET /api/v1/listings/search`

  **Query parameters:**
  ```go
  type SearchRequest struct {
      Query      string   `form:"q"`
      CategoryID *int64   `form:"category_id"`
      MinPrice   *float64 `form:"min_price"`
      MaxPrice   *float64 `form:"max_price"`
      Condition  string   `form:"condition"`  // new, used, refurbished
      Latitude   *float64 `form:"lat"`
      Longitude  *float64 `form:"lng"`
      Radius     *int     `form:"radius"`     // km, default 10
      City       string   `form:"city"`
      Country    string   `form:"country"`
      SortBy     string   `form:"sort_by"`    // relevance, price_asc, price_desc, date, distance
      Page       int      `form:"page"     validate:"min=1"`
      PerPage    int      `form:"per_page" validate:"min=1,max=100"`
  }
  ```

  **Additional endpoints:**
  - `GET /api/v1/listings/suggestions?q=iphone` — autocomplete (ILIKE title, limit 10)

  **Response includes facets:**
  ```json
  {
    "results": [...],
    "total": 1234,
    "facets": {
      "categories": [{"id": 1, "name": "Electronics", "count": 450}],
      "price_ranges": [{"min": 0, "max": 100, "count": 320}],
      "conditions": [{"value": "new", "count": 670}]
    }
  }
  ```

  **Redis caching:**
  - Cache popular search results (TTL: 5 minutes)
  - Cache facet counts (TTL: 5 minutes)
  - Store recent searches: `user:{uid}:recent_searches` (TTL: 30 days, max 20 items)

  ---

  ## Phase 5: Notification System

  ### Task 5.1 — Complete Notification System

  **Database schema:**
  ```sql
  CREATE TABLE notifications (
      id         BIGSERIAL PRIMARY KEY,
      user_id    BIGINT NOT NULL REFERENCES users(id),
      type       VARCHAR(50) NOT NULL,
      title      VARCHAR(255),
      body       TEXT,
      data       JSONB,
      read       BOOLEAN DEFAULT FALSE,
      read_at    TIMESTAMP,
      created_at TIMESTAMP DEFAULT NOW()
  );
  CREATE INDEX idx_notifications_user   ON notifications(user_id, created_at DESC);
  CREATE INDEX idx_notifications_unread ON notifications(user_id, read) WHERE read = FALSE;

  CREATE TABLE notification_preferences (
      user_id                BIGINT PRIMARY KEY REFERENCES users(id),
      email_new_bid          BOOLEAN DEFAULT TRUE,
      email_outbid           BOOLEAN DEFAULT TRUE,
      email_message          BOOLEAN DEFAULT TRUE,
      email_listing_approved BOOLEAN DEFAULT TRUE,
      push_new_bid           BOOLEAN DEFAULT TRUE,
      push_outbid            BOOLEAN DEFAULT TRUE,
      push_message           BOOLEAN DEFAULT TRUE,
      in_app_enabled         BOOLEAN DEFAULT TRUE,
      created_at             TIMESTAMP DEFAULT NOW(),
      updated_at             TIMESTAMP DEFAULT NOW()
  );

  CREATE TABLE push_tokens (
      id         BIGSERIAL PRIMARY KEY,
      user_id    BIGINT REFERENCES users(id),
      token      TEXT NOT NULL,
      platform   VARCHAR(20),  -- web, ios, android
      created_at TIMESTAMP DEFAULT NOW()
  );
  ```

  **Notification types:**
  ```go
  const (
      NotifNewBid          = "new_bid"
      NotifOutbid          = "outbid"
      NotifAuctionWon      = "auction_won"
      NotifAuctionEnded    = "auction_ended"
      NotifNewMessage      = "new_message"
      NotifListingApproved = "listing_approved"
      NotifListingRejected = "listing_rejected"
      NotifPaymentSuccess  = "payment_success"
      NotifPaymentFailed   = "payment_failed"
      NotifEscrowReleased  = "escrow_released"
      NotifNewReview       = "new_review"
  )
  ```

  **API Endpoints:**

  | Method | Path | Description |
  |--------|------|-------------|
  | GET | `/notifications` | List (paginated, filter read/unread) |
  | GET | `/notifications/unread-count` | Badge count |
  | PUT | `/notifications/:id/read` | Mark single as read |
  | PUT | `/notifications/mark-all-read` | Mark all as read |
  | DELETE | `/notifications/:id` | Delete |
  | POST | `/notifications/register-push-token` | Register FCM token |
  | DELETE | `/notifications/push-tokens/:id` | Remove token |
  | GET | `/notifications/preferences` | Get preferences |
  | PUT | `/notifications/preferences` | Update preferences |

  **WebSocket:** `WS /ws/notifications` — real-time delivery

  **Delivery pipeline:**
  1. Check user preferences
  2. Create in-app notification (DB)
  3. Queue email (Redis) → async worker
  4. Queue push (Redis) → FCM async worker
  5. Broadcast via WebSocket (immediate)

  **Push:** Firebase Cloud Messaging (FCM) — supports Web, iOS, Android

  ---

  ## Phase 6: Admin Dashboard

  ### Task 6.1 — Admin Panel

  **Backend middleware:**
  ```go
  func AdminOnly() gin.HandlerFunc  // role must be "admin" or "super_admin"
  ```

  **Database additions:**
  ```sql
  ALTER TABLE users ADD COLUMN is_banned  BOOLEAN DEFAULT FALSE;
  ALTER TABLE users ADD COLUMN ban_reason TEXT;

  CREATE TABLE admin_logs (
      id          BIGSERIAL PRIMARY KEY,
      admin_id    BIGINT REFERENCES users(id),
      action      VARCHAR(100),
      target_type VARCHAR(50),
      target_id   BIGINT,
      details     JSONB,
      ip_address  INET,
      created_at  TIMESTAMP DEFAULT NOW()
  );
  ```

  **Admin API endpoints:**

  | Method | Path | Description |
  |--------|------|-------------|
  | GET | `/admin/stats` | Dashboard metrics |
  | GET | `/admin/users` | List users (search + filters) |
  | GET | `/admin/users/:id` | User detail + activity |
  | PUT | `/admin/users/:id` | Update user |
  | DELETE | `/admin/users/:id` | Soft delete |
  | POST | `/admin/users/:id/ban` | Ban + reason |
  | POST | `/admin/users/:id/unban` | Unban |
  | GET | `/admin/listings` | All listings with filters |
  | PUT | `/admin/listings/:id/approve` | Approve listing |
  | PUT | `/admin/listings/:id/reject` | Reject with reason |
  | DELETE | `/admin/listings/:id` | Hard delete |
  | GET | `/admin/reports` | All reports |
  | PUT | `/admin/reports/:id/resolve` | Resolve report |
  | GET | `/admin/revenue` | Revenue (daily/weekly/monthly) |
  | GET | `/admin/transactions` | All payments + CSV export |

  **Stats response:**
  ```json
  {
    "total_users": 15234,
    "active_users_today": 1243,
    "total_listings": 8932,
    "active_listings": 7521,
    "total_auctions": 432,
    "live_auctions": 87,
    "total_revenue": 125430.50,
    "revenue_today": 3421.20,
    "pending_moderation": 23,
    "reports_pending": 12
  }
  ```

  **Frontend (React + Vite — already at `artifacts/admin/`):**
  - Pages: Dashboard, Users, Listings, Reports, Revenue, Settings
  - Components: Stats cards, Recharts charts, DataTable with search/filter/pagination/bulk-actions
  - Libraries: shadcn/ui · Recharts · React Query
  - Features: Approve/reject queue, ban/unban modal, CSV export, revenue charts

  ---

  ## Phase 7: Testing & Optimization

  ### Task 7.1 — Comprehensive Testing Suite

  **Test structure:**
  ```
  backend/
  ├── internal/
  │   ├── auth/
  │   │   ├── auth.go
  │   │   └── auth_test.go
  │   ├── listings/
  │   │   ├── service.go
  │   │   └── service_test.go
  ```

  **Unit test coverage:**
  - Auth: Register, Login, JWT generation/validation, social login, password reset
  - Listings: CRUD, search with all filter combinations
  - Auctions: bid placement, outbid, auto-bid, end-of-auction
  - Payments: intent creation, escrow, refund

  **Integration tests:**
  ```go
  func setupTestDB(t *testing.T) *gorm.DB   // PostgreSQL test DB with migrations
  func teardownTestDB(t *testing.T, db *gorm.DB)

  func TestAuthFlow(t *testing.T)      // register → verify → login → protected endpoint
  func TestListingFlow(t *testing.T)   // create → upload images → search → update → delete
  func TestAuctionFlow(t *testing.T)   // create → bid → outbid → end
  ```

  **E2E tests (Playwright):**
  - `tests/e2e/auth.spec.ts` — register + login flow
  - `tests/e2e/listing.spec.ts` — create listing with image upload
  - `tests/e2e/auction.spec.ts` — place bid flow

  **Load tests (k6):**
  - Target: 100 concurrent users, 95th percentile < 500ms, error rate < 1%
  - File: `tests/load/listings_search.js`

  **Target coverage:** 80%+
  ```bash
  go test ./... -coverprofile=coverage.out
  go tool cover -html=coverage.out
  ```

  **CI/CD (`.github/workflows/test.yml`):**
  - On every push + PR
  - Services: postgres:16, redis:7
  - Steps: unit tests → integration tests → coverage upload (Codecov)

  ---

  ## Phase 8: Production Deployment

  ### Task 8.1 — Docker + Kubernetes + Monitoring

  **Backend Dockerfile** (`backend/Dockerfile.prod`):
  ```dockerfile
  FROM golang:1.23-alpine AS builder
  WORKDIR /app
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

  FROM alpine:latest
  RUN apk --no-cache add ca-certificates
  WORKDIR /root/
  COPY --from=builder /app/main .
  EXPOSE 8080
  CMD ["./main"]
  ```

  **Kubernetes manifests** (`k8s/`):

  | File | Resource |
  |------|----------|
  | `namespace.yaml` | Namespace: `geocore-prod` |
  | `configmap.yaml` | DB_HOST, REDIS_HOST, etc. |
  | `secret.yaml` | DB_PASSWORD, JWT_SECRET, STRIPE_SECRET_KEY |
  | `backend-deployment.yaml` | 3 replicas, resource limits, health checks |
  | `frontend-deployment.yaml` | 2 replicas |
  | `postgres-statefulset.yaml` | 1 replica, 50Gi PVC |
  | `ingress.yaml` | SSL via cert-manager + Let's Encrypt |

  **CI/CD (`.github/workflows/deploy.yml`):**
  - Trigger: push to `main`
  - Steps: build images → push to registry → `kubectl set image` → rollout status

  **Health endpoints:**
  ```go
  GET /health  →  {"status": "healthy"}
  GET /ready   →  checks DB + Redis connections
  ```

  **Monitoring stack:**
  - Prometheus (scrapes `/metrics` from backend pods)
  - Grafana dashboards: API response times, error rates, DB connections, Redis stats
  - ELK Stack / Loki for centralized logs (30-day retention)

  **Backup strategy:**
  ```bash
  #!/bin/bash
  # Runs daily via CronJob
  TIMESTAMP=$(date +%Y%m%d_%H%M%S)
  pg_dump -h $DB_HOST -U $DB_USER geocore > backup_$TIMESTAMP.sql
  aws s3 cp backup_$TIMESTAMP.sql s3://geocore-backups/
  ```

  ---

  ## 🌍 GCC Region Configuration

  | Setting | Value |
  |---------|-------|
  | Default currency | AED (UAE Dirham) |
  | Supported currencies | AED, SAR, KWD, QAR, BHD, OMR |
  | Supported countries | UAE, KSA, Kuwait, Qatar, Bahrain, Oman |
  | Primary cities | Dubai, Abu Dhabi, Riyadh, Jeddah, Kuwait City, Doha |
  | Languages | Arabic (RTL) + English |
  | Phone format | E.164 international (`+971...`, `+966...`, etc.) |

  ---

  ## 🔐 Environment Variables Reference

  ```env
  # Database
  DATABASE_URL=postgresql://user:pass@host:5432/geocore

  # Redis
  REDIS_HOST=localhost
  REDIS_PORT=6379
  REDIS_PASSWORD=

  # Auth
  JWT_SECRET=<64-char-random>
  JWT_EXPIRY=24h

  # Email (SMTP)
  SMTP_HOST=
  SMTP_PORT=587
  SMTP_USER=
  SMTP_PASS=
  SMTP_FROM=noreply@geocore.app
  APP_BASE_URL=https://geocore.app

  # Social Auth
  GOOGLE_CLIENT_ID=
  GOOGLE_CLIENT_SECRET=
  GOOGLE_REDIRECT_URL=

  # Stripe
  STRIPE_SECRET_KEY=sk_live_...
  STRIPE_PUBLISHABLE_KEY=pk_live_...
  STRIPE_WEBHOOK_SECRET=whsec_...

  # Cloudflare R2
  R2_ACCOUNT_ID=
  R2_ACCESS_KEY_ID=
  R2_SECRET_ACCESS_KEY=
  R2_BUCKET_NAME=geocore-images
  R2_PUBLIC_URL=https://images.geocore.com

  # FCM (Push Notifications)
  FCM_SERVER_KEY=

  # App
  APP_ENV=production
  PORT=8080
  FRONTEND_URL=https://geocore.app
  RATE_LIMIT_WHITELIST=127.0.0.1,::1
  ```

  ---

  ## 📁 Backend Directory Structure

  ```
  backend/
  ├── cmd/api/main.go
  ├── internal/
  │   ├── auth/
  │   │   ├── handler.go
  │   │   ├── social_handler.go
  │   │   ├── password_reset_handler.go
  │   │   └── routes.go
  │   ├── users/
  │   │   └── model.go
  │   ├── listings/
  │   ├── auctions/
  │   ├── chat/
  │   ├── payments/
  │   └── notifications/
  ├── pkg/
  │   ├── database/
  │   ├── email/
  │   ├── middleware/
  │   │   ├── auth.go          (JWT + revocation check)
  │   │   └── ratelimit.go     (⬜ Task 1.4)
  │   ├── response/
  │   │   └── response.go
  │   └── validator/
  │       └── validator.go     (⬜ Task 1.5)
  └── k8s/                     (⬜ Task 8.1)
  ```

  ---

  *Last updated: 2026-03-24 — Phase 1 complete | GeoCore Next v1.0*
  