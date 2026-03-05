# GeoCore Next 🚀

> Modern Global Classifieds & Auctions Platform
> Built with Go + Next.js 15 — competing with OLX & Craigslist

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?logo=go)](https://go.dev)
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
│   │   └── payments/      # Stripe integration
│   └── pkg/
│       ├── database/      # PostgreSQL + GORM + PostGIS
│       ├── redis/         # Caching + Pub/Sub
│       ├── middleware/    # JWT Auth middleware
│       └── response/      # Standardized API responses
├── frontend/              # Next.js 15 + React 19
│   └── src/
│       ├── app/           # App Router pages
│       ├── components/    # Reusable UI components
│       ├── hooks/         # Custom hooks (WebSocket, etc.)
│       ├── store/         # Zustand state management
│       ├── lib/           # API client + utilities
│       └── types/         # TypeScript types
├── infra/
│   ├── docker/
│   └── k8s/               # Kubernetes configs
└── .github/workflows/     # CI/CD

```

## 🚀 Quick Start

### With Docker (recommended)

```bash
git clone https://github.com/your-org/geocore-next
cd geocore-next
cp .env.example .env
make dev
```

Then visit:
- Frontend: http://localhost:3000
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

## 🗺 Roadmap

- [x] Core API (Go + Gin + GORM)
- [x] Authentication (JWT)
- [x] Listings CRUD + Search
- [x] Standard Auctions + Auto-bid
- [x] Real-time Chat (WebSocket)
- [x] Stripe Payments
- [x] Docker + CI/CD
- [ ] AI Search (pgvector + OpenAI)
- [ ] Image Upload (Cloudflare R2)
- [ ] Mobile App (React Native / Flutter)
- [ ] Admin Dashboard
- [ ] Kubernetes Production Deployment

## 🛠 Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.23, Gin, GORM |
| Database | PostgreSQL 16 + PostGIS + pgvector |
| Cache | Redis 7 |
| Frontend | Next.js 15, React 19, TypeScript |
| State | Zustand + React Query |
| Styling | Tailwind CSS + shadcn/ui |
| Payments | Stripe |
| Real-time | WebSocket (gorilla/websocket) |
| CI/CD | GitHub Actions |
| Infra | Docker, Kubernetes |

## 📄 License
MIT
