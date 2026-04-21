# 🛠️ GeoCore Next — Admin Dashboard Complete Rebuild Prompt
> **المرجع:** geocore-community (PHP) — النظام الأصلي اللي بُني على مدار 15+ سنة
> **الهدف:** بناء admin dashboard كامل في GeoCore Next (Go + Next.js) يغطي كل section من الأصل
> **الاستخدام:** افتح GeoCore Next في Cursor وقوله "نفّذ هذا الـ prompt كاملاً"

---

## السياق الكامل

الـ original GeoCore (PHP) كان عنده admin panel من أقوى الـ admin panels في صناعة الـ classifieds. المشروع الجديد (GeoCore Next) عنده admin dashboard ضعيف مقارنة بالأصل. مهمتك بناء admin dashboard شامل يغطي كل الـ sections دي.

**الـ Stack المطلوب:**
- Backend: Go + Gin (الموجود في `internal/admin/`)
- Frontend: Next.js 15 + shadcn/ui + Tailwind + Recharts
- Auth: Admin middleware موجود — استخدمه
- Route prefix: `/admin` للـ frontend، `/api/v1/admin` للـ backend

---

## 📐 البنية العامة للـ Dashboard

```
/admin
├── / (Dashboard Home — Stats Overview)
├── /users
│   ├── / (Users List)
│   ├── /[id] (User Details)
│   ├── /groups (User Groups)
│   └── /fields (Custom User Fields)
├── /listings
│   ├── / (All Listings)
│   ├── /pending (Awaiting Approval)
│   ├── /[id] (Edit Listing)
│   └── /extras (Listing Extras Management)
├── /auctions
│   ├── / (All Auctions)
│   ├── /pending (Awaiting Approval)
│   └── /[id] (Auction Details)
├── /categories
│   ├── / (Categories Tree)
│   ├── /[id] (Edit Category)
│   └── /fields (Category Custom Fields)
├── /pricing
│   ├── /plans (Price Plans)
│   ├── /plans/[id] (Edit Plan)
│   ├── /gateways (Payment Gateways)
│   ├── /invoices (Invoices)
│   └── /discount-codes (Discount Codes)
├── /storefronts
│   ├── / (All Storefronts)
│   └── /[id] (Storefront Details)
├── /content
│   ├── /pages (Static Pages)
│   ├── /emails (Email Templates)
│   └── /announcements (Site Announcements)
├── /geography
│   ├── /regions (Regions/Countries)
│   ├── /states (States/Governorates)
│   └── /cities (Cities)
├── /reports
│   ├── / (Reports List)
│   └── /[id] (Report Details)
├── /kyc
│   ├── / (KYC Queue)
│   └── /[id] (KYC Review)
├── /settings
│   ├── /general (General Settings)
│   ├── /features (Feature Flags)
│   ├── /seo (SEO Settings)
│   └── /notifications (Notification Settings)
└── /addons (Addons Management)
```

---

## 1️⃣ Dashboard Home — `/admin`

### Backend: `GET /api/v1/admin/dashboard`
يرجع:
```json
{
  "stats": {
    "total_users": 0,
    "new_users_today": 0,
    "new_users_week": 0,
    "total_listings": 0,
    "active_listings": 0,
    "pending_listings": 0,
    "total_auctions": 0,
    "active_auctions": 0,
    "total_revenue": 0,
    "revenue_today": 0,
    "revenue_month": 0,
    "pending_reports": 0,
    "pending_kyc": 0
  },
  "charts": {
    "daily_signups": [{"date": "2026-04-01", "count": 12}],
    "daily_revenue": [{"date": "2026-04-01", "amount": 450.00}],
    "listings_by_category": [{"category": "سيارات", "count": 234}],
    "listings_by_status": {"active": 0, "pending": 0, "sold": 0, "expired": 0}
  },
  "recent_activity": [
    {"type": "new_user", "description": "...", "created_at": "..."},
    {"type": "new_listing", "description": "...", "created_at": "..."}
  ]
}
```

### Frontend: Dashboard Page
- **Stats Cards Row:** Total Users | Active Listings | Active Auctions | Monthly Revenue
- **Alert Badges:** Pending Listings (N) | Pending KYC (N) | Open Reports (N)
- **Charts:**
  - Line chart: Daily signups آخر 30 يوم (Recharts)
  - Bar chart: Revenue آخر 30 يوم
  - Pie chart: Listings by category
- **Recent Activity Feed:** آخر 20 نشاط في الموقع

---

## 2️⃣ Users Management — `/admin/users`

### Backend APIs:

```go
// internal/admin/users_handler.go

GET    /api/v1/admin/users              // list مع pagination + filters
GET    /api/v1/admin/users/:id          // user details كامل
PUT    /api/v1/admin/users/:id          // edit user data
DELETE /api/v1/admin/users/:id          // soft delete
PUT    /api/v1/admin/users/:id/ban      // ban مع reason + duration
PUT    /api/v1/admin/users/:id/unban
PUT    /api/v1/admin/users/:id/suspend  // suspend مؤقت
PUT    /api/v1/admin/users/:id/verify   // verify manually
PUT    /api/v1/admin/users/:id/role     // تغيير role (user|moderator|admin)
PUT    /api/v1/admin/users/:id/group    // تغيير user group
GET    /api/v1/admin/users/:id/listings // listings الـ user
GET    /api/v1/admin/users/:id/orders   // orders الـ user
POST   /api/v1/admin/users/:id/impersonate // "login as user" لرؤية ما يراه
```

**Query params للـ list:**
`?page=1&limit=20&q=search&role=user&status=active&group_id=1&verified=true&sort=created_at&order=desc`

### Frontend: Users Page
**قايمة المستخدمين:**
- Columns: Avatar | Name | Email | Role | Group | Status | KYC | Join Date | Listings Count | Actions
- Filters: Role | Status | User Group | Verified | Date Range
- Search: by name/email
- Bulk actions: Ban selected | Change group | Export CSV

**User Details Page `/admin/users/[id]`:**
- Profile info + Edit form
- Tabs: Overview | Listings | Orders | Wallet | KYC | Activity Log | Notes
- Quick actions: Ban | Verify | Change Role | Reset Password | Impersonate

### User Groups — `/admin/users/groups`

```sql
CREATE TABLE user_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    description TEXT,
    price_plan_id INTEGER REFERENCES price_plans(id),
    permissions JSONB DEFAULT '{}',
    max_active_listings INTEGER DEFAULT 10,
    can_place_auctions BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**APIs:**
```
GET  /api/v1/admin/user-groups
POST /api/v1/admin/user-groups
PUT  /api/v1/admin/user-groups/:id
DELETE /api/v1/admin/user-groups/:id
```

### Custom User Fields — `/admin/users/fields`
إضافة fields إضافية في registration form

```sql
CREATE TABLE user_custom_fields (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    label VARCHAR(200),
    field_type VARCHAR(20) CHECK (field_type IN ('text','select','boolean','date','textarea')),
    options JSONB DEFAULT '[]',
    is_required BOOLEAN DEFAULT FALSE,
    show_in_profile BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE
);
```

---

## 3️⃣ Listings Management — `/admin/listings`

### Backend APIs:

```go
GET    /api/v1/admin/listings                    // all listings
GET    /api/v1/admin/listings/pending            // pending approval
GET    /api/v1/admin/listings/:id
PUT    /api/v1/admin/listings/:id                // full edit
DELETE /api/v1/admin/listings/:id
PUT    /api/v1/admin/listings/:id/approve
PUT    /api/v1/admin/listings/:id/reject         // body: {reason: string}
PUT    /api/v1/admin/listings/:id/feature        // mark as featured
PUT    /api/v1/admin/listings/:id/extend         // extend duration
POST   /api/v1/admin/listings/:id/extras         // add listing extra free
DELETE /api/v1/admin/listings/:id/extras/:extraId
PUT    /api/v1/admin/listings/:id/plan           // change price plan
POST   /api/v1/admin/listings/bulk-approve       // bulk action
POST   /api/v1/admin/listings/bulk-reject
POST   /api/v1/admin/listings/bulk-delete
```

**Query params:** `?status=pending&category_id=1&user_id=1&q=search&type=classified|auction&featured=true&date_from=...&date_to=...`

### Frontend:
**Listings List:**
- Columns: Thumbnail | Title | Category | Seller | Type | Price | Status | Date | Actions
- Status tabs: All | Pending | Active | Rejected | Expired | Sold
- Filters: Category | Type (Classified/Auction) | Date Range | Price Range | Featured
- Bulk select + bulk actions

**Pending Listings Queue `/admin/listings/pending`:**
- Priority view — يعرض الـ listings دي أول
- Quick approve/reject بدون فتح صفحة جديدة (inline)
- Reject مع modal لكتابة سبب الرفض
- Preview listing قبل القرار

**Edit Listing `/admin/listings/[id]`:**
- كل حاجة الـ user يقدر يعملها + extra admin powers:
  - تغيير الـ seller
  - تغيير الـ price plan
  - extend expiry date manually
  - add/remove listing extras بدون رسوم
  - تغيير الـ status يدوياً

### Listing Extras — `/admin/listings/extras`

```sql
CREATE TABLE listing_extras (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    description TEXT,
    type VARCHAR(20) CHECK (type IN ('featured','bold','highlight','gallery','video')),
    price DECIMAL(10,2) DEFAULT 0,
    duration_days INTEGER,
    is_active BOOLEAN DEFAULT TRUE
);
```

---

## 4️⃣ Auctions Management — `/admin/auctions`

### Backend APIs:

```go
GET /api/v1/admin/auctions             // all auctions مع filters
GET /api/v1/admin/auctions/pending
GET /api/v1/admin/auctions/:id
PUT /api/v1/admin/auctions/:id/approve
PUT /api/v1/admin/auctions/:id/reject
PUT /api/v1/admin/auctions/:id/cancel  // إلغاء + إشعار الـ bidders
PUT /api/v1/admin/auctions/:id/extend  // extend end time
GET /api/v1/admin/auctions/:id/bids    // bid history
DELETE /api/v1/admin/auctions/:id/bids/:bidId  // حذف bid مشبوه
```

**Auction Types:**
الـ admin يقدر يرى ويدير: Standard | Dutch | Reverse | Buy Now Only | Inventory

**الـ frontend:**
- نفس pattern الـ listings بس مع auction-specific columns: Current Bid | Start Price | End Time | Bid Count | Reserve Met?
- Dutch auctions: يظهر current price + next decrement time
- Reverse auctions: يظهر lowest offer + offers count
- قدرة على تمديد وقت المزاد

---

## 5️⃣ Categories Management — `/admin/categories`

### Backend APIs:

```go
GET    /api/v1/admin/categories              // tree structure
POST   /api/v1/admin/categories
PUT    /api/v1/admin/categories/:id
DELETE /api/v1/admin/categories/:id
PUT    /api/v1/admin/categories/:id/reorder  // drag & drop order
GET    /api/v1/admin/categories/:id/fields   // custom fields
POST   /api/v1/admin/categories/:id/fields
PUT    /api/v1/admin/categories/:id/fields/:fieldId
DELETE /api/v1/admin/categories/:id/fields/:fieldId
```

**الـ Category Settings (per category):**

```sql
ALTER TABLE categories ADD COLUMN settings JSONB DEFAULT '{}';
-- settings يحتوي على:
-- {
--   "allow_classifieds": true,
--   "allow_auctions": true,
--   "allow_dutch_auctions": false,
--   "allow_reverse_auctions": false,
--   "require_admin_approval": false,
--   "default_listing_duration": 30,
--   "max_listing_duration": 90,
--   "template": "default",
--   "meta_title": "",
--   "meta_description": "",
--   "icon": "",
--   "color": ""
-- }
```

**Frontend: Categories Tree**
- Tree view مع drag & drop لإعادة الترتيب
- Expand/collapse subcategories
- Quick edit inline (الاسم بس)
- Full edit page لكل category (settings + custom fields)

**Category Custom Fields `/admin/categories/fields`:**
اتعمل في GAP-003 — اربطه هنا في الـ admin UI

---

## 6️⃣ Pricing — `/admin/pricing`

### Price Plans — `/admin/pricing/plans`

**الـ Price Plan Model الكامل:**

```sql
CREATE TABLE price_plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) UNIQUE,
    description TEXT,
    price DECIMAL(10,2) DEFAULT 0,
    billing_period VARCHAR(20) DEFAULT 'one_time' 
        CHECK (billing_period IN ('one_time','monthly','yearly','per_listing')),
    
    -- Listing limits
    max_active_listings INTEGER DEFAULT 10,      -- -1 = unlimited
    max_images_per_listing INTEGER DEFAULT 5,
    listing_duration_days INTEGER DEFAULT 30,
    max_listing_duration_days INTEGER DEFAULT 90,
    
    -- Classifieds settings
    allow_classifieds BOOLEAN DEFAULT TRUE,
    classified_cost DECIMAL(10,2) DEFAULT 0,
    
    -- Auction settings  
    allow_auctions BOOLEAN DEFAULT TRUE,
    allow_dutch_auctions BOOLEAN DEFAULT FALSE,
    allow_reverse_auctions BOOLEAN DEFAULT FALSE,
    allow_buy_now BOOLEAN DEFAULT TRUE,
    allow_buy_now_only BOOLEAN DEFAULT FALSE,
    allow_inventory_auctions BOOLEAN DEFAULT FALSE,
    auction_cost DECIMAL(10,2) DEFAULT 0,
    commission_percent DECIMAL(5,2) DEFAULT 0,   -- % من قيمة البيع
    
    -- Features
    allow_featured BOOLEAN DEFAULT FALSE,
    featured_cost DECIMAL(10,2) DEFAULT 0,
    allow_storefront BOOLEAN DEFAULT FALSE,
    require_approval BOOLEAN DEFAULT FALSE,
    
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

**APIs:**
```
GET    /api/v1/admin/price-plans
POST   /api/v1/admin/price-plans
GET    /api/v1/admin/price-plans/:id
PUT    /api/v1/admin/price-plans/:id
DELETE /api/v1/admin/price-plans/:id
```

**Frontend: Price Plans Page**
- Cards لكل plan مع quick overview
- Edit plan: form كامل مقسّم بـ sections:
  - Basic Info (name, price, billing)
  - Listing Limits
  - Classifieds Settings
  - Auction Settings (مع toggle لكل نوع)
  - Features & Extras

### Payment Gateways — `/admin/pricing/gateways`

```sql
CREATE TABLE payment_gateways (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    slug VARCHAR(50) UNIQUE,  -- stripe | paypal | paymob | fawry | vodafone_cash
    display_name VARCHAR(100),
    is_active BOOLEAN DEFAULT FALSE,
    is_sandbox BOOLEAN DEFAULT TRUE,
    config JSONB DEFAULT '{}',  -- encrypted credentials
    supported_currencies JSONB DEFAULT '["EGP","USD"]',
    fee_percent DECIMAL(5,2) DEFAULT 0,
    fee_fixed DECIMAL(10,2) DEFAULT 0,
    sort_order INTEGER DEFAULT 0
);
```

**APIs:**
```
GET /api/v1/admin/payment-gateways
PUT /api/v1/admin/payment-gateways/:slug/toggle
PUT /api/v1/admin/payment-gateways/:slug/config
PUT /api/v1/admin/payment-gateways/:slug/test  -- test connection
```

**Frontend: Gateways Page**
- Card لكل gateway: Logo | Status Toggle | Config Button | Test Button
- Config modal: credentials inputs (masked) + sandbox/live toggle
- Supported: Stripe | PayPal | Paymob (مصر) | Fawry | Vodafone Cash

### Invoices — `/admin/pricing/invoices`

```sql
CREATE TABLE invoices (
    id SERIAL PRIMARY KEY,
    invoice_number VARCHAR(20) UNIQUE,  -- INV-2026-00001
    user_id INTEGER REFERENCES users(id),
    items JSONB NOT NULL,   -- [{description, quantity, unit_price, total}]
    subtotal DECIMAL(10,2),
    discount DECIMAL(10,2) DEFAULT 0,
    tax DECIMAL(10,2) DEFAULT 0,
    total DECIMAL(10,2),
    status VARCHAR(20) DEFAULT 'pending'
        CHECK (status IN ('pending','paid','refunded','cancelled')),
    gateway_id INTEGER,
    gateway_reference VARCHAR(200),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    paid_at TIMESTAMPTZ
);
```

**APIs + Frontend:** List invoices | Filter by status/date/user | View details | Print-friendly view | Refund

### Discount Codes — `/admin/pricing/discount-codes`

```sql
CREATE TABLE discount_codes (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    discount_type VARCHAR(20) CHECK (discount_type IN ('percent','fixed')),
    discount_value DECIMAL(10,2),
    applies_to VARCHAR(20) CHECK (applies_to IN ('all','classifieds','auctions','subscriptions')),
    min_order_amount DECIMAL(10,2) DEFAULT 0,
    max_uses INTEGER DEFAULT NULL,  -- NULL = unlimited
    uses_per_user INTEGER DEFAULT 1,
    current_uses INTEGER DEFAULT 0,
    user_group_id INTEGER,  -- NULL = all groups
    valid_from TIMESTAMPTZ,
    valid_until TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 7️⃣ Storefronts — `/admin/storefronts`

### Backend APIs:

```go
GET /api/v1/admin/storefronts           // all storefronts
GET /api/v1/admin/storefronts/:id
PUT /api/v1/admin/storefronts/:id/approve
PUT /api/v1/admin/storefronts/:id/suspend
PUT /api/v1/admin/storefronts/:id/feature  // featured store
DELETE /api/v1/admin/storefronts/:id
```

**Frontend:**
- List: Store Name | Owner | Slug | Status | Listings Count | Created At
- Quick approve/suspend
- Preview link

---

## 8️⃣ Content Management — `/admin/content`

### Email Templates — `/admin/content/emails`

```sql
CREATE TABLE email_templates (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE,  -- user_registered, listing_approved, etc.
    name VARCHAR(200),
    subject VARCHAR(300),
    body_html TEXT,
    body_text TEXT,
    variables JSONB DEFAULT '[]',  -- available placeholders
    is_active BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

**Default Templates:**
- `user_registered` — مرحباً بتسجيلك
- `listing_approved` — تم الموافقة على إعلانك
- `listing_rejected` — تم رفض إعلانك (مع السبب)
- `auction_won` — مبروك! فزت بالمزاد
- `bid_outbid` — تم تخطي عرضك
- `auction_ending_soon` — المزاد ينتهي قريباً
- `payment_received` — تم استلام دفعتك
- `password_reset` — إعادة تعيين كلمة المرور
- `kyc_approved` — تم التحقق من هويتك
- `kyc_rejected` — تم رفض التحقق من هويتك

**APIs:**
```
GET /api/v1/admin/email-templates
GET /api/v1/admin/email-templates/:slug
PUT /api/v1/admin/email-templates/:slug
POST /api/v1/admin/email-templates/:slug/test  -- إرسال test email
POST /api/v1/admin/email-templates/:slug/reset -- إعادة لـ default
```

**Frontend: Email Templates**
- List كل الـ templates مع status
- Edit template: Subject input + WYSIWYG HTML editor
- Variables panel: يعرض الـ placeholders المتاحة مع copy button
- Preview rendered email
- Send test email button

### Static Pages — `/admin/content/pages`

```sql
CREATE TABLE static_pages (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200),
    slug VARCHAR(200) UNIQUE,
    content TEXT,
    meta_title VARCHAR(200),
    meta_description TEXT,
    is_published BOOLEAN DEFAULT FALSE,
    show_in_footer BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

Default pages: About Us | Contact | Privacy Policy | Terms of Service | FAQ | Help

### Announcements — `/admin/content/announcements`

```sql
CREATE TABLE announcements (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200),
    content TEXT,
    type VARCHAR(20) CHECK (type IN ('info','warning','success','error')),
    display_location VARCHAR(20) CHECK (display_location IN ('homepage','all','listing_form','auction_form')),
    target_group_id INTEGER,  -- NULL = all users
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 9️⃣ Geography — `/admin/geography`

```sql
CREATE TABLE geo_regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    name_ar VARCHAR(100),
    code VARCHAR(10),
    type VARCHAR(20) CHECK (type IN ('country','state','city')),
    parent_id INTEGER REFERENCES geo_regions(id),
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    is_active BOOLEAN DEFAULT TRUE,
    sort_order INTEGER DEFAULT 0
);
```

**APIs:**
```
GET    /api/v1/admin/geography           -- tree: countries > states > cities
POST   /api/v1/admin/geography
PUT    /api/v1/admin/geography/:id
DELETE /api/v1/admin/geography/:id
POST   /api/v1/admin/geography/import    -- import from CSV
```

**Frontend:**
- Collapsible tree: Countries → States → Cities
- Search
- Add/Edit/Delete
- Import CSV (seed initial data: مصر + دول عربية + عالمية)

---

## 🔟 Reports — `/admin/reports`

(موجود جزئياً — وسّعه)

**إضافة report types:**

```sql
ALTER TABLE reports ADD COLUMN type VARCHAR(30) 
    CHECK (type IN ('spam','fraud','inappropriate','duplicate','wrong_category','other'));
ALTER TABLE reports ADD COLUMN priority VARCHAR(10) DEFAULT 'normal'
    CHECK (priority IN ('low','normal','high','urgent'));
ALTER TABLE reports ADD COLUMN admin_notes TEXT;
ALTER TABLE reports ADD COLUMN resolved_by INTEGER REFERENCES users(id);
```

**Frontend Improvements:**
- Priority badges (urgent باللون الأحمر)
- Bulk resolve
- Admin internal notes
- Link مباشر للـ content المُبلَّغ عنه مع preview

---

## 1️⃣1️⃣ KYC Review — `/admin/kyc`

(موجود — وسّعه)

**Frontend Improvements:**
- Image viewer مدمج (بدون فتح tab جديد)
- Side-by-side view: صورة الـ ID + الـ selfie
- Approve/Reject مع reason dropdown
- Auto-notify user عند القرار
- Stats: Pending | Approved this week | Rejection rate

---

## 1️⃣2️⃣ Site Settings — `/admin/settings`

### General Settings — `/admin/settings/general`

```sql
CREATE TABLE site_settings (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    value TEXT,
    type VARCHAR(20) DEFAULT 'string'  -- string|boolean|integer|json
);
```

**الـ Settings المطلوبة:**

```
-- Site Info
site_name           = "GeoCore Next"
site_tagline        = ""
site_url            = ""
site_logo_url       = ""
site_favicon_url    = ""
site_email          = ""
support_email       = ""
default_currency    = "EGP"
default_language    = "ar"

-- Registration
allow_registration  = true
require_email_verify = true
allow_social_login  = false
min_password_length = 8

-- Listings
require_listing_approval = false
default_listing_duration = 30
max_images_per_listing   = 10
max_image_size_mb        = 5
allow_anonymous_listings = false

-- Auctions  
allow_auctions          = true
allow_dutch_auctions    = true
allow_reverse_auctions  = true
auto_end_auctions       = true
outbid_notification     = true

-- Monetization
platform_commission_percent = 5
allow_free_listings         = true
max_free_listings_per_user  = 5

-- SEO
meta_title        = ""
meta_description  = ""
google_analytics  = ""
facebook_pixel    = ""

-- Social
facebook_url   = ""
twitter_url    = ""
instagram_url  = ""
```

### Feature Flags — `/admin/settings/features`

```sql
CREATE TABLE feature_flags (
    id SERIAL PRIMARY KEY,
    key VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(200),
    description TEXT,
    is_enabled BOOLEAN DEFAULT FALSE,
    category VARCHAR(50)  -- core|payments|social|experimental
);
```

**Default Flags:**
```
wallet_enabled          = true
escrow_enabled          = true
crowdshipping_enabled   = false
crypto_payments_enabled = false
loyalty_points_enabled  = true
ai_search_enabled       = false
storefront_enabled      = true
kyc_required_for_wallet = true
```

**Frontend: Feature Flags Page**
- Toggle switches لكل feature
- Grouped by category
- Warning لو feature تانية بتعتمد عليها
- "تجريبي" badge للـ experimental features

---

## 1️⃣3️⃣ Addons — `/admin/addons`

مش محتاج يتنفذ دلوقتي — بس ابني الـ UI skeleton:

```sql
CREATE TABLE addons (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    slug VARCHAR(100) UNIQUE,
    version VARCHAR(20),
    description TEXT,
    is_active BOOLEAN DEFAULT FALSE,
    is_installed BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}'
);
```

**Frontend:** Cards لكل addon مع: Name | Version | Status Toggle | Configure

---

## 🎨 متطلبات الـ UI/UX

### Layout:
- **Sidebar** ثابتة على اليسار (collapsible على الموبايل)
- الـ sidebar مقسّمة بـ sections مع icons
- **Topbar:** Search | Notifications Bell | Quick Links (Pending X) | Profile
- **Content area:** Breadcrumbs + Page Title + Actions

### Shared Components:
```typescript
// components/admin/
DataTable.tsx     // reusable table مع sort + filter + pagination + bulk select
StatusBadge.tsx   // colored badge لكل status
QuickActions.tsx  // dropdown actions على كل row
ConfirmDialog.tsx // modal للـ destructive actions مع reason input
StatsCard.tsx     // card للـ KPIs مع icon + trend
FilterBar.tsx     // reusable filter component
```

### Design Tokens:
- استخدم shadcn/ui components بالكامل
- Colors: Tailwind CSS classes فقط (مش hardcoded)
- Dark mode: مدعوم عن طريق shadcn
- Responsive: يشتغل على موبايل (collapsed sidebar)
- RTL support: الـ admin ممكن يكون عربي — استخدم `dir="rtl"` option

---

## 🔌 ترتيب التنفيذ المقترح

```
1. Dashboard Home (stats + charts) ← أهم صفحة
2. Users List + User Details
3. User Groups
4. Listings Management (list + approve/reject)
5. Categories Tree + Custom Fields
6. Price Plans (full form)
7. Payment Gateways
8. Email Templates
9. Site Settings + Feature Flags
10. Geography
11. Auctions Management
12. Storefronts
13. Reports (improved)
14. KYC (improved)
15. Invoices + Discount Codes
16. Announcements + Static Pages
17. Addons (skeleton)
```

---

## ⚠️ قواعد مهمة للـ Agent

1. **لا تكسر الـ existing admin code** — وسّع وأضف، لا تمسح
2. **كل form** يحتوي على validation بالـ frontend والـ backend
3. **كل destructive action** (حذف، ban، reject) يحتاج confirmation dialog مع reason
4. **الـ pagination** على كل list page — 20 item افتراضي
5. **الـ audit log** — كل action من الـ admin يتسجّل:
```sql
CREATE TABLE admin_audit_log (
    id BIGSERIAL PRIMARY KEY,
    admin_id INTEGER REFERENCES users(id),
    action VARCHAR(100),
    entity_type VARCHAR(50),
    entity_id INTEGER,
    changes JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```
6. **الـ search** على كل list page — server-side search مع debounce
7. **Export CSV** على كل list page تقريباً

---

*مرجع: geocore-community (PHP) — 15+ سنة من admin panel development*
*الهدف: نفس القدرات، modern stack، UX أحسن*
