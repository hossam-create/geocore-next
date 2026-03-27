# 🤖 DOCUMENT 2: CLAUDE.md — GeoCore Next Developer Guide

---

## Project Overview

**What this rebuilds:** [geocore-community](https://github.com/hossam-create/geocore-community) (PHP classifieds + auctions) → [geocore-next](https://github.com/hossam-create/geocore-next) (Go + Next.js)

**Stack:**
- **Backend:** Go 1.23 · Gin · GORM · PostgreSQL 16 (PostGIS) · Redis 7 · Gorilla WebSocket · Stripe · Zap logger
- **Frontend:** Next.js 15 · App Router · TypeScript · Tailwind · shadcn/ui · Zustand · React Query · Zod
- **Infra:** Docker Compose · Meilisearch · Adminer · Cloudinary

### Repo Tree (Actual)
```
geocore-next/
├── docker-compose.yml                 # 5 services: api, postgres, redis, meilisearch, adminer
├── backend/
│   ├── go.mod                         # module github.com/geocore-next/backend
│   ├── cmd/api/main.go                # Entry point — Gin setup, WS hubs, routes, graceful shutdown
│   ├── pkg/
│   │   ├── database/database.go       # Connect(), AutoMigrate(), SeedCategories
│   │   ├── middleware/auth.go         # Auth() JWT middleware, AdminOnly()
│   │   ├── response/response.go      # R{} struct, OK/Created/BadRequest/etc helpers
│   │   └── redis/redis.go            # Set/Get/Del/Publish/Subscribe wrappers
│   └── internal/
│       ├── auth/
│       │   ├── handler.go             # Register, Login, Me, generateToken()
│       │   └── routes.go              # /auth/register, /auth/login, /auth/me
│       ├── listings/
│       │   ├── model.go               # Category, Listing, ListingImage, Favorite
│       │   ├── handler.go             # List, Get, Create, Update, Delete, GetCategories, ToggleFavorite, GetMyListings
│       │   ├── routes.go              # /categories, /listings CRUD
│       │   └── seed.go                # SeedCategories — 10 categories EN+AR
│       ├── auctions/
│       │   ├── model.go               # Auction, Bid (with IsAuto + MaxAmount)
│       │   ├── handler.go             # List, Get, Create, PlaceBid, GetBids
│       │   ├── routes.go              # /auctions CRUD + /auctions/:id/bid
│       │   └── websocket.go           # Hub (register/unregister/broadcast) + ServeWS + read/writePump
│       ├── chat/
│       │   ├── model.go               # Conversation, ConversationMember, Message
│       │   ├── handler.go             # GetConversations, CreateOrGetConversation, GetMessages, SendMessage
│       │   ├── routes.go              # /chat/* routes + WS endpoint
│       │   └── websocket.go           # Hub (same pattern as auctions) + ServeWS
│       ├── users/
│       │   ├── model.go               # User, PublicUser, ToPublic()
│       │   ├── handler.go             # GetProfile, UpdateMe, GetMe
│       │   └── routes.go              # /users/:id/profile, /users/me
│       └── payments/
│           ├── handler.go             # CreatePaymentIntent, GetPublishableKey
│           └── routes.go              # /payments/intent, /payments/key
└── frontend/                          # Not yet created
```

### Quick Start
```bash
docker-compose up -d
# API on http://localhost:8080
# Adminer on http://localhost:8081
# Meilisearch on http://localhost:7700
```

---

## Architecture Rules (Non-Negotiable)

### 1. Layer Law — What Goes Where

Every domain module lives in `backend/internal/<domain>/` and has exactly these files:

| File | Purpose | Example |
|---|---|---|
| `model.go` | GORM structs, no business logic | `auctions.Auction`, `auctions.Bid` |
| `handler.go` | HTTP handlers, request parsing, response writing | `Handler.PlaceBid(c *gin.Context)` |
| `routes.go` | Route registration, middleware binding | `RegisterRoutes(r *gin.RouterGroup, db, rdb)` |
| `websocket.go` | WS Hub + Client (only if domain has real-time) | `auctions.Hub`, `chat.Hub` |
| `seed.go` | Initial data seeding (only if needed) | `listings.SeedCategories()` |

**Example from actual code** — every handler follows this constructor pattern:
```go
// From backend/internal/auctions/handler.go
type Handler struct {
    db  *gorm.DB
    rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
    return &Handler{db, rdb}
}
```

**NEVER** put handler logic into `routes.go`. Routes only declares the mapping:
```go
// From backend/internal/auctions/routes.go
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
    h := NewHandler(db, rdb)
    a := r.Group("/auctions")
    {
        a.GET("", h.List)
        a.GET("/:id", h.Get)
        a.GET("/:id/bids", h.GetBids)
        a.Use(middleware.Auth())    // ← Auth applied AFTER public routes
        a.POST("", h.Create)
        a.POST("/:id/bid", h.PlaceBid)
    }
}
```

### 2. Error Handling

Use the standard `response` package. **Never** call `c.JSON()` directly:

```go
// ✅ CORRECT — from backend/internal/auth/handler.go
response.BadRequest(c, err.Error())    // 400
response.Unauthorized(c)               // 401
response.Forbidden(c)                  // 403
response.NotFound(c, "Auction")        // 404 → {"error": "Auction not found"}
response.InternalError(c, err)         // 500 (err is logged, not exposed)
response.Created(c, auction)           // 201
response.OK(c, data)                   // 200
response.OKMeta(c, data, meta)         // 200 with pagination

// ❌ WRONG
c.JSON(200, gin.H{"data": auction})
```

**Response envelope** (from `pkg/response/response.go`):
```go
type R struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    interface{} `json:"meta,omitempty"`
}
```

### 3. Auth Context

JWT middleware (`pkg/middleware/auth.go`) sets `user_id` as a **string** in gin context:

```go
// From backend/pkg/middleware/auth.go — Claims struct
type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

// Middleware sets:
c.Set("user_id", claims.UserID)
c.Set("user_email", claims.Email)
```

**Correct way to read user_id in a handler:**
```go
// From backend/internal/auctions/handler.go — PlaceBid
userID, _ := uuid.Parse(c.MustGet("user_id").(string))  // string → uuid.UUID
```

**⚠ NEVER** assume `user_id` is `uuid.UUID` directly — it's stored as `string` in the gin context and must be parsed.

### 4. GORM Patterns

**UUID Primary Keys** — every model uses `uuid.UUID` with DB-side generation:
```go
// From backend/internal/listings/model.go
ID uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
```

**Soft Deletes** — Listing, Auction, User all have:
```go
DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
```

**Relations with Preload** — from `listings/handler.go`:
```go
h.db.Preload("Images").Preload("Category").First(&listing, "id = ?", id)
```

**Conditional Preload** — from `auctions/handler.go`:
```go
h.db.Preload("Bids", func(db *gorm.DB) *gorm.DB {
    return db.Order("amount DESC").Limit(20)
}).First(&auction, "id = ?", id)
```

**Atomic updates** — from `listings/handler.go`:
```go
h.db.Model(&Listing{}).Where("id = ?", listingID).
    UpdateColumn("favorite_count", gorm.Expr("favorite_count + 1"))
```

**Transaction** — from `auctions/handler.go`:
```go
h.db.Transaction(func(tx *gorm.DB) error {
    tx.Create(&bid)
    tx.Model(&auction).Updates(map[string]interface{}{
        "current_bid": req.Amount,
        "bid_count":   gorm.Expr("bid_count + 1"),
    })
    return nil
})
```

### 5. WebSocket Hub Pattern

Both `auctions/websocket.go` and `chat/websocket.go` use the same Hub architecture:

```
                    ┌──────────┐
    register ──────►│          │
    unregister ────►│   Hub    │◄──── broadcast channel
                    │          │
                    └─────┬────┘
                          │
               ┌──────────┼──────────┐
               ▼          ▼          ▼
           Client₁    Client₂    Client₃
           (room A)   (room A)   (room B)
```

**Key components** (from `auctions/websocket.go`):
```go
type Hub struct {
    clients    map[string]map[*Client]bool  // roomID → set of clients
    broadcast  chan *BroadcastMsg
    register   chan *Client
    unregister chan *Client
    rdb        *redis.Client
    mu         sync.RWMutex                 // protects clients map
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:    // Add client to room
        case client := <-h.unregister:  // Remove + close send channel
        case msg := <-h.broadcast:      // Fan out to room members
        }
    }
}
```

**To add a new WS feature,** copy `auctions/websocket.go`, rename types, and:
1. Create Hub in `routes.go` → `go hub.Run()`
2. Add WS route → `ServeWS(hub, c, db)`
3. Use Redis Pub/Sub for cross-pod broadcasting

---

## Adding a New Feature — Step-by-Step Template

### Example: "Saved Searches with Email Alerts"

#### Step 1: Create model (`backend/internal/searches/model.go`)
```go
package searches

import (
    "time"
    "github.com/google/uuid"
)

type SavedSearch struct {
    ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
    UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
    Query      string     `gorm:"not null" json:"query"`
    CategoryID *uuid.UUID `gorm:"type:uuid" json:"category_id,omitempty"`
    Country    string     `json:"country,omitempty"`
    City       string     `json:"city,omitempty"`
    MinPrice   *float64   `json:"min_price,omitempty"`
    MaxPrice   *float64   `json:"max_price,omitempty"`
    AlertEmail bool       `gorm:"default:true" json:"alert_email"`
    LastAlerted *time.Time `json:"last_alerted,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
}
```

#### Step 2: Register model in `database.go`
```go
// Add to AutoMigrate call in pkg/database/database.go
import "github.com/geocore-next/backend/internal/searches"

// Inside AutoMigrate():
&searches.SavedSearch{},
```

#### Step 3: Create handler (`backend/internal/searches/handler.go`)
```go
package searches

import (
    "github.com/geocore-next/backend/pkg/response"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

type Handler struct {
    db  *gorm.DB
    rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
    return &Handler{db, rdb}
}

func (h *Handler) Create(c *gin.Context) {
    userID, _ := uuid.Parse(c.MustGet("user_id").(string))
    var req struct {
        Query      string   `json:"query" binding:"required,min=2"`
        CategoryID *string  `json:"category_id"`
        Country    string   `json:"country"`
        MinPrice   *float64 `json:"min_price"`
        MaxPrice   *float64 `json:"max_price"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, err.Error())
        return
    }
    search := SavedSearch{
        ID:     uuid.New(),
        UserID: userID,
        Query:  req.Query,
        // ... map remaining fields
    }
    if err := h.db.Create(&search).Error; err != nil {
        response.InternalError(c, err)
        return
    }
    response.Created(c, search)
}

func (h *Handler) List(c *gin.Context) {
    userID := c.MustGet("user_id").(string)
    var searches []SavedSearch
    h.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&searches)
    response.OK(c, searches)
}

func (h *Handler) Delete(c *gin.Context) {
    userID := c.MustGet("user_id").(string)
    id, _ := uuid.Parse(c.Param("id"))
    result := h.db.Where("id = ? AND user_id = ?", id, userID).Delete(&SavedSearch{})
    if result.RowsAffected == 0 {
        response.NotFound(c, "Saved search")
        return
    }
    response.OK(c, gin.H{"message": "Deleted"})
}
```

#### Step 4: Create routes (`backend/internal/searches/routes.go`)
```go
package searches

import (
    "github.com/geocore-next/backend/pkg/middleware"
    "github.com/gin-gonic/gin"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
    h := NewHandler(db, rdb)
    s := r.Group("/searches", middleware.Auth())
    {
        s.POST("", h.Create)
        s.GET("", h.List)
        s.DELETE("/:id", h.Delete)
    }
}
```

#### Step 5: Wire into main.go
```go
// Add import
"github.com/geocore-next/backend/internal/searches"

// Add route registration (after existing routes)
searches.RegisterRoutes(v1, db, rdb)
```

#### Step 6: Add cron job for email alerts (future)
Create a goroutine in `main.go` that runs every hour, queries saved searches with new matching listings since `last_alerted`, and sends email notifications.

---

## Go Conventions (Based on Actual Code)

| Convention | Pattern Used | Example |
|---|---|---|
| Package naming | Lowercase, singular domain | `listings`, `auctions`, `chat` |
| Handler struct | `Handler` with `db` + `rdb` fields | `Handler{db *gorm.DB, rdb *redis.Client}` |
| Constructor | `NewHandler(db, rdb) *Handler` | Every module |
| Route registration | `RegisterRoutes(r *gin.RouterGroup, db, rdb)` | Every module |
| UUID handling | `uuid.Parse()` from gin param/context | `uuid.Parse(c.Param("id"))` |
| Default values | `defaultStr(s, d string) string` helper | `defaultStr(req.Currency, "USD")` |
| Pagination | `page` + `per_page` query params, max 50 | `listings/handler.go` |
| Error from binding | `c.ShouldBindJSON(&req)` → `response.BadRequest` | Every POST handler |
| Timestamps | `time.Time` with `json:"created_at"` | All models |
| Soft delete | `gorm.DeletedAt` field, tagged `json:"-"` | Listing, Auction, User |

---

## Frontend Conventions (Target)

| Convention | Approach |
|---|---|
| RSC vs Client | Server Components by default; `"use client"` only for interactivity (forms, WebSocket, state) |
| State Management | Zustand for auth store, cart; React Query for server data (listings, auctions) |
| Forms | Zod schema → `useForm` from react-hook-form with Zod resolver |
| API Client | `api.ts` — axios instance, `baseURL` from `NEXT_PUBLIC_API_URL`, interceptor for 401 → refresh |
| Styling | Tailwind + shadcn/ui component variants |
| WebSocket | Custom hooks: `useAuctionWebSocket(auctionId)`, `useChatWebSocket(conversationId)` |
| Routing | App Router, `app/(auth)/login`, `app/(main)/listings/[id]`, `app/admin/*` |

---

## Environment Variables

| Variable | Description | Required | Default | Example |
|---|---|---|---|---|
| `APP_ENV` | Runtime environment | yes | `development` | `production` |
| `PORT` | HTTP listen port | no | `8080` | `8080` |
| `DB_HOST` | PostgreSQL host | yes | `localhost` | `postgres` |
| `DB_PORT` | PostgreSQL port | no | `5432` | `5432` |
| `DB_USER` | DB username | yes | — | `geocore` |
| `DB_PASSWORD` | DB password | yes | — | `geocore_secret` |
| `DB_NAME` | Database name | yes | — | `geocore_dev` |
| `DB_SSLMODE` | SSL mode | no | `disable` | `require` |
| `REDIS_HOST` | Redis host | yes | `localhost` | `redis` |
| `REDIS_PORT` | Redis port | no | `6379` | `6379` |
| `REDIS_PASSWORD` | Redis password | no | — | `secret` |
| `JWT_SECRET` | Signing key (min 32 chars) | yes | — | `dev_secret_geocore_next_32_chars_min` |
| `FRONTEND_URL` | CORS origin | yes | `http://localhost:3000` | `https://geocore.com` |
| `STRIPE_SECRET_KEY` | Stripe secret key | yes | — | `sk_live_...` |
| `STRIPE_PUBLISHABLE_KEY` | Stripe publishable key | yes | — | `pk_live_...` |
| `CLOUDINARY_URL` | Cloudinary DSN | yes | — | `cloudinary://k:s@cloud` |
| `MEILI_HOST` | Meilisearch URL | no | `http://localhost:7700` | `http://meilisearch:7700` |
| `MEILI_MASTER_KEY` | Meilisearch admin key | yes | — | `dev_meili_key` |

---

## Common Commands

```bash
# ── Docker ──────────────────────────────────────────
docker-compose up -d                              # Start all 5 services
docker-compose logs -f api                        # Tail API logs
docker-compose exec postgres psql -U geocore geocore_dev  # DB shell
docker-compose down -v                            # Stop + remove volumes

# ── Go Backend ──────────────────────────────────────
cd backend
go run ./cmd/api/main.go                          # Run API locally
go test ./...                                     # Run all tests
go build -o bin/api ./cmd/api/main.go             # Build binary
go vet ./...                                      # Lint

# ── Frontend (once created) ─────────────────────────
cd frontend
npm run dev                                       # Dev server on :3000
npm run build                                     # Production build
npm run lint                                      # ESLint
```

---

## Top 10 Pitfalls (From Actual Code Analysis)

### 1. Route ordering — `/me` vs `/:id` collision
In `listings/routes.go`, `GET /listings/me` is registered AFTER `GET /listings/:id`. Gin will match `/me` as an `:id` parameter. **Fix:** register `/me` before `/:id`, or use a separate route group.

### 2. Chat WebSocket param mismatch
In `chat/websocket.go`, `ServeWS` reads `c.Param("conversationId")` but the route uses `:id`. This means the WS connection gets an empty string for conversationID. **Fix:** change to `c.Param("id")`.

### 3. Duplicate utility functions
`defaultStr()` is defined in both `listings/handler.go` and `chat/handler.go`. `getenv()` is defined in both `main.go` and `database.go`. **Fix:** extract to `pkg/util/util.go`.

### 4. Redis package unused
`pkg/redis/redis.go` defines `Connect()`, `Set()`, `Get()`, `Publish()` etc., but `main.go` creates its own `redis.Client` directly. **Fix:** use `pkg/redis` consistently, or remove the unused package.

### 5. AdminOnly middleware reads wrong context key
`AdminOnly()` reads `user_role` from context, but `Auth()` middleware only sets `user_id` and `user_email`. The role is never in the JWT claims or context. **Fix:** add `Role` to `Claims` struct and set it in auth middleware.

### 6. PlaceBid has race condition
`PlaceBid` reads `auction.CurrentBid`, checks `req.Amount > currentBid`, then updates — but without `SELECT FOR UPDATE` or pessimistic locking. Two concurrent bids can both pass the check. **Fix:** use `tx.Clauses(clause.Locking{Strength: "UPDATE"})` inside the transaction.

### 7. No refresh token
`generateToken()` creates a single 30-day JWT. There's no access/refresh split, no token rotation, and no way to invalidate tokens. **Fix:** implement dual tokens (15min access + 30d refresh) with Redis blacklist for revocation.

### 8. No DB connection retry
`database.Connect()` fails immediately if PostgreSQL is not ready. In Docker, the API may start before Postgres despite `depends_on`. **Fix:** add retry loop (5 attempts, 2s backoff).

### 9. Chat Hub started twice
In `main.go`, `chatHub := chat.NewHub(rdb); go chatHub.Run()` creates one hub, but `chat/routes.go` creates a SECOND hub: `hub := NewHub(rdb); go hub.Run()`. Messages sent via the REST handler won't reach the WS hub from main. **Fix:** pass the hub from main to `RegisterRoutes`, or use only one initialization point.

### 10. Password hash exposed in auth response
`auth/handler.go` returns `{"user": user}` in Register/Login responses, and `User.PasswordHash` is tagged `json:"-"`, so it's correctly hidden. However, `auth/handler.go:Me()` returns the full user — if anyone changes the json tag on PasswordHash, it leaks. **Fix:** always use `ToPublic()` or a dedicated response struct for user data over the wire.
