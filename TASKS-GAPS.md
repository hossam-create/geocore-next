# GeoCore Next — GAPS ONLY TASKS.md
> **للـ AI Agent:** ده الملف النهائي للـ gaps الفعلية بس. كل حاجة تانية موجودة.
> نفّذ بالترتيب. لا تمس الكود الموجود إلا لو الـ task بيتطلب ربط صريح.

---

## GAP-001: Dutch Auction — Background Ticker + WebSocket Broadcast
**الأولوية:** P0 🔴  
**الجهد:** نص يوم  
**الملفات المتأثرة:** `internal/auctions/`

**السياق:**
`GetCurrentDutchPrice()` و`completeDutchAuction()` موجودين. الناقص هو الـ goroutine اللي بيشغّل الـ price decrement تلقائياً وبيبعته على الـ WebSocket.

**المطلوب:**

- [ ] إنشاء `internal/auctions/dutch_ticker.go`:
```go
// DutchAuctionManager manages all active dutch auctions
type DutchAuctionManager struct {
    db    *gorm.DB
    hub   *Hub  // existing WebSocket hub
    tickers map[uint]*time.Ticker
    mu   sync.Mutex
}

// StartTicker — يبدأ ticker لـ auction معين
// يشتغل كل `decrement_interval_minutes` دقيقة
// بيحسب السعر الجديد، بيحدثه في DB، بيبعته على WS
// لو وصل لـ reserve_price → يوقف نفسه + يكمّل completeDutchAuction()

// StopTicker — يوقف ticker لـ auction معين

// RestoreOnStartup — لما السيرفر يقوم، يجيب كل dutch auctions النشطة ويبدأ tickersها
```

- [ ] تحديث `POST /api/v1/auctions` — لو `auction_type = dutch` يبدأ `StartTicker()` تلقائياً

- [ ] تحديث `POST /api/v1/auctions/:id/buy-now` — يوقف الـ ticker بعد الشراء

- [ ] WebSocket message format للـ price update:
```json
{
  "type": "dutch_price_update",
  "auction_id": 123,
  "current_price": 850.00,
  "next_decrement_at": "2026-04-05T14:30:00Z",
  "reserve_price": 500.00
}
```

- [ ] تحديث frontend Dutch Auction page — يستقبل `dutch_price_update` ويحدّث:
  - السعر الحالي live
  - عداد تنازلي للخصم القادم (`next_decrement_at`)

**اختبار النجاح:**
1. ابدأ dutch auction بـ start_price=1000, decrement=50, interval=1min
2. بعد دقيقة السعر يبقى 950 تلقائياً ويتبعت على WS
3. اضغط buy-now → يوقف الـ ticker، يكمّل الصفقة

---

## GAP-002: Reverse Auctions — Full Buyer-Posts-Request Flow
**الأولوية:** P0 🔴  
**الجهد:** يومين  
**الملفات المتأثرة:** `internal/auctions/`, `frontend/`

**السياق:**
الموجود هو "lowest bid wins" على auction عادي. المطلوب هو flow مختلف تماماً: المشتري ينشر طلب، البائعين يتقدموا بعروض، المشتري يختار.

**المطلوب:**

**Backend:**

- [ ] Migration جديد: `migrations/YYYYMMDD_reverse_auction_requests.sql`
```sql
CREATE TABLE reverse_auction_requests (
    id SERIAL PRIMARY KEY,
    buyer_id INTEGER NOT NULL REFERENCES users(id),
    title VARCHAR(200) NOT NULL,
    description TEXT,
    category_id INTEGER REFERENCES categories(id),
    max_budget DECIMAL(15,2),
    deadline TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open','closed','fulfilled','expired')),
    images JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE reverse_auction_offers (
    id SERIAL PRIMARY KEY,
    request_id INTEGER NOT NULL REFERENCES reverse_auction_requests(id),
    seller_id INTEGER NOT NULL REFERENCES users(id),
    price DECIMAL(15,2) NOT NULL,
    description TEXT,
    delivery_days INTEGER,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending','accepted','rejected','withdrawn')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(request_id, seller_id)
);

CREATE INDEX idx_rar_status ON reverse_auction_requests(status);
CREATE INDEX idx_rar_buyer ON reverse_auction_requests(buyer_id);
CREATE INDEX idx_rao_request ON reverse_auction_offers(request_id);
CREATE INDEX idx_rao_seller ON reverse_auction_offers(seller_id);
```

- [ ] إنشاء `internal/reverse_auctions/model.go` — الـ structs
- [ ] إنشاء `internal/reverse_auctions/handler.go` — الـ endpoints:

| Method | Endpoint | وصف | Auth |
|--------|----------|-----|------|
| POST | `/api/v1/reverse-auctions` | إنشاء طلب | buyer |
| GET | `/api/v1/reverse-auctions` | list الطلبات المفتوحة | public |
| GET | `/api/v1/reverse-auctions/:id` | تفاصيل طلب | public |
| PUT | `/api/v1/reverse-auctions/:id` | تعديل طلب | owner |
| DELETE | `/api/v1/reverse-auctions/:id` | إلغاء طلب | owner |
| POST | `/api/v1/reverse-auctions/:id/offers` | تقديم عرض | seller |
| GET | `/api/v1/reverse-auctions/:id/offers` | عروض الطلب | owner أو sellers |
| PUT | `/api/v1/reverse-auctions/:id/offers/:offerId/accept` | قبول عرض | owner فقط |
| PUT | `/api/v1/reverse-auctions/:id/offers/:offerId/reject` | رفض عرض | owner |
| DELETE | `/api/v1/reverse-auctions/:id/offers/:offerId` | سحب العرض | offer owner |

- [ ] لما يتقبل عرض:
  1. status الطلب يبقى `fulfilled`
  2. باقي العروض يترفضوا تلقائياً
  3. لو Wallet موجود → `wallet.Hold()` للمبلغ
  4. إشعار للبائع صاحب العرض المقبول

**Frontend:**

- [ ] صفحة `/reverse-auctions` — list الطلبات المفتوحة مع filter (category, budget, deadline)
- [ ] صفحة `/reverse-auctions/new` — form لإنشاء طلب (buyers فقط)
- [ ] صفحة `/reverse-auctions/:id` — تفاصيل + list العروض
  - لو المشتري: يشوف كل العروض + زرار قبول/رفض
  - لو بائع: يشوف العروض الأخرى (بدون أسماء) + form يقدم عرضه
- [ ] في navbar أو sidebar: "اطلب منتج" button

**اختبار النجاح:**
1. مشتري ينشر "عايز لاب توب Core i7 بحد أقصى 15,000 جنيه"
2. بائعان يتقدموا بعروض (12,000 و 13,500)
3. المشتري يقبل عرض الـ 12,000
4. الطلب يتغلق، البائع التاني يتبلغ إن عرضه اترفض

---

## GAP-003: Custom Fields per Category
**الأولوية:** P1 🟡  
**الجهد:** يوم إلى يومين  
**الملفات المتأثرة:** `internal/categories/`, `internal/listings/`, `frontend/`

**المطلوب:**

**Backend:**

- [ ] Migration: `migrations/YYYYMMDD_category_fields.sql`
```sql
CREATE TABLE category_fields (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,         -- مفتاح برمجي: "engine_size"
    label VARCHAR(100) NOT NULL,        -- للعرض: "سعة المحرك"
    label_en VARCHAR(100),              -- "Engine Size"
    field_type VARCHAR(20) NOT NULL CHECK (
        field_type IN ('text','number','select','boolean','range','date')
    ),
    options JSONB DEFAULT '[]',         -- للـ select: [{"value":"automatic","label":"أوتوماتيك"}]
    is_required BOOLEAN DEFAULT FALSE,
    placeholder VARCHAR(200),
    unit VARCHAR(20),                   -- مثلاً: "cc", "m²", "km"
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_cf_category ON category_fields(category_id) WHERE is_active = TRUE;
```

- [ ] إضافة `custom_fields JSONB DEFAULT '{}'` في `listings` table (migration منفصل)

- [ ] Seed data للـ categories الرئيسية:
```sql
-- سيارات
INSERT INTO category_fields (category_id, name, label, label_en, field_type, options, is_required, unit, sort_order)
VALUES
  (CAR_CATEGORY_ID, 'year', 'سنة الصنع', 'Year', 'number', '[]', true, NULL, 1),
  (CAR_CATEGORY_ID, 'mileage', 'الكيلومتراج', 'Mileage', 'number', '[]', false, 'km', 2),
  (CAR_CATEGORY_ID, 'transmission', 'ناقل الحركة', 'Transmission', 'select',
    '[{"value":"manual","label":"مانيوال"},{"value":"automatic","label":"أوتوماتيك"}]', false, NULL, 3),
  (CAR_CATEGORY_ID, 'fuel_type', 'نوع الوقود', 'Fuel Type', 'select',
    '[{"value":"petrol","label":"بنزين"},{"value":"diesel","label":"ديزل"},{"value":"electric","label":"كهربائي"}]', false, NULL, 4);

-- عقارات
INSERT INTO category_fields (category_id, name, label, label_en, field_type, is_required, unit, sort_order)
VALUES
  (REAL_ESTATE_ID, 'area', 'المساحة', 'Area', 'number', true, 'm²', 1),
  (REAL_ESTATE_ID, 'rooms', 'عدد الغرف', 'Rooms', 'number', false, NULL, 2),
  (REAL_ESTATE_ID, 'bathrooms', 'عدد الحمامات', 'Bathrooms', 'number', false, NULL, 3),
  (REAL_ESTATE_ID, 'floor', 'الدور', 'Floor', 'number', false, NULL, 4),
  (REAL_ESTATE_ID, 'furnished', 'مفروشة', 'Furnished', 'boolean', false, NULL, 5);
```

- [ ] تحديث `GET /api/v1/categories/:id/fields` — يرجع الـ fields للـ category

- [ ] تحديث `POST /api/v1/listings` — validate الـ required custom fields قبل الحفظ

- [ ] تحديث `GET /api/v1/listings` — دعم filter على custom fields:
  - مثال: `?cf[year_min]=2020&cf[year_max]=2023&cf[transmission]=automatic`

**Frontend:**

- [ ] Hook: `useCategoryFields(categoryId)` — يجيب الـ fields عند تغيير الـ category

- [ ] Component: `<DynamicFieldsForm fields={fields} onChange={...} />` — يعرض الـ fields المناسبة ديناميكياً حسب نوعها (text input / select / checkbox / range)

- [ ] تحديث listing create/edit form — بعد اختيار الـ category تظهر الـ custom fields تلقائياً

- [ ] تحديث صفحة الـ Search/Filter — إضافة الـ custom fields كـ filters جانبية لما الـ user يختار category

- [ ] تحديث listing details page — عرض الـ custom fields في جدول مرتب

**اختبار النجاح:**
1. اختر category "سيارات" في إنشاء listing → يظهر fields: سنة الصنع، الكيلومتراج، ناقل الحركة
2. احفظ listing مع custom fields
3. ابحث بـ filter "ناقل الحركة: أوتوماتيك" → تظهر السيارات الأوتوماتيك فقط

---

## GAP-004: CSV Import/Export for Listings
**الأولوية:** P2 🟢  
**الجهد:** نص يوم  
**الملفات المتأثرة:** `internal/listings/`

**المطلوب:**

- [ ] `GET /api/v1/listings/export` (authenticated):
  - يرجع CSV بكل listings الـ user
  - Columns: id, title, description, price, category, status, created_at, views, custom_fields
  - Content-Type: `text/csv`
  - Content-Disposition: `attachment; filename="my-listings-{date}.csv"`

- [ ] `GET /api/v1/listings/export/template`:
  - يرجع CSV template فاضي مع header row فقط
  - يساعد الـ user يعرف الـ format المطلوب

- [ ] `POST /api/v1/listings/import` (authenticated):
  - يقبل CSV file (multipart)
  - Max 500 rows
  - Validate كل row قبل الـ import
  - يرجع: `{ success: N, failed: M, errors: [{row: X, reason: "..."}] }`
  - يحفظ الـ valid rows فقط، مش يوقف عند أول error

- [ ] Frontend:
  - في `/dashboard/listings`: زرار "Export CSV" + زرار "Import CSV"
  - Import: file picker + progress indicator + results summary

---

## GAP-005: Listing Views Tracking + Analytics Export
**الأولوية:** P3 🔵  
**الجهد:** نص يوم  
**الملفات المتأثرة:** `internal/analytics/`

**السياق:**
`internal/analytics/` موجود بـ seller endpoints. الناقص هو tracking الـ views فعلياً + export.

**المطلوب:**

- [ ] Migration: `migrations/YYYYMMDD_listing_views.sql`
```sql
CREATE TABLE listing_views (
    id BIGSERIAL PRIMARY KEY,
    listing_id INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    viewer_id INTEGER REFERENCES users(id) ON DELETE SET NULL,  -- NULL لو مش logged in
    ip_hash VARCHAR(64),   -- hash الـ IP مش الـ IP نفسه (privacy)
    user_agent VARCHAR(500),
    viewed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_lv_listing ON listing_views(listing_id);
CREATE INDEX idx_lv_date ON listing_views(viewed_at);

-- View للـ daily stats (أسرع في الـ queries)
CREATE MATERIALIZED VIEW listing_daily_views AS
SELECT 
    listing_id,
    DATE(viewed_at) as view_date,
    COUNT(*) as total_views,
    COUNT(DISTINCT COALESCE(viewer_id::text, ip_hash)) as unique_views
FROM listing_views
GROUP BY listing_id, DATE(viewed_at);

CREATE INDEX idx_ldv_listing ON listing_daily_views(listing_id);
```

- [ ] Middleware أو helper: `TrackView(listingID, userID, ip, userAgent)` — يتنادى في `GET /api/v1/listings/:id`
  - لا يحسب نفس الـ user أكتر من مرة كل ساعة (Redis TTL)

- [ ] تحديث analytics endpoints عشان يستخدم `listing_views`:
  - `GET /api/v1/analytics/listings/:id/views` — views per day (last 30 days)
  - `GET /api/v1/analytics/summary` — total views, top listings

- [ ] `GET /api/v1/analytics/export` — CSV export للـ analytics data

- [ ] Frontend: إضافة "Views" chart في seller analytics dashboard (موجود) باستخدام البيانات الحقيقية

---

## ملخص الأولويات

| Gap | الجهد | متى |
|-----|-------|-----|
| GAP-001: Dutch Ticker | نص يوم | **دلوقتي** |
| GAP-002: Reverse Auctions Full Flow | يومين | **دلوقتي** |
| GAP-003: Custom Fields | يوم-يومين | بعد اللانش |
| GAP-004: CSV Import/Export | نص يوم | بعد اللانش |
| GAP-005: Listing Views Tracking | نص يوم | بعدين |

**إجمالي الجهد: ~5 أيام للـ P0+P1، نص يوم للـ P2، نص يوم للـ P3**

---

*GeoCore Next — Gaps Implementation Plan | April 2026*
