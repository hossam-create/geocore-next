# GeoCore Next — Senior Engineer Onboarding
**تاريخ الفحص:** 2026-04-05
**الفاحص:** AI Senior Engineer Audit

---

## 📊 نظرة عامة
- إجمالي ملفات Go: **170**
- إجمالي ملفات TypeScript: **652**
- عدد الـ packages في `backend/internal/`: **39**
- عدد الـ API endpoints (Gin route registrations): **326**
- عدد الـ DB migrations (`backend/migrations/*.sql`): **44**
- عدد صفحات frontend (`frontend/app/**/page.tsx`): **78**

> ملاحظة: يوجد مجلد mirror إضافي داخل المشروع باسم `geocore-next/`، وتم استبعاده من العد الرئيسي لتفادي تضخيم الأرقام.

---

## 🗂️ الـ Packages وحالتها

| Package | الغرض | الحالة | متربط بـ |
|---------|-------|--------|----------|
| admin | Admin APIs + management | ✅ | main, auth middleware |
| aichat | AI chat endpoints | ✅ | main |
| analytics | analytics endpoints | ✅ | main |
| arpreview | AR preview APIs | ✅ | listings |
| auctions | auctions + realtime hub | ✅ | notifications, payments |
| auth | auth + JWT + registration | ✅ | users, middleware, notifications |
| blockchain | blockchain escrow APIs | ✅ | payments |
| bnpl | BNPL webhooks/APIs | ✅ | payments/orders (signature + status mapping added) |
| cart | cart APIs | ✅ | users, listings |
| chat | chat APIs + notification hooks | ✅ | notifications |
| crowdshipping | traveler/request logistics | ✅ | requests |
| crypto | crypto webhook handlers | ✅ | payments (charge status reconciliation added) |
| deals | deals endpoints | ✅ | main, listings, users |
| disputes | disputes APIs | ✅ | orders, users |
| fraud | fraud APIs/rules | ✅ | admin/ops |
| images | media upload/image APIs | ✅ | listings |
| kyc | KYC APIs | ✅ | users |
| listings | listings CRUD/search | ✅ | images, categories, moderation |
| livestream | livestream APIs | ✅ | listings/users |
| loyalty | points/referrals balance APIs | ✅ | users/orders |
| moderation | moderation core logic | ✅ | listings (logic layer) |
| notifications | in-app + push notifications | ✅ | auth/auctions/chat/ops |
| ops | operations dashboards + jobs | ✅ | jobs, analytics |
| order | orders lifecycle | ✅ | payments/referrals (escrow release/refund transitions added) |
| p2p | p2p exchange APIs | ✅ | wallet/users |
| payments | payments + webhooks | ✅ | orders/auctions |
| plugins | plugin marketplace APIs | ✅ | listings |
| referral | referral APIs | ✅ | users/orders |
| reports | reporting APIs | ✅ | users/admin |
| requests | product request APIs | ✅ | listings |
| reverseauctions | reverse auction APIs | ✅ | wallet/orders |
| reviews | reviews/ratings APIs | ✅ | users/listings |
| search | semantic/AI search APIs | ✅ | main, listings |
| stores | seller stores APIs | ✅ | users/listings |
| subscriptions | plan subscriptions APIs | ✅ | payments |
| support | support tickets APIs | ✅ | main, users |
| users | profile/user APIs | ✅ | auth/listings/orders |
| wallet | wallet + escrow endpoints | ✅ | payments/orders/p2p |
| watchlist | favorites/watchlist APIs | ✅ | users/listings |

---

## ⚠️ المشاكل اللي لقيتها

### 🔴 Critical (لازم يتصلح قبل launch)
1. **Repository duplication**: وجود مجلد nested `geocore-next/` داخل root يزيد احتمالات التشغيل من مصدر خاطئ وتضارب التعديلات.

### 🟡 Important (بعد launch)
1. **`frontend-admin/` ما زال موجود** بجانب `admin/` (ازدواجية admin frontends).
2. **BNPL webhook order matching يعتمد على `reference_id` الموجود في payload**؛ يلزم توحيد contract ثابت بين create + webhook لكل provider لتتبع أدق.

### 🟢 Minor (لاحقاً)
1. مجلد غريب تحت `backend/internal/{auth,listings,auctions,chat,payments,users}/` يبدو artifact فارغ.
2. توجد ملفات empty داخل `attached_assets` و`services/xyops/htdocs` وتحتاج مراجعة غرضها قبل التنظيف.

---

## 🧹 الـ Junk اللي اتمسح/اتنقل
- نُقل لـ /archive/: **لم يتم النقل تلقائيًا في هذه الجولة** (awaiting confirmation قبل أي حذف/نقل).
- اتحذف: **لم يتم الحذف تلقائيًا**.

### Junk candidates (للمراجعة قبل التنفيذ)
- Prompt artifacts (`attached_assets/Pasted-*Prompt*.txt` ونسخها في frontend/).
- PowerShell scripts:
  - `frontend/fix-env.ps1`
  - `frontend/migrate.ps1`
  - `services/xyops/internal/sat-install.ps1`
  - `services/xyops/internal/sat-upgrade.ps1`

---

## 📦 Dead Code / Unused Packages
- لا يوجد package dead من الثلاثة (`deals/search/support`) بعد توصيلهم في bootstrap.

---

## 🔗 الـ Missing Connections
- `internal/notifications` موصول مع auctions/chat، لكن coverage مع flows إضافية (مثل payments lifecycle) يحتاج استكمال على مستوى business events.
- `internal/auth` notification service أصبح موصول عبر `auth.SetNotificationService(notifSvc)`.

---

## ✅ الـ .env.example المحدّث
تم تحديث `.env.example` ليتماشى مع المتغيرات المستخدمة فعليًا في الكود.

---

## 🧪 نتايج الاختبار (Step 9)
- تم التنفيذ بعد الموافقة، والنتيجة كالتالي:
  1. `docker-compose up -d postgres redis` ❌
     - السبب: Docker daemon غير متاح على الجهاز (`//./pipe/dockerDesktopLinuxEngine` not found).
  2. `go run cmd/migrate/main.go` ❌
     - السبب: مسار migration command غير موجود في repo الحالي (`cmd/migrate/main.go` not found).
  3. `go run cmd/api/main.go` ✅
     - تم إصلاح compile error في `internal/auth/handler.go`.
     - تم إصلاح تضارب routes على `/api/v1/escrow` بين `wallet` و`blockchain` عبر تغيير prefix blockchain إلى `/api/v1/blockchain/escrow`.
  4. اختبار `GET /health` ✅
     - الاستجابة: `{"status":"ok"...}`.
  5. اختبار `POST /api/v1/auth/register`:
     - ❌ محاولة أولى (password أقل من 10 حروف)
     - ✅ محاولة ثانية ناجحة (user + token returned).
  6. اختبار `GET /api/v1/listings` ✅
     - الاستجابة ناجحة مع `success: true` وmeta pagination.
  7. اختبار `GET /api/v1/search/trending` ✅
  8. اختبار `GET /api/v1/support/subjects` ✅
  9. اختبار `GET /api/v1/deals` ✅
     - كان يرجع `500` بسبب غياب جدول `deals` في قاعدة البيانات.
     - تم الحل بإضافة `deals.Deal` إلى `database.AutoMigrate`.
  10. اختبار `POST /api/v1/support/contact` ✅
      - كان يرجع `500` سابقًا بسبب غياب support models في AutoMigrate.
      - تم الحل بإضافة `support.ContactMessage/SupportTicket/TicketMessage` إلى `database.AutoMigrate`.
  11. اختبار BNPL webhooks (`/api/v1/bnpl/tamara/webhook`, `/api/v1/bnpl/tabby/webhook`) ✅
      - تم تطبيق HMAC verification (عند توفر secrets) + status mapping على orders.
  12. اختبار Coinbase webhook (`/api/v1/crypto/coinbase/webhook`) ✅
      - تم تطبيق verification + charge status reconciliation على payments.

---

## 📋 الـ TODO list للخطوة الجاية
بالترتيب:
1. [ ] توحيد `reference_id` contract بين BNPL create + webhook وتوثيقه رسميًا (Tamara/Tabby).
2. [ ] حسم ازدواجية admin dashboards: الإبقاء على `admin/` وأرشفة `frontend-admin/` بعد مراجعتك.
3. [ ] تنظيف junk candidates ونقل scripts/prompts إلى `archive/` حسب policy.
