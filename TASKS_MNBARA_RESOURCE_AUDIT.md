# mnbara-platform Resource Audit

> Reference repo: `E:\New Computer\Development Coding\Projects\Repos\geo\mnbara-platform`
> Audited: 2026-04-05

## Summary

The abandoned mnbara-platform is a massive NestJS/Prisma microservices monorepo with **80+ services** and a React/Next.js frontend. Below is a mapping of reusable assets → geocore-next tasks.

---

## 1. Active Services (`services/`)

| Service | Stack | Key Patterns | Maps To |
|---|---|---|---|
| `crowdshipping/trips-service` | NestJS, Prisma | Trip CRUD, route management | **TASK-F02** |
| `crowdshipping/matching-service` | NestJS, Prisma | Haversine distance, match scoring (price/rating/KYC/timing/country), geofencing, AI embeddings | **TASK-F02** |
| `financial/wallet-service` | NestJS, Prisma | Multi-currency wallet, ledger, escrow, transfer, conversion | Enhance existing wallet |
| `financial/escrow-service` | NestJS, Prisma, Blockchain | Smart contract escrow, fund integration | **TASK-F09** |
| `financial/payment-service` | NestJS, Prisma | Stripe Connect, PayMob, wallet-ledger | Enhance existing payments |
| `financial/settlement-service` | NestJS, Prisma | Seller payouts, settlement batching | Future enhancement |
| `marketplace/product-service` | NestJS, Prisma | Product CRUD, categories | Already built |
| `marketplace/order-service` | NestJS, Prisma | Order lifecycle, fulfillment | Already built |
| `marketplace/cart-service` | NestJS, Prisma | Cart management | Already built |

---

## 2. Legacy Services (`archive/legacy-services/`) — 80+ services

| Service | Relevance | Maps To |
|---|---|---|
| `ai-agent-service`, `ai-chatbot-service` | AI assistant patterns | TASK-F04 ✅ Done |
| `ai-pricing-service` | Dynamic pricing ML | Future |
| `ar-preview-service`, `vr-showroom-service` | AR/VR preview | **TASK-F08** |
| `blockchain-service` | Smart contracts, Solidity | **TASK-F09** |
| `bnpl-service` | BNPL integration | TASK-F06 ✅ Done |
| `crowdship-service` | Older crowdshipping version | **TASK-F02** |
| `crypto-service` | Crypto payments | TASK-F07 ✅ Done |
| `ebay-live-service` | Live auction streaming | TASK-F01 ✅ Done |
| `fraud-detection-service` | ML fraud detection | **TASK-F05** |
| `p2p-exchange-service` | P2P currency/item exchange | **TASK-F03** |
| `plugin-system` | Plugin SDK, marketplace | **TASK-F10** |
| `recommendation-engine-service` | Recommendation ML | Future |
| `social-commerce-service` | Social features | Future |
| `voice-commerce-service` | Voice commerce | Future |

---

## 3. Frontend Components (`apps/web/src/components/`)

| Directory | Files | Key Components | Maps To |
|---|---|---|---|
| `live-streaming/` | 8 components + CSS | LiveStreamPlayer, LiveStreamAuction, StreamModeration, LiveStreamChat, LiveStreamCreator, LiveStreamDiscovery | TASK-F01 ✅ Done |
| `p2p-exchange/` | 17 components | ExchangeRequestForm, MarketplaceBrowser, MatchChat, ProofUpload, SecurityDepositCard, TrustLevelBadge, ReceiptConfirmation | **TASK-F03** |
| `plugin-marketplace/` | 7 components | PluginMarketplace, PluginCard, PluginDetails, PluginInstallModal, PluginCategories | **TASK-F10** |
| `traveler/` | — | (see pages below) | **TASK-F02** |

---

## 4. Frontend Pages (`apps/web/src/pages/traveler/`) — 15 pages

| Page | Description | Maps To |
|---|---|---|
| `TravelerDashboard.tsx` | Dashboard with active orders + trips + earnings | **TASK-F02** |
| `TripCreation.tsx` / `TripCreationPage.tsx` | Trip creation form (origin/dest/capacity/dates) | **TASK-F02** |
| `AvailableOrdersPage.tsx` | Browse delivery requests | **TASK-F02** |
| `DeliveryMatchingPage.tsx` | Matching UI for order→trip pairing | **TASK-F02** |
| `DeliveryStatusTimeline.tsx` | Delivery tracking timeline | **TASK-F02** |
| `RouteDetailsPage.tsx` / `RouteMapPage.tsx` | Route visualization | **TASK-F02** |
| `ActiveRoutesPage.tsx` | Active route management | **TASK-F02** |
| `BecomeTravelerPage.tsx` | Onboarding for travelers | **TASK-F02** |
| `TravelerOffersPage.tsx` | Offers management | **TASK-F02** |
| `TravelerProfilePage.tsx` | Traveler profile + ratings | **TASK-F02** |
| `TravelerRatingPage.tsx` | Rating system | **TASK-F02** |

---

## 5. Key Algorithms Extracted

### Match Scoring Algorithm (from matching-service)
```
Score = 100 (base)
  - min(estimatedCost/100 * 30, 30)     // price penalty
  + min(travelerRating * 10, 50)         // rating bonus
  + 20 if KYC verified                   // trust bonus
  + 15/10/5 based on departure proximity // timing bonus
  + countryCompatibilityScore - 100      // country match
Clamped to [0, 200]
```

### Haversine Distance (from matching-service)
Standard implementation for km distance between lat/lon coordinates.

### Trip→Order Matching Flow
1. Find compatible trips by country route + weight capacity
2. Score each match (price, rating, KYC, timing, country)
3. Sort by score descending
4. Request match → MATCHED status
5. Traveler accepts/rejects → ACCEPTED/PENDING
6. Capacity decremented/restored atomically in transaction

---

## 6. DB Schema Patterns (from matching-service schema.prisma)

Key tables to port:
- `match_candidates` — orderId, tripId, score, status, geospatial deviations, country fields
- `geofences` — GeoJSON polygon, priority levels, fence types
- `match_weight_configs` — configurable scoring weights
- `user_behaviors` — event tracking for AI recommendations
- `match_history` — outcome tracking for ML training

---

## Task Status After Audit

| Task | Status | Source Used |
|---|---|---|
| TASK-F01: Live Streaming | ✅ Done | `components/live-streaming/` |
| TASK-F02: Crowdshipping | ✅ Done | `services/crowdshipping/`, `pages/traveler/` |
| TASK-F03: P2P Exchange | ✅ Done | `components/p2p-exchange/`, `archive/p2p-exchange-service/` |
| TASK-F04: AI Chatbot | ✅ Done | `archive/ai-chatbot-service/` |
| TASK-F05: Fraud Detection | ✅ Done | `archive/fraud-detection-service/` |
| TASK-F06: BNPL | ✅ Done | `archive/bnpl-service/` |
| TASK-F07: Crypto Payments | ✅ Done | `archive/crypto-service/` |
| TASK-F08: AR/VR Preview | ✅ Done | `archive/ar-preview-service/`, `archive/vr-showroom-service/` |
| TASK-F09: Blockchain | ✅ Done | `archive/blockchain-service/`, `services/escrow-service/` |
| TASK-F10: Plugin Marketplace | ✅ Done | `archive/plugin-system/`, `components/plugin-marketplace/` |

---

## Implementation Summary (All F-Tasks Complete)

### Backend packages added (all build ✅):
- `internal/crowdshipping/` — Trip CRUD, delivery requests, Haversine matching (migration 015)
- `internal/p2p/` — Exchange requests, matching, chat messages (migration 016)
- `internal/fraud/` — Risk scoring engine, alerts, rules, profiles (migration 017)
- `internal/arpreview/` — 3D model management for listings (migration 018)
- `internal/blockchain/` — Escrow contracts with fund/release/dispute (migration 019)
- `internal/plugins/` — Plugin CRUD, install/uninstall, marketplace (migration 020)

### Frontend pages added (all build ✅):
- `/traveler` — Dashboard, trip creation, browse requests, request detail + matching
- `/p2p` — Marketplace, create request, request detail + chat, my requests
- `/admin/fraud` — Fraud dashboard with alerts, rules toggle, stats
- `/ar-preview` — 3D model viewer with model-viewer web component
- `/escrow` — Escrow contracts list + detail with fund/release/dispute actions
- `/plugins` — Plugin marketplace, detail, create, install/uninstall
