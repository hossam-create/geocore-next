# Workspace

## Overview

pnpm workspace monorepo using TypeScript. Each package manages its own dependencies.

## Stack

- **Monorepo tool**: pnpm workspaces
- **Node.js version**: 24
- **Package manager**: pnpm
- **TypeScript version**: 5.9
- **API framework**: Express 5
- **Database**: PostgreSQL + Drizzle ORM
- **Validation**: Zod (`zod/v4`), `drizzle-zod`
- **API codegen**: Orval (from OpenAPI spec)
- **Build**: esbuild (CJS bundle)

## Structure

```text
artifacts-monorepo/
├── artifacts/              # Deployable applications
│   └── api-server/         # Express API server
├── lib/                    # Shared libraries
│   ├── api-spec/           # OpenAPI spec + Orval codegen config
│   ├── api-client-react/   # Generated React Query hooks
│   ├── api-zod/            # Generated Zod schemas from OpenAPI
│   └── db/                 # Drizzle ORM schema + DB connection
├── scripts/                # Utility scripts (single workspace package)
│   └── src/                # Individual .ts scripts, run via `pnpm --filter @workspace/scripts run <script>`
├── pnpm-workspace.yaml     # pnpm workspace (artifacts/*, lib/*, lib/integrations/*, scripts)
├── tsconfig.base.json      # Shared TS options (composite, bundler resolution, es2022)
├── tsconfig.json           # Root TS project references
└── package.json            # Root package with hoisted devDeps
```

## TypeScript & Composite Projects

Every package extends `tsconfig.base.json` which sets `composite: true`. The root `tsconfig.json` lists all packages as project references. This means:

- **Always typecheck from the root** — run `pnpm run typecheck` (which runs `tsc --build --emitDeclarationOnly`). This builds the full dependency graph so that cross-package imports resolve correctly. Running `tsc` inside a single package will fail if its dependencies haven't been built yet.
- **`emitDeclarationOnly`** — we only emit `.d.ts` files during typecheck; actual JS bundling is handled by esbuild/tsx/vite...etc, not `tsc`.
- **Project references** — when package A depends on package B, A's `tsconfig.json` must list B in its `references` array. `tsc --build` uses this to determine build order and skip up-to-date packages.

## Root Scripts

- `pnpm run build` — runs `typecheck` first, then recursively runs `build` in all packages that define it
- `pnpm run typecheck` — runs `tsc --build --emitDeclarationOnly` using project references

## Packages

### `artifacts/api-server` (`@workspace/api-server`)

Express 5 API server. Routes live in `src/routes/` and use `@workspace/api-zod` for request and response validation and `@workspace/db` for persistence.

- Entry: `src/index.ts` — reads `PORT`, starts Express
- App setup: `src/app.ts` — mounts CORS, JSON/urlencoded parsing, routes at `/api`
- Routes: `src/routes/index.ts` mounts sub-routers; `src/routes/health.ts` exposes `GET /health` (full path: `/api/health`)
- Depends on: `@workspace/db`, `@workspace/api-zod`
- `pnpm --filter @workspace/api-server run dev` — run the dev server
- `pnpm --filter @workspace/api-server run build` — production esbuild bundle (`dist/index.cjs`)
- Build bundles an allowlist of deps (express, cors, pg, drizzle-orm, zod, etc.) and externalizes the rest

### `lib/db` (`@workspace/db`)

Database layer using Drizzle ORM with PostgreSQL. Exports a Drizzle client instance and schema models.

- `src/index.ts` — creates a `Pool` + Drizzle instance, exports schema
- `src/schema/index.ts` — barrel re-export of all models
- `src/schema/<modelname>.ts` — table definitions with `drizzle-zod` insert schemas (no models definitions exist right now)
- `drizzle.config.ts` — Drizzle Kit config (requires `DATABASE_URL`, automatically provided by Replit)
- Exports: `.` (pool, db, schema), `./schema` (schema only)

Production migrations are handled by Replit when publishing. In development, we just use `pnpm --filter @workspace/db run push`, and we fallback to `pnpm --filter @workspace/db run push-force`.

### `lib/api-spec` (`@workspace/api-spec`)

Owns the OpenAPI 3.1 spec (`openapi.yaml`) and the Orval config (`orval.config.ts`). Running codegen produces output into two sibling packages:

1. `lib/api-client-react/src/generated/` — React Query hooks + fetch client
2. `lib/api-zod/src/generated/` — Zod schemas

Run codegen: `pnpm --filter @workspace/api-spec run codegen`

### `lib/api-zod` (`@workspace/api-zod`)

Generated Zod schemas from the OpenAPI spec (e.g. `HealthCheckResponse`). Used by `api-server` for response validation.

### `lib/api-client-react` (`@workspace/api-client-react`)

Generated React Query hooks and fetch client from the OpenAPI spec (e.g. `useHealthCheck`, `healthCheck`).

### `artifacts/mobile` (`@workspace/mobile`)

Expo React Native app for GCC marketplace. 18+ screens with Zustand auth, TanStack Query, Expo Router, and a custom component library. Brand colors: Walmart Blue `#0071CE` + Yellow `#FFC220`.

- Entry: `app/(tabs)/` — tab navigation (Home, Search, Auctions, Messages, Profile)
- Auth: `store/authStore.ts` — Zustand store with Expo SecureStore persistence
- API client: `utils/api.ts` — Axios pointing at `https://geo-core-next.replit.app/api/v1`
- Colors: `constants/colors.ts`

### `artifacts/web` (`@workspace/web`)

React + Vite web frontend for GCC marketplace. Pre-installed: TanStack Query, Radix UI, Wouter, Lucide, Framer Motion, Tailwind CSS 4, Zod, Axios, Zustand.

- Entry: `src/App.tsx` — Wouter router, TanStack Query provider, auth session restore
- API client: `src/lib/api.ts` — Axios with JWT auth interceptor, using Vite proxy `/api` → Go backend on port 9000
- Auth store: `src/store/auth.ts` — Zustand store with localStorage persistence
- Pages: `HomePage`, `ListingsPage`, `AuctionsPage`, `ListingDetailPage`, `LoginPage`, `RegisterPage`, `SellPage`, `SellerPage`, `ProfilePage`, `WalletPage`, `MyStorefrontPage`, `StoreListPage`, `StorefrontPage`
- Components: `layout/Header`, `layout/Footer`, `home/HeroBanner`, `home/CategorySection`, `home/LiveAuctions`, `home/FeaturedListings`, `listings/ListingCard`, `listings/AuctionCard`, `listings/FiltersPanel`, `ui/CountdownTimer`, `ui/LoadingGrid`
- Libs: `api.ts` (Axios client), `categoryFields.ts` (category custom field schemas), `auctionTypes.ts` (Dutch/Reverse type detection), `utils.ts`
- Theme: Walmart Blue `#0071CE` as primary, Yellow `#FFC220` as secondary, `#F5F5F5` background
- Routes: `/` (home), `/listings`, `/listings/:id`, `/auctions`, `/sell`, `/login`, `/register`, `/profile`, `/wallet`, `/my-store`, `/stores`, `/stores/:slug`, `/sellers/:id`
- Header: shows wallet balance (AED) + My Store link when authenticated
- All pages with auth requirements redirect to `/login?next=<page>` when unauthenticated
- Preview path: `/web/` · Port: 22333

### Phase 5: Mobile App, Storefronts & Notifications

- **Go backend Storefronts** (`backend/internal/stores/`) — Full Storefront CRUD: `GET /api/v1/stores`, `GET /api/v1/stores/:slug` (with view count), `GET/POST/PUT /api/v1/stores/me`. Auto-generates slug from name with timestamp collision protection. AutoMigrated `storefronts` table with unique slug + user_id indexes.
- **Push notifications pipeline** — FCM client reads `FIREBASE_SERVICE_ACCOUNT_JSON` env var; gracefully degrades if not set. `POST /api/v1/notifications/register-push-token` and `DELETE /api/v1/notifications/push-tokens/:id` routes wired. Auction `PlaceBid` triggers `new_bid` + `outbid` notifications; chat `SendMessage` triggers `new_message` to all other conversation members.
- **Email notifications** (`backend/pkg/email/transactional.go`) — SMTP-based: `SendWelcomeEmail`, `SendAuctionWonEmail`, `SendAuctionEndedSellerEmail`, `SendPurchaseConfirmationEmail`, `SendOutbidEmail`. Falls back to stdout logging when `SMTP_HOST`/`SMTP_FROM` not set. Welcome email fires on registration; outbid emails fire via notification service.
- **Mobile API integration** (`mobile/utils/api.ts`, `mobile/app/notifications.tsx`) — `notificationsAPI` and `storesAPI` added; notifications screen uses real API with react-query. Push notification setup in `mobile/utils/pushNotifications.ts` — registers FCM token via `POST /notifications/register-push-token` after login.
- **Frontend storefronts** — `BrandOutletPage.tsx` added to both artifact directories, loads real `/api/v1/stores` API alongside curated brands. `MyStorefrontPage.tsx` enhanced with listings display and store stats. Routes `/brand-outlet`, `/stores`, `/stores/:slug`, `/my-store` all wired.

### Phase 4: Trust, Safety & Admin

- **Seller Reviews** (`frontend/artifacts/web/src/components/reviews/SellerReviews.tsx`) — Review display + submission on SellerPage. Star rating picker, real API data only (no mock fallback), POST `/users/:id/reviews`. Eligibility gated: reviewer must have a completed purchase from the seller.
- **KYC Verification** (`frontend/artifacts/web/src/components/kyc/KYCSection.tsx`) — KYC status banner + file upload form embedded in ProfilePage. Document images uploaded via `POST /api/v1/media/upload-url` (presigned URL), then submitted to `POST /kyc/submit`. Shows pending/approved/rejected state.
- **Buyer-Seller Chat** (`frontend/artifacts/web/src/components/chat/ChatPanel.tsx`) — Floating chat panel with real-time WebSocket (`/api/v1/chat/conversations/:id/ws?token=<jwt>`). Auth via JWT query param; membership enforced before WS upgrade. Messages sent via REST (`POST /chat/conversations/:id/messages`) which persists and broadcasts to WS subscribers. WS is server-push only (client frames are discarded to prevent spoofing). Vite proxy has `ws: true` for WS forwarding.
- **Admin Dashboard** (`frontend/artifacts/admin/`) — Full admin UI with ban/unban users, approve/reject listings, KYC approval/rejection, reports queue. All wired to Go backend endpoints.
- **Security Hardening** (commit `2ec1336`) — Four layers:
  - *CORS allowlist*: `ALLOWED_ORIGINS` env var (comma-separated) replaces single `FRONTEND_URL`. Dev allows all; prod locks to allowlist; prod without env var warns + defaults to localhost.
  - *Per-user rate limits*: `LimitByUser` wired to three high-value endpoints: 10 listings/hour, 30 bids/minute, 60 chat messages/minute. Sliding-window Redis implementation, fails open if Redis unavailable.
  - *Fraud detector* (`internal/fraud/detector.go`): Evaluates three live DB signals — listing velocity (24 h), bid velocity (1 h), new-account flag (10 min). Composite weighted score 0–1. Fires `slog.Warn` on score ≥ 0.70. Called async (non-blocking) from Create listing and PlaceBid handlers.
  - *Admin defense-in-depth*: `requireAdmin()` helper added to `admin/handler.go`; called at the top of all 14 admin handler functions. Guards against any future middleware mis-ordering.

### `artifacts/admin` (`@workspace/admin`)

React + Vite + shadcn admin dashboard. Pages: Dashboard, Listings, Auctions, Users, KYC Verification, Reports, Payments, Pricing, Categories, Storefronts, Settings.

- KYC page (`pages/kyc.tsx`) — full admin KYC review UI with stat cards, data table, document viewer, approve/reject actions
- Layout sidebar shows pending KYC badge count fetched from API
- AI Bid Suggestions available via `POST /api/v1/ai/predict`
- API: Express api-server running locally (mock data) + real Go backend on production

### `artifacts/api-server` routes (extended)

Additional local development routes:
- `routes/kyc.ts` — KYC CRUD (stats, list, getOne, approve, reject, under-review) with 4 mock profiles
- `routes/ai-pricing.ts` — AI pricing endpoints (predict, strategies, categories) — TypeScript port of Python engine
- `routes/media.ts` — Image upload routes: `POST /api/v1/media/upload-url` (presigned R2 URL), `DELETE /api/v1/media/delete`, `GET /api/v1/media/config`
- `routes/auth.ts` — Mock auth: login/register/refresh/me with 3 demo users (demo@geocore.com/demo1234, seller@geocore.com/seller123, test@test.com/test123)
- `app.ts` — Catch-all proxy: unknown routes forward to Go backend (`geo-core-next.replit.app`)

### Image Upload (Cloudflare R2)

- **Component**: `artifacts/web/src/components/ui/ImageUploader.tsx`
  - Drag-and-drop + file browser, multi-image grid (up to 8), "Main" badge on first image
  - Presigned URL upload flow: `POST /api/v1/media/upload-url` → PUT direct to R2 → public URL stored
  - Mock fallback: returns picsum.photos URLs when R2 env vars not set
  - Allowed: JPEG/PNG/WebP/GIF/AVIF · Max 10MB per image
- **SellPage integration**: `artifacts/web/src/pages/SellPage.tsx` — Step 3 uses ImageUploader; form state uses `uploadedImages[]` (key, url, file_name)
- **Auth mock fallback**: `artifacts/web/src/store/auth.ts` — when Go backend unavailable, falls back to local mock users

### AI Search (SearchPage)

- **File**: `artifacts/web/src/pages/SearchPage.tsx`
- **Approach**: Client-side mock data (12 GCC listings) + 600ms AI simulation delay
- **API (local api-server)**: POST `/api/v1/ai/search`, GET `/api/v1/ai/search/suggest`, GET `/api/v1/ai/search/trending`
  - Real OpenAI query understanding (gpt-4o-mini) + keyword extraction + intent parsing
  - Route: `artifacts/api-server/src/routes/ai-search.ts`
- **Go backend**: `backend/internal/search/handler.go` — pgvector cosine similarity + OpenAI embeddings + text fallback
  - Migration: `backend/migrations/20260324_001_pgvector_search.sql`
- **Vite proxy**: `/api` → Go backend on port 9000 (dev only)
- **Features**: Autocomplete suggestions, AI intent card with category/location tags, "Best Match" badge, filters panel (Category/Price/Location), trending searches, Arabic + English support
- **Route**: `/search?q=<query>` (accessible from Header search bar with Sparkles AI badge)

### GitHub repo: `hossam-create/geocore-next` (Go backend)

Production Go backend with all 8 phases + new integrations:
- `backend/internal/kyc/` — KYCProfile, KYCDocument, KYCAuditLog models + full CRUD handler + RequireKYC middleware
- `backend/internal/auctions/ai_pricing_client.go` — HTTP client for Python AI pricing service
- `ai-service/` — Python Flask pricing microservice (inspired by T51-AI-Bidding-and-Auction-Pricing-Agent)
  - Endpoints: POST /predict, GET /strategies, GET /categories, GET /health
  - Statistical model: urgency score + competition pressure + category multipliers
  - GCC currency rounding (AED/SAR/KWD/QAR/BHD/OMR)
- `backend/internal/search/handler.go` — Semantic search with pgvector + OpenAI text-embedding-3-small + text fallback
- `backend/migrations/20260324_001_pgvector_search.sql` — pgvector extension + listing_embeddings + search_queries tables
- Kubernetes: `k8s/` — Full production deployment (Deployments, Services, HPA, Ingress, PodDisruptionBudget, NetworkPolicy, RBAC, Sealed Secrets)
  - Domains: geocore.app / api.geocore.app / admin.geocore.app
  - Images: ghcr.io/hossam-create/geocore-{api,frontend,admin}:latest

### `scripts` (`@workspace/scripts`)

Utility scripts package. Each script is a `.ts` file in `src/` with a corresponding npm script in `package.json`. Run scripts via `pnpm --filter @workspace/scripts run <script>`. Scripts can import any workspace package (e.g., `@workspace/db`) by adding it as a dependency in `scripts/package.json`.
