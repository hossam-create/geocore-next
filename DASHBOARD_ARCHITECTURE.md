# GeoCore — Dashboard Architecture & Design System

## 1. الطبقات (Layers)

```
┌─────────────────────────────────────────────────────┐
│  Layer 1 — Admin Control Center  (frontend-admin)   │
│  Port: 5173 | Route: /admin/*                       │
│  Roles: super_admin, admin, ops_admin, finance_admin│
│         support_admin                               │
├─────────────────────────────────────────────────────┤
│  Layer 2 — Seller Dashboard      (frontend)         │
│  Port: 3000 | Route: /seller/*                      │
│  Guard: authenticated + seller_verified             │
├─────────────────────────────────────────────────────┤
│  Layer 3 — Buyer Dashboard       (frontend)         │
│  Port: 3000 | Route: /dashboard                     │
│  Guard: authenticated                               │
├─────────────────────────────────────────────────────┤
│  Layer 4 — Traveler Dashboard    (frontend)         │
│  Port: 3000 | Route: /traveler/*                    │
│  Guard: authenticated + traveler_active flag        │
└─────────────────────────────────────────────────────┘
```

---

## 2. نموذج الصلاحيات (RBAC Model)

| Role           | Permissions (Key Ones)                                       |
|----------------|--------------------------------------------------------------|
| super_admin    | ALL                                                          |
| admin          | dashboard, users, listings, finance, audit, settings, catalog|
| ops_admin      | dashboard, listings.moderate, reports.review, ops.*          |
| finance_admin  | dashboard, finance.read, audit.logs.read                     |
| support_admin  | dashboard, support.tickets.*, reports.review                 |
| seller         | seller.listings.*, seller.orders.*, seller.analytics         |
| buyer          | buyer.orders.*, buyer.watchlist, buyer.wallet                |
| traveler       | traveler.trips.*, traveler.deliveries.*, traveler.earnings   |

**Enforcement Points:**
- Backend: Gin middleware per route group
- Admin Frontend: `RequirePermission` wrapper on each `<Route>`
- Sidebar: `hasAnyPermission()` filters visible nav items
- UI Buttons: `hasPermission()` guards on destructive actions
- User Frontend: `requireAuth` server-side + role check

---

## 3. الأمان والامتثال (Security & Compliance)

### Auth
- JWT stored in `localStorage` under `admin_token` (admin) / `auth_token` (user)
- Token parsed for `role`, `user_id`, `exp`
- Auto-logout on expiry + refresh flow
- All API requests carry `Authorization: Bearer <token>`

### Session Security
- `restore()` on app mount — validates token locally before trusting
- Admin panel isolated on separate port (5173) — no shared cookies with user frontend
- CORS restricted to known origins on backend

### Audit Trail
- All write actions on `/admin/*` routes logged to `audit_logs` table
- Audit Logs page accessible only to `audit.logs.read` permission

### Compliance Checklist
- [ ] PII masking on user listing (email/phone partial display)
- [ ] Read-only notice for non-write roles
- [ ] Escrow release BLOCKED unless Custodii APPROVED
- [ ] All destructive actions require confirmation dialog
- [ ] No business logic in UI — all decisions via API

---

## 4. الإطار البصري (Design System)

### Design Tokens
```
Background:    #0F172A (sidebar) / #F8FAFC (content)
Brand Primary: #0071CE (blue)
Brand Accent:  #F59E0B (amber — alerts)
Success:       #10B981
Danger:        #EF4444
Warning:       #F59E0B
Text Primary:  #0F172A
Text Muted:    #64748B
Border:        #E2E8F0
Card BG:       #FFFFFF
```

### Layout Pattern (Every Screen)
```
┌──────────────┬────────────────────────────────────┐
│              │  [Topbar: search + alerts + user]  │
│   Sidebar    ├────────────────────────────────────┤
│   (240px)    │  [Page Header: title + actions]    │
│              ├───────────────────┬────────────────┤
│  Logo        │                   │  Action Panel  │
│  Nav Sections│   Main Content    │  (contextual)  │
│  User Card   │   Table / Cards   │  (details)     │
└──────────────┴───────────────────┴────────────────┘
```

### UI Principles (World-Class Standard)
1. **Data density** — show maximum info without crowding (Saleor pattern)
2. **Status everywhere** — every entity has a visible status badge
3. **Action proximity** — actions adjacent to the data they affect
4. **Progressive disclosure** — summary → detail → action
5. **Alert-first** — urgent items surface immediately at top
6. **Real-time feel** — relative timestamps ("2 min ago")
7. **Keyboard-friendly** — Cmd+K search, shortcuts on key actions

---

## 5. خريطة الصفحات (Page Map)

### Admin Control Center (frontend-admin)
```
/admin                  Dashboard (KPIs + Activity + Queue)
/admin/listings         Listing Moderation Queue
/admin/auctions         Auction Monitor
/admin/users            User Management
/admin/reports          Report Review Center
/admin/payments         Finance & Revenue
/admin/transactions     Transaction Ledger
/admin/categories       Catalog Management
/admin/pricing          Plans & Pricing
/admin/settings         Site Settings
/admin/logs             Audit Trail
```

### User-Facing Dashboards (frontend)
```
/dashboard              Buyer Dashboard (orders, wallet, bids)
/seller                 Seller Home
/seller/listings        My Listings
/seller/orders          My Orders
/seller/analytics       Revenue Analytics
/seller/settings        Store Settings
/traveler               Traveler Home (trips + deliveries)
/traveler/trips         My Trips
/traveler/orders        Available Delivery Orders
/traveler/earnings      Earnings History
```

---

## 6. خطة التنفيذ (Implementation Phases)

### Phase 1 — Admin Panel Redesign ← CURRENT
- [x] Permissions & RBAC guards
- [ ] Sidebar redesign (world-class nav)
- [ ] Header redesign (topbar with system status)
- [ ] DashboardPage redesign (KPIs + Chart + Activity)

### Phase 2 — Seller Dashboard
- [ ] /seller page with KPIs + recent orders + quick actions
- [ ] RBAC guard (seller_verified)

### Phase 3 — Buyer Dashboard
- [ ] /dashboard with orders + wallet + watchlist + bids
- [ ] RBAC guard (authenticated)

### Phase 4 — Traveler Dashboard
- [ ] /traveler with trips + delivery orders + earnings
- [ ] RBAC guard (authenticated + traveler)
