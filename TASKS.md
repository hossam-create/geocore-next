# TASKS.md — GeoCore Next: Master Development Roadmap

> **Generated from gap analysis vs. Mnbara Platform.**
> Last updated: 2026-03-30 | Stack: Go 1.23 backend · Next.js 15 App Router frontend

---

## How to Use This File

1. Find the first task with **Status: `[ ] Not started`** that has no unmet dependencies.
2. Read its full section carefully — especially Context and Files.
3. Read the referenced source files before writing any code.
4. Implement **ONLY** this task — do not touch other tasks.
5. When done: change `[ ]` to `[x]`, tick each acceptance criterion.
6. Report which files were created/modified, then move to next task.
7. To reference the Mnbara codebase: path is `E:\New computer\Development Coding\Projects\Repos\geo\mnbara-platform`

---

# PHASE 0 — Foundation
**Goal:** App has complete data models and API for all commerce flows. No crashes. All new endpoints return data.

---

## TASK-001: Order Management — Backend Models & API

**Phase:** 0 — Foundation
**Priority:** CRITICAL
**Effort:** L (2-3 days)
**Layer:** Backend (Go)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
When a user wins an auction or buys a fixed-price listing, nothing tracks what was purchased, from whom, at what price, or the delivery state. The checkout page (`frontend/app/checkout/page.tsx`) calls the Stripe payment flow, but after payment succeeds there is no order record. This is the single most important missing backend domain.

### Acceptance Criteria
- [x] `orders` table exists in PostgreSQL with migration file
- [x] Order states: `pending -> confirmed -> processing -> shipped -> delivered -> completed -> cancelled -> disputed`
- [x] `OrderItem` sub-struct tracks listing_id, auction_id, quantity, unit_price, snapshot of title
- [x] `POST /api/v1/orders` — creates order from a completed payment intent (called by Stripe webhook)
- [x] `GET /api/v1/orders` — paginated buyer order list
- [x] `GET /api/v1/orders/selling` — paginated seller order list
- [x] `GET /api/v1/orders/:id` — full order detail (buyer or seller only)
- [x] `PATCH /api/v1/orders/:id/confirm` — seller confirms order
- [x] `PATCH /api/v1/orders/:id/ship` — seller marks shipped (requires tracking_number in body)
- [x] `PATCH /api/v1/orders/:id/deliver` — buyer confirms delivery, triggers escrow release job
- [x] `PATCH /api/v1/orders/:id/cancel` — cancel if still `pending`
- [x] All endpoints require JWT auth
- [x] `go build ./...` still passes

### Verification Evidence (2026-04-02)
- [x] `backend/internal/order/` — model.go, handler.go, repository.go, routes.go created
- [x] `backend/migrations/005_create_orders.up.sql` — orders + order_items tables with JSONB status_history
- [x] `order.RegisterRoutes` called in `backend/cmd/api/main.go`
- [x] `payments/handler.go` → `handlePaymentSuccess` calls `createOrderFromPayment` after escrow creation (idempotent)
- [x] `go build ./...` → exit 0, no errors

### Files to Create / Modify
```
CREATE:
  backend/internal/order/model.go          -- Order, OrderItem, OrderStatus structs + GORM tags
  backend/internal/order/handler.go        -- HTTP handlers for all endpoints
  backend/internal/order/routes.go         -- RegisterRoutes(v1, db, rdb)
  backend/migrations/005_create_orders.up.sql
  backend/migrations/005_create_orders.down.sql

MODIFY:
  backend/cmd/api/main.go                  -- import order pkg, call order.RegisterRoutes(v1, db, rdb)
  backend/internal/payments/webhook.go     -- on payment_intent.succeeded, call order creation logic
  backend/internal/auctions/handler.go     -- on auction end (winner determined), enqueue order creation job
```

### Phase Gate Contribution
Required before Phase 0 gate can pass.

---

## TASK-002: Shopping Cart — Backend Service

**Phase:** 0 — Foundation
**Priority:** CRITICAL
**Effort:** M (1 day)
**Layer:** Backend (Go)
**Status:** [x] Completed
**Depends on:** None

### Context
GeoCore Next has a `CheckoutPage` but no cart. For fixed-price listings, there is no path for a user to collect multiple items before paying. Cart state is stored in Redis (ephemeral, per-session) — not PostgreSQL — because carts are temporary and high-write.

### Acceptance Criteria
- [x] `POST /api/v1/cart/items` — add listing to cart (body: `listing_id`, `quantity`)
- [x] `GET /api/v1/cart` — return current user's cart with line items and total
- [x] `DELETE /api/v1/cart/items/:listing_id` — remove item
- [x] `DELETE /api/v1/cart` — clear entire cart
- [x] Cart stored in Redis under key `cart:{user_id}` as JSON, TTL 7 days
- [x] Adding a listing that is already sold or expired returns 400
- [x] Cart item count exposed in response for header badge
- [x] `go build ./...` still passes

### Files to Create / Modify
```
CREATE:
  backend/internal/cart/model.go           -- CartItem, Cart structs
  backend/internal/cart/handler.go         -- HTTP handlers
  backend/internal/cart/routes.go          -- RegisterRoutes(v1, db, rdb)

MODIFY:
  backend/cmd/api/main.go                  -- import cart pkg, call cart.RegisterRoutes(v1, db, rdb)
```

---

## TASK-003: Watchlist / Favorites — Backend

**Phase:** 0 — Foundation
**Priority:** CRITICAL
**Effort:** S (half day)
**Layer:** Backend (Go)
**Status:** [x] Completed
**Depends on:** None

### Context
No watchlist or favorites feature exists. Buyers have no way to save listings they are interested in. The `WatchlistPage.tsx` stub exists in the frontend but has no backend to call.

### Acceptance Criteria
- [x] `watchlist_items` table: user_id, listing_id, created_at (composite PK)
- [x] `POST /api/v1/watchlist/:listing_id` — add to watchlist (idempotent)
- [x] `DELETE /api/v1/watchlist/:listing_id` — remove from watchlist
- [x] `GET /api/v1/watchlist` — paginated list of watched listings with full listing data joined
- [x] `GET /api/v1/listings/:id` response includes `is_watched: bool` when auth present
- [x] Auth required for all endpoints
- [x] `go build ./...` still passes

### Files to Create / Modify
```
CREATE:
  backend/internal/watchlist/model.go      -- WatchlistItem struct
  backend/internal/watchlist/handler.go    -- HTTP handlers
  backend/internal/watchlist/routes.go     -- RegisterRoutes(v1, db)
  backend/migrations/006_create_watchlist.up.sql
  backend/migrations/006_create_watchlist.down.sql

MODIFY:
  backend/cmd/api/main.go                  -- register watchlist routes
  backend/internal/listings/handler.go     -- add is_watched field to GetListing response
```

---

## TASK-004: Refund & Dispute Resolution — Backend Completion

**Phase:** 0 — Foundation
**Priority:** CRITICAL
**Effort:** M (1 day)
**Layer:** Backend (Go)
**Status:** [x] Completed
**Depends on:** TASK-001

### Context
`backend/internal/disputes/` exists with partial implementation but the escrow release flow and refund webhook handlers are incomplete. Orders can get stuck in `disputed` state forever without resolution.

Reference: `mnbara-platform/services/payment-service/` — refund logic and escrow release patterns.

### Acceptance Criteria
- [x] `POST /api/v1/disputes` — buyer opens dispute (requires order_id, reason, evidence text)
- [x] `GET /api/v1/disputes/:id` — dispute detail (buyer or seller or admin)
- [x] `PATCH /api/v1/disputes/:id/resolve` — admin resolves with outcome: `refund_buyer` or `release_seller`
- [x] When resolved as `refund_buyer`: call Stripe refund API, update order status to `refunded`
- [x] When resolved as `release_seller`: call escrow release job, update order status to `completed`
- [x] Escrow release background job (`HandleEscrowRelease`) moves held funds to seller wallet
- [x] `go build ./...` still passes

### Files to Create / Modify
```
MODIFY:
  backend/internal/disputes/handler.go     -- complete Resolve handler, add Stripe refund call
  backend/internal/disputes/routes.go      -- verify all routes registered
  backend/pkg/jobs/handlers.go             -- implement HandleEscrowRelease stub
  backend/cmd/api/main.go                  -- verify disputes routes registered
```

---

## TASK-005: Seller Analytics — Backend Data Endpoints

**Phase:** 0 — Foundation
**Priority:** CRITICAL
**Effort:** M (1 day)
**Layer:** Backend (Go)
**Status:** [x] Completed
**Depends on:** TASK-001

### Context
`backend/internal/analytics/` exists but exposes only admin-level metrics. Sellers have no API to query their own performance data (revenue, views, conversion rate). The frontend analytics pages (Phase 3) depend on these endpoints.

### Acceptance Criteria
- [x] `GET /api/v1/analytics/seller/summary` — seller's own metrics: total_revenue, total_orders, active_listings, total_views, avg_rating
- [x] `GET /api/v1/analytics/seller/revenue?period=7d|30d|90d|1y` — revenue time-series (date, amount)
- [x] `GET /api/v1/analytics/seller/listings` — per-listing breakdown: title, views, favorites, orders, conversion_rate
- [x] All endpoints auth-required, only return data for the requesting seller
- [x] `go build ./...` still passes

### Files to Create / Modify
```
MODIFY:
  backend/internal/analytics/handler.go    -- add SellerSummary, SellerRevenue, SellerListings handlers
  backend/internal/analytics/routes.go     -- register new routes under /analytics/seller/*
  backend/cmd/api/main.go                  -- verify analytics routes registered
```

### Phase 0 Gate
> `go build ./...` passes with zero errors.
> All 5 new backend services have routes registered in `main.go`.
> `GET /api/v1/orders`, `GET /api/v1/cart`, `GET /api/v1/watchlist`, `POST /api/v1/disputes`, `GET /api/v1/analytics/seller/summary` all return 200 with a valid test JWT.

---

# PHASE 1 — Critical Frontend (Launch Blockers)
**Goal:** A user can complete the full buy/sell flow end-to-end. All legal pages exist.

---

## TASK-006: Order Management Pages

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** L (2 days)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-001

### Context
After a buyer pays, they have nowhere to see their order status. After a seller receives an order, they have no UI to confirm shipment. These are the most critical missing frontend flows.

Reference: `mnbara-platform/apps/web/src/pages/orders/` — OrdersPage, OrderDetailPage, SellerOrdersPage

### Acceptance Criteria
- [x] Route `/orders` -> `OrdersPage` — paginated list of buyer's orders with status badges
- [x] Route `/orders/:id` -> `OrderDetailPage` — full order detail: items, status timeline, shipping info, action buttons
- [x] Route `/selling/orders` -> `SellerOrdersPage` — seller's incoming orders list
- [x] Buyer can confirm delivery on `OrderDetailPage` (calls `PATCH /api/v1/orders/:id/deliver`)
- [x] Seller can confirm and mark shipped from `SellerOrdersPage` detail view
- [x] Order status displayed with color-coded badges
- [x] Auth required; redirect to `/login` if not authenticated

### Files to Create / Modify
```
CREATE:
  frontend/app/orders/OrdersPage.tsx
  frontend/app/orders/OrderDetailPage.tsx
  frontend/app/orders/SellerOrdersPage.tsx
  frontend/components/orders/OrderStatusBadge.tsx
  frontend/components/orders/OrderTimeline.tsx

MODIFY:
  frontend/app/layout.tsx           -- add /orders, /orders/:id, /selling/orders routes
  frontend/app/DashboardPage.tsx  -- add "My Orders" and "My Sales" links
```

---

## TASK-007: Cart Page + Cart Icon Component

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** M (1 day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-002

### Context
No cart UI exists. The cart backend (TASK-002) provides the API. This task adds the cart page and the persistent cart icon in the header with item count badge.

### Acceptance Criteria
- [x] Route `/cart` -> `CartPage` — list of cart items, quantities, subtotal, "Proceed to Checkout" button
- [x] Each cart item shows: listing image, title, price, quantity, remove button
- [x] "Proceed to Checkout" navigates to existing `/checkout` with cart items pre-loaded
- [x] `CartIcon` component in `Header` shows item count badge from `GET /api/v1/cart`
- [x] Cart count updates in real-time when items are added or removed
- [x] Empty cart state with "Browse Listings" link
- [x] Auth required; redirect to `/login`

### Files to Create / Modify
```
CREATE:
  frontend/app/CartPage.tsx
  frontend/components/cart/CartIcon.tsx
  frontend/components/cart/CartItem.tsx

MODIFY:
  frontend/app/layout.tsx           -- add /cart route
  frontend/components/layout/Header.tsx  -- add CartIcon
  frontend/app/CheckoutPage.tsx  -- read cart items from API
```

---

## TASK-008: Connect Listing to Cart to Checkout Flow

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** M (1 day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-007

### Context
The listing detail page and cart page exist but are not connected. A user cannot add a listing to the cart from the listing detail page, and the checkout does not know which items to charge for.

### Acceptance Criteria
- [x] "Add to Cart" button on `ListingDetailPage` calls `POST /api/v1/cart/items`
- [x] Button shows loading state during API call and success/error feedback
- [x] For Buy Now listings, "Buy Now" button adds to cart then redirects to `/cart`
- [x] `CheckoutPage` reads items from cart API and passes to Stripe
- [x] After successful payment, cart is cleared and user is redirected to `/orders/:id`
- [x] Sold-out or expired listings show disabled state

### Files to Create / Modify
```
MODIFY:
  frontend/app/ListingDetailPage.tsx   -- add Add to Cart / Buy Now buttons
  frontend/app/CheckoutPage.tsx        -- load cart items, clear after payment
  frontend/lib/api.ts                    -- add cart API functions
```

---

## TASK-009: Watchlist / Favorites Page

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-003

### Context
`WatchlistPage.tsx` exists as a stub but has no real data. The backend (TASK-003) provides the API. This connects the stub to the real API.

### Acceptance Criteria
- [x] Route `/watchlist` -> `WatchlistPage` — grid of watched listings
- [x] Each listing card has a filled heart icon; clicking removes from watchlist
- [x] Heart icon on all listing cards toggles watchlist membership (calls POST/DELETE watchlist API)
- [x] Listing cards show price-drop indicator if price changed since added
- [x] Empty state with "Start Browsing" link
- [x] Auth required

### Files to Create / Modify
```
MODIFY:
  frontend/app/WatchlistPage.tsx       -- connect to GET /api/v1/watchlist
  frontend/components/listings/ListingCard.tsx  -- add heart toggle button
  frontend/lib/api.ts                    -- add watchlist API functions
```

---

## TASK-010: Legal Pages — Terms, Privacy, Cookie Policy

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No legal pages exist. Publishing a marketplace without Terms of Service and Privacy Policy is a legal liability. These pages must exist before any public launch. Content is placeholder/template — legal review is a separate concern.

### Acceptance Criteria
- [x] Route `/legal/terms` -> `TermsOfServicePage` — Terms of Service content
- [x] Route `/legal/privacy` -> `PrivacyPolicyPage` — Privacy Policy content
- [x] Route `/legal/cookies` -> `CookiePolicyPage` — Cookie Policy content
- [x] All three pages linked from `Footer`
- [x] No auth required
- [x] Last updated date shown on each page

### Files to Create / Modify
```
CREATE:
  frontend/app/legal/TermsOfServicePage.tsx
  frontend/app/legal/PrivacyPolicyPage.tsx
  frontend/app/legal/CookiePolicyPage.tsx

MODIFY:
  frontend/app/layout.tsx           -- add /legal/* routes
  frontend/components/layout/Footer.tsx  -- add legal links
```

---

## TASK-011: Refund Page + Chargeback Page

**Phase:** 1 — Critical Frontend
**Priority:** CRITICAL
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-004

### Context
Buyers have no way to initiate a refund or dispute from the UI. The backend dispute system (TASK-004) is implemented. This task adds the user-facing dispute flow.

Reference: `mnbara-platform/apps/web/src/pages/RefundPolicyPage.tsx`

### Acceptance Criteria
- [x] Route `/refund-policy` -> `RefundPolicyPage` — static refund policy content (no auth)
- [x] Route `/disputes/new` -> `NewDisputePage` — form to open a dispute against an order
  - [x] Requires: order_id (dropdown from user's orders), reason (dropdown), evidence (textarea)
  - [x] Submits to `POST /api/v1/disputes`
  - [x] Success state shows dispute ID and next steps
- [x] Route `/disputes` -> `MyDisputesPage` — list of user's disputes with status
- [x] Auth required for `/disputes/*`

### Files to Create / Modify
```
CREATE:
  frontend/app/RefundPolicyPage.tsx
  frontend/app/disputes/NewDisputePage.tsx
  frontend/app/disputes/MyDisputesPage.tsx

MODIFY:
  frontend/app/layout.tsx           -- add /refund-policy and /disputes/* routes
  frontend/components/layout/Footer.tsx  -- add Refund Policy link
```

### Phase 1 Gate
> A test user can: register, browse listings, add to cart, proceed to checkout, complete Stripe test payment, see order at `/orders`, see order details at `/orders/:id`.
> `/legal/terms`, `/legal/privacy`, `/legal/cookies` all render without errors.
> Watchlist heart icon toggles on listing cards.

---

# PHASE 2 — Trust & Guidance Pages
**Goal:** New users understand the platform and feel safe using it.

---

## TASK-012: Help Center & FAQ Page

**Phase:** 2 — Trust & Guidance
**Priority:** HIGH
**Effort:** M (1 day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No help documentation exists. New users have no guidance on how to use the platform. This is a static content page with search functionality. Content is seeded with 20+ common Q&A items.

Reference: `mnbara-platform/apps/web/src/pages/HelpCenterPage.tsx` (38KB) and `mnbara-platform/apps/web/src/pages/FAQPage.tsx`

### Acceptance Criteria
- [x] Route `/help` -> `HelpCenterPage` — categories grid (Buying, Selling, Payments, Account, Safety)
- [x] Route `/help/faq` -> `FAQPage` — accordion with 20+ Q&A items grouped by category
- [x] Route `/help/buying` -> `HelpBuyingPage` — buyer guide content
- [x] Route `/help/selling` -> `HelpSellingPage` — seller guide content
- [x] Client-side search filters FAQ items by keyword
- [x] No auth required
- [x] Footer links to `/help` and `/help/faq`

### Files to Create / Modify
```
CREATE:
  frontend/app/HelpCenterPage.tsx
  frontend/app/FAQPage.tsx
  frontend/app/HelpBuyingPage.tsx
  frontend/app/HelpSellingPage.tsx
  frontend/data/faq.ts         -- FAQ content array

MODIFY:
  frontend/app/layout.tsx             -- add /help/* routes
  frontend/components/layout/Footer.tsx  -- add Help links
```

---

## TASK-013: How It Works Page

**Phase:** 2 — Trust & Guidance
**Priority:** HIGH
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No onboarding or platform explanation page exists. New visitors have no context for what GeoCore is or how to use it. This page drives conversion from visitor to registered user.

Reference: `mnbara-platform/apps/web/src/pages/HowItWorksPage.tsx` (10KB)

### Acceptance Criteria
- [x] Route `/how-it-works` -> `HowItWorksPage`
- [x] Three sections: For Buyers, For Sellers, For Auction Bidders
- [x] Step-by-step numbered steps with icons per section
- [x] CTA buttons: "Start Buying" -> `/listings`, "Start Selling" -> `/sell`
- [x] No auth required
- [x] Linked from `Header` navigation and `Footer`

### Files to Create / Modify
```
CREATE:
  frontend/app/HowItWorksPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /how-it-works route
  frontend/components/layout/Header.tsx  -- add nav link
  frontend/components/layout/Footer.tsx  -- add footer link
```

---

## TASK-014: Buyer Protection Page

**Phase:** 2 — Trust & Guidance
**Priority:** HIGH
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No buyer trust signals exist. Without a visible "Buyer Protection" guarantee, conversion rates suffer. This page explains the escrow system, dispute process, and money-back guarantees.

Reference: `mnbara-platform/apps/web/src/pages/BuyerProtectionPage.tsx` (8KB)

### Acceptance Criteria
- [x] Route `/buyer-protection` -> `BuyerProtectionPage`
- [x] Sections: Escrow System, Dispute Resolution, Money-Back Guarantee, Verified Sellers
- [x] Trust badges/icons for each protection feature
- [x] CTA: "Shop with Confidence" -> `/listings`
- [x] No auth required
- [x] Footer link

### Files to Create / Modify
```
CREATE:
  frontend/app/BuyerProtectionPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /buyer-protection route
  frontend/components/layout/Footer.tsx
```

---

## TASK-015: Seller Protection Page

**Phase:** 2 — Trust & Guidance
**Priority:** HIGH
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
Sellers need to understand the platform's protections for them — fraud prevention, chargeback protection, secure payouts. Without this, seller acquisition suffers.

Reference: `mnbara-platform/apps/web/src/pages/SellerProtectionPage.tsx` (7KB)

### Acceptance Criteria
- [x] Route `/seller-protection` -> `SellerProtectionPage`
- [x] Sections: Fraud Prevention, Secure Payouts, Chargeback Coverage, Verified Buyers
- [x] CTA: "Start Selling" -> `/sell`
- [x] No auth required
- [x] Footer link

### Files to Create / Modify
```
CREATE:
  frontend/app/SellerProtectionPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /seller-protection route
  frontend/components/layout/Footer.tsx
```

---

## TASK-016: About Us Page

**Phase:** 2 — Trust & Guidance
**Priority:** MEDIUM
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No About page exists. Investors, sellers, and press need to understand the company mission. Content is placeholder — replace with real content before launch.

### Acceptance Criteria
- [x] Route `/about` -> `AboutPage`
- [x] Sections: Mission, Team (placeholder), Values, Contact
- [x] No auth required
- [x] Footer link

### Files to Create / Modify
```
CREATE:
  frontend/app/AboutPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /about route
  frontend/components/layout/Footer.tsx
```

---

## TASK-017: Shipping & Delivery Info Page

**Phase:** 2 — Trust & Guidance
**Priority:** MEDIUM
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
Buyers need to understand shipping timelines, delivery options, and international shipping policies before they purchase. Absence of this info reduces conversion.

Reference: `mnbara-platform/apps/web/src/pages/ShippingInfoPage.tsx`

### Acceptance Criteria
- [x] Route `/shipping` -> `ShippingInfoPage`
- [x] Sections: Domestic Shipping, International Shipping, Delivery Times, Tracking
- [x] Table of typical delivery times by region
- [x] No auth required
- [x] Footer link

### Files to Create / Modify
```
CREATE:
  frontend/app/ShippingInfoPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /shipping route
  frontend/components/layout/Footer.tsx
```

---

## TASK-018: Fee Calculator Page

**Phase:** 2 — Trust & Guidance
**Priority:** MEDIUM
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
Sellers do not know how much they will receive after platform and payment fees. A fee calculator builds trust and reduces seller churn. All calculations are done client-side using constants.

Reference: `mnbara-platform/apps/web/src/pages/FeesPricingPage.tsx` (12KB)

### Acceptance Criteria
- [x] Route `/fees` -> `FeesPricingPage` — fee schedule table (listing fees, success fees by category)
- [x] Route `/fees/calculator` -> `FeeCalculatorPage` — interactive calculator
  - [x] Input: sale price
  - [x] Shows: platform fee, payment processing fee, net payout
  - [x] Real-time calculation (no API call, use frontend constants)
- [x] No auth required

### Files to Create / Modify
```
CREATE:
  frontend/app/FeesPricingPage.tsx
  frontend/app/FeeCalculatorPage.tsx

MODIFY:
  frontend/app/layout.tsx             -- add /fees and /fees/calculator routes
  frontend/components/layout/Footer.tsx
  frontend/app/HelpSellingPage.tsx  -- link to fee calculator
```

### Phase 2 Gate
> All 7 pages (TASK-012 through TASK-018) render at their routes without errors.
> Footer has working links to: /legal/terms, /legal/privacy, /how-it-works, /buyer-protection, /seller-protection, /help, /fees, /about, /shipping.

---

# PHASE 3 — Seller Tools
**Goal:** Sellers can manage their business, see analytics, and engage with buyers.

---

## TASK-019: Seller Analytics Dashboard Page

**Phase:** 3 — Seller Tools
**Priority:** HIGH
**Effort:** M (1 day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** TASK-005

### Context
Sellers have no visibility into their performance. This page connects to the analytics backend from TASK-005.

Reference: `mnbara-platform/apps/web/src/pages/seller/SellerAnalytics.tsx` (8KB)

### Acceptance Criteria
- [x] Route `/seller/analytics` -> `SellerAnalyticsPage`
- [x] Metric cards: Total Revenue, Total Orders, Active Listings, Total Views, Avg Rating
- [x] Revenue chart: 30-day line chart (using data from `GET /api/v1/analytics/seller/revenue?period=30d`)
- [x] Listings breakdown table: title, views, favorites, orders per listing
- [x] Period filter: 7d / 30d / 90d / 1y
- [x] Auth required, only shows own data

### Files to Create / Modify
```
CREATE:
  frontend/app/seller/SellerAnalyticsPage.tsx
  frontend/components/seller/RevenueChart.tsx
  frontend/components/seller/MetricCard.tsx

MODIFY:
  frontend/app/layout.tsx                          -- add /seller/analytics route
  frontend/app/DashboardPage.tsx          -- add "View Analytics" link
```

---

## TASK-020: Seller Storefront Analytics

**Phase:** 3 — Seller Tools
**Priority:** MEDIUM
**Effort:** S (half day)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-019, TASK-005

### Context
`MyStorefrontPage.tsx` (27KB) exists but has no analytics tab. Sellers need to know how much traffic their storefront gets.

### Acceptance Criteria
- [x] `GET /api/v1/analytics/storefront` — returns storefront-specific metrics (page views, follower count, conversion rate)
- [x] `MyStorefrontPage` gains an "Analytics" tab alongside the existing listings tab
- [x] Tab shows: storefront views (7d/30d), top performing listings, conversion rate

### Files to Create / Modify
```
MODIFY:
  backend/internal/analytics/handler.go                       -- add StorefrontAnalytics handler
  backend/internal/analytics/routes.go                        -- add GET /analytics/storefront
  frontend/app/MyStorefrontPage.tsx       -- add Analytics tab
```

---

## TASK-021: Loyalty Program Frontend

**Phase:** 3 — Seller Tools
**Priority:** MEDIUM
**Effort:** M (1 day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
`backend/internal/loyalty/` is fully implemented (handler, model, routes -- 14KB handler). There is no frontend for it. Users and sellers cannot see their points, tier, or redeem rewards.

### Acceptance Criteria
- [x] Route `/loyalty` -> `LoyaltyPage` — user's loyalty dashboard
  - Current tier badge (Bronze/Silver/Gold/Platinum)
  - Points balance with progress bar to next tier
  - Points history list (earn/spend events)
  - Available rewards to redeem
- [x] `PointsDisplay` widget added to `DashboardPage` and `ProfilePage`
- [x] `TierBadge` component shown on user profile cards
- [x] Connects to existing `GET /api/v1/loyalty/...` endpoints

### Files to Create / Modify
```
CREATE:
  frontend/app/LoyaltyPage.tsx
  frontend/components/loyalty/PointsDisplay.tsx
  frontend/components/loyalty/TierBadge.tsx
  frontend/components/loyalty/RewardCard.tsx

MODIFY:
  frontend/app/layout.tsx                          -- add /loyalty route
  frontend/app/DashboardPage.tsx          -- add PointsDisplay widget
  frontend/app/ProfilePage.tsx            -- add TierBadge
```

---

## TASK-022: Notification Settings Page

**Phase:** 3 — Seller Tools
**Priority:** MEDIUM
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
Users have no way to control what notifications they receive. The notifications backend (`internal/notifications/`) handles delivery but preferences are not exposed to users.

Reference: `mnbara-platform/apps/web/src/pages/features/NotificationSettingsPage.tsx` (9KB)

### Acceptance Criteria
- [x] Route `/settings/notifications` -> `NotificationSettingsPage`
- [x] Toggle groups: Email Notifications, Push Notifications, SMS Notifications
- [x] Per-event toggles: new message, auction outbid, order update, price drop on watchlist, promo offers
- [x] Save button calls `PATCH /api/v1/users/me/notification-preferences`
- [x] Backend endpoint added to `users/handler.go`

### Files to Create / Modify
```
CREATE:
  frontend/app/settings/NotificationSettingsPage.tsx

MODIFY:
  backend/internal/users/handler.go                           -- add PATCH notification-preferences endpoint
  backend/internal/users/routes.go                            -- register route
  frontend/app/layout.tsx                          -- add /settings/notifications route
  frontend/app/ProfilePage.tsx            -- link to notification settings
```

---

## TASK-023: Deals & Promotions — Backend + Frontend

**Phase:** 3 — Seller Tools
**Priority:** MEDIUM
**Effort:** L (2 days)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-001

### Context
No deals or promotional listings exist. Sellers cannot run sales. The platform cannot feature time-limited deals to drive traffic.

Reference: `mnbara-platform/apps/web/src/pages/DealsPage.tsx` (17KB)

### Acceptance Criteria
- [x] `deals` table: deal_id, listing_id, seller_id, original_price, deal_price, discount_pct, start_at, end_at, status
- [x] `POST /api/v1/deals` — seller creates a deal for their listing
- [x] `GET /api/v1/deals` — public list of active deals, sorted by discount_pct desc
- [x] `GET /api/v1/deals/:id` — single deal detail
- [x] Cron job marks expired deals as `expired` (similar to listing expiry cron)
- [x] Route `/deals` -> `DealsPage` — grid of active deals with countdown timer and discount badge
- [x] `DealBadge` component shown on listing cards when listing has active deal
- [x] Auth required to create; no auth to view

### Files to Create / Modify
```
CREATE:
  backend/internal/deals/model.go
  backend/internal/deals/handler.go
  backend/internal/deals/routes.go
  backend/migrations/007_create_deals.up.sql
  backend/migrations/007_create_deals.down.sql
  frontend/app/DealsPage.tsx
  frontend/components/listings/DealBadge.tsx

MODIFY:
  backend/cmd/api/main.go                                      -- register deals routes
  frontend/app/layout.tsx                           -- add /deals route
  frontend/components/layout/Header.tsx      -- add Deals nav link
```

### Phase 3 Gate
> All Phase 3 deliverables complete. Seller analytics, loyalty frontend, notification settings, and deals pages render without errors.
> All new backend routes return 200 with valid test JWT.

---

# PHASE 4 — Support & Communication
**Goal:** Users can get help. Sellers can be reached. Admins see business metrics.

---

## TASK-024: Contact & Support Page

**Phase:** 4 — Support & Communication
**Priority:** ?? HIGH
**Effort:** S (half day)
**Layer:** Frontend (React/Vite)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No contact mechanism exists. Users encountering problems have no way to reach support.

Reference: `mnbara-platform/apps/web/src/pages/ContactSupportPage.tsx` (27KB)

### Acceptance Criteria
- [x] Route `/contact` -> `ContactSupportPage`
- [x] Form fields: name, email, subject (dropdown), message (min 20 chars)
- [x] Submit calls `POST /api/v1/support/contact` (new backend endpoint)
- [x] Backend stores message and sends email to admin via existing SMTP job
- [x] Success state shown after submission
- [x] No auth required, but pre-fill name/email if logged in
- [x] Linked from `Footer` and from Help pages

### Files to Create / Modify
```
CREATE:
  frontend/app/ContactSupportPage.tsx
  backend/internal/support/handler.go         — ContactForm handler
  backend/internal/support/routes.go          — RegisterRoutes(v1, db)

MODIFY:
  backend/cmd/api/main.go                      — register support routes
  frontend/app/layout.tsx           — add /contact route
  frontend/components/layout/Footer.tsx
```

---

## TASK-025: Support Ticket System

**Phase:** 4 — Support & Communication
**Priority:** ?? MEDIUM
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-024

### Context
One-off contact forms don't scale. Users need to track the status of their support requests. This adds a ticketing system on top of the contact backend.

Reference: `mnbara-platform/apps/web/src/pages/features/SupportTicketsPage.tsx` (10KB)

### Acceptance Criteria
- [x] `support_tickets` table: ticket_id, user_id, subject, status (open/in_progress/resolved/closed), priority, messages[]
- [x] `GET /api/v1/support/tickets` — user's ticket list
- [x] `GET /api/v1/support/tickets/:id` — ticket detail with message thread
- [x] `POST /api/v1/support/tickets/:id/messages` — add reply to ticket
- [x] Admin can change ticket status via existing admin panel
- [x] Route `/support/tickets` -> `SupportTicketsPage` — list of user's tickets
- [x] Route `/support/tickets/:id` -> `SupportTicketDetailPage` — message thread view
- [x] Auth required

### Files to Create / Modify
```
CREATE:
  backend/internal/support/model.go            — Ticket, TicketMessage structs
  backend/migrations/008_create_support_tickets.up.sql
  backend/migrations/008_create_support_tickets.down.sql
  frontend/app/support/SupportTicketsPage.tsx
  frontend/app/support/SupportTicketDetailPage.tsx

MODIFY:
  backend/internal/support/handler.go          — add ticket CRUD handlers
  backend/internal/support/routes.go           — add ticket routes
  frontend/app/layout.tsx           — add /support/tickets routes
  frontend/app/DashboardPage.tsx  — add "My Tickets" link
```

---

## TASK-026: Founder / Owner Dashboard

**Phase:** 4 — Support & Communication
**Priority:** ?? MEDIUM
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-005

### Context
The existing admin panel (`/admin`) is operational. However, there is no high-level business overview — P&L, user growth, GMV. The Founder Dashboard is a read-only view for business owners.

Reference: `mnbara-platform/apps/web/src/pages/founder/FounderDashboard.tsx` (8KB)

### Acceptance Criteria
- [x] `GET /api/v1/analytics/platform` — platform-wide metrics (admin-only):
  - `total_users`, `new_users_7d`, `new_users_30d`
  - `total_listings`, `active_listings`
  - `total_orders`, `gmv_30d` (gross merchandise value)
  - `total_revenue` (platform fee collected)
  - `open_disputes`, `resolved_disputes`
- [x] Route `/founder` -> `FounderDashboard` — role-gated to `admin` or `super_admin`
- [x] Metric cards with sparklines (7-day mini-chart)
- [x] Top categories by revenue table
- [x] Redirect non-admin users to `/` with 403 message

### Files to Create / Modify
```
CREATE:
  frontend/app/founder/FounderDashboard.tsx
  frontend/components/founder/PlatformMetricCard.tsx

MODIFY:
  backend/internal/analytics/handler.go        — add PlatformMetrics handler (admin-only)
  backend/internal/analytics/routes.go         — add GET /analytics/platform
  frontend/app/layout.tsx           — add /founder route with admin guard
```

### Phase 4 Gate
> Admin sees platform metrics at `/founder`.
> User can submit contact form and receive confirmation.
> User can view their support tickets at `/support/tickets`.

---

# PHASE 5 — Infrastructure & Observability
**Goal:** Production is monitored. Errors are visible. All job stubs are implemented. PayPal works.

---

## TASK-027: Sentry Error Tracking — Frontend + Backend

**Phase:** 5 — Infrastructure & Observability
**Priority:** ?? HIGH
**Effort:** S (half day)
**Layer:** Both
**Status:** [x] Completed
**Depends on:** None

### Context
GeoCore Next has no error tracking. Production bugs are invisible. Sentry catches and groups unhandled errors with full stack traces.

### Acceptance Criteria
- [x] Frontend: `@sentry/react` installed, initialized in `main.tsx` with `VITE_SENTRY_DSN` env var
- [x] Frontend: `ErrorBoundary` wraps `<App />` to catch React render errors
- [x] Frontend: Release version injected at build time via `VITE_APP_VERSION`
- [x] Backend: Sentry Go SDK (`github.com/getsentry/sentry-go`) installed
- [x] Backend: `SentryMiddleware` added to Gin router (captures panics + 5xx)
- [x] Backend: `SENTRY_DSN` read from env, Sentry no-op if DSN is empty (safe for dev)
- [ ] Test: trigger a deliberate error, verify it appears in Sentry dashboard

### Files to Create / Modify
```
CREATE:
  backend/pkg/middleware/sentry.go             — Sentry Gin middleware

MODIFY:
  frontend/artifacts/web/src/main.tsx          — initialize Sentry
  frontend/artifacts/web/.env.local.example   — add VITE_SENTRY_DSN
  backend/cmd/api/main.go                      — init Sentry, add SentryMiddleware
  backend/.env.example                         — add SENTRY_DSN
  go.mod                                       — add sentry-go dependency
```

---

## TASK-028: Prometheus Metrics Endpoint — Backend

**Phase:** 5 — Infrastructure & Observability
**Priority:** ?? MEDIUM
**Effort:** S (half day)
**Layer:** Backend (Go)
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
No metrics endpoint exists. Prometheus cannot scrape the API. Without metrics, there is no alerting on latency spikes, error rate, or DB pool exhaustion.

### Acceptance Criteria
- [x] `GET /metrics` endpoint exposes Prometheus text format
- [x] Default Go runtime metrics exposed (goroutines, GC, memory)
- [x] Custom counters/histograms:
  - `http_requests_total` labeled by method, route, status_code
  - `http_request_duration_seconds` histogram per route
  - `db_connections_open` gauge
- [x] `/metrics` endpoint protected by `METRICS_TOKEN` env var (Bearer token check)
- [x] `go build ./...` still passes

### Files to Create / Modify
```
CREATE:
  backend/pkg/metrics/metrics.go               — register custom Prometheus metrics
  backend/pkg/middleware/prometheus.go         — Gin middleware for request metrics

MODIFY:
  backend/cmd/api/main.go                      — add GET /metrics, apply prometheus middleware
  backend/.env.example                         — add METRICS_TOKEN
  go.mod                                       — add prometheus/client_golang dependency
```

---

## TASK-029: Grafana + Prometheus — Docker Compose Setup

**Phase:** 5 — Infrastructure & Observability
**Priority:** ?? MEDIUM
**Effort:** S (half day)
**Layer:** Infra
**Status:** [x] Completed
**Depends on:** TASK-028

### Context
`monitoring/` directory exists but `grafana/` and `prometheus/` subdirectories are empty. No monitoring stack is configured.

### Acceptance Criteria
- [x] `monitoring/prometheus/prometheus.yml` — scrapes `api:8080/metrics` every 15s
- [x] `monitoring/grafana/provisioning/datasources/prometheus.yml` — auto-provisions Prometheus datasource
- [x] `monitoring/grafana/provisioning/dashboards/geocore.json` — pre-built dashboard with:
  - HTTP request rate per route
  - P99 response time per route
  - Error rate (5xx/total)
  - Active DB connections
- [x] `docker-compose.monitoring.yml` — separate compose file adding Grafana + Prometheus services
- [x] `docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up` starts cleanly
- [x] Grafana accessible at `http://localhost:3001`, default login `admin/admin`

### Files to Create / Modify
```
CREATE:
  monitoring/prometheus/prometheus.yml
  monitoring/grafana/provisioning/datasources/prometheus.yml
  monitoring/grafana/provisioning/dashboards/geocore.json
  docker-compose.monitoring.yml
```

---

## TASK-030: Complete All Job Handler Stubs

**Phase:** 5 — Infrastructure & Observability
**Priority:** ?? CRITICAL
**Effort:** M (1 day)
**Layer:** Backend (Go)
**Status:** [x] Completed
**Depends on:** TASK-004

### Context
`backend/pkg/jobs/handlers.go` has multiple TODO stubs. In production, failed background jobs silently do nothing. Each stub must be fully implemented.

Stubs to implement: `HandleEmail`, `HandleSMS`, `HandleAuctionEnd`, `HandleEscrowRelease`, `HandlePushNotification`, `HandleAnalyticsEvent`

### Acceptance Criteria
- [x] `HandleEmail` sends email using `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` env vars
- [x] `HandleSMS` calls Twilio API using `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM`
- [x] `HandleAuctionEnd` queries auction winner, creates order via order service, notifies both parties
- [x] `HandleEscrowRelease` fully implemented (from TASK-004)
- [x] `HandlePushNotification` calls FCM via existing `notifications/fcm.go`
- [x] `HandleAnalyticsEvent` calls PostHog using existing `pkg/analytics` client
- [x] All handlers log errors with zap and re-queue on transient failures
- [x] `go build ./...` still passes

### Runtime Smoke Evidence (2026-04-01)
- [x] Enqueued representative jobs for `email`, `sms`, `push_notification`, `analytics`, `escrow_release`, `auction_end` via Redis queue
- [x] Verified escrow release side effect in DB (`escrow_accounts.status = released`)
- [x] Verified `auction_end` end-to-end after adding missing `orders` / `order_items` tables in environment:
  - `auctions.status = sold`, winner assigned
  - `orders` + `order_items` records created
  - seller/winner notifications inserted
- [x] Added defensive fail-fast guard in `HandleAuctionEnd` for missing order tables

### Files to Create / Modify
```
MODIFY:
  backend/pkg/jobs/handlers.go                 — implement all stubs
  backend/pkg/jobs/dependencies.go             — add all required service deps to HandlerDependencies struct
  backend/cmd/api/main.go                      — pass real dependencies to RegisterDefaultHandlers
```

---

## TASK-031: PayPal Payment Integration

**Phase:** 5 — Infrastructure & Observability
**Priority:** ?? MEDIUM
**Effort:** L (2 days)
**Layer:** Both
**Status:** [~] In progress
**Depends on:** TASK-001

### Context
GeoCore Next only supports Stripe and PayMob. PayPal is dominant in international markets and required for GCC buyers who prefer it.

### Acceptance Criteria
- [x] `POST /api/v1/payments/paypal/create` — creates PayPal order, returns approval URL
- [x] `POST /api/v1/payments/paypal/capture` — captures approved PayPal order and marks payment succeeded with escrow hold
- [x] `POST /api/v1/payments/paypal/webhook` — PayPal webhook endpoint uses signature verification API and event handling
- [x] Frontend checkout shows "Pay with PayPal" button alongside Stripe
- [x] PayPal button redirects to PayPal approval URL, then capture flow redirects to `/orders/:id/success` when order is available
- [x] `PAYPAL_CLIENT_ID` and `PAYPAL_CLIENT_SECRET` read from env vars
- [x] `PAYPAL_WEBHOOK_ID` read from env vars for webhook signature verification
- [x] Supports sandbox mode when `APP_ENV != production`

### Verification Evidence (2026-04-01)
- [x] Backend compile passed: `go build ./...` (after PayPal webhook verification + handlers refactor)
- [x] Frontend compile passed: `npm run build` (Next.js build + type check)
- [x] Webhook verification path exercised: unsigned POST to `/api/v1/payments/paypal/webhook` returned `400` (expected rejection)
- [ ] Full PayPal sandbox E2E charge/capture in this environment
  - Blocker: runtime container has no `PAYPAL_CLIENT_ID`, `PAYPAL_CLIENT_SECRET`, `PAYPAL_WEBHOOK_ID` configured yet
  - Next action after env setup: run buyer checkout via `/checkout` → PayPal approve → return capture, then verify `payments.status=succeeded` and `escrow_accounts.status=held`

### Files to Create / Modify
```
CREATE:
  backend/internal/payments/paypal_client.go   — PayPal REST API wrapper
  backend/internal/payments/paypal_handler.go  — create/capture/webhook handlers

MODIFY:
  backend/internal/payments/routes.go          — add PayPal routes
  backend/.env.example                         — add PAYPAL_CLIENT_ID, PAYPAL_CLIENT_SECRET
  frontend/app/CheckoutPage.tsx  — add PayPal button
  frontend/artifacts/web/.env.local.example   — add VITE_PAYPAL_CLIENT_ID
  go.mod                                       — add paypal SDK or use raw HTTP
```

### Phase 5 Gate
> `GET /metrics` returns Prometheus text format.
> A deliberate frontend error appears in Sentry.
> All job handlers in `handlers.go` have real implementations (no TODO stubs).
> PayPal sandbox checkout completes end-to-end.

---

## TASK-032: XyOps Control Center Integration

**Phase:** 5 — Infrastructure & Observability
**Priority:** HIGH
**Effort:** L (2 days)
**Layer:** Backend
**Status:** [x] Complete
**Depends on:** TASK-030, TASK-031

### Context
Integrate a selective subset of the xyOps workflow automation platform as an operational Control Center for GeoCore Next. Replaces fragmented `time.Sleep` background goroutines with a proper cron scheduler, adds a threshold-based alerting engine, and introduces a runtime config store so payment keys (PayPal, Stripe) are manageable via API instead of requiring container restarts.

### What's included (selective integration)
- **Cron Scheduler** — DB-driven 5-field cron expressions, minute-tick loop, dispatches to existing Redis job queue or internal builtin actions
- **Alert Engine** — threshold rules on job failures, queue depth, payment failures, new users, active auctions; throttled firing + history log
- **Runtime Config Store** — `ops_configs` DB table, Redis cache (5 min TTL), env var fallback; used by PayPal and Stripe init

### What's excluded
- Full xyOps platform / workflow editor GUI
- Server fleet management
- Agent-based monitoring

### Acceptance Criteria
- [x] `GET  /api/v1/ops/status`            — system health: DB, Redis, job queue depth, alerts last 24h
- [x] `GET/POST/PUT/DELETE /api/v1/ops/cron`   — manage cron schedules
- [x] `GET/POST/PUT/DELETE /api/v1/ops/alerts` — manage alert rules
- [x] `GET /api/v1/ops/alerts/history`     — last 100 alert firings
- [x] `GET/POST /api/v1/ops/config`        — read/write runtime config (keys masked for secrets)
- [x] `POST /api/v1/ops/config/bulk`       — set multiple keys at once
- [x] `GET/POST /api/v1/ops/jobs/stats`    — job queue stats + retry failed jobs
- [x] All ops routes protected by `middleware.Auth() + middleware.AdminWithDB(db)`
- [x] `PAYPAL_CLIENT_ID`, `PAYPAL_CLIENT_SECRET`, `PAYPAL_WEBHOOK_ID`, `PAYPAL_BASE_URL` read via `ops.ConfigGet()` (DB → env fallback)
- [x] `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` read via `ops.ConfigGet()` (DB → env fallback)
- [x] Built-in cron schedules seeded: `expire-deals` (*/5 * * * *), `activate-deals` (*/5 * * * *), `cleanup-sessions` (0 3 * * *)
- [x] `go build ./...` passes

### Verification Evidence (2026-04-02)
- [x] `go build ./...` → exit code 0

### Files Created / Modified
```
CREATE:
  backend/internal/ops/models.go        — CronSchedule, AlertRule, OpsConfig, AlertHistory
  backend/internal/ops/config_store.go  — ConfigGet / ConfigSet with Redis cache + env fallback
  backend/internal/ops/cron.go          — CronScheduler, 5-field cron parser, builtin actions
  backend/internal/ops/alerting.go      — AlertEngine, metric collectors, throttled firing
  backend/internal/ops/handler.go       — REST handlers for cron/alerts/config/jobs
  backend/internal/ops/routes.go        — RegisterRoutes, seeds default schedules

MODIFY:
  backend/internal/payments/paypal_client.go   — os.Getenv → ops.ConfigGet
  backend/internal/payments/stripe_client.go   — os.Getenv → ops.ConfigGet
  backend/pkg/database/database.go             — AutoMigrateOps added
  backend/cmd/api/main.go                      — InitConfigStore, RegisterRoutes, Start/Stop scheduler+alerter
```

---

# PHASE 6 — Growth & Acquisition
**Goal:** Platform grows itself through referrals and subscription upsells.

---

## TASK-032: Referral / Affiliate Program

**Phase:** 6 — Growth & Acquisition
**Priority:** ?? MEDIUM
**Effort:** L (2 days)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-001

### Context
No user acquisition mechanism exists beyond organic discovery. A referral program lets existing users invite friends in exchange for loyalty points or wallet credit.

Reference: `mnbara-platform/apps/web/src/pages/affiliate/ProgramPage.tsx` and `ReferralProgramPage.tsx`

### Acceptance Criteria
- [x] `referrals` table: referral_id, referrer_id, referee_id, code, status (pending/completed), reward_amount
- [x] Each user gets a unique referral code on registration (UUID-based slug)
- [x] `GET /api/v1/referral/code` — returns current user's referral code + share URL
- [x] `GET /api/v1/referral/stats` — referral count, pending, completed, total earned
- [x] Registration flow accepts `?ref=CODE` query param and links referral
- [x] On referee's first completed order → referrer receives loyalty points (100 pts, configurable)
- [x] Route `/referral` → `ReferralPage` — show user's code, share buttons, stats
- [x] Auth required

### Verification Evidence (2026-04-02)
- [x] `backend/migrations/007_create_referrals.up.sql` — referrals table + users.referral_code column
- [x] `backend/internal/referral/` — model.go, handler.go, routes.go created
- [x] `users.ReferralCode` field added; generated via `referral.GenerateCode` on Register
- [x] `auth/handler.go` — `?ref=CODE` param processed; `referral.LinkReferral` called async
- [x] `order/handler.go` — `referral.CompleteReferral` called on buyer delivery confirmation
- [x] `referral.RegisterRoutes` registered in `main.go`
- [x] `frontend/app/referral/page.tsx` — ReferralPage with code display, copy, share, stats
- [x] `go build ./...` → exit 0, no errors

### Files to Create / Modify
```
CREATE:
  backend/internal/referral/model.go
  backend/internal/referral/handler.go
  backend/internal/referral/routes.go
  backend/migrations/009_create_referrals.up.sql
  backend/migrations/009_create_referrals.down.sql
  frontend/app/ReferralPage.tsx

MODIFY:
  backend/cmd/api/main.go                      — register referral routes
  backend/internal/auth/handler.go             — process ?ref param on register
  backend/internal/order/handler.go            — trigger referral completion on first order
  frontend/app/layout.tsx           — add /referral route
  frontend/app/DashboardPage.tsx  — add referral widget
```

---

## TASK-033: Subscription / Plans System

**Phase:** 6 — Growth & Acquisition
**Priority:** ?? MEDIUM
**Effort:** XL (3–4 days)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** TASK-001
**? FREELANCER RECOMMENDED**

### Context
No subscription or premium tier system exists. Sellers cannot unlock advanced features (more listings, analytics, promoted listings). This adds a SaaS subscription layer.

Reference: `mnbara-platform/apps/web/src/pages/subscription/` and `mnbara-platform/services/subscription-service/`

### Acceptance Criteria
- [x] `plans` table: plan_id, name (Free/Basic/Pro/Enterprise), price_monthly, features[] JSON
- [x] `subscriptions` table: sub_id, user_id, plan_id, status, current_period_end, stripe_subscription_id
- [x] `GET /api/v1/plans` — public list of available plans
- [x] `POST /api/v1/subscriptions` — create Stripe subscription, returns checkout URL
- [x] `GET /api/v1/subscriptions/me` — current user's plan and status
- [x] `DELETE /api/v1/subscriptions/me` — cancel subscription (end of period)
- [x] Stripe webhook handles `customer.subscription.updated` and `customer.subscription.deleted`
- [x] Route `/plans` → `PlansPage` — pricing table with plan comparison
- [x] Route `/settings/subscription` → `SubscriptionSettingsPage` — current plan + cancel
- [x] Free plan enforces listing limit (5 active listings default)

### Verification Evidence (2026-04-02)
- [x] `backend/migrations/009_create_subscriptions.up.sql` — plans (seeded) + subscriptions tables
- [x] `backend/internal/subscriptions/` — model.go, handler.go, routes.go created
- [x] `subscriptions.RegisterRoutes` registered in `main.go`
- [x] `listings/handler.go` — plan limit check via `subscriptions.GetUserPlanLimits` at Create
- [x] `payments/webhook.go` — `customer.subscription.updated/deleted` events handled
- [x] `frontend/app/plans/page.tsx` — pricing table with all 4 plans
- [x] `frontend/app/settings/subscription/page.tsx` — current plan, cancel, renewal date
- [x] `go build ./...` → exit 0, no errors

### Files to Create / Modify
```
CREATE:
  backend/internal/subscriptions/model.go
  backend/internal/subscriptions/handler.go
  backend/internal/subscriptions/routes.go
  backend/migrations/010_create_subscriptions.up.sql
  backend/migrations/010_create_subscriptions.down.sql
  frontend/app/PlansPage.tsx
  frontend/app/settings/SubscriptionSettingsPage.tsx

MODIFY:
  backend/cmd/api/main.go                      — register subscription routes
  backend/internal/payments/webhook.go         — handle subscription Stripe events
  backend/internal/listings/handler.go         — check plan limits on Create
  frontend/app/layout.tsx           — add /plans and /settings/subscription routes
```

---

## TASK-034: Product Request System

**Phase:** 6 — Growth & Acquisition
**Priority:** ?? LOW
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Completed (verified)
**Depends on:** None

### Context
Buyers sometimes can't find what they're looking for. A "Request a Product" feature lets buyers signal demand, which sellers can then fulfill.

Reference: `mnbara-platform/apps/web/src/components/features/ProductRequest/`

### Acceptance Criteria
- [x] `product_requests` table: request_id, user_id, title, description, category_id, budget, status (open/fulfilled)
- [x] `POST /api/v1/requests` — create product request
- [x] `GET /api/v1/requests` — public list of open requests, filterable by category
- [x] `POST /api/v1/requests/:id/respond` — seller responds with a matching listing
- [x] Route `/requests` → `ProductRequestsPage` — list of open requests
- [x] Route `/requests/new` → `NewProductRequestPage` — form to submit request
- [x] Auth required to create; no auth to view

### Verification Evidence (2026-04-02)
- [x] `backend/migrations/008_create_product_requests.up.sql` — product_requests + product_request_responses tables
- [x] `backend/internal/requests/` — model.go, handler.go, routes.go created
- [x] `requests.RegisterRoutes` registered in `main.go`
- [x] `frontend/app/requests/page.tsx` — public list with search + pagination
- [x] `frontend/app/requests/new/page.tsx` — create form with category, budget, currency
- [x] `go build ./...` → exit 0, no errors

### Files to Create / Modify
```
CREATE:
  backend/internal/requests/model.go
  backend/internal/requests/handler.go
  backend/internal/requests/routes.go
  backend/migrations/011_create_product_requests.up.sql
  backend/migrations/011_create_product_requests.down.sql
  frontend/app/ProductRequestsPage.tsx
  frontend/app/NewProductRequestPage.tsx

MODIFY:
  backend/cmd/api/main.go                      — register request routes
  frontend/app/layout.tsx           — add /requests routes
```

### Phase 6 Gate
> User can generate referral link and see stats at `/referral`.
> Seller can view and upgrade subscription plan at `/plans`.
> Buyer can submit a product request at `/requests/new`.

---

# PHASE 7+ — Future Roadmap
**These tasks are defined but NOT to be built until Phases 0–6 are complete.**
No acceptance criteria — define them when the phase begins.

---

## TASK-F01: Live Streaming Auctions ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Both
**Status:** [x] Completed
**Depends on:** TASK-001

### Context
Sellers host live video streams where viewers bid in real-time. Requires WebRTC/media server (e.g. LiveKit, Agora, or Cloudflare Stream).

Reference: `mnbara-platform/apps/web/src/components/live-stream/` — 8 components including `LiveStreamAuction.tsx`, `LiveStreamChat.tsx`, `LiveStreamCreator.tsx`, `LiveStreamPlayer.tsx`, `StreamModeration.tsx`

---

## TASK-F02: Crowdshipping / Traveler Delivery System ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Both
**Status:** [x] Completed

### Context
Travelers post their routes. Buyers attach items to travelers for physical delivery. A unique platform differentiator.

Reference: `mnbara-platform/apps/web/src/pages/traveler/` — 15 page files. `mnbara-platform/services/trips-service/` and `matching-service/`

---

## TASK-F03: P2P Currency / Item Exchange ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Both
**Status:** [x] Completed

### Context
Peer-to-peer exchange with security deposit, trust level badges, and proof of delivery upload.

Reference: `mnbara-platform/apps/web/src/components/p2p-exchange/` — 15+ components

---

## TASK-F04: AI Chatbot Widget

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** L
**Layer:** Both
**Status:** [x] Completed

### Context
In-platform AI assistant for buyer/seller guidance. Integrates with OpenAI or a self-hosted LLM.

Reference: `mnbara-platform/apps/web/src/components/ai/AIChatWidget.tsx`, `mnbara-platform/services/ai-agent/`

---

## TASK-F05: Fraud Detection Service ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Backend
**Status:** [x] Completed

### Context
ML-based detection of fake listings, bid manipulation, and payment fraud. Requires a separate model training pipeline.

Reference: `mnbara-platform/services/fraud-detection-service/`

---

## TASK-F06: BNPL (Buy Now Pay Later) Integration

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** L
**Layer:** Both
**Status:** [x] Completed

### Context
Installment payment option, critical for GCC markets. Providers: Tamara, Tabby, Spotii.

Reference: `mnbara-platform/services/bnpl-service/`

---

## TASK-F07: Crypto Payments

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** L
**Layer:** Both
**Status:** [x] Completed

### Context
Accept cryptocurrency payments. Requires a crypto payment gateway (e.g. Coinbase Commerce, NOWPayments).

Reference: `mnbara-platform/apps/web/src/pages/CryptoPaymentPage.tsx`, `mnbara-platform/services/crypto-service/`

---

## TASK-F08: AR / VR Product Preview ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Frontend
**Status:** [x] Completed

### Context
Augmented reality for viewing products in real space (WebXR API). VR showroom for browsing.

Reference: `mnbara-platform/apps/web/src/pages/ARPreviewPage.tsx` (10KB), `VRShowroomPage.tsx` (5KB)

---

## TASK-F09: Blockchain / Smart Contracts ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Both
**Status:** [x] Completed

### Context
On-chain escrow, platform token (MNBToken), DAO governance, and staking.

Reference: `mnbara-platform/blockchain/contracts/` — 6 Solidity contracts: `MNBToken.sol`, `MNBWallet.sol`, `MNBAuctionEscrow.sol`, `MNBExchange.sol`, `MNBGovernance.sol`, `MNBStaking.sol`

---

## TASK-F10: Plugin Marketplace ? FREELANCER RECOMMENDED

**Phase:** Future
**Priority:** ?? FUTURE
**Effort:** XL
**Layer:** Both
**Status:** [x] Completed

### Context
Extensible plugin system allowing merchants to add features (custom checkout flows, analytics integrations, shipping providers).

Reference: `mnbara-platform/packages/plugin-sdk/`, `mnbara-platform/apps/web/src/components/plugin-marketplace/`

---

# PHASE 8 — Admin Dashboard Gaps (Mnbara Migration)
**Goal:** Port remaining Mnbara admin features visible in the reference design screenshots.

---

## TASK-035: Banner Ads Manager — Backend + Admin Frontend

**Phase:** 8 — Admin Dashboard Gaps
**Priority:** HIGH
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Done
**Depends on:** None

### Context
The admin dashboard reference screenshots (Layer 5, Screenshot 5) show a "Banner Ads Manager" feature. Mnbara has a full `ad-service` with campaigns, placements (hero carousel, sponsored deals, category spotlight), and ad CRUD. GeoCore Next has no ads/banners system at all.

Reference: `mnbara-platform/services/ad-service/` and `mnbara-platform/apps/web/src/pages/admin/AdsManager.tsx`

### Acceptance Criteria
- [x] `ads` table: id, title, image_url, link_url, placement (hero/sidebar/category/listing_footer), position (sort order), enabled, start_date, end_date, click_count, view_count, created_by, created_at, updated_at
- [x] `GET /api/v1/admin/ads` — list all ads (admin only), filterable by placement/status
- [x] `POST /api/v1/admin/ads` — create ad (admin only)
- [x] `PUT /api/v1/admin/ads/:id` — update ad (admin only)
- [x] `DELETE /api/v1/admin/ads/:id` — delete ad (admin only)
- [x] `PATCH /api/v1/admin/ads/:id/toggle` — enable/disable ad (admin only)
- [x] `GET /api/v1/ads` — public endpoint returns active ads by placement (for frontend homepage banners)
- [x] `POST /api/v1/ads/:id/click` — increment click counter (public, rate-limited)
- [x] Admin page `/content/banners` — CRUD UI with placement selector, date range, image upload URL, enable/disable toggle
- [x] `go build ./...` passes
- [x] `npm run build` passes

### Files to Create / Modify
```
CREATE:
  backend/internal/ads/model.go          — Ad struct + GORM tags
  backend/internal/ads/handler.go        — admin CRUD + public list + click tracking
  backend/internal/ads/routes.go         — RegisterRoutes(v1, db)
  admin/app/content/banners/page.tsx     — Banner Ads Manager page

MODIFY:
  backend/cmd/api/main.go               — register ads routes
  backend/pkg/database/database.go      — add Ad to AutoMigrate
```

---

## TASK-036: Admin Finance CSV/PDF Export

**Phase:** 8 — Admin Dashboard Gaps
**Priority:** HIGH
**Effort:** S (half day)
**Layer:** Backend (Go)
**Status:** [x] Done
**Depends on:** None

### Context
Screenshot 1 shows "CSV / PDF تقارير تصدير" (Export Reports CSV/PDF) in the P&L section. The admin backend has CSV export for transactions but no PDF export and no comprehensive finance report endpoint.

### Acceptance Criteria
- [x] `GET /api/v1/admin/finance/report?format=csv` — export financial summary as CSV (revenue, fees, refunds, escrow balance, payouts)
- [x] `GET /api/v1/admin/finance/report?format=pdf` — export financial summary as PDF
- [x] Report covers configurable date range via `?from=YYYY-MM-DD&to=YYYY-MM-DD`
- [x] PDF generated server-side using Go PDF library (gopdf)
- [x] Admin page `/finance` gets "Export CSV" and "Export PDF" buttons
- [x] `go build ./...` passes

### Files to Create / Modify
```
CREATE:
  backend/internal/admin/finance_export.go  — CSV + PDF export handlers

MODIFY:
  backend/internal/admin/routes.go          — add finance export routes
  admin/app/finance/page.tsx                — add export buttons
```

---

## TASK-037: Chargeback Management

**Phase:** 8 — Admin Dashboard Gaps
**Priority:** MEDIUM
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Done
**Depends on:** None

### Context
Screenshot 3 shows "Chargeback Management" in the Reviews & Disputes section. When a buyer disputes a charge with their bank, the platform needs to track and respond to the chargeback. Currently disputes exist but chargebacks (bank-initiated) are not tracked.

### Acceptance Criteria
- [x] `chargebacks` table: id, payment_id, order_id, stripe_dispute_id, amount, currency, reason, status (open/won/lost/under_review), evidence_due_by, created_at, updated_at
- [x] Stripe webhook handles `charge.dispute.created`, `charge.dispute.updated`, `charge.dispute.closed`
- [x] `GET /api/v1/admin/chargebacks` — list all chargebacks (admin only)
- [x] `POST /api/v1/admin/chargebacks/:id/evidence` — submit evidence to Stripe
- [x] Admin page `/support/chargebacks` — list + detail + submit evidence
- [x] `go build ./...` passes

### Files to Create / Modify
```
CREATE:
  backend/internal/chargebacks/model.go     — Chargeback struct
  backend/internal/chargebacks/handler.go   — admin CRUD + webhook handlers
  backend/internal/chargebacks/routes.go    — RegisterRoutes(v1, db)
  admin/app/support/chargebacks/page.tsx    — Chargeback management page

MODIFY:
  backend/cmd/api/main.go                   — register chargeback routes
  backend/internal/payments/webhook.go      — handle charge.dispute.* events
  backend/pkg/database/database.go          — add Chargeback to AutoMigrate
```

---

## TASK-038: Email Templates Manager

**Phase:** 8 — Admin Dashboard Gaps
**Priority:** MEDIUM
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Done
**Depends on:** None

### Context
Screenshot 5 shows "Email Templates — لكل حدث" (per-event email templates). Currently emails are sent via SMTP job handlers but templates are hardcoded. Admins need to customize email content without code changes.

### Acceptance Criteria
- [x] `email_templates` table: id, event_type (welcome/order_confirmed/password_reset/etc.), subject, body_html, body_text, variables (JSON), is_active, updated_at, updated_by
- [x] `GET /api/v1/admin/email-templates` — list all templates
- [x] `PUT /api/v1/admin/email-templates/:event_type` — update template
- [x] `POST /api/v1/admin/email-templates/:event_type/preview` — render preview with sample data
- [x] Job handlers use DB templates (fallback to hardcoded if not found)
- [x] Admin page `/content/emails` enhanced with template editor + preview
- [x] Seed 10 default templates (welcome, order_confirmed, order_shipped, password_reset, etc.)
- [x] `go build ./...` passes

### Files to Create / Modify
```
CREATE:
  backend/internal/emailtpl/model.go        — EmailTemplate struct
  backend/internal/emailtpl/handler.go      — CRUD + preview
  backend/internal/emailtpl/routes.go       — RegisterRoutes(v1, db)
  backend/internal/emailtpl/seed.go         — default templates

MODIFY:
  backend/cmd/api/main.go                   — register email template routes
  backend/pkg/database/database.go          — add EmailTemplate to AutoMigrate
  backend/pkg/jobs/handlers.go              — load template from DB before sending
  admin/app/content/emails/page.tsx         — enhance with template CRUD
```

---

## TASK-039: Addon Marketplace

**Phase:** 9 — Extensibility & Integrations
**Priority:** HIGH
**Effort:** M (1 day)
**Layer:** Both
**Status:** [x] Done
**Depends on:** None

### Context
Inspired by Mnbara's plugin-system (PluginManager, PluginRegistry, PluginMarketplace), this adds a full addon marketplace to the admin dashboard. Admins can browse, install, enable/disable, and configure platform addons/integrations without code changes.

### Acceptance Criteria
- [x] `addons` table: id, slug, name, description, category, tags, author, version, download_count, avg_rating, rating_count, is_free, price, currency, is_verified, is_official, permissions, hooks, config_schema, status, config, installed_at
- [x] `addon_versions` table: id, addon_id, version, changelog, download_url, min/max_core_version, dependencies, manifest
- [x] `addon_reviews` table: id, addon_id, user_id, rating (1-5), review, version
- [x] `GET /api/v1/admin/addons` — list addons with search, category, status filters
- [x] `GET /api/v1/admin/addons/stats` — marketplace statistics
- [x] `POST /api/v1/admin/addons/:id/install` — install addon
- [x] `POST /api/v1/admin/addons/:id/uninstall` — uninstall addon
- [x] `POST /api/v1/admin/addons/:id/enable` — enable installed addon
- [x] `POST /api/v1/admin/addons/:id/disable` — disable enabled addon
- [x] `PUT /api/v1/admin/addons/:id/config` — update addon configuration
- [x] `GET/POST /api/v1/admin/addons/:id/reviews` — list/add reviews
- [x] Admin page `/marketplace` — browse, install, enable/disable addons with stats dashboard
- [x] Seed 8 default addons (Stripe Payments, GA4, Mailchimp, AR Viewer, WhatsApp, Loyalty, SEO, Fraud Shield)
- [x] `go build ./...` passes

### Files Created / Modified
```
CREATE:
  backend/internal/addons/model.go     — Addon, AddonVersion, AddonReview structs
  backend/internal/addons/handler.go   — marketplace CRUD + lifecycle handlers
  backend/internal/addons/routes.go    — RegisterRoutes(v1, db, rdb)
  backend/internal/addons/seed.go      — 8 default marketplace addons
  admin/app/marketplace/page.tsx      — Marketplace admin page

MODIFY:
  backend/cmd/api/main.go             — register addon routes
  backend/pkg/database/database.go    — AutoMigrate + seed
  admin/lib/api.ts                    — addonsApi
```

---

## TASK-040: CMS — WordPress-like Content Management

**Phase:** 9 — Extensibility & Integrations
**Priority:** HIGH
**Effort:** L (2 days)
**Layer:** Both
**Status:** [x] Done
**Depends on:** None

### Context
The user wants WordPress-like control over the site — manage banners, sliders, content, media, settings, and navigation from the admin dashboard without needing a developer.

### Acceptance Criteria
- [x] `hero_slides` table — banner slider with title, subtitle, image, link, badge, scheduling, reorder
- [x] `content_blocks` table — reusable content sections (HTML, hero, CTA, FAQ, features, testimonial, markdown, image)
- [x] `media_files` table — upload and manage images/videos/documents with folders
- [x] `site_settings` table — global settings (branding, contact, social, SEO, general) with typed inputs
- [x] `nav_menus` table — drag-and-drop menu builder (header, footer, mobile, sidebar)
- [x] Admin CRUD for all 5 CMS entities
- [x] Public API endpoints (no auth) for frontend to consume CMS data
- [x] File upload with folder organization
- [x] Bulk settings update
- [x] Navigation reorder
- [x] Seed default data (3 hero slides, 5 content blocks, 26 site settings, 11 nav items)
- [x] Admin page `/cms` with 5 tabs (Slides, Blocks, Media, Settings, Nav)
- [x] `go build ./...` passes

### Files Created / Modified
```
CREATE:
  backend/internal/cms/model.go      — HeroSlide, ContentBlock, MediaFile, SiteSetting, NavMenu
  backend/internal/cms/handler.go    — CRUD + upload + public API
  backend/internal/cms/routes.go     — admin + public routes
  backend/internal/cms/seed.go      — default CMS data
  admin/app/cms/page.tsx            — CMS admin page with 5 tabs

MODIFY:
  backend/cmd/api/main.go           — register CMS routes + static files
  backend/pkg/database/database.go  — AutoMigrate + seed
  admin/lib/api.ts                  — cmsApi
```

---

# Progress Tracker

| Phase | Description | Total Tasks | Done | In Progress | Blocked |
|-------|-------------|-------------|------|-------------|---------|
| Phase 0 | Foundation (Backend) | 5 | 5 | 0 | 0 |
| Phase 1 | Critical Frontend | 6 | 6 | 0 | 0 |
| Phase 2 | Trust & Guidance Pages | 7 | 7 | 0 | 0 |
| Phase 3 | Seller Tools | 5 | 5 | 0 | 0 |
| Phase 4 | Support & Communication | 3 | 3 | 0 | 0 |
| Phase 5 | Infrastructure & Observability | 5 | 5 | 0 | 0 |
| Phase 6 | Growth & Acquisition | 3 | 3 | 0 | 0 |
| Phase 7+ | Future Roadmap | 10 | 10 | 0 | 0 |
| Phase 8 | Admin Dashboard Gaps | 4 | 4 | 0 | 0 |
| Phase 9 | Extensibility & Integrations | 2 | 2 | 0 | 0 |
| **TOTAL** | | **50** | **50** | **0** | **0** |

---

# Quick Reference — Task Index

| ID | Phase | Layer | Effort | Title |
|----|-------|-------|--------|-------|
| TASK-001 | 0 | Backend | L | Order Management — Backend Models & API |
| TASK-002 | 0 | Backend | M | Shopping Cart — Backend Service |
| TASK-003 | 0 | Backend | S | Watchlist / Favorites — Backend |
| TASK-004 | 0 | Backend | M | Refund & Dispute Resolution — Backend Completion |
| TASK-005 | 0 | Backend | M | Seller Analytics — Backend Data Endpoints |
| TASK-006 | 1 | Frontend | L | Order Management Pages |
| TASK-007 | 1 | Frontend | M | Cart Page + Cart Icon Component |
| TASK-008 | 1 | Frontend | M | Connect Listing ? Cart ? Checkout Flow |
| TASK-009 | 1 | Frontend | S | Watchlist / Favorites Page |
| TASK-010 | 1 | Frontend | S | Legal Pages — Terms, Privacy, Cookie Policy |
| TASK-011 | 1 | Frontend | S | Refund Page + Chargeback Page |
| TASK-012 | 2 | Frontend | M | Help Center & FAQ Page |
| TASK-013 | 2 | Frontend | S | How It Works Page |
| TASK-014 | 2 | Frontend | S | Buyer Protection Page |
| TASK-015 | 2 | Frontend | S | Seller Protection Page |
| TASK-016 | 2 | Frontend | S | About Us Page |
| TASK-017 | 2 | Frontend | S | Shipping & Delivery Info Page |
| TASK-018 | 2 | Frontend | S | Fee Calculator Page |
| TASK-019 | 3 | Frontend | M | Seller Analytics Dashboard Page |
| TASK-020 | 3 | Both | S | Seller Storefront Analytics |
| TASK-021 | 3 | Frontend | M | Loyalty Program Frontend |
| TASK-022 | 3 | Frontend | S | Notification Settings Page |
| TASK-023 | 3 | Both | L | Deals & Promotions — Backend + Frontend |
| TASK-024 | 4 | Frontend | S | Contact & Support Page |
| TASK-025 | 4 | Both | M | Support Ticket System |
| TASK-026 | 4 | Both | M | Founder / Owner Dashboard |
| TASK-027 | 5 | Both | S | Sentry Error Tracking |
| TASK-028 | 5 | Backend | S | Prometheus Metrics Endpoint |
| TASK-029 | 5 | Infra | S | Grafana + Prometheus Docker Compose |
| TASK-030 | 5 | Backend | M | Complete All Job Handler Stubs |
| TASK-031 | 5 | Both | L | PayPal Payment Integration |
| TASK-032 | 6 | Both | L | Referral / Affiliate Program |
| TASK-033 | 6 | Both | XL | Subscription / Plans System ? |
| TASK-034 | 6 | Both | M | Product Request System |
| TASK-F01 | 7+ | Both | XL | Live Streaming Auctions ? |
| TASK-F02 | 7+ | Both | XL | Crowdshipping / Traveler Delivery ? |
| TASK-F03 | 7+ | Both | XL | P2P Currency / Item Exchange ? |
| TASK-F04 | 7+ | Both | L | AI Chatbot Widget |
| TASK-F05 | 7+ | Backend | XL | Fraud Detection Service ? |
| TASK-F06 | 7+ | Both | L | BNPL Integration |
| TASK-F07 | 7+ | Both | L | Crypto Payments |
| TASK-F08 | 7+ | Frontend | XL | AR / VR Product Preview ? |
| TASK-F09 | 7+ | Both | XL | Blockchain / Smart Contracts ? |
| TASK-F10 | 7+ | Both | XL | Plugin Marketplace ? |

> ? = Freelancer Recommended for XL tasks requiring specialized domain knowledge

---

## 🔐 Production Readiness Roadmap

> All items below are tracked as part of the security hardening initiative (Phase PR).
> Evidence for each item is captured in `git` commit history and the sign-off template.

### Phase PR-1: Security Hardening

#### PR-1.1 — Security Headers & HTTPS
- [x] `SecurityHeaders` middleware adds `X-Frame-Options`, `X-Content-Type-Options`, `X-XSS-Protection`, `Strict-Transport-Security`, `Referrer-Policy`
- [x] `ContentSecurityPolicy` middleware registered globally
- [x] HTTPS enforcement: production binds to TLS / behind reverse proxy; non-prod allows HTTP

#### PR-1.2 — Input Validation & SQL Injection Prevention
- [x] `security.SanitizeText` / `security.SanitizeHTML` applied to all user-supplied string fields
- [x] KYC fields sanitized before DB write
- [x] Search query sanitized; `offset` validated non-negative
- [x] Profile update fields (`UpdateMe`) sanitized
- [x] Payment free-text fields (notes, description) sanitized and currency normalized
- [x] All GORM queries use parameterized values — no raw string interpolation

#### PR-1.3 — Password Security
- [x] Argon2id hashing with tuned cost parameters
- [x] `POST /auth/change-password` endpoint — old password verification + session revocation
- [x] Password strength enforced on registration, reset, and change

#### PR-1.4 — Token Security
- [x] RS256-signed JWTs (asymmetric, private key not exposed via API)
- [x] Refresh token rotation: single-use tokens in Redis, reuse detection triggers full session revocation
- [x] `POST /auth/logout` revokes refresh token; audit log emitted
- [x] Access token expiry: 15 min; refresh token expiry: 7 days

#### PR-1.5 — Security Audit Log
- [x] `security_audit_logs` table with async writes (never blocks request)
- [x] Risk scoring per event type (0–100)
- [x] Events covered: `login`, `logout`, `register`, `password_reset`, `kyc_submitted`, `kyc_approved`, `kyc_rejected`, `payment_attempt`, `escrow_released`, `refund_requested`, `wallet_topup`, `rate_limited`, `session_revoked`
- [x] `session_revoked` elevates to risk 90 when reason is `refresh_token_reuse_detected`

#### PR-1.6 — KYC Field Encryption
- [x] PII fields (`full_name`, `id_number`) encrypted at rest with XChaCha20-Poly1305
- [x] `FIELD_ENCRYPTION_KEY` (32-byte base64) loaded from env; required in production
- [x] Decrypt-on-read in status and admin endpoints (callers see plaintext)

---

### Phase PR-2: Fintech Security

#### PR-2.1 — Rate Limiting
- [x] Redis sliding-window rate limiter middleware (`pkg/middleware/ratelimit.go`)
- [x] Per-route limits: register (3/hr), login (5/15min), refresh (10/min), forgot-password (3/hr)
- [x] Global API bucket: 100 req/min per IP
- [x] `rate_limited` audit log event emitted on every 429 (both global and auth-specific limiters)

#### PR-2.2 — Idempotency & Race Condition Prevention
- [x] `IdempotentRequest` table: unique `(user_id, idempotency_key)` with 24-hr TTL
- [x] `X-Idempotency-Key` header honoured by `Deposit`, `Withdraw`, `CreateEscrow`
- [x] All financial writes (`Deposit`, `Withdraw`, `CreateEscrow`, `ReleaseEscrow`) wrapped in `db.Transaction`
- [x] `SELECT ... FOR UPDATE` row-level locking on `wallet_balances` rows prevents TOCTOU over-debit

#### PR-2.3 — IDOR Protection
- [x] `wallet.ReleaseEscrow` — admin-only route (`AdminWithDB` + `AdminOnly` middleware)
- [x] `payments.ReleaseEscrow` — caller must be the buyer (`escrow.BuyerID == user_id` check)
- [x] Deposit / Withdraw / CreateEscrow scoped to authenticated `user_id` from JWT

#### PR-2.4 — Escrow State Machine
- [x] `wallet.ReleaseEscrow` locks escrow row with `FOR UPDATE` before checking state
- [x] Only `PENDING → COMPLETED` transition allowed; `already_processed` sentinel error returned otherwise
- [x] All balance mutations and escrow status update happen atomically in one DB transaction

#### PR-2.5 — Stripe Webhook Verification & Idempotency
- [x] `webhook.ConstructEvent` HMAC-SHA256 verification (`STRIPE_WEBHOOK_SECRET` env var)
- [x] `ProcessedStripeEvent` table with unique `stripe_event_id` index
- [x] Duplicate events (Stripe retries) acknowledged with 200 and skipped without re-processing
- [x] Event record inserted **before** dispatch to survive crash-restart replays

#### PR-2.6 — Fraud Detection Baseline
- [x] `fraud.AnalyzeTransaction` wired into `CreatePaymentIntent`
- [x] Score ≥ 80 → transaction declined + `payment_attempt` audit log with `fraud_declined: true`
- [x] Score ≥ 50 → transaction allowed but flagged with `fraud_flagged: true` in audit log
- [x] `FraudAlert` created automatically for high-risk scores via existing `fraud` package

---

### Phase PR-3: Performance

#### PR-3.1 — Database Indexes
- [x] `idx_sal_user_created` — `security_audit_logs (user_id, created_at DESC)`
- [x] `idx_sal_event_created` — `security_audit_logs (event_type, created_at DESC)`
- [x] `idx_payments_user_status` — `payments (user_id, status)`
- [x] `idx_wallet_tx_wallet_created` — `wallet_transactions (wallet_id, created_at DESC)`
- [x] `idx_escrows_status` / `idx_escrows_buyer` — escrow state & owner queries
- [x] `idx_processed_stripe_event` — webhook dedup lookup
- [x] `idx_idempotent_req_lookup` — idempotency check lookup

#### PR-3.2 — Redis Caching
- [x] `pkg/cache` helper: `Get / Set / Del` with JSON marshalling
- [x] `GET /listings` — cached 2 min for unfiltered page-1 requests
- [x] `GET /listings/:id` — cached 5 min for unauthenticated reads
- [x] Cache invalidated on `PUT /listings/:id` and `DELETE /listings/:id`
