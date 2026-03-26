# GeoCore Next 🚀

  > Modern GCC Classifieds & Auctions Marketplace Platform
  > Built with Go + React Native + Next.js 15 — competing with OLX, Craigslist & eBay

  [![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
  [![React Native](https://img.shields.io/badge/React_Native-Expo-black?logo=expo)](https://expo.dev)
  [![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](https://nextjs.org)
  [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-blue?logo=postgresql)](https://postgresql.org)
  [![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

  ## 🏗 Architecture

  ```
  geocore-next/
  ├── backend/               # Go 1.23 REST API
  │   ├── cmd/api/           # Entry point
  │   ├── internal/
  │   │   ├── auth/          # JWT Authentication
  │   │   ├── listings/      # Classifieds (CRUD, search, favorites)
  │   │   ├── auctions/      # Real-time bidding + WebSocket
  │   │   ├── chat/          # P2P messaging + WebSocket
  │   │   ├── users/         # User profiles
  │   │   ├── payments/      # Stripe integration
  │   │   └── kyc/           # KYC identity verification (GCC compliance)
  │   └── pkg/
  │       ├── database/      # PostgreSQL + GORM + PostGIS
  │       ├── redis/         # Caching + Pub/Sub
  │       ├── middleware/    # JWT Auth + KYC middleware
  │       └── response/      # Standardized API responses
  ├── frontend/              # Next.js 15 + React 19 (Web Marketplace)
  │   └── src/
  │       ├── app/           # App Router pages
  │       ├── components/    # Reusable UI components
  │       ├── hooks/         # Custom hooks (WebSocket, etc.)
  │       ├── store/         # Zustand state management
  │       ├── lib/           # API client + utilities
  │       └── types/         # TypeScript types
  ├── mobile/                # React Native (Expo) — iOS & Android
  │   └── src/
  │       ├── screens/       # Home, Listings, Auctions, Chat, Profile
  │       ├── components/    # Mobile UI components
  │       └── store/         # Zustand + React Query
  ├── admin/                 # React + Vite + shadcn/ui (Admin Panel)
  │   └── src/
  │       ├── pages/         # Dashboard, Listings, Auctions, KYC, Users
  │       ├── components/    # saleor-dashboard inspired UI patterns
  │       └── hooks/         # Admin data hooks with mock fallback
  ├── ai-service/            # Python Flask — AI Auction Pricing (DQN-inspired)
  │   └── pricing.py         # GCC-aware bid prediction engine
  ├── infra/
  │   ├── docker/
  │   └── k8s/               # Kubernetes configs
  └── .github/workflows/     # CI/CD

  ```

  ## 🚀 Quick Start

  ### With Docker (recommended)

  ```bash
  git clone https://github.com/hossam-create/geocore-next
  cd geocore-next
  cp .env.example .env
  make dev
  ```

  Then visit:
  - Web Marketplace: http://localhost:3000
  - Admin Panel: http://localhost:3001
  - API: http://localhost:8080
  - DB Admin: http://localhost:8083

  ### Local Development

  ```bash
  # Start dependencies
  make dev-deps

  # Backend
  make run-backend

  # Frontend (new terminal)
  make install
  make run-frontend
  ```

  ## 🔑 API Endpoints

  | Method | Endpoint | Description |
  |--------|----------|-------------|
  | POST | /api/v1/auth/register | Register |
  | POST | /api/v1/auth/login | Login |
  | GET | /api/v1/listings | Browse listings |
  | POST | /api/v1/listings | Create listing |
  | GET | /api/v1/auctions | Active auctions |
  | POST | /api/v1/auctions/:id/bid | Place bid |
  | WS | /ws/auctions/:id | Live auction feed |
  | WS | /ws/chat/:conversationId | Real-time chat |
  | GET | /api/v1/kyc/admin/list | KYC submissions (admin) |
  | PUT | /api/v1/kyc/admin/:id/approve | Approve KYC |
  | POST | /api/v1/ai/predict | AI bid prediction |

  ## 🗺 Roadmap

  ### Backend & Infrastructure
  - [x] Core API (Go + Gin + GORM)
  - [x] Authentication (JWT)
  - [x] Listings CRUD + Search
  - [x] Standard Auctions + Auto-bid
  - [x] Real-time Chat (WebSocket)
  - [x] Stripe Payments
  - [x] Docker + CI/CD
  - [x] KYC Identity Verification (GCC — UAE/KSA/Kuwait)
  - [ ] AI Search (pgvector + OpenAI embeddings)
  - [ ] Image Upload (Cloudflare R2)
  - [x] Kubernetes Production Deployment

  ### Frontend & Mobile
  - [x] Mobile App (React Native / Expo) — iOS & Android
  - [x] Admin Dashboard (React + Vite + shadcn/ui)
  - [x] Web Marketplace (Next.js 15)
  - [x] Seller Dashboard (listings, orders, analytics)
  - [x] Real-time Auction Bidding UI
  - [x] In-app Chat (WebSocket)
  - [x] Wallet & Payments UI

  ### AI & ML
  - [x] AI Auction Pricing Engine (DQN-inspired, GCC currencies)
  - [ ] AI Semantic Search (pgvector + OpenAI)
  - [ ] AI Recommendation Engine

  ## 🛠 Tech Stack

  | Layer | Technology |
  |-------|-----------|
  | Backend | Go 1.23, Gin, GORM |
  | Database | PostgreSQL 16 + PostGIS + pgvector |
  | Cache | Redis 7 |
  | Web Frontend | Next.js 15, React 19, TypeScript |
  | Mobile | React Native, Expo |
  | Admin Panel | React + Vite + shadcn/ui |
  | AI Service | Python Flask (DQN-inspired pricing) |
  | State | Zustand + React Query |
  | Styling | Tailwind CSS + shadcn/ui |
  | Payments | Stripe |
  | KYC | Custom GCC compliance (UAE/KSA/Kuwait) |
  | Real-time | WebSocket (gorilla/websocket) |
  | CI/CD | GitHub Actions |
  | Infra | Docker, Kubernetes |

  ## 📄 License
  MIT
  