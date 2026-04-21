# Engineering Delivery & Verification Playbook

> **Scope:** هذا الدليل إلزامي على كل مهمة تم تنفيذها سابقًا، وكل مهمة جديدة لاحقًا.
> 
> **Audience:** أي مبرمج داخل المشروع (Backend / Frontend / Fullstack / QA).

---

## 1) الهدف من الدليل

هذا الدليل يضمن أن أي Feature أو Fix:
1. يتنفذ بنفس التقنية والمعمارية المتفق عليها.
2. يندمج بدون كسر أجزاء أخرى (No Regression).
3. يتوثق بأدلة قابلة للمراجعة.
4. لا يتم اعتباره "Done" إلا بعد اختبارات واضحة.

---

## 2) القاعدة الذهبية (Definition of Done)

لا يتم تعليم أي Task كـ `Completed` في `TASKS.md` إلا إذا تحققت كل البنود التالية:

- [ ] Acceptance Criteria بالكامل متحقق.
- [ ] Build ناجح (بدون أخطاء).
- [ ] اختبارات Unit/Integration المطلوبة ناجحة.
- [ ] Auth/Authorization متحقق (خصوصًا الـ scoped data).
- [ ] Side effects validated (DB updates, jobs, external APIs).
- [ ] Logs/metrics لا تظهر أخطاء غير متوقعة.
- [ ] Evidence مرفق (أوامر + نتائج + لقطات/JSON responses عند اللزوم).

---

## 3) Unified Execution Flow (لكل مهمة)

### Step 1: فهم المهمة قبل الكود
- اقرأ Acceptance Criteria حرفيًا من `TASKS.md`.
- حدد Dependencies (DB, Redis, Queue, Stripe, إلخ).
- اعمل "Impact Map": ما الملفات/الخدمات المتأثرة؟

### Step 2: التنفيذ
- التزم بنفس Stack الحالية (لا تغييرات تقنية بدون قرار صريح).
- نفذ أقل تغيير يحقق المطلوب (Minimal, Root-cause fix).
- حافظ على نمط الكود القائم (naming, structure, middleware usage).

### Step 3: التحقق المحلي
- Backend:
  - `go build ./...`
  - `go test ./...` (إن وجدت/تمت إضافتها)
- Frontend:
  - `npm run build`
  - `npm run dev` + smoke flows

### Step 4: Verification Matrix
- حول كل Acceptance Criterion إلى حالة اختبار صريحة (pass/fail).
- اختبر:
  - Happy path
  - Validation failures
  - Authorization failures
  - Idempotency / retry behavior (للـ jobs + payments)

### Step 5: Evidence & Sign-off
- وثق النتائج في ملف التحقق (انظر Template بالأسفل).
- بعد اكتمال الأدلة فقط:
  - حدث `TASKS.md` إلى Completed.
  - اذكر الأوامر المنفذة ونتائجها.

---

## 4) Backend Verification Checklist

استخدم هذا الجزء في أي Task Backend:

### A) API Contract
- [ ] المسارات الصحيحة موجودة.
- [ ] Methods صحيحة (GET/POST/PATCH/...).
- [ ] HTTP codes صحيحة (200/201/400/401/403/404/500).
- [ ] Response schema مطابق للمطلوب.

### B) Security & Access
- [ ] `Auth()` مفعل حيث يلزم.
- [ ] Admin-only endpoints محمية فعليًا.
- [ ] المستخدم لا يرى بيانات مستخدم آخر.

### C) Data Integrity
- [ ] التحديثات في DB تحدث مرة واحدة وبالحالة الصحيحة.
- [ ] Status transitions valid.
- [ ] لا يوجد partial update بدون rollback عند الخطأ.

### D) Async Jobs / External Services
- [ ] Jobs يتم enqueue وprocess بشكل صحيح.
- [ ] External API failures handled (timeouts/errors/retries).
- [ ] No duplicate financial effects (refund/release).

### E) Performance Baseline
- [ ] p95 ضمن الهدف المتفق عليه.
- [ ] Error rate < 1% تحت حمل الاختبار.
- [ ] Queue lag ضمن الحدود.

---

## 5) Frontend Verification Checklist

- [ ] الصفحات المطلوبة موجودة ومساراتها صحيحة.
- [ ] API integration تعمل مع backend الحقيقي.
- [ ] States واضحة: loading / empty / error / success.
- [ ] Mobile + desktop usable.
- [ ] لا أخطاء SSR/CSR (خصوصًا App Router hooks).
- [ ] Build ناجح بدون crash.

---

## 6) Regression Guardrails

قبل إغلاق أي Task:
- [ ] شغّل smoke على أهم الرحلات (Auth → Listings → Buy/Bid → Checkout/Orders).
- [ ] تأكد أن تغييراتك لم تكسر مهام مكتملة سابقًا.
- [ ] راجع logs بعد التشغيل للتأكد من عدم ظهور أخطاء جديدة.

---

## 7) Load/Stress Standard (مختصر موحّد)

اختبر endpoints الجديدة بهذه الأهداف الافتراضية (ما لم يحدد task خلاف ذلك):

- Error rate < **1%**
- p95 latency:
  - Read endpoints: < **300ms**
  - Write/financial endpoints: < **700ms**
- p99 < **2x p95**
- No sustained queue lag > **60s**

> في حال فشل أي حد: المهمة لا تعتبر مكتملة حتى يتم الإصلاح وإعادة الاختبار.

---

## 8) Sign-off Template (انسخها لكل Task)

```md
## Task Sign-off: TASK-XXX

### 1) Acceptance Criteria Mapping
- [ ] AC-1: ... (evidence: ...)
- [ ] AC-2: ... (evidence: ...)

### 2) Commands Executed
- [ ] go build ./...  -> PASS
- [ ] go test ./...   -> PASS/NA
- [ ] npm run build   -> PASS/NA

### 3) Functional Test Cases
- [ ] Case-1 ... -> PASS
- [ ] Case-2 ... -> PASS

### 4) Auth & Access Checks
- [ ] Unauthorized -> blocked
- [ ] Cross-user access -> blocked

### 5) Data/Side Effects Validation
- [ ] DB state correct
- [ ] Jobs processed
- [ ] External API side effects correct

### 6) Performance Snapshot
- [ ] Error rate: ...
- [ ] p95: ...
- [ ] p99: ...

### 7) Final Decision
- [ ] DONE (all gates passed)
- [ ] NOT DONE (issues listed below)

### 8) Open Issues (if any)
- ...
```

---

## 9) Mandatory Team Rule

أي مبرمج يعدّل حالة Task إلى `Completed` بدون Evidence واضح يعتبر الإغلاق غير صالح ويجب إعادة الفتح.

---

## 10) Quick Start لأي مبرمج جديد

1. اقرأ `TASKS.md` وحدد الـ Task الحالية.
2. نفّذ التعديل بأقل scope ممكن.
3. اتبع checklists في هذا الملف.
4. املأ Sign-off Template.
5. بعد نجاح كل gate فقط علّم المهمة Completed.
