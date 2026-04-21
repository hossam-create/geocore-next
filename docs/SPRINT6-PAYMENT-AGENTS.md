# 💳 Sprint 6 — Agent Payment System
> **النموذج:** Agent Banking (مثل Fawry + M-Pesa + Western Union Agents)
> **الهدف:** نظام دفع كامل عبر وكلاء معتمدين — جاهز للترخيص
> **القاعدة:** كل عملية مالية تمر عبر Escrow — مفيش استثناء

---

## 🏗️ الهيكل الكامل

```
internal/payments/
├── agents.go          ← Agent management + KYC
├── deposit.go         ← Deposit via agent flow
├── withdraw.go        ← Withdraw via agent flow
├── liquidity.go       ← Agent balance limits + utilization
├── fx.go              ← Currency layer (USD base)
└── sprint6_test.go    ← Tests

internal/wallet/
└── escrow.go          ← Add new states (MODIFY)
```

---

## STEP 1 — DB Migrations

```sql
-- migrations/YYYYMMDD_payment_agents.sql

CREATE TABLE payment_agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    status          VARCHAR(20) DEFAULT 'pending'
                    CHECK (status IN ('pending','active','suspended','terminated')),
    country         VARCHAR(3) NOT NULL,
    currency        VARCHAR(3) NOT NULL,
    balance_limit   DECIMAL(15,2) NOT NULL DEFAULT 10000.00,
    current_balance DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    collateral_held DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    trust_score     INTEGER NOT NULL DEFAULT 50
                    CHECK (trust_score BETWEEN 0 AND 100),
    payment_methods JSONB DEFAULT '[]',
    -- مثال: [{"type":"instapay","identifier":"01xxxxxxxxx"},
    --         {"type":"vodafone_cash","number":"01xxxxxxxxx"}]
    approved_by     UUID REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    suspension_reason TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_pa_user ON payment_agents(user_id);
CREATE INDEX idx_pa_country_currency ON payment_agents(country, currency)
    WHERE status = 'active';

-- -------------------------------------------------------

CREATE TABLE deposit_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    agent_id    UUID NOT NULL REFERENCES payment_agents(id),
    amount      DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency    VARCHAR(3) NOT NULL,
    usd_amount  DECIMAL(15,2) NOT NULL,  -- المبلغ بالـ base currency
    fx_rate     DECIMAL(10,6) NOT NULL,  -- السعر المستخدم
    status      VARCHAR(20) DEFAULT 'pending'
                CHECK (status IN ('pending','paid','confirmed','rejected','expired')),
    proof_url   VARCHAR(500),            -- screenshot الإيصال
    proof_uploaded_at TIMESTAMPTZ,
    confirmed_by_agent_at TIMESTAMPTZ,
    rejection_reason TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    expires_at  TIMESTAMPTZ DEFAULT NOW() + INTERVAL '30 minutes',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_dr_user ON deposit_requests(user_id, created_at DESC);
CREATE INDEX idx_dr_agent ON deposit_requests(agent_id, status);
CREATE INDEX idx_dr_expires ON deposit_requests(expires_at)
    WHERE status = 'pending';

-- -------------------------------------------------------

CREATE TABLE withdraw_requests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    agent_id    UUID REFERENCES payment_agents(id),
    amount      DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency    VARCHAR(3) NOT NULL,
    usd_amount  DECIMAL(15,2) NOT NULL,
    fx_rate     DECIMAL(10,6) NOT NULL,
    recipient_details JSONB NOT NULL,
    -- {"type":"instapay","identifier":"01xxxxxxxxx","name":"Ahmed"}
    status      VARCHAR(20) DEFAULT 'pending'
                CHECK (status IN ('pending','assigned','processing','completed','failed','cancelled')),
    assigned_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failure_reason TEXT,
    idempotency_key VARCHAR(255) UNIQUE,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wr_user ON withdraw_requests(user_id, created_at DESC);
CREATE INDEX idx_wr_agent ON withdraw_requests(agent_id, status);

-- -------------------------------------------------------

CREATE TABLE agent_liquidity_log (
    id          BIGSERIAL PRIMARY KEY,
    agent_id    UUID NOT NULL REFERENCES payment_agents(id),
    event_type  VARCHAR(30) NOT NULL,
    -- deposit_confirmed | withdraw_completed | balance_adjusted
    amount      DECIMAL(15,2) NOT NULL,
    balance_before DECIMAL(15,2) NOT NULL,
    balance_after  DECIMAL(15,2) NOT NULL,
    reference_id UUID,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_all_agent ON agent_liquidity_log(agent_id, created_at DESC);

-- -------------------------------------------------------

-- VIP Users table
CREATE TABLE vip_users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) UNIQUE,
    tier            VARCHAR(20) DEFAULT 'silver'
                    CHECK (tier IN ('silver','gold','platinum')),
    daily_limit     DECIMAL(15,2) NOT NULL,
    monthly_limit   DECIMAL(15,2) NOT NULL,
    transfer_fee_pct DECIMAL(5,4) DEFAULT 0.005,  -- 0.5% بدل 1%
    priority_matching BOOLEAN DEFAULT TRUE,
    fast_track_kyc  BOOLEAN DEFAULT FALSE,
    dedicated_agent_id UUID REFERENCES payment_agents(id),
    activated_at    TIMESTAMPTZ DEFAULT NOW(),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- VIP Limits:
-- Silver:   daily $1,000 / monthly $10,000 / fee 0.8%
-- Gold:     daily $5,000 / monthly $50,000 / fee 0.5%
-- Platinum: daily $20,000 / monthly $200,000 / fee 0.3%
```

---

## STEP 2 — Agents System (`agents.go`)

```go
package payments

import (
    "github.com/geocore-next/backend/internal/kyc"
    "github.com/geocore-next/backend/internal/wallet"
    "github.com/geocore-next/backend/internal/security"
)

// ========== MODELS ==========

type PaymentAgent struct {
    ID             uuid.UUID  `gorm:"primaryKey;type:uuid"`
    UserID         uuid.UUID  `gorm:"not null;uniqueIndex"`
    Status         string     `gorm:"default:'pending'"`
    Country        string     `gorm:"not null;size:3"`
    Currency       string     `gorm:"not null;size:3"`
    BalanceLimit   decimal.Decimal
    CurrentBalance decimal.Decimal
    CollateralHeld decimal.Decimal
    TrustScore     int        `gorm:"default:50"`
    PaymentMethods datatypes.JSON
    ApprovedBy     *uuid.UUID
    ApprovedAt     *time.Time
    SuspensionReason *string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// ========== HANDLERS ==========

// POST /api/v1/admin/agents/register
func (h *Handler) RegisterAgent(c *gin.Context) {
    var req struct {
        UserID         uuid.UUID       `json:"user_id" binding:"required"`
        Country        string          `json:"country" binding:"required,len=2"`
        Currency       string          `json:"currency" binding:"required,len=3"`
        BalanceLimit   decimal.Decimal `json:"balance_limit" binding:"required"`
        CollateralAmount decimal.Decimal `json:"collateral_amount" binding:"required"`
        PaymentMethods []PaymentMethod `json:"payment_methods" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.R(c, 400, nil, err.Error())
        return
    }

    // تحقق KYC مكتمل
    var kycProfile kyc.KYCProfile
    if err := h.db.Where("user_id = ? AND status = 'approved'", req.UserID).
        First(&kycProfile).Error; err != nil {
        response.R(c, 400, nil, "agent must have approved KYC")
        return
    }

    err := h.db.Transaction(func(tx *gorm.DB) error {
        agent := PaymentAgent{
            ID:             uuid.New(),
            UserID:         req.UserID,
            Country:        req.Country,
            Currency:       req.Currency,
            BalanceLimit:   req.BalanceLimit,
            CollateralHeld: req.CollateralAmount,
            PaymentMethods: req.PaymentMethods,
            TrustScore:     50,
        }
        if err := tx.Create(&agent).Error; err != nil {
            return err
        }

        // احجز الـ collateral في escrow
        if err := wallet.HoldCollateral(tx, req.UserID, agent.ID,
            req.CollateralAmount); err != nil {
            return fmt.Errorf("collateral hold failed: %w", err)
        }

        security.LogAdminAction(tx, c.GetUint("user_id"),
            "agent_registered", "payment_agent", agent.ID, nil)
        return nil
    })

    if err != nil {
        response.R(c, 500, nil, err.Error())
        return
    }
    response.R(c, 201, gin.H{"message": "agent registered, pending approval"}, "")
}

// PUT /api/v1/admin/agents/:id/approve
func (h *Handler) ApproveAgent(c *gin.Context) {
    agentID := uuid.MustParse(c.Param("id"))
    adminID := c.GetUUID("user_id")

    var agent PaymentAgent
    h.db.Set("gorm:query_option", "FOR UPDATE").First(&agent, agentID)
    if agent.Status != "pending" {
        response.R(c, 400, nil, "agent is not in pending status")
        return
    }

    h.db.Model(&agent).Updates(map[string]interface{}{
        "status":      "active",
        "approved_by": adminID,
        "approved_at": time.Now(),
    })

    security.LogAdminAction(h.db, adminID, "agent_approved",
        "payment_agent", agentID, nil)
    response.R(c, 200, gin.H{"message": "agent approved"}, "")
}

// PUT /api/v1/admin/agents/:id/suspend
func (h *Handler) SuspendAgent(c *gin.Context) {
    // suspend + freeze pending requests
}

// GET /api/v1/payments/agents/available?country=EG&currency=EGP
func (h *Handler) GetAvailableAgents(c *gin.Context) {
    country := c.Query("country")
    currency := c.Query("currency")
    amount, _ := decimal.NewFromString(c.Query("amount"))

    var agents []AgentPublicView
    h.db.Select("id, trust_score, payment_methods, "+
        "(balance_limit - current_balance) as available_capacity").
        Where("country = ? AND currency = ? AND status = 'active'"+
              " AND (balance_limit - current_balance) >= ?",
              country, currency, amount).
        Order("trust_score DESC, available_capacity DESC").
        Find(&agents)

    response.R(c, 200, agents, "")
}
```

---

## STEP 3 — Deposit Flow (`deposit.go`)

```go
// POST /api/v1/payments/deposit/initiate
func (h *Handler) InitiateDeposit(c *gin.Context) {
    var req struct {
        AgentID        uuid.UUID       `json:"agent_id" binding:"required"`
        Amount         decimal.Decimal `json:"amount" binding:"required"`
        Currency       string          `json:"currency" binding:"required"`
        IdempotencyKey string          `json:"idempotency_key" binding:"required"`
    }
    c.ShouldBindJSON(&req)
    userID := c.GetUUID("user_id")

    // Idempotency check
    var existing DepositRequest
    if h.db.Where("idempotency_key = ? AND user_id = ?",
        req.IdempotencyKey, userID).First(&existing).Error == nil {
        response.R(c, 200, existing, "")
        return
    }

    // Trust gate check
    if result := fraud.CheckTrustGate(h.db, h.redis, userID, "deposit",
        req.Amount); !result.Allowed {
        response.R(c, 403, nil, "deposit blocked: "+strings.Join(result.Flags, ", "))
        return
    }

    // جيب الـ agent وتحقق من الـ capacity
    var agent PaymentAgent
    h.db.Set("gorm:query_option", "FOR UPDATE").First(&agent, req.AgentID)
    if agent.Status != "active" {
        response.R(c, 400, nil, "agent not available")
        return
    }

    // FX conversion
    fxRate := h.fx.GetRate(req.Currency, "USD")
    usdAmount := req.Amount.Div(fxRate)

    // VIP daily/monthly limit check
    if err := checkUserLimits(h.db, userID, usdAmount, "deposit"); err != nil {
        response.R(c, 400, nil, err.Error())
        return
    }

    deposit := DepositRequest{
        ID:             uuid.New(),
        UserID:         userID,
        AgentID:        req.AgentID,
        Amount:         req.Amount,
        Currency:       req.Currency,
        USDAmount:      usdAmount,
        FXRate:         fxRate,
        IdempotencyKey: req.IdempotencyKey,
        ExpiresAt:      time.Now().Add(30 * time.Minute),
    }
    h.db.Create(&deposit)

    // رجّع تعليمات الدفع للـ user
    var agentMethods []PaymentMethod
    json.Unmarshal(agent.PaymentMethods, &agentMethods)

    response.R(c, 201, gin.H{
        "deposit_id":        deposit.ID,
        "amount":            req.Amount,
        "currency":          req.Currency,
        "payment_methods":   agentMethods,
        "instructions":      buildPaymentInstructions(agentMethods, req.Amount, req.Currency),
        "expires_at":        deposit.ExpiresAt,
        "reference":         deposit.ID.String()[:8], // رقم مرجعي قصير
    }, "")
}

// POST /api/v1/payments/deposit/:id/upload-proof
func (h *Handler) UploadDepositProof(c *gin.Context) {
    // المستخدم يرفع screenshot الإيصال
    // يرجع proof_url بعد رفعها على R2
}

// POST /api/v1/payments/deposit/:id/confirm (agents only)
func (h *Handler) AgentConfirmDeposit(c *gin.Context) {
    depositID := uuid.MustParse(c.Param("id"))
    agentUserID := c.GetUUID("user_id")

    err := h.db.Transaction(func(tx *gorm.DB) error {
        var deposit DepositRequest
        tx.Set("gorm:query_option", "FOR UPDATE").First(&deposit, depositID)

        // تحقق إن الـ agent هو صاحب الـ request
        var agent PaymentAgent
        tx.Where("user_id = ?", agentUserID).First(&agent)
        if agent.ID != deposit.AgentID {
            return errors.New("unauthorized")
        }
        if deposit.Status != "pending" && deposit.Status != "paid" {
            return errors.New("invalid deposit status")
        }
        if time.Now().After(deposit.ExpiresAt) {
            return errors.New("deposit expired")
        }

        // 1. Credit user wallet
        if err := wallet.ApplyDeposit(tx, deposit.UserID,
            deposit.USDAmount, deposit.ID); err != nil {
            return err
        }

        // 2. Update agent balance
        tx.Model(&agent).Update("current_balance",
            gorm.Expr("current_balance + ?", deposit.USDAmount))

        // 3. Log liquidity event
        tx.Create(&AgentLiquidityLog{
            AgentID:       agent.ID,
            EventType:     "deposit_confirmed",
            Amount:        deposit.USDAmount,
            BalanceBefore: agent.CurrentBalance,
            BalanceAfter:  agent.CurrentBalance.Add(deposit.USDAmount),
            ReferenceID:   &depositID,
        })

        // 4. Update deposit status
        tx.Model(&deposit).Updates(map[string]interface{}{
            "status":                  "confirmed",
            "confirmed_by_agent_at":  time.Now(),
        })

        // 5. Financial audit log
        financial.LogEvent(tx, financial.FinancialEvent{
            UserID:    deposit.UserID,
            EventType: "deposit_confirmed",
            Amount:    deposit.USDAmount,
            Status:    "success",
            Metadata: map[string]interface{}{
                "agent_id": agent.ID,
                "currency": deposit.Currency,
                "fx_rate":  deposit.FXRate,
            },
        })

        return nil
    })

    if err != nil {
        response.R(c, 400, nil, err.Error())
        return
    }

    // Notify user
    h.notifications.Notify(deposit.UserID, "deposit_confirmed",
        fmt.Sprintf("تم إيداع $%.2f في محفظتك", deposit.USDAmount.InexactFloat64()))

    response.R(c, 200, gin.H{"message": "deposit confirmed"}, "")
}
```

---

## STEP 4 — Withdraw Flow (`withdraw.go`)

```go
// POST /api/v1/payments/withdraw/request
func (h *Handler) RequestWithdraw(c *gin.Context) {
    var req struct {
        Amount           decimal.Decimal        `json:"amount" binding:"required"`
        Currency         string                 `json:"currency" binding:"required"`
        RecipientDetails map[string]interface{} `json:"recipient_details" binding:"required"`
        // {"type":"instapay","identifier":"01xxxxxxxxx","name":"Ahmed Mohamed"}
        IdempotencyKey   string                 `json:"idempotency_key" binding:"required"`
    }
    userID := c.GetUUID("user_id")

    // Fraud gate
    result := fraud.CheckRiskBeforeWithdraw(h.db, h.redis, userID, req.Amount)
    if !result.Allowed {
        response.R(c, 403, nil, "withdraw blocked: high risk")
        return
    }

    // Fast withdraw after deposit check (24h cooldown for new users)
    if err := checkFastWithdrawAbuse(h.db, h.redis, userID); err != nil {
        response.R(c, 429, nil, err.Error())
        return
    }

    fxRate := h.fx.GetRate("USD", req.Currency)
    usdAmount := req.Amount.Div(fxRate)

    err := h.db.Transaction(func(tx *gorm.DB) error {
        // Lock wallet balance
        var walletBal wallet.WalletBalance
        tx.Set("gorm:query_option", "FOR UPDATE").
            Where("user_id = ?", userID).First(&walletBal)

        if walletBal.AvailableBalance.LessThan(usdAmount) {
            return wallet.ErrInsufficientBalance
        }

        // Debit user wallet immediately (hold it)
        tx.Model(&walletBal).Update("available_balance",
            gorm.Expr("available_balance - ?", usdAmount))
        tx.Model(&walletBal).Update("pending_balance",
            gorm.Expr("pending_balance + ?", usdAmount))

        // Find best available agent
        agent := findBestAgent(tx, req.Currency, usdAmount)
        var agentID *uuid.UUID
        if agent != nil {
            agentID = &agent.ID
        }

        withdraw := WithdrawRequest{
            ID:               uuid.New(),
            UserID:           userID,
            AgentID:          agentID,
            Amount:           req.Amount,
            Currency:         req.Currency,
            USDAmount:        usdAmount,
            FXRate:           fxRate,
            RecipientDetails: req.RecipientDetails,
            IdempotencyKey:   req.IdempotencyKey,
        }
        return tx.Create(&withdraw).Error
    })

    response.R(c, 201, gin.H{"message": "withdraw request submitted"}, "")
}

// PUT /api/v1/payments/withdraw/:id/complete (agents only)
func (h *Handler) AgentCompleteWithdraw(c *gin.Context) {
    // Agent confirms money sent → release pending_balance
}
```

---

## STEP 5 — Liquidity Control (`liquidity.go`)

```go
func GetAgentUtilization(db *gorm.DB, agentID uuid.UUID) AgentUtilization {
    var agent PaymentAgent
    db.First(&agent, agentID)

    utilization := agent.CurrentBalance.Div(agent.BalanceLimit).
        Mul(decimal.NewFromInt(100))

    var level string
    switch {
    case utilization.GreaterThanOrEqual(decimal.NewFromInt(90)):
        level = "critical"   // لازم يجيب سيولة
    case utilization.GreaterThanOrEqual(decimal.NewFromInt(70)):
        level = "warning"
    default:
        level = "healthy"
    }

    return AgentUtilization{
        AgentID:        agentID,
        Utilization:    utilization,
        Available:      agent.BalanceLimit.Sub(agent.CurrentBalance),
        Level:          level,
    }
}

func checkAgentCapacity(db *gorm.DB, agentID uuid.UUID,
    amount decimal.Decimal) error {
    var agent PaymentAgent
    db.First(&agent, agentID)
    if agent.CurrentBalance.Add(amount).GreaterThan(agent.BalanceLimit) {
        return errors.New("agent capacity exceeded")
    }
    return nil
}

func findBestAgent(db *gorm.DB, currency string,
    amount decimal.Decimal) *PaymentAgent {
    var agent PaymentAgent
    db.Where("currency = ? AND status = 'active'"+
             " AND (balance_limit - current_balance) >= ?"+
             " AND trust_score >= 60",
             currency, amount).
        Order("trust_score DESC, (balance_limit - current_balance) DESC").
        First(&agent)
    if agent.ID == uuid.Nil {
        return nil
    }
    return &agent
}
```

---

## STEP 6 — FX Layer (`fx.go`)

```go
// MVP: USD as base currency
// Agents handle local conversion externally
// Rates from exchange_rates table (Sprint 5)

type FXService struct {
    db    *gorm.DB
    redis *redis.Client
}

func (fx *FXService) GetRate(from, to string) decimal.Decimal {
    if from == to {
        return decimal.NewFromInt(1)
    }

    // Cache في Redis (5 دقايق)
    cacheKey := fmt.Sprintf("fx_rate:%s:%s", from, to)
    cached, err := fx.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        rate, _ := decimal.NewFromString(cached)
        return rate
    }

    // من DB (exchange_rates table)
    var rate ExchangeRate
    fx.db.Where("base_currency = ? AND quote_currency = ? AND is_active = true",
        from, to).First(&rate)
    if rate.Rate.IsZero() {
        return decimal.NewFromInt(1) // fallback
    }

    fx.redis.Set(ctx, cacheKey, rate.Rate.String(), 5*time.Minute)
    return rate.Rate
}
```

---

## STEP 7 — VIP System

```go
// VIP Tiers & Limits
var VIPTiers = map[string]VIPTierConfig{
    "silver": {
        DailyLimit:      decimal.NewFromFloat(1000),
        MonthlyLimit:    decimal.NewFromFloat(10000),
        TransferFeePct:  decimal.NewFromFloat(0.008),   // 0.8%
        PriorityMatching: true,
        WithdrawSpeedH:   4,   // 4 ساعات مثل Western Union
    },
    "gold": {
        DailyLimit:      decimal.NewFromFloat(5000),
        MonthlyLimit:    decimal.NewFromFloat(50000),
        TransferFeePct:  decimal.NewFromFloat(0.005),   // 0.5%
        PriorityMatching: true,
        FastTrackKYC:    true,
        WithdrawSpeedH:   2,   // ساعتين
        DedicatedAgent:  true,
    },
    "platinum": {
        DailyLimit:      decimal.NewFromFloat(20000),
        MonthlyLimit:    decimal.NewFromFloat(200000),
        TransferFeePct:  decimal.NewFromFloat(0.003),   // 0.3%
        PriorityMatching: true,
        FastTrackKYC:    true,
        WithdrawSpeedH:   1,   // ساعة واحدة — زي Western Union Gold
        DedicatedAgent:  true,
        ZeroFeeTransfers: 3,  // 3 تحويلات مجانية شهرياً
    },
}

// POST /api/v1/admin/users/:id/vip
func (h *Handler) UpgradeToVIP(c *gin.Context) {
    // Admin يرفع المستخدم لـ VIP tier
    // بعد KYC approved + minimum transaction history
}
```

---

## STEP 8 — Fraud Checks Integration

```go
// internal/fraud/payment_gates.go

func CheckTrustGate(db *gorm.DB, redis *redis.Client,
    userID uuid.UUID, operation string, amount decimal.Decimal) FraudResult {

    result := FraudResult{Allowed: true, RiskScore: 0}

    switch operation {
    case "deposit":
        // Rule 1: أكثر من 3 deposits في ساعة
        key := fmt.Sprintf("deposit_count:%s", userID)
        count, _ := redis.Incr(ctx, key).Result()
        redis.Expire(ctx, key, time.Hour)
        if count > 3 {
            result.RiskScore += 40
            result.Flags = append(result.Flags, "repeated_deposits")
        }

        // Rule 2: نفس الـ agent دايماً (agent abuse)
        if checkSameAgentAbuse(db, redis, userID) {
            result.RiskScore += 30
            result.Flags = append(result.Flags, "same_agent_abuse")
        }

    case "withdraw":
        // Rule 3: سحب سريع بعد إيداع (< 2 ساعة)
        if checkFastWithdrawAfterDeposit(db, userID) {
            result.RiskScore += 50
            result.Flags = append(result.Flags, "fast_withdraw_after_deposit")
        }
    }

    if result.RiskScore >= 70 {
        result.Allowed = false
        result.Action = "block"
    } else if result.RiskScore >= 40 {
        result.Action = "review"
    }

    return result
}
```

---

## STEP 9 — Routes

```go
// wire في payments/routes.go

payments := r.Group("/api/v1/payments", authMiddleware)
{
    // Deposit
    payments.POST("/deposit/initiate", h.InitiateDeposit)
    payments.POST("/deposit/:id/upload-proof", h.UploadDepositProof)
    payments.GET("/deposit/:id/status", h.GetDepositStatus)
    payments.GET("/deposit/history", h.GetDepositHistory)

    // Withdraw
    payments.POST("/withdraw/request", h.RequestWithdraw)
    payments.DELETE("/withdraw/:id/cancel", h.CancelWithdraw)
    payments.GET("/withdraw/history", h.GetWithdrawHistory)

    // Agents (public)
    payments.GET("/agents/available", h.GetAvailableAgents)
}

// Agent endpoints (agents only)
agentRoutes := r.Group("/api/v1/agent", authMiddleware, agentOnlyMiddleware)
{
    agentRoutes.GET("/requests", h.GetAgentPendingRequests)
    agentRoutes.POST("/deposit/:id/confirm", h.AgentConfirmDeposit)
    agentRoutes.POST("/deposit/:id/reject", h.AgentRejectDeposit)
    agentRoutes.POST("/withdraw/:id/complete", h.AgentCompleteWithdraw)
    agentRoutes.GET("/liquidity", h.GetMyLiquidity)
}

// Admin
adminPayments := r.Group("/api/v1/admin/payments", adminMiddleware)
{
    adminPayments.POST("/agents/register", h.RegisterAgent)
    adminPayments.PUT("/agents/:id/approve", h.ApproveAgent)
    adminPayments.PUT("/agents/:id/suspend", h.SuspendAgent)
    adminPayments.GET("/agents", h.ListAllAgents)
    adminPayments.GET("/agents/:id/utilization", h.GetAgentUtilization)
    adminPayments.GET("/dashboard", h.GetPaymentDashboard)

    // VIP
    adminPayments.POST("/users/:id/vip", h.UpgradeToVIP)
    adminPayments.PUT("/users/:id/vip/tier", h.UpdateVIPTier)
}
```

---

## STEP 10 — Tests (`sprint6_test.go`)

```go
func TestDepositFlow(t *testing.T) {
    // 1. Register + approve agent
    // 2. User initiates deposit
    // 3. User uploads proof
    // 4. Agent confirms
    // 5. Assert: wallet balance increased
    // 6. Assert: agent current_balance increased
    // 7. Assert: financial_events log entry exists
}

func TestWithdrawFlow(t *testing.T) {
    // 1. User has balance
    // 2. Request withdraw
    // 3. Assert: pending_balance increased
    // 4. Agent completes
    // 5. Assert: balance debited
    // 6. Assert: agent current_balance decreased
}

func TestAgentCapacityLimit(t *testing.T) {
    // Agent with limit $1000, current $900
    // Try deposit $200 → should reject
    // Try deposit $50 → should accept
}

func TestFraudGate_FastWithdraw(t *testing.T) {
    // Deposit → immediately try withdraw → should block
}

func TestVIPLimits(t *testing.T) {
    // Silver user: try deposit $2000 → block (daily $1000)
    // Gold user: try deposit $2000 → allow
}
```

---

## ⚡ Western Union Speed — كيف نحققها

```
عادي:      Deposit → manual review → confirm (2-4 ساعة)
VIP Gold:  Dedicated agent → auto-confirm threshold ≤$500 (2 ساعة)
VIP Plat:  Dedicated agent + auto-confirm threshold ≤$2000 (1 ساعة)

السر في السرعة:
1. Pre-approved agents بـ trust_score ≥ 80 → auto-confirm صغار
2. Dedicated agents للـ VIP → مش بيتشاركوا مع عامة الناس
3. Smart agent matching → أقرب agent بـ أعلى capacity
4. Timeout 30 دقيقة → agent مش بيأخر
```

---

## 📊 Compliance Hooks (جاهزة للترخيص)

```go
// كل حاجة موثقة وجاهزة لـ CBE/FRA audit:

// 1. كل transaction فيها audit trail كامل
// 2. KYC إجباري للـ agents
// 3. Balance limits per agent
// 4. Transaction limits per user tier
// 5. Fraud detection مدمج
// 6. Full financial_events log
// 7. Agent collateral (ضمان)

// لما تجيب الترخيص:
// - Replace agent confirmation بـ bank API
// - Keep نفس الـ flow والـ models
// - Upgrade فقط الـ confirmation mechanism
```

---

## ✅ Verification

```bash
go build ./...
go test ./internal/payments/... -v -run TestDepositFlow
go test ./internal/payments/... -v -run TestWithdrawFlow
go test ./internal/payments/... -v -run TestAgentCapacity
go test ./internal/payments/... -v -run TestFraudGate
go test ./internal/payments/... -v -run TestVIPLimits

# Check DB
psql $DATABASE_URL -c "SELECT * FROM payment_agents LIMIT 5;"
psql $DATABASE_URL -c "SELECT * FROM deposit_requests LIMIT 5;"
```
