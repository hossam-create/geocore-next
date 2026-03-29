# ✅ DOCUMENT 3: TASKS.md — GeoCore Next Development Tasks

---

## EPIC 1 — Foundation (TASK-001 → TASK-010)

---

## TASK-001: Verify Go Build Passes
**Epic:** Foundation | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** none | **Go Status:** Done

### Description
Run `go build ./...` and fix any import issues. The codebase has duplicate utility functions (`getenv` in `main.go` and `database.go`, `defaultStr` in `listings/handler.go` and `chat/handler.go`) that need consolidation.

### Technical Notes
Extract shared helpers to `pkg/util/util.go`. Check that all internal package imports resolve correctly. The `pkg/redis/redis.go` package is defined but unused in `main.go` — decide whether to use it or remove it.

### Acceptance Criteria
- [x] `go build ./...` completes with zero errors
- [x] `go vet ./...` has no warnings
- [x] Duplicate `getenv` and `defaultStr` functions consolidated into `pkg/util/`
- [x] `pkg/redis/redis.go` either used or removed

### Files
- `backend/pkg/util/util.go` — create with shared helpers
- `backend/cmd/api/main.go` — remove duplicate `getenv`, import from `pkg/util`
- `backend/pkg/database/database.go` — remove duplicate `getenv`, import from `pkg/util`
- `backend/internal/listings/handler.go` — remove `defaultStr`, import from `pkg/util`
- `backend/internal/chat/handler.go` — remove `defaultStr`, import from `pkg/util`

---

## TASK-002: PostgreSQL Connection with Retry
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-001 | **Go Status:** Partial

### Description
Add retry logic to `database.Connect()`. Currently it fails immediately if PostgreSQL isn't ready. In Docker, the API container often starts before Postgres despite `depends_on` health checks.

### Technical Notes
Implement exponential backoff: 5 attempts, starting at 2s. Use `time.Sleep` between attempts. Log each retry attempt with `zap.Warn`. Pattern: wrap the `gorm.Open()` call in a loop.

### Acceptance Criteria
- [ ] `database.Connect()` retries up to 5 times with 2s backoff
- [ ] Each retry attempt is logged with attempt number
- [ ] After 5 failures, returns error with clear message
- [ ] `docker-compose up` starts cleanly even with slow Postgres startup

### Files
- `backend/pkg/database/database.go` — add retry loop around `gorm.Open()`

---

## TASK-003: Redis Connection with Health Check
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-001 | **Go Status:** Partial

### Description
Add PING health check on Redis startup with retry, matching the DB retry pattern. Currently `main.go` has a basic ping but no retry.

### Technical Notes
Use the same 5-attempt, 2s-backoff pattern as TASK-002. Consider using `pkg/redis/redis.go` `Connect()` instead of inline client creation in `main.go`.

### Acceptance Criteria
- [ ] Redis connection retries up to 5 times with 2s backoff
- [ ] `redis-cli PING` returns PONG before API accepts traffic
- [ ] Health check logged with zap
- [ ] Consistent with DB retry pattern

### Files
- `backend/cmd/api/main.go` — add Redis retry logic or use `pkg/redis`

---

## TASK-004: JWT Middleware with Dual Tokens
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-001 | **Go Status:** Partial

### Description
Upgrade from single 30-day JWT to access (15min) + refresh (30d) token pair. Add `Role` field to Claims so `AdminOnly()` middleware works.

### Technical Notes
Current `generateToken()` in `auth/handler.go` creates one 30-day token. Split into `generateAccessToken(15min)` and `generateRefreshToken(30d)`. Store refresh token hash in Redis with TTL. Add `Role string` to `middleware.Claims` and set `user_role` in gin context.

### Acceptance Criteria
- [ ] Access token expires in 15 minutes
- [ ] Refresh token expires in 30 days, stored in Redis
- [ ] `Claims` struct includes `Role` field
- [ ] Auth middleware sets `user_id`, `user_email`, and `user_role` in gin context
- [ ] `AdminOnly()` correctly reads `user_role` from context

### Files
- `backend/pkg/middleware/auth.go` — add `Role` to Claims, set `user_role` in context
- `backend/internal/auth/handler.go` — split `generateToken` into access+refresh, include role

---

## TASK-005: Rate Limiting
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-003 | **Go Status:** Not started

### Description
Add IP-based rate limiting using Redis INCR with sliding window. 100 requests per minute per IP. Skip OPTIONS preflight requests.

### Technical Notes
Create `pkg/middleware/ratelimit.go`. Use Redis key `rate:{ip}` with INCR and EXPIRE 60s. Return 429 with `Retry-After` header when exceeded. Check `c.Request.Method == "OPTIONS"` to skip preflight.

### Acceptance Criteria
- [ ] 100 req/min per IP enforced via Redis
- [ ] OPTIONS requests are exempt
- [ ] 429 response includes `Retry-After` header
- [ ] Rate limit headers (`X-RateLimit-Remaining`, `X-RateLimit-Limit`) on every response

### Files
- `backend/pkg/middleware/ratelimit.go` — create rate limiter middleware
- `backend/cmd/api/main.go` — apply middleware globally

---

## TASK-006: Standard Response Package Verification
**Epic:** Foundation | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-001 | **Go Status:** Done

### Description
Verify the existing `response` package works correctly for all status codes. The package already has OK/Created/BadRequest/Unauthorized/Forbidden/NotFound/InternalError helpers. Verify they are used consistently across all handlers.

### Technical Notes
`pkg/response/response.go` defines `R{Success, Data, Error, Meta}` struct. Audit all handlers to ensure none use `c.JSON()` directly. The `InternalError` function takes an error but doesn't expose it — verify this is intentional for security.

### Acceptance Criteria
- [ ] No handler calls `c.JSON()` directly — all use `response.*` helpers
- [ ] `InternalError` logs the actual error (add zap logging if missing)
- [ ] Write unit test for each response helper function
- [ ] `Meta` struct used consistently for pagination

### Files
- `backend/pkg/response/response.go` — add error logging to InternalError
- `backend/pkg/response/response_test.go` — create unit tests

---

## TASK-007: Docker Compose Health Verification
**Epic:** Foundation | **Type:** chore | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-002, TASK-003 | **Go Status:** Partial

### Description
Verify all 5 Docker services start healthy. Current docker-compose.yml has postgres and redis health checks but API has none. Add API health check and Meilisearch health check.

### Technical Notes
The API already has `GET /health` returning `{"status":"ok"}`. Add a `healthcheck` block to the `api` service in docker-compose.yml. Meilisearch has a built-in `/health` endpoint.

### Acceptance Criteria
- [ ] `docker-compose up -d` starts all 5 services
- [ ] `docker-compose ps` shows all services as "healthy"
- [ ] API health check: `curl localhost:8080/health` returns 200
- [ ] No service restarts within 60 seconds of startup

### Files
- `docker-compose.yml` — add healthcheck for api and meilisearch services

---

## TASK-008: GitHub Actions CI
**Epic:** Foundation | **Type:** chore | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-001 | **Go Status:** Not started

### Description
Create GitHub Actions workflow for CI: run `go vet`, `go test`, and `go build` on every push and pull request.

### Technical Notes
Use `actions/setup-go@v5` with Go 1.23. Cache go modules. Run against push to main and all PRs. Add PostgreSQL and Redis service containers for integration tests.

### Acceptance Criteria
- [ ] `.github/workflows/ci.yml` exists and runs on push/PR
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes
- [ ] `go build ./cmd/api/main.go` passes
- [ ] CI completes in under 3 minutes

### Files
- `.github/workflows/ci.yml` — create CI workflow

---

## TASK-009: Next.js 15 Frontend Init
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** none | **Go Status:** Not started

### Description
Initialize Next.js 15 frontend with App Router, TypeScript strict mode, Tailwind CSS, and shadcn/ui. Set up project structure.

### Technical Notes
Use `npx create-next-app@latest ./frontend --typescript --tailwind --eslint --app --src-dir`. Install shadcn/ui via `npx shadcn-ui@latest init`. Configure `NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1`.

### Acceptance Criteria
- [ ] `frontend/` directory created with Next.js 15
- [ ] TypeScript strict mode enabled
- [ ] Tailwind CSS configured
- [ ] shadcn/ui initialized with at least Button, Input, Card components
- [ ] `npm run dev` starts on port 3000
- [ ] `npm run build` completes without errors

### Files
- `frontend/` — entire new Next.js project
- `frontend/src/lib/api.ts` — create API client stub
- `frontend/src/app/layout.tsx` — root layout with providers

---

## TASK-010: Frontend API Client
**Epic:** Foundation | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-009 | **Go Status:** Not started

### Description
Create an axios-based API client with auto-refresh token on 401 response. Base URL from `NEXT_PUBLIC_API_URL`.

### Technical Notes
Create `api.ts` with axios instance. Add response interceptor: on 401, try refresh token, retry original request. Store tokens in httpOnly cookies or localStorage (localStorage for simplicity in MVP). Export typed API functions.

### Acceptance Criteria
- [ ] `api.ts` exports configured axios instance
- [ ] Base URL reads from `NEXT_PUBLIC_API_URL` env var
- [ ] 401 interceptor attempts token refresh before failing
- [ ] Typed API functions for auth (login, register, refresh)
- [ ] Error handling wraps axios errors into user-friendly messages

### Files
- `frontend/src/lib/api.ts` — create API client with interceptors
- `frontend/src/lib/auth.ts` — token storage and refresh logic

---

## EPIC 2 — Auth & Users (TASK-011 → TASK-020)

---

## TASK-011: POST /auth/register — Dual Tokens
**Epic:** Auth & Users | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-004 | **Go Status:** Partial

### Description
Update Register handler to return access + refresh token pair instead of single token. bcrypt cost 12 is already correct.

### Technical Notes
Current `auth/handler.go:Register()` calls `generateToken()` which returns one 30-day JWT. After TASK-004 splits tokens, update Register to return both `access_token` and `refresh_token` in response. Store refresh token hash in Redis with key `refresh:{userID}:{tokenID}`.

### Acceptance Criteria
- [ ] Register returns `{access_token, refresh_token, user}` (user without password_hash)
- [ ] Access token expires in 15 minutes
- [ ] Refresh token expires in 30 days
- [ ] Refresh token stored in Redis with correct TTL
- [ ] Duplicate email returns 400 "Email already in use"
- [ ] Password validation enforces min 8 chars

### Files
- `backend/internal/auth/handler.go` — update Register to return dual tokens

---

## TASK-012: POST /auth/login — Dual Tokens
**Epic:** Auth & Users | **Type:** feature | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-011 | **Go Status:** Partial

### Description
Update Login handler to return access + refresh token pair. Currently returns single token.

### Technical Notes
Same pattern as TASK-011 but for Login. Verify password with `bcrypt.CompareHashAndPassword`, then generate dual tokens. Update last_seen_at on successful login.

### Acceptance Criteria
- [ ] Login returns `{access_token, refresh_token, user}`
- [ ] Invalid email returns 401 (not 404, to prevent enumeration)
- [ ] Invalid password returns 401
- [ ] `last_seen_at` updated on successful login

### Files
- `backend/internal/auth/handler.go` — update Login response

---

## TASK-013: POST /auth/refresh — Token Rotation
**Epic:** Auth & Users | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-012 | **Go Status:** Not started

### Description
Implement refresh token endpoint. Validates the refresh token, rotates both tokens, and invalidates the old refresh token in Redis.

### Technical Notes
Accept `{refresh_token}` in body. Parse and validate JWT. Check Redis for `refresh:{userID}:{tokenID}` — if missing, token was already used (possible theft, invalidate all user tokens). Generate new access+refresh pair, store new refresh in Redis, delete old.

### Acceptance Criteria
- [ ] POST /auth/refresh accepts `{refresh_token}`
- [ ] Valid refresh → returns new access+refresh pair
- [ ] Old refresh token invalidated in Redis after use
- [ ] Expired refresh → 401
- [ ] Reused refresh token → invalidate ALL user tokens (security)
- [ ] Missing/malformed token → 400

### Files
- `backend/internal/auth/handler.go` — add `Refresh` method
- `backend/internal/auth/routes.go` — add `POST /auth/refresh` route

---

## TASK-014: POST /auth/google — OAuth 2.0
**Epic:** Auth & Users | **Type:** feature | **Priority:** P1 | **Estimate:** 4h | **Depends on:** TASK-013 | **Go Status:** Not started

### Description
Implement Google OAuth 2.0 login. Accept Google ID token from frontend, verify with Google, upsert user on first login.

### Technical Notes
Use `google.golang.org/api/oauth2/v2` to verify the ID token. Extract email, name, avatar from Google claims. If user exists with that email, link Google account. If new, create user with `auth_provider=google` and no password_hash. Return dual tokens.

### Acceptance Criteria
- [ ] POST /auth/google accepts `{id_token}`
- [ ] Verifies token with Google's public keys
- [ ] New Google user → creates account, returns tokens
- [ ] Existing email → links Google, returns tokens
- [ ] Invalid token → 401

### Files
- `backend/internal/auth/handler.go` — add `GoogleAuth` method
- `backend/internal/auth/routes.go` — add `POST /auth/google`
- `backend/internal/users/model.go` — add `AuthProvider` field

---

## TASK-015: GET /users/me — Full Profile
**Epic:** Auth & Users | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-004 | **Go Status:** Done

### Description
Verify `/users/me` returns full user object excluding password_hash. Currently implemented in both `auth/handler.go:Me()` and `users/handler.go:GetMe()`.

### Technical Notes
There's duplication: `auth/handler.go` has `Me()` and `users/handler.go` has `GetMe()`. Consolidate to one. Ensure `PasswordHash` has `json:"-"` tag (already present). Add test.

### Acceptance Criteria
- [ ] GET /auth/me or GET /users/me returns full user object
- [ ] `password_hash` is NOT in the response
- [ ] Duplicate endpoint consolidated
- [ ] Unit test verifies password_hash exclusion

### Files
- `backend/internal/auth/handler.go` — remove duplicate `Me()` or redirect
- `backend/internal/users/handler.go` — keep `GetMe()`

---

## TASK-016: PUT /users/me — Profile Update
**Epic:** Auth & Users | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-015 | **Go Status:** Done

### Description
Verify PUT /users/me correctly updates allowed fields: name, bio, location, language, currency.

### Technical Notes
Current implementation in `users/handler.go:UpdateMe()` manually checks each field. Consider using a whitelist map pattern like `listings/handler.go:Update()` for consistency.

### Acceptance Criteria
- [ ] Only whitelisted fields can be updated
- [ ] Empty string for `name` does not clear it
- [ ] `language` and `currency` validated against allowed values
- [ ] Response includes updated user object

### Files
- `backend/internal/users/handler.go` — verify/improve `UpdateMe()`

---

## TASK-017: GET /users/:id/profile — Public Profile
**Epic:** Auth & Users | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-015 | **Go Status:** Done

### Description
Verify public profile endpoint returns only non-PII data using `ToPublic()` method.

### Technical Notes
Current `users/handler.go:GetProfile()` calls `user.ToPublic()` which returns `PublicUser{ID, Name, AvatarURL, Rating, ReviewCount, IsVerified, Location, CreatedAt}`. Verify no email, phone, or password_hash leaks.

### Acceptance Criteria
- [ ] Response matches `PublicUser` struct exactly
- [ ] No email, phone, or password_hash in response
- [ ] 404 for non-existent user ID
- [ ] Blocked users return 404 (not their profile)

### Files
- `backend/internal/users/handler.go` — verify `GetProfile()`

---

## TASK-018: POST /users/me/avatar — Cloudinary Upload
**Epic:** Auth & Users | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-016 | **Go Status:** Not started

### Description
Add avatar upload endpoint. Accept multipart file, upload to Cloudinary, update user's `avatar_url`.

### Technical Notes
Use Cloudinary Go SDK. Accept `multipart/form-data` with `avatar` field. Validate file type (jpeg, png, webp) and size (< 5MB). Upload to `geocore/avatars/{userID}` folder. Return updated user object.

### Acceptance Criteria
- [ ] POST /users/me/avatar accepts multipart file upload
- [ ] File type validated: jpeg, png, webp only
- [ ] File size validated: max 5MB
- [ ] Uploaded to Cloudinary with proper folder structure
- [ ] User `avatar_url` updated in database
- [ ] Old avatar deleted from Cloudinary (if exists)

### Files
- `backend/internal/users/handler.go` — add `UploadAvatar` method
- `backend/internal/users/routes.go` — add route
- `backend/pkg/cloudinary/cloudinary.go` — create Cloudinary upload helper

---

## TASK-019: Frontend Login Page
**Epic:** Auth & Users | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-009, TASK-010 | **Go Status:** Not started

### Description
Create login page with email/password form, Zod validation, and Zustand auth store. Redirect to homepage on success.

### Technical Notes
Use shadcn/ui Input and Button. Zod schema: email required, password min 8. On submit, call `api.post('/auth/login')`, store tokens in auth store, redirect to `/`. Show error toast on failure.

### Acceptance Criteria
- [ ] Login form with email and password fields
- [ ] Client-side Zod validation with error messages
- [ ] Successful login stores tokens and redirects to `/`
- [ ] Failed login shows error message without revealing if email exists
- [ ] Loading state during API call
- [ ] Link to register page

### Files
- `frontend/src/app/(auth)/login/page.tsx` — login page
- `frontend/src/stores/auth.ts` — Zustand auth store
- `frontend/src/lib/validations/auth.ts` — Zod schemas

---

## TASK-020: Frontend Register Page + Protected Routes
**Epic:** Auth & Users | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-019 | **Go Status:** Not started

### Description
Create registration page and protected route HOC/middleware. Unauthenticated users are redirected to `/login`.

### Technical Notes
Registration form: name, email, password, confirm password. Create `ProtectedRoute` component or Next.js middleware that checks auth store for valid token. Use `useEffect` to redirect.

### Acceptance Criteria
- [ ] Register form with name, email, password, confirm password
- [ ] Zod validation: name 2-100 chars, email valid, password 8+ chars, passwords match
- [ ] Successful registration auto-logs in and redirects to `/`
- [ ] Protected route HOC redirects to `/login` if unauthenticated
- [ ] Auth middleware in Next.js checks token on protected pages
- [ ] Link to login page

### Files
- `frontend/src/app/(auth)/register/page.tsx` — register page
- `frontend/src/components/auth/protected-route.tsx` — HOC
- `frontend/src/middleware.ts` — Next.js auth middleware

---

## EPIC 3 — Categories & Listings (TASK-021 → TASK-035)

---

## TASK-021: Seed 10 Categories with EN + AR Names
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-001 | **Go Status:** Done

### Description
Verify `listings/seed.go:SeedCategories()` runs idempotently on startup with 10 categories (EN+AR).

### Technical Notes
Already implemented with count check. Seeds: Vehicles, Real Estate, Electronics, Furniture, Clothing, Jobs, Services, Animals & Pets, Sports & Hobbies, Kids & Baby. Each has emoji icon and sort_order.

### Acceptance Criteria
- [ ] 10 categories seeded on first startup
- [ ] Re-running does not duplicate categories (idempotent)
- [ ] Each category has `name_en`, `name_ar`, `slug`, `icon`, `sort_order`
- [ ] Categories queryable via GET /categories

### Files
- `backend/internal/listings/seed.go` — verify existing implementation

---

## TASK-022: GET /categories with Redis Cache
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-021 | **Go Status:** Partial

### Description
Add Redis caching (5min TTL) to GET /categories. Currently queries DB on every request.

### Technical Notes
Use `pkg/redis` helpers. Cache key: `categories:tree`. Serialize to JSON before storing. Handle cache miss gracefully. Invalidate cache when categories are modified (future admin feature).

### Acceptance Criteria
- [ ] First request populates Redis cache
- [ ] Subsequent requests served from Redis (< 5ms)
- [ ] Cache expires after 5 minutes
- [ ] Cache miss falls back to DB query
- [ ] Response includes tree with children

### Files
- `backend/internal/listings/handler.go` — add caching to `GetCategories()`

---

## TASK-023: GET /listings — Full Filtering
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-001 | **Go Status:** Done

### Description
Verify listing search/filter endpoint handles all query params: q, category_id, country, city, min_price, max_price, condition, type, sort, page, per_page.

### Technical Notes
Already implemented in `listings/handler.go:List()`. Uses ILIKE for text search, supports `newest`, `price_asc`, `price_desc`, `popular` sorts. Per page capped at 50. Returns `response.Meta` with pagination.

### Acceptance Criteria
- [ ] All query params work correctly
- [ ] Empty results return `[]` not `null`
- [ ] Pagination meta is accurate
- [ ] Featured listings appear first with default sort
- [ ] Only active listings are returned
- [ ] SQL injection safe (parameterized via GORM)

### Files
- `backend/internal/listings/handler.go` — verify `List()` implementation

---

## TASK-024: GET /listings/:id — Preload + View Count
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-023 | **Go Status:** Done

### Description
Verify listing detail preloads Images + Category and increments view_count asynchronously.

### Technical Notes
Already implemented with `go h.db.Model(&listing).UpdateColumn("view_count", gorm.Expr("view_count + 1"))`. The async goroutine means view count may lag by a fraction of a second.

### Acceptance Criteria
- [ ] Images and Category preloaded in response
- [ ] `view_count` incremented without blocking response
- [ ] 404 for non-existent or non-active listings
- [ ] UUID validation on ID parameter

### Files
- `backend/internal/listings/handler.go` — verify `Get()`

---

## TASK-025: POST /listings — Create with Expiry
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-004 | **Go Status:** Done

### Description
Verify listing creation sets `expires_at = now + 60 days` and `status = active`. Currently sets 2 months which is approximately 60 days.

### Technical Notes
Current code: `expires := time.Now().AddDate(0, 2, 0)` — this gives ~60 days but varies (Feb = 59d). Consider using `time.Now().Add(60 * 24 * time.Hour)` for exact 60 days. Images saved inline from URL list.

### Acceptance Criteria
- [ ] `expires_at` set to approximately 60 days from creation
- [ ] `status` defaults to `active`
- [ ] All required fields validated (title 5-200, description 10+, country, city)
- [ ] Images created with sort_order, first image marked `is_cover = true`
- [ ] Returns created listing with assigned UUID

### Files
- `backend/internal/listings/handler.go` — verify `Create()`

---

## TASK-026: PUT /listings/:id — Owner-Only Update
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-025 | **Go Status:** Done

### Description
Verify update is owner-only and only allows whitelisted fields.

### Technical Notes
Current whitelist: `title, description, price, currency, price_type, condition, country, city, address, status`. Uses `map[string]interface{}` pattern. Verify `user_id` ownership check.

### Acceptance Criteria
- [ ] Only listing owner can update
- [ ] Non-whitelisted fields (e.g., `user_id`, `view_count`) are ignored
- [ ] 404 if listing doesn't exist or user doesn't own it
- [ ] Updated listing returned in response

### Files
- `backend/internal/listings/handler.go` — verify `Update()`

---

## TASK-027: DELETE /listings/:id — Soft Delete with Auction Check
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-026 | **Go Status:** Partial

### Description
Enhance soft delete to block deletion if an active auction exists on the listing.

### Technical Notes
Current `Delete()` does owner-only soft delete but doesn't check for active auctions. Add query: `SELECT id FROM auctions WHERE listing_id = ? AND status = 'active'`. Return 409 Conflict if found.

### Acceptance Criteria
- [ ] Soft delete succeeds for listings without active auctions
- [ ] 409 Conflict returned if listing has an active auction
- [ ] Error message: "Cannot delete listing with active auction"
- [ ] 404 if listing doesn't exist or user doesn't own it

### Files
- `backend/internal/listings/handler.go` — enhance `Delete()` with auction check

---

## TASK-028: POST /listings/:id/images — Cloudinary Upload
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-025 | **Go Status:** Not started

### Description
Add image upload endpoint. Accept multipart files (up to 10), upload to Cloudinary, save as ListingImage records.

### Technical Notes
Currently images are saved from URL strings in the Create handler. This task adds actual file upload. Max 10 images per listing. First image (or explicitly marked one) becomes `is_cover`. Validate file types: jpeg, png, webp.

### Acceptance Criteria
- [ ] POST accepts multipart form with up to 10 files
- [ ] Files validated: jpeg/png/webp, max 5MB each
- [ ] Uploaded to Cloudinary under `geocore/listings/{listingID}/`
- [ ] ListingImage records created with correct sort_order
- [ ] First image has `is_cover = true`
- [ ] 400 if total images would exceed 10

### Files
- `backend/internal/listings/handler.go` — add `UploadImages` method
- `backend/internal/listings/routes.go` — add route
- `backend/pkg/cloudinary/cloudinary.go` — create if not from TASK-018

---

## TASK-029: POST /listings/:id/favorite — Toggle
**Epic:** Categories & Listings | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-001 | **Go Status:** Done

### Description
Verify favorite toggle works correctly with atomic counter update.

### Technical Notes
Already implemented in `listings/handler.go:ToggleFavorite()`. Uses `gorm.Expr("favorite_count + 1")` and `gorm.Expr("favorite_count - 1")` for atomicity. Check for negative count edge case.

### Acceptance Criteria
- [ ] Toggle on: creates Favorite record, increments `favorite_count`
- [ ] Toggle off: deletes Favorite record, decrements `favorite_count`
- [ ] `favorite_count` never goes below 0
- [ ] Works with concurrent requests (atomic DB operations)

### Files
- `backend/internal/listings/handler.go` — verify `ToggleFavorite()`

---

## TASK-030: GET /listings/me — User's Listings
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-023 | **Go Status:** Partial

### Description
Fix route ordering issue and add status filter to user's listings endpoint.

### Technical Notes
**Critical bug:** In `routes.go`, `GET /listings/me` is registered AFTER `GET /listings/:id`, so Gin matches `/me` as an `:id` param. Fix by registering `/me` before `/:id`, or use a separate group. Add optional `?status=active|sold|expired|draft` filter.

### Acceptance Criteria
- [ ] `GET /listings/me` correctly returns current user's listings (not treated as `:id`)
- [ ] Optional `?status=` filter works
- [ ] Images preloaded
- [ ] Ordered by `created_at DESC`

### Files
- `backend/internal/listings/routes.go` — fix route ordering
- `backend/internal/listings/handler.go` — add status filter to `GetMyListings()`

---

## TASK-031: Cron — Expire Listings
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-025 | **Go Status:** Not started

### Description
Create a background goroutine that runs at midnight to expire listings where `expires_at < NOW()` and `status = 'active'`.

### Technical Notes
Use `time.NewTicker` or a cron library like `robfig/cron`. Run as a goroutine started in `main.go`. Update query: `UPDATE listings SET status = 'expired' WHERE expires_at < NOW() AND status = 'active'`. Log number of expired listings.

### Acceptance Criteria
- [ ] Cron runs at midnight UTC daily
- [ ] Expired listings have status changed to `expired`
- [ ] Number of expired listings logged
- [ ] Does not affect deleted (soft-deleted) listings
- [ ] Idempotent — running twice has no side effects

### Files
- `backend/internal/listings/cron.go` — create expiry cron
- `backend/cmd/api/main.go` — start cron goroutine

---

## TASK-032: Frontend Homepage
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 5h | **Depends on:** TASK-009, TASK-010 | **Go Status:** Not started

### Description
Create homepage with category bar, search input, featured listings grid, and active auction countdowns.

### Technical Notes
Fetch categories for the horizontal scrollbar. Fetch featured listings with `?sort=newest&per_page=12`. Fetch active auctions with countdown timers. Use React Query for data fetching. shadcn/ui Card for listing cards.

### Acceptance Criteria
- [ ] Hero section with search bar
- [ ] Horizontal scrollable category bar with emoji icons
- [ ] Featured listings grid (responsive: 1/2/3 columns)
- [ ] Active auctions section with countdown timers
- [ ] Recently added section
- [ ] Loading skeletons during data fetch
- [ ] Mobile responsive

### Files
- `frontend/src/app/(main)/page.tsx` — homepage
- `frontend/src/components/listings/listing-card.tsx`
- `frontend/src/components/listings/category-bar.tsx`
- `frontend/src/components/auctions/auction-countdown.tsx`

---

## TASK-033: Frontend Listing Detail
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-032 | **Go Status:** Not started

### Description
Create listing detail page with image gallery, info panel, map, contact seller CTA, and similar listings.

### Technical Notes
Use React Query to fetch single listing. Image gallery with swipe/arrows. Leaflet or Mapbox for location map. "Contact Seller" button initiates chat (TASK-059). Similar listings from same category.

### Acceptance Criteria
- [ ] Image gallery with navigation (swipe + arrows)
- [ ] Price, condition, location, description displayed
- [ ] Seller card with name, rating, join date
- [ ] Map showing listing location
- [ ] "Contact Seller" button
- [ ] Similar listings row (same category)
- [ ] Share button, favorite toggle
- [ ] SEO: dynamic meta tags from listing data

### Files
- `frontend/src/app/(main)/listings/[id]/page.tsx`
- `frontend/src/components/listings/image-gallery.tsx`
- `frontend/src/components/listings/seller-card.tsx`

---

## TASK-034: Frontend Create Listing Wizard
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 5h | **Depends on:** TASK-032 | **Go Status:** Not started

### Description
3-step wizard for creating a listing with Zod validation per step.

### Technical Notes
Step 1: Category + Title + Description. Step 2: Price + Condition + Location (with map picker). Step 3: Photos (drag-and-drop) + Review summary. Use Zustand for wizard state persistence across steps. Zod schema per step.

### Acceptance Criteria
- [ ] 3-step wizard with progress indicator
- [ ] Step 1: category picker, title (5-200), description (10+)
- [ ] Step 2: price, price type, condition, country, city, map location picker
- [ ] Step 3: drag-and-drop image upload (up to 10), review all data
- [ ] Zod validation errors shown inline per field
- [ ] Back/Next navigation preserves state
- [ ] Submit creates listing via API

### Files
- `frontend/src/app/(main)/listings/create/page.tsx`
- `frontend/src/components/listings/create-wizard/step-1.tsx`
- `frontend/src/components/listings/create-wizard/step-2.tsx`
- `frontend/src/components/listings/create-wizard/step-3.tsx`

---

## TASK-035: Frontend Search Results
**Epic:** Categories & Listings | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-032 | **Go Status:** Not started

### Description
Search results page with filters sidebar, listing grid, sort dropdown, and pagination.

### Technical Notes
URL query params drive filters (synced with API params). Sidebar: category, price range slider, condition checkboxes, type. Sort: newest, price asc/desc, popular. Pagination with page numbers.

### Acceptance Criteria
- [ ] Filter sidebar with category, price range, condition, type
- [ ] URL params sync with filter state (bookmarkable)
- [ ] Grid view of listing cards
- [ ] Sort dropdown
- [ ] Pagination controls with page numbers
- [ ] "No results found" empty state
- [ ] Filters update results in real-time (debounced)

### Files
- `frontend/src/app/(main)/search/page.tsx`
- `frontend/src/components/search/filter-sidebar.tsx`
- `frontend/src/components/search/sort-dropdown.tsx`

---

## EPIC 4 — Auctions (TASK-036 → TASK-047)

---

## TASK-036: POST /auctions — Create with Validation
**Epic:** Auctions | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-025 | **Go Status:** Done

### Description
Verify auction creation validates time range (max 720 hours = 30 days) and links to existing listing.

### Technical Notes
Already implemented in `auctions/handler.go:Create()` with `DurationHrs` binding `min=1,max=720`. Verify ownership of the linked listing. Verify no existing active auction on same listing (unique index).

### Acceptance Criteria
- [ ] Duration validated: 1-720 hours
- [ ] Linked listing must exist and belong to the seller
- [ ] No duplicate active auction on same listing (DB unique index)
- [ ] Start price, reserve price, buy-now price validated
- [ ] Returns created auction with computed `ends_at`

### Files
- `backend/internal/auctions/handler.go` — verify `Create()`

---

## TASK-037: GET /auctions — Active List with time_remaining
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-036 | **Go Status:** Partial

### Description
Enhance auction list to include computed `time_remaining` field (seconds until end).

### Technical Notes
Currently returns raw `ends_at`. Add a computed field in the response: `time_remaining = ends_at - now()` in seconds. Could be done in Go code after query, or as a DB computed column.

### Acceptance Criteria
- [ ] Each auction in list has `time_remaining` in seconds
- [ ] Only active auctions with `ends_at > now()` returned
- [ ] Sorted by `ends_at ASC` (ending soonest first)
- [ ] Paginated with meta

### Files
- `backend/internal/auctions/handler.go` — add `time_remaining` to `List()` response

---

## TASK-038: GET /auctions/:id — Detail with Reserve Status
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-037 | **Go Status:** Partial

### Description
Enhance auction detail to include `is_reserve_met` boolean and `time_remaining` field.

### Technical Notes
`is_reserve_met = auction.CurrentBid >= *auction.ReservePrice` (if reserve exists). Already preloads top 20 bids sorted by amount DESC.

### Acceptance Criteria
- [ ] Response includes `is_reserve_met` boolean
- [ ] Response includes `time_remaining` in seconds
- [ ] Top 20 bids included, sorted by amount DESC
- [ ] Listing details preloaded (title, images)

### Files
- `backend/internal/auctions/handler.go` — enhance `Get()` response

---

## TASK-039: POST /auctions/:id/bid — SELECT FOR UPDATE TX
**Epic:** Auctions | **Type:** bug | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-036 | **Go Status:** Partial

### Description
Fix race condition in PlaceBid by adding `SELECT FOR UPDATE` pessimistic locking inside the transaction.

### Technical Notes
Current code reads auction outside TX then updates inside TX — classic TOCTOU race. Fix: move the entire read-check-update into a single TX with `tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&auction, ...)`. Validate `amount > current_bid`, block self-bid, check auction not ended.

### Acceptance Criteria
- [ ] Auction row locked with `SELECT FOR UPDATE` during bid
- [ ] Concurrent bids serialize correctly (only highest wins)
- [ ] Self-bid rejected with 400
- [ ] Bid on ended auction rejected with 400
- [ ] Bid below current_bid rejected with 400 and message showing minimum
- [ ] Redis Pub/Sub broadcast after successful bid

### Files
- `backend/internal/auctions/handler.go` — rewrite `PlaceBid()` with locking

---

## TASK-040: Auto-Bid Proxy
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-039 | **Go Status:** Not started

### Description
After a new bid, check other bidders' auto-bid max amounts and automatically outbid incrementally.

### Technical Notes
PHP had this as a core feature. After each bid: query all Bids where `is_auto = true AND max_amount > current_bid AND user_id != current_bidder`. Find the highest `max_amount` among them. Place an automatic bid at `current_bid + increment` on their behalf. The increment should be configurable (default: 1% of current bid or $1, whichever is higher).

### Acceptance Criteria
- [ ] Auto-bid triggers after any manual bid
- [ ] Finds highest competing auto-bid max_amount
- [ ] Places incremental bid on behalf of auto-bidder
- [ ] Auto-bid marked with `is_auto = true`
- [ ] Stops when max_amount reached
- [ ] Broadcasts auto-bid via WebSocket
- [ ] Does not create infinite self-bidding loops

### Files
- `backend/internal/auctions/handler.go` — add auto-bid logic after PlaceBid
- `backend/internal/auctions/autobid.go` — extract auto-bid logic

---

## TASK-041: Anti-Sniping Extension
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-039 | **Go Status:** Not started

### Description
If a bid is placed within 5 minutes of auction end, extend `ends_at` by 5 minutes. Max 3 extensions.

### Technical Notes
PHP had `auction_extension_check()`. Track extension count (add `extension_count int` to Auction model, default 0). After bid: if `ends_at - now() < 5min && extension_count < 3`, update `ends_at += 5min` and increment `extension_count`. Broadcast updated `ends_at` via WebSocket.

### Acceptance Criteria
- [ ] Bid within 5 min of end extends by 5 min
- [ ] Maximum 3 extensions (15 min total)
- [ ] `extension_count` field tracks extensions
- [ ] WebSocket broadcasts updated `ends_at`
- [ ] Extensions visible in auction detail response

### Files
- `backend/internal/auctions/model.go` — add `ExtensionCount` field
- `backend/internal/auctions/handler.go` — add anti-sniping logic in PlaceBid

---

## TASK-042: Auction WebSocket Hub — Broadcast Format
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-039 | **Go Status:** Partial

### Description
Standardize WS broadcast format to `{type, current_bid, bid_count, ends_at}` on every bid.

### Technical Notes
Currently broadcasts raw string via Redis Pub/Sub: `{"bid": X, "user": "Y"}`. Standardize to `{type: "bid_update", current_bid, bid_count, ends_at, bidder_name}`. Add integration between HTTP handler broadcast and WS Hub (currently disconnected — Hub only receives from WS clients, not from HTTP handlers).

### Acceptance Criteria
- [ ] Bid broadcasts include `{type, current_bid, bid_count, ends_at}`
- [ ] HTTP handler PlaceBid triggers WS Hub broadcast (not just Redis)
- [ ] Auction end broadcasts `{type: "auction_ended", winner_id}`
- [ ] All connected WS clients receive updates within 100ms

### Files
- `backend/internal/auctions/websocket.go` — standardize broadcast format
- `backend/internal/auctions/handler.go` — integrate Hub broadcast into PlaceBid

---

## TASK-043: GET /ws/auctions/:id — WebSocket Upgrade
**Epic:** Auctions | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-042 | **Go Status:** Done

### Description
Verify WebSocket upgrade endpoint works. Already implemented with `ServeWS` in `main.go`.

### Technical Notes
Route: `r.GET("/ws/auctions/:id", ...)` in `main.go`. Uses Gorilla WebSocket upgrader with `CheckOrigin: true` (should be restricted in production). Verify client lifecycle: register → readPump + writePump → unregister on disconnect.

### Acceptance Criteria
- [ ] WS connection upgrades successfully
- [ ] Client registered in correct auction room
- [ ] Client removed on disconnect
- [ ] Multiple clients in same room all receive broadcasts
- [ ] Production: restrict `CheckOrigin` to `FRONTEND_URL`

### Files
- `backend/internal/auctions/websocket.go` — verify and restrict CheckOrigin

---

## TASK-044: Auction End Cron
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-041 | **Go Status:** Not started

### Description
Create a cron job that runs every minute to end expired auctions, set winner_id, and notify participants.

### Technical Notes
Query: `SELECT * FROM auctions WHERE status = 'active' AND ends_at <= NOW()`. For each: set `status = 'ended'`, determine winner (highest bid), set `winner_id`. If reserve price not met, set `status = 'ended'` with no winner. Broadcast `{type: "auction_ended"}` via WS. Future: send email notifications.

### Acceptance Criteria
- [ ] Cron runs every 60 seconds
- [ ] Expired auctions have status set to `ended`
- [ ] Winner determined by highest bid amount
- [ ] If reserve not met, no winner set
- [ ] WS broadcast `auction_ended` to all connected clients
- [ ] Log auction end events with auction ID and winner info

### Files
- `backend/internal/auctions/cron.go` — create auction end cron
- `backend/cmd/api/main.go` — start cron goroutine

---

## TASK-045: Frontend Auctions List
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-032 | **Go Status:** Not started

### Description
Create auctions listing page with cards showing live countdown timers, current bid, and bid count.

### Technical Notes
Fetch from `GET /api/v1/auctions`. Each card shows: title, current bid, bid count, countdown timer. Timer updates in real-time using `setInterval`. Clicking card navigates to auction detail page.

### Acceptance Criteria
- [ ] Grid of auction cards
- [ ] Live countdown timer on each card (updates every second)
- [ ] Current bid and bid count displayed
- [ ] Cards sorted by ending soonest
- [ ] "Ending Soon" badge for auctions ending in < 1 hour
- [ ] Mobile responsive

### Files
- `frontend/src/app/(main)/auctions/page.tsx`
- `frontend/src/components/auctions/auction-card.tsx`
- `frontend/src/components/auctions/countdown-timer.tsx`

---

## TASK-046: Frontend Auction Detail — Live
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 5h | **Depends on:** TASK-045 | **Go Status:** Not started

### Description
Create auction detail page with live bid ticker, countdown, bid history, and place bid modal.

### Technical Notes
WebSocket connection for live updates. Bid ticker shows bids as they come in. Place bid modal with amount input and auto-bid option (max amount). Bid history table with timestamps. Countdown timer with anti-sniping extension indicator.

### Acceptance Criteria
- [ ] Live countdown timer (large, prominent)
- [ ] Current bid display updates in real-time via WebSocket
- [ ] Bid history table with bidder name, amount, time
- [ ] "Place Bid" button opens modal
- [ ] Bid modal: amount input (min = current + increment), auto-bid toggle with max
- [ ] Toast notification on outbid
- [ ] Anti-sniping indicator (countdown extension notification)
- [ ] "Auction Ended" state with winner display

### Files
- `frontend/src/app/(main)/auctions/[id]/page.tsx`
- `frontend/src/components/auctions/bid-ticker.tsx`
- `frontend/src/components/auctions/place-bid-modal.tsx`
- `frontend/src/components/auctions/bid-history.tsx`

---

## TASK-047: Frontend useAuctionWebSocket Hook
**Epic:** Auctions | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-046 | **Go Status:** Not started

### Description
Create custom React hook for auction WebSocket connection with auto-reconnect.

### Technical Notes
Connect to `ws://localhost:8080/ws/auctions/{id}`. Parse incoming messages: `bid_update` → update bid state, `auction_ended` → update status. Auto-reconnect on disconnect with exponential backoff (1s, 2s, 4s, max 30s). Clean up on component unmount.

### Acceptance Criteria
- [ ] Hook connects to auction WS on mount
- [ ] Parses `bid_update` messages and updates auction state
- [ ] Parses `auction_ended` messages
- [ ] Auto-reconnects on disconnect with backoff
- [ ] Cleans up connection on unmount
- [ ] Connection status exposed (connecting/connected/disconnected)

### Files
- `frontend/src/hooks/use-auction-websocket.ts`

---

## EPIC 5 — Chat (TASK-048 → TASK-059)

---

## TASK-048: POST /chat/conversations — Create or Return Existing
**Epic:** Chat | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-004 | **Go Status:** Done

### Description
Verify conversation creation logic: finds existing conversation between two users, or creates new one linked to listing.

### Technical Notes
Already implemented in `chat/handler.go:CreateOrGetConversation()`. Uses subquery to find common conversations. Verify the deduplicate logic works when listing_id is null (general chat).

### Acceptance Criteria
- [ ] Existing conversation returned if one exists between the pair
- [ ] New conversation created with both members if none exists
- [ ] `listing_id` optionally linked
- [ ] Members preloaded in response
- [ ] Cannot create conversation with yourself

### Files
- `backend/internal/chat/handler.go` — verify `CreateOrGetConversation()`

---

## TASK-049: GET /chat/conversations — Ordered with Unread
**Epic:** Chat | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-048 | **Go Status:** Partial

### Description
Verify conversation list is ordered by last_msg_at DESC and includes unread count per conversation.

### Technical Notes
Current implementation queries ConversationMember first, then fetches Conversations. The `unread_count` on ConversationMember tracks unreads per user. Verify ordering handles NULL `last_msg_at` (conversations with no messages).

### Acceptance Criteria
- [ ] Conversations ordered by `last_msg_at DESC NULLS LAST`
- [ ] Each conversation includes `unread_count` for requesting user
- [ ] Members preloaded (to show other user's name/avatar)
- [ ] Empty conversations (no messages yet) appear at end

### Files
- `backend/internal/chat/handler.go` — verify `GetConversations()`

---

## TASK-050: GET /chat/conversations/:id/messages — Paginated
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-049 | **Go Status:** Partial

### Description
Add cursor-based pagination to messages endpoint. Currently returns last 100 messages.

### Technical Notes
Add `?before={message_id}` cursor for loading older messages. Return in ASC order (oldest first within page). Limit 50 per page. Membership verification already exists.

### Acceptance Criteria
- [ ] Returns messages in chronological order (ASC)
- [ ] `?before={id}` returns older messages (cursor pagination)
- [ ] Max 50 messages per request
- [ ] Membership verified — 403 for non-members
- [ ] Read receipts updated on fetch (unread_count reset)
- [ ] Response includes `has_more` boolean

### Files
- `backend/internal/chat/handler.go` — enhance `GetMessages()` with pagination

---

## TASK-051: POST /chat/conversations/:id/messages — Send
**Epic:** Chat | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-050 | **Go Status:** Done

### Description
Verify message sending updates `last_msg_at` and increments `unread_count` for other members.

### Technical Notes
Already implemented in `chat/handler.go:SendMessage()`. Verifies membership, creates message, updates conversation `last_msg_at`, increments other members' `unread_count`. Verify message `type` field defaults to "text".

### Acceptance Criteria
- [ ] Message saved with correct `sender_id` and `conversation_id`
- [ ] `last_msg_at` updated on conversation
- [ ] Other members' `unread_count` incremented
- [ ] Non-member gets 403
- [ ] Content required, not empty

### Files
- `backend/internal/chat/handler.go` — verify `SendMessage()`

---

## TASK-052: Chat WS Hub — Room Broadcast
**Epic:** Chat | **Type:** bug | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-051 | **Go Status:** Partial

### Description
Fix Chat WS Hub: route param mismatch and duplicate Hub creation. Messages sent via REST must reach WS clients.

### Technical Notes
**Bug 1:** `chat/websocket.go:ServeWS()` reads `c.Param("conversationId")` but route defines `:id`. Fix: change to `c.Param("id")`. **Bug 2:** `main.go` creates chatHub and `chat/routes.go` creates another Hub. Consolidate. **Bug 3:** REST `SendMessage()` doesn't broadcast to WS Hub — add Hub broadcast after message creation.

### Acceptance Criteria
- [ ] WS connects to correct conversation room
- [ ] Single Hub instance (not duplicated)
- [ ] REST message send triggers WS broadcast to room
- [ ] WS message includes `{type, content, sender_id, created_at}`
- [ ] Only connected members of conversation receive message

### Files
- `backend/internal/chat/websocket.go` — fix `c.Param("id")`
- `backend/internal/chat/routes.go` — receive Hub from outside or consolidate
- `backend/internal/chat/handler.go` — add Hub broadcast in SendMessage
- `backend/cmd/api/main.go` — fix Hub creation

---

## TASK-053: GET /chat/conversations/:id/ws — WS Upgrade with JWT
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-052 | **Go Status:** Partial

### Description
Verify WS upgrade for chat with JWT authentication from query param (since WS doesn't support headers).

### Technical Notes
WebSocket connections can't send Authorization headers. Accept token as query param: `?token=xxx`. Parse and validate JWT in ServeWS before upgrading. Verify user is a member of the conversation.

### Acceptance Criteria
- [ ] WS accepts `?token=xxx` query param
- [ ] JWT validated before WebSocket upgrade
- [ ] Non-member gets 403 (connection refused)
- [ ] Invalid/expired token gets 401
- [ ] User ID extracted and stored on WS client struct

### Files
- `backend/internal/chat/websocket.go` — add JWT validation from query param

---

## TASK-054: Read Receipts
**Epic:** Chat | **Type:** chore | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-050 | **Go Status:** Done

### Description
Verify read receipts work: fetching messages resets `unread_count` for requesting user.

### Technical Notes
Already implemented as async update in `GetMessages()`: `go h.db.Model(&ConversationMember{}).Where(...).Updates({"unread_count": 0, "last_read_at": now})`.

### Acceptance Criteria
- [ ] GET messages resets unread_count to 0 for requesting user
- [ ] `last_read_at` updated to current time
- [ ] Async operation doesn't block response
- [ ] Other members' unread_count unchanged

### Files
- `backend/internal/chat/handler.go` — verify read receipt logic

---

## TASK-055: GET /chat/unread — Total Unread Count
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 1h | **Depends on:** TASK-054 | **Go Status:** Not started

### Description
Add endpoint to get total unread message count across all conversations for the current user.

### Technical Notes
Query: `SELECT SUM(unread_count) FROM conversation_members WHERE user_id = ?`. Return single integer. Used for navbar badge count.

### Acceptance Criteria
- [ ] Returns `{unread_total: N}` where N is sum of all unread_count
- [ ] Returns 0 if no unreads (not null)
- [ ] Auth required

### Files
- `backend/internal/chat/handler.go` — add `GetUnreadCount` method
- `backend/internal/chat/routes.go` — add `GET /chat/unread`

---

## TASK-056: Frontend Chat List
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-032 | **Go Status:** Not started

### Description
Create chat list page showing conversations with avatar, name, last message preview, and unread badge.

### Technical Notes
Fetch from `GET /api/v1/chat/conversations`. Show other user's avatar, name. Last message preview (truncated). Unread count as badge. Click to open conversation thread.

### Acceptance Criteria
- [ ] List of conversations with other user's info
- [ ] Last message preview (truncated to ~50 chars)
- [ ] Unread badge (number) on conversations with unreads
- [ ] Sorted by most recent message
- [ ] Click navigates to message thread
- [ ] Empty state: "No conversations yet"

### Files
- `frontend/src/app/(main)/chat/page.tsx`
- `frontend/src/components/chat/conversation-list-item.tsx`

---

## TASK-057: Frontend Message Thread
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 5h | **Depends on:** TASK-056 | **Go Status:** Not started

### Description
Create message thread page with bubbles, timestamps, read ticks, and auto-scroll to bottom.

### Technical Notes
Message bubbles: user's on right (blue), other's on left (gray). Timestamps grouped by day. Double checkmark (✓✓) for read messages. Auto-scroll to newest message. "Load older" button for pagination. Compose bar at bottom with text input and send button.

### Acceptance Criteria
- [ ] Message bubbles (own = right, other = left)
- [ ] Grouped timestamps ("Today", "Yesterday", date)
- [ ] Read receipts (✓ sent, ✓✓ read)
- [ ] Auto-scroll to newest message on load and new message
- [ ] "Load older messages" at top for pagination
- [ ] Compose bar with text input + send button
- [ ] Real-time updates via WebSocket

### Files
- `frontend/src/app/(main)/chat/[id]/page.tsx`
- `frontend/src/components/chat/message-bubble.tsx`
- `frontend/src/components/chat/compose-bar.tsx`

---

## TASK-058: Frontend useChatWebSocket Hook
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-057 | **Go Status:** Not started

### Description
Create custom React hook for chat WebSocket — send/receive messages, update message list, auto-reconnect.

### Technical Notes
Connect to `ws://localhost:8080/api/v1/chat/conversations/{id}/ws?token=xxx`. Send messages as JSON. Parse incoming messages and append to message list. Auto-reconnect with exponential backoff. Clean up on unmount.

### Acceptance Criteria
- [ ] Connects to chat WS with JWT token
- [ ] Receives new messages and updates state
- [ ] Sends messages via WS
- [ ] Auto-reconnects on disconnect
- [ ] Connection status exposed
- [ ] Clean disconnect on unmount

### Files
- `frontend/src/hooks/use-chat-websocket.ts`

---

## TASK-059: Frontend Contact Seller Button
**Epic:** Chat | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-056 | **Go Status:** Not started

### Description
Add "Contact Seller" button on listing detail that creates/gets conversation and redirects to chat.

### Technical Notes
On click: call `POST /chat/conversations` with `{other_user_id: seller_id, listing_id}`. On response: redirect to `/chat/{conversation_id}`. If user not logged in, redirect to login first.

### Acceptance Criteria
- [ ] Button visible on listing detail page
- [ ] Creates new conversation if needed, reuses existing
- [ ] Redirects to chat thread after creation
- [ ] Unauthenticated users redirected to login first
- [ ] Loading state during API call

### Files
- `frontend/src/components/listings/contact-seller-button.tsx`

---

## EPIC 6 — Payments (TASK-060 → TASK-067)

---

## TASK-060: Stripe Webhook Handler
**Epic:** Payments | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-004 | **Go Status:** Not started

### Description
Implement Stripe webhook endpoint with signature verification and idempotent event processing.

### Technical Notes
Create `POST /payments/webhook`. Verify signature with `stripe.ConstructEvent(body, sig, endpointSecret)`. Handle `payment_intent.succeeded`, `payment_intent.failed`, `checkout.session.completed`. Use idempotency: store event ID in Redis with 24h TTL to prevent duplicate processing.

### Acceptance Criteria
- [ ] Webhook signature verification
- [ ] Handles `payment_intent.succeeded` → update payment record
- [ ] Handles `payment_intent.failed` → log failure
- [ ] Idempotent event processing (duplicate webhook ignored)
- [ ] Returns 200 immediately to Stripe

### Files
- `backend/internal/payments/handler.go` — add `HandleWebhook` method
- `backend/internal/payments/routes.go` — add `POST /payments/webhook` (no auth)

---

## TASK-061: Payment Model & Records
**Epic:** Payments | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-060 | **Go Status:** Not started

### Description
Create Payment model to track all payment records with status, amount, Stripe ID, and purpose.

### Technical Notes
Create `internal/payments/model.go` with Payment struct: ID, UserID, StripePaymentIntentID, Amount, Currency, Status (pending/succeeded/failed), Purpose (featured_listing/deposit/subscription), ListingID (optional), metadata JSON, timestamps.

### Acceptance Criteria
- [ ] Payment model with all fields
- [ ] AutoMigrate in database.go
- [ ] Payment created on PaymentIntent creation
- [ ] Payment updated on webhook event
- [ ] GET /payments/history endpoint for user's payments

### Files
- `backend/internal/payments/model.go` — create Payment model
- `backend/internal/payments/handler.go` — add `GetHistory` method
- `backend/pkg/database/database.go` — add to AutoMigrate

---

## TASK-062: Featured Listing Payment Flow
**Epic:** Payments | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-061 | **Go Status:** Not started

### Description
Complete flow: user pays → webhook fires → listing marked as featured for 7d or 30d.

### Technical Notes
Tiers: 7d ($5), 30d ($15). When `payment_intent.succeeded` fires with `purpose=featured_listing`: set `listing.is_featured = true`, `listing.featured_until = now + duration`. Create cron to unfeature expired listings.

### Acceptance Criteria
- [ ] POST /payments/featured with listing_id and tier (7d/30d)
- [ ] Creates PaymentIntent with correct amount
- [ ] Webhook success → sets `is_featured = true` and `featured_until`
- [ ] Cron unfeatures expired listings daily
- [ ] Already-featured listings extend duration

### Files
- `backend/internal/payments/handler.go` — add `CreateFeaturedPayment`
- `backend/internal/listings/model.go` — add `FeaturedUntil` field
- `backend/internal/listings/cron.go` — add unfeature cron

---

## TASK-063: Frontend Payment Flow
**Epic:** Payments | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-062 | **Go Status:** Not started

### Description
Stripe Elements integration for featured listing payment on the frontend.

### Technical Notes
Use `@stripe/react-stripe-js` and `@stripe/stripe-js`. Create payment modal on listing detail: select tier, enter card via Stripe Elements, confirm payment. On success: refresh listing data to show featured badge.

### Acceptance Criteria
- [ ] Payment modal with tier selection (7d/$5, 30d/$15)
- [ ] Stripe Elements card input
- [ ] Payment processing with loading state
- [ ] Success → show featured badge on listing
- [ ] Error handling with user-friendly messages
- [ ] Payment confirmation screen

### Files
- `frontend/src/components/payments/featured-payment-modal.tsx`
- `frontend/src/lib/stripe.ts` — Stripe client setup

---

## TASK-064 – TASK-067: Additional payment tasks (deposit, refund, subscription flows)
**Epic:** Payments | **Type:** feature | **Priority:** P1

These tasks cover additional payment workflows that mirror the PHP platform's payment gateway capabilities:
- **TASK-064:** User deposit/wallet balance system (P1, 4h)
- **TASK-065:** Payment refund handling via Stripe (P1, 3h)
- **TASK-066:** Subscription tiers for premium sellers (P1, 5h)
- **TASK-067:** Frontend payment history page (P0, 3h)

---

## EPIC 7 — Reviews & Admin (TASK-068 → TASK-078)

---

## TASK-068: Review Model & POST /users/:id/reviews
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-017 | **Go Status:** Not started

### Description
Create Review model and endpoint for leaving reviews. Rating 1-5, one review per user pair per listing.

### Technical Notes
Create `internal/reviews/model.go`: Review{ID, ReviewerID, RevieweeID, ListingID, Rating(1-5), Comment, CreatedAt}. Unique constraint on (reviewer_id, reviewee_id, listing_id). After review creation, update user's `rating` and `review_count` using AVG query.

### Acceptance Criteria
- [ ] Review model with all fields and unique constraint
- [ ] POST /users/:id/reviews creates review
- [ ] Self-review returns 400
- [ ] Duplicate review returns 409
- [ ] User rating and review_count updated atomically
- [ ] Rating validated: 1-5

### Files
- `backend/internal/reviews/model.go`
- `backend/internal/reviews/handler.go`
- `backend/internal/reviews/routes.go`

---

## TASK-069: GET /users/:id/reviews — Paginated
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-068

### Description
Paginated review list for a user, including reviewer name and avatar. Sorted by newest first.

### Acceptance Criteria
- [ ] Paginated with page/per_page
- [ ] Includes reviewer name and avatar_url
- [ ] Sorted by `created_at DESC`
- [ ] Average rating in meta

### Files
- `backend/internal/reviews/handler.go` — add `ListReviews` method

---

## TASK-070: Report Model & Endpoints
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-004 | **Go Status:** Not started

### Description
Create Report model for flagging listings and users. Reports feed into admin review queue.

### Technical Notes
Model: Report{ID, ReporterID, TargetType(listing/user), TargetID, Reason(enum), Description, Status(pending/reviewed/dismissed), ReviewedBy, ReviewedAt, CreatedAt}.

### Acceptance Criteria
- [ ] POST /listings/:id/report — report listing
- [ ] POST /users/:id/report — report user
- [ ] Reason enum: spam, scam, inappropriate, counterfeit, other
- [ ] One active report per reporter per target
- [ ] Report status defaults to `pending`

### Files
- `backend/internal/reports/model.go`
- `backend/internal/reports/handler.go`
- `backend/internal/reports/routes.go`

---

## TASK-071: Admin Middleware — JWT Role Check
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-004 | **Go Status:** Partial

### Description
Fix AdminOnly middleware to properly check user role from JWT claims.

### Technical Notes
After TASK-004 adds `Role` to JWT claims, update `AdminOnly()` to read `user_role` from gin context and check against "admin". Return 403 if not admin.

### Acceptance Criteria
- [ ] AdminOnly reads `user_role` from context (set by Auth middleware)
- [ ] Non-admin users get 403
- [ ] Admin users pass through

### Files
- `backend/pkg/middleware/auth.go` — fix `AdminOnly()`

---

## TASK-072: Admin Stats Dashboard API
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-071

### Description
GET /admin/stats endpoint returning platform totals: users, listings, auctions, revenue, new today.

### Acceptance Criteria
- [ ] Total users, listings, auctions, active auctions
- [ ] Revenue total from succeeded payments
- [ ] New users/listings/auctions today
- [ ] Response time < 100ms (use COUNT queries)

### Files
- `backend/internal/admin/handler.go`
- `backend/internal/admin/routes.go`

---

## TASK-073: Admin User Management API
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-071

### Description
Admin endpoints for listing, searching, blocking, and unblocking users.

### Acceptance Criteria
- [ ] GET /admin/users — search by name/email, filter by is_blocked
- [ ] POST /admin/users/:id/block — set is_blocked = true
- [ ] POST /admin/users/:id/unblock — set is_blocked = false
- [ ] Blocking user with active auctions → end all auctions

### Files
- `backend/internal/admin/handler.go` — add user management methods

---

## TASK-074: Admin Listing Management API
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-071

### Description
Admin endpoints for listing moderation: approve, reject, search all listings.

### Acceptance Criteria
- [ ] GET /admin/listings — all statuses, search, pagination
- [ ] POST /admin/listings/:id/approve
- [ ] POST /admin/listings/:id/reject with reason

### Files
- `backend/internal/admin/handler.go` — add listing management methods

---

## TASK-075: Admin Report Queue API
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-070

### Description
Admin endpoints for reviewing and resolving reports.

### Acceptance Criteria
- [ ] GET /admin/reports — filter by type, status, paginated
- [ ] POST /admin/reports/:id/resolve with action (dismiss/warn/block)
- [ ] Reporter and target details preloaded

### Files
- `backend/internal/admin/handler.go` — add report management methods

---

## TASK-076: Frontend Admin Dashboard
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 5h | **Depends on:** TASK-072

### Description
Admin dashboard with stats cards, user management table, listing queue, and report queue.

### Acceptance Criteria
- [ ] Stats cards (total users, listings, auctions, revenue)
- [ ] User management data table with search and block/unblock
- [ ] Listing moderation queue
- [ ] Report queue with resolve actions
- [ ] Admin-only route protection

### Files
- `frontend/src/app/admin/page.tsx`
- `frontend/src/app/admin/users/page.tsx`
- `frontend/src/app/admin/listings/page.tsx`
- `frontend/src/app/admin/reports/page.tsx`

---

## TASK-077: Frontend Reviews Display
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-068

### Description
Reviews tab on user profile pages with star ratings, reviewer info, and write review form.

### Acceptance Criteria
- [ ] Review list with star ratings and comments
- [ ] Average rating displayed prominently
- [ ] Write review form for authenticated users
- [ ] Review submitted via API

### Files
- `frontend/src/components/reviews/review-list.tsx`
- `frontend/src/components/reviews/review-form.tsx`
- `frontend/src/components/reviews/star-rating.tsx`

---

## TASK-078: Frontend Report Dialog
**Epic:** Reviews & Admin | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-070

### Description
Report dialog on listings and user profiles with reason dropdown and description textarea.

### Acceptance Criteria
- [ ] Report button on listing and user profile pages
- [ ] Dialog with reason dropdown and description
- [ ] Confirmation toast on submission
- [ ] Disabled if already reported

### Files
- `frontend/src/components/reports/report-dialog.tsx`

---

## EPIC 8 — Search & AI (TASK-079 → TASK-085)

---

## TASK-079: Meilisearch Integration
**Epic:** Search & AI | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-023 | **Go Status:** Not started

### Description
Sync listings to Meilisearch for fast full-text search, replacing ILIKE queries.

### Technical Notes
Use `meilisearch-go` SDK. Create `listings` index with searchable attributes: title, description, category. Filterable: category_id, country, city, price, condition, status. Sortable: price, created_at. Sync on create/update/delete via GORM hooks or explicit calls.

### Acceptance Criteria
- [ ] Meilisearch client initialized on startup
- [ ] Listings synced to Meilisearch on create/update/delete
- [ ] Search endpoint uses Meilisearch when available, falls back to ILIKE
- [ ] Faceted search with counts per category
- [ ] Search results < 50ms

### Files
- `backend/pkg/search/meilisearch.go` — create client wrapper
- `backend/internal/listings/handler.go` — integrate Meilisearch in `List()`

---

## TASK-080: pgvector Extension & Embedding Model
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 4h | **Depends on:** TASK-079

### Description
Add pgvector extension and embedding column to listings for semantic search.

### Acceptance Criteria
- [ ] pgvector extension enabled in PostgreSQL
- [ ] `embedding vector(1536)` column on listings table
- [ ] Migration script to add column
- [ ] OpenAI embedding API integration for generating vectors

### Files
- `backend/pkg/ai/embeddings.go`
- `backend/internal/listings/model.go` — add Embedding field

---

## TASK-081: POST /ai/categorize — Auto-Categorization
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 3h | **Depends on:** TASK-021

### Description
GPT-4o-powered category suggestion based on listing title and description.

### Acceptance Criteria
- [ ] Accepts title and description
- [ ] Returns top 3 category suggestions with confidence scores
- [ ] Uses category list from DB as context
- [ ] Response time < 3s

### Files
- `backend/internal/ai/handler.go` — add `Categorize` method
- `backend/internal/ai/routes.go`

---

## TASK-082: GET /ai/price-suggest — Price Suggestion
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 3h | **Depends on:** TASK-081

### Description
Suggest a price based on similar listings in same category, condition, and location.

### Acceptance Criteria
- [ ] Queries 20 similar listings
- [ ] Returns min, max, suggested price
- [ ] Factors in condition and location

### Files
- `backend/internal/ai/handler.go` — add `PriceSuggest` method

---

## TASK-083: POST /ai/moderate — Content Moderation
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 3h | **Depends on:** TASK-081

### Description
GPT-4o moderation check before listing goes active. Auto-flag inappropriate content.

### Acceptance Criteria
- [ ] Checks title, description, and images
- [ ] Returns pass/flag/reject with reason
- [ ] Flagged listings require admin review
- [ ] Rejected listings cannot be published

### Files
- `backend/internal/ai/handler.go` — add `Moderate` method

---

## TASK-084: POST /ai/search — Semantic Vector Search
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 4h | **Depends on:** TASK-080

### Description
Natural language search using pgvector cosine similarity.

### Acceptance Criteria
- [ ] Embed query string using OpenAI API
- [ ] Cosine similarity search against listing embeddings
- [ ] Return top 20 results with similarity score
- [ ] Combine with Meilisearch results for hybrid search

### Files
- `backend/internal/ai/handler.go` — add `SemanticSearch` method

---

## TASK-085: Frontend AI Features
**Epic:** Search & AI | **Type:** feature | **Priority:** P2 | **Estimate:** 4h | **Depends on:** TASK-081

### Description
Integrate AI features into create listing wizard and search.

### Acceptance Criteria
- [ ] Category suggestion chips in Step 1 of create wizard
- [ ] Price suggestion tooltip in Step 2
- [ ] "Search with AI" toggle in search bar
- [ ] Loading states for AI operations

### Files
- `frontend/src/components/ai/category-suggest.tsx`
- `frontend/src/components/ai/price-suggest.tsx`

---

## EPIC 9 — DevOps & Infrastructure (TASK-086 → TASK-092)

---

## TASK-086: Structured Logging with Zap
**Epic:** DevOps | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-001 | **Go Status:** Partial

### Description
Replace fmt.Println/log calls with structured Zap logging throughout the codebase.

### Acceptance Criteria
- [ ] Global zap logger initialized in main.go (already partially done)
- [ ] All handlers use zap for error logging
- [ ] Request logging middleware with method, path, status, duration
- [ ] JSON format in production, console in development

### Files
- `backend/cmd/api/main.go` — ensure zap is used consistently
- `backend/pkg/middleware/logger.go` — create request logger middleware

---

## TASK-087: Redis Caching Layer
**Epic:** DevOps | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-003

### Description
Add Redis caching for frequently accessed data: categories (5min), popular listings (2min), user profiles (1min).

### Acceptance Criteria
- [ ] Cache wrapper with get-or-set pattern
- [ ] Categories cached for 5 minutes
- [ ] Popular/featured listings cached for 2 minutes
- [ ] Cache invalidation on data mutation
- [ ] Cache miss logged with zap

### Files
- `backend/pkg/cache/cache.go` — create cache wrapper
- `backend/internal/listings/handler.go` — add caching

---

## TASK-088: Kubernetes Manifests
**Epic:** DevOps | **Type:** feature | **Priority:** P1 | **Estimate:** 4h | **Depends on:** TASK-008

### Description
Create K8s manifests for production deployment: Deployment, Service, Ingress, HPA, ConfigMap, Secrets.

### Acceptance Criteria
- [ ] API Deployment with 3 replicas, resource limits, health probes
- [ ] Service (ClusterIP) and Ingress (with TLS)
- [ ] HPA: min 3, max 10, target CPU 70%
- [ ] ConfigMap for non-sensitive env vars
- [ ] Secret for sensitive env vars
- [ ] PostgreSQL and Redis as managed services (external)

### Files
- `k8s/deployment.yaml`
- `k8s/service.yaml`
- `k8s/ingress.yaml`
- `k8s/hpa.yaml`
- `k8s/configmap.yaml`

---

## TASK-089: Production Dockerfile
**Epic:** DevOps | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-001

### Description
Multi-stage Dockerfile for production build: build in Go image, run in distroless/static.

### Acceptance Criteria
- [ ] Stage 1: Build with `golang:1.23-alpine`
- [ ] Stage 2: Run with `gcr.io/distroless/static-debian12`
- [ ] Binary size < 20MB
- [ ] Non-root user
- [ ] Health check included

### Files
- `backend/Dockerfile` — create production Dockerfile

---

## TASK-090: GitHub Actions CD
**Epic:** DevOps | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-008, TASK-089

### Description
CD pipeline: build Docker image, push to registry, deploy to K8s on merge to main.

### Acceptance Criteria
- [ ] Triggered on merge to main
- [ ] Builds and tags Docker image
- [ ] Pushes to container registry (GHCR or ECR)
- [ ] Deploys to K8s cluster via kubectl or Helm

### Files
- `.github/workflows/cd.yml`

---

## TASK-091: Database Migrations with goose
**Epic:** DevOps | **Type:** feature | **Priority:** P1 | **Estimate:** 3h | **Depends on:** TASK-001

### Description
Replace GORM AutoMigrate with goose for production-safe migrations.

### Technical Notes
AutoMigrate is convenient for dev but dangerous in production (can't rollback, doesn't handle column renames). Use `pressly/goose` for versioned SQL migrations. Keep AutoMigrate for dev mode only.

### Acceptance Criteria
- [ ] goose installed and configured
- [ ] Initial migration from current GORM schema
- [ ] Migration runner in main.go (only in production mode)
- [ ] AutoMigrate still used when APP_ENV=development
- [ ] Rollback tested for at least one migration

### Files
- `backend/migrations/` — create migration directory
- `backend/pkg/database/database.go` — conditional migrate strategy

---

## TASK-092: Monitoring & Observability
**Epic:** DevOps | **Type:** feature | **Priority:** P1 | **Estimate:** 4h | **Depends on:** TASK-086

### Description
Add Prometheus metrics endpoint and OpenTelemetry tracing.

### Acceptance Criteria
- [ ] GET /metrics endpoint with Prometheus format
- [ ] Request count, latency histogram, error rate
- [ ] DB query duration metrics
- [ ] WebSocket connection count gauge
- [ ] Grafana dashboard template

### Files
- `backend/pkg/middleware/metrics.go`
- `backend/cmd/api/main.go` — register /metrics

---

## EPIC 10 — Polish & i18n (TASK-093 → TASK-098)

---

## TASK-093: Frontend User Profile Page
**Epic:** Polish | **Type:** feature | **Priority:** P0 | **Estimate:** 4h | **Depends on:** TASK-032

### Description
User profile page with avatar, bio, listings tab, reviews tab, and edit profile form.

### Acceptance Criteria
- [ ] Avatar display with upload option (own profile)
- [ ] Name, bio, location, join date
- [ ] Tabs: Active Listings, Reviews, About
- [ ] Edit profile form (own profile only)
- [ ] Star rating display

### Files
- `frontend/src/app/(main)/profile/[id]/page.tsx`
- `frontend/src/app/(main)/profile/edit/page.tsx`

---

## TASK-094: Frontend Responsive Navigation
**Epic:** Polish | **Type:** feature | **Priority:** P0 | **Estimate:** 3h | **Depends on:** TASK-032

### Description
Responsive navbar with logo, search, create listing CTA, chat badge, notifications dropdown, and user avatar menu.

### Acceptance Criteria
- [ ] Desktop: full navbar with all elements
- [ ] Mobile: hamburger menu + bottom tab bar
- [ ] Chat unread badge (from GET /chat/unread)
- [ ] User dropdown: My Listings, My Auctions, Profile, Settings, Logout
- [ ] Search bar with autocomplete suggestions

### Files
- `frontend/src/components/layout/navbar.tsx`
- `frontend/src/components/layout/mobile-nav.tsx`
- `frontend/src/components/layout/user-menu.tsx`

---

## TASK-095: Arabic (RTL) Support
**Epic:** Polish | **Type:** feature | **Priority:** P1 | **Estimate:** 4h | **Depends on:** TASK-094

### Description
Full Arabic localization with RTL layout, translation files, and language switcher.

### Technical Notes
Use `next-intl` or `i18next`. Category names already have `name_ar`. User model has `language` preference. RTL support via `dir="rtl"` on HTML element.

### Acceptance Criteria
- [ ] Language switcher (EN/AR) in navbar
- [ ] All static text translated
- [ ] RTL layout for Arabic
- [ ] Category names display in user's language
- [ ] Date/number formatting per locale
- [ ] Persisted in user preferences

### Files
- `frontend/src/i18n/en.json`
- `frontend/src/i18n/ar.json`
- `frontend/src/components/layout/language-switcher.tsx`

---

## TASK-096: Dark Mode
**Epic:** Polish | **Type:** feature | **Priority:** P1 | **Estimate:** 2h | **Depends on:** TASK-094

### Description
Dark mode toggle with system preference detection and Tailwind dark classes.

### Acceptance Criteria
- [ ] Toggle in navbar (sun/moon icon)
- [ ] System preference detection on first visit
- [ ] Persisted in localStorage
- [ ] All components styled for dark mode
- [ ] Smooth transition between modes

### Files
- `frontend/src/components/layout/theme-toggle.tsx`
- `frontend/src/lib/theme.ts`

---

## TASK-097: SEO & Meta Tags
**Epic:** Polish | **Type:** feature | **Priority:** P0 | **Estimate:** 2h | **Depends on:** TASK-033

### Description
Dynamic meta tags for all pages: title, description, Open Graph, Twitter Cards.

### Technical Notes
Use Next.js `generateMetadata` in page components. Listing detail: title = listing title, description = truncated description, image = cover image. Auction detail: include current bid in description.

### Acceptance Criteria
- [ ] Every page has unique title and description
- [ ] Open Graph tags for social sharing
- [ ] Twitter Card tags
- [ ] Listing images as og:image
- [ ] Structured data (JSON-LD) for listings

### Files
- `frontend/src/app/(main)/listings/[id]/page.tsx` — add generateMetadata
- `frontend/src/app/(main)/auctions/[id]/page.tsx` — add generateMetadata

---

## TASK-098: E2E Test Suite
**Epic:** Polish | **Type:** feature | **Priority:** P0 | **Estimate:** 5h | **Depends on:** TASK-089

### Description
End-to-end test suite covering critical user flows: register → create listing → search → chat → bid → payment.

### Technical Notes
Use Playwright for browser E2E tests. Test against Docker Compose environment. Seed test data before each suite. Cover both happy path and key error cases.

### Acceptance Criteria
- [ ] User registration and login flow
- [ ] Create listing with images
- [ ] Search and filter listings
- [ ] Start chat from listing detail
- [ ] Create auction and place bid
- [ ] Featured listing payment flow
- [ ] Admin login and moderation
- [ ] Tests pass in CI (GitHub Actions)
- [ ] Test coverage report generated

### Files
- `frontend/e2e/auth.spec.ts`
- `frontend/e2e/listings.spec.ts`
- `frontend/e2e/auctions.spec.ts`
- `frontend/e2e/chat.spec.ts`
- `frontend/e2e/payments.spec.ts`
- `frontend/playwright.config.ts`


  ---

  ## TASK-099: React Native Mobile App — GeoCore
  **Status:** ✅ Done
  **Type:** feature
  **Priority:** P1
  **Completed:** 2026-03-22

  ### What was built
  - React Native + Expo SDK 53 (Latest)
  - TypeScript strict mode — 0 errors
  - eBay-style UI with Walmart colors (#0071CE Blue + #FFC220 Yellow)
  - Zustand state management + JWT auth with expo-secure-store
  - EAS Build configured for APK + IPA

  ### Screens
  - HomeScreen (search + categories + live auctions + listings grid)
  - ListingDetailScreen (images, price, bid CTA, seller info)
  - AuctionsScreen (live countdown timers, real-time bidding UI)
  - SearchScreen (filters: category, price range, location, condition)
  - CreateListingScreen (image upload, category, pricing type)
  - ChatScreen (conversation threads, unread badges)
  - ProfileScreen (stats, wallet balance, menu, sign in/out)
  - LoginScreen + RegisterScreen (email + password, form validation)

  ### Components
  - ListingCard (AUCTION / BUY NOW badges, countdown overlay)
  - AuctionCard (live countdown, current bid, bidder count)
  - CountdownTimer (updates every second via useEffect interval)
  - SearchBar (full-width, keyboard-aware)
  - FloatingActionButton (yellow + sell button, bottom-right)

  ### API Integration
  - axios client pointing at https://geo-core-next.replit.app/api/v1
  - Bearer token injection via request interceptor
  - Automatic token refresh on 401 via response interceptor
  - listingsAPI, authAPI, auctionsAPI, messagesAPI, walletAPI modules

  ### Build
  - APK: `eas build --platform android --profile preview`
  - IPA: `eas build --platform ios --profile production`
  - Bundle ID: com.geocore.next (iOS + Android)

  ---

  ## TASK-100: App Store Submission
  **Status:** ❌ Not Started
  **Type:** chore
  **Priority:** P2
  **Depends on:** TASK-099

  ### Description
  Submit app to Google Play Store and Apple App Store.

  ### Requirements
  - Google Play Developer account ($25 one-time)
  - Apple Developer account ($99/year)
  - App icons (1024×1024)
  - Screenshots for all screen sizes (phone + tablet)
  - App description in EN + AR
  - Privacy policy URL

  ### Checklist
  - [ ] Generate signed keystore for Android
  - [ ] Run `eas build --platform android --profile production`
  - [ ] Upload AAB to Google Play Console
  - [ ] Run `eas build --platform ios --profile production`
  - [ ] Upload IPA to App Store Connect via Transporter
  - [ ] Submit for review
  