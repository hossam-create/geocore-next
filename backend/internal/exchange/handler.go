package exchange

// handler.go — Sprint 19/20 HTTP handlers for the P2P Exchange domain.
//
// All endpoints require authentication (user_id from JWT middleware).
// Trust + freeze checks run on every mutating action.
// Platform is NEVER a payment party — all fund flows are P2P external.

import (
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/reputation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ─── Safety Context (Part 8) ─────────────────────────────────────────────────

// SafetyContext is embedded in responses that involve a counterparty.
type SafetyContext struct {
	Disclaimer        string  `json:"disclaimer"`
	RiskLevel         string  `json:"risk_level"`
	CounterpartyTrust string  `json:"counterparty_trust_level"` // low|medium|high
	CounterpartyScore float64 `json:"counterparty_trust_score"`
	PaymentMethodRisk string  `json:"payment_method_risk,omitempty"`
}

func (h *Handler) buildSafetyContext(counterpartyID uuid.UUID, paymentMethod string) SafetyContext {
	score := reputation.GetOverallScore(h.db, counterpartyID)
	trustLevel := "high"
	switch {
	case score < 40:
		trustLevel = "low"
	case score < 70:
		trustLevel = "medium"
	}
	return SafetyContext{
		Disclaimer:        legalDisclaimer,
		RiskLevel:         RiskLevelForUser(h.db, counterpartyID),
		CounterpartyTrust: trustLevel,
		CounterpartyScore: score,
		PaymentMethodRisk: PaymentMethodRiskLabel(paymentMethod),
	}
}

// Handler holds the DB and Redis dependencies for exchange endpoints.
type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewHandler returns a new exchange Handler.
func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func (h *Handler) mustUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id"})
		return uuid.Nil, false
	}
	return id, true
}

// trustCheck blocks frozen or low-trust users from the exchange.
func (h *Handler) trustCheck(c *gin.Context, userID uuid.UUID) bool {
	if freeze.IsUserFrozen(h.db, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "account is frozen"})
		return false
	}
	if err := reputation.CheckTrustGate(h.db, userID, TrustGateExchange); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return false
	}
	return true
}

// featureCheck aborts when neither exchange flag is on.
// ENABLE_EXCHANGE_SYSTEM is the full VIP product flag (superset of ENABLE_P2P_EXCHANGE).
func featureCheck(c *gin.Context) bool {
	f := config.GetFlags()
	if !f.EnableP2PExchange && !f.EnableExchangeSystem {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Community Exchange is currently disabled"})
		return false
	}
	return true
}

// ─── CreateRequest — POST /exchange/requests ────────────────────────────────

func (h *Handler) CreateRequest(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}
	if !h.trustCheck(c, userID) {
		return
	}

	// Tier limit check — free users capped at 1 active request
	if err := CanCreateRequest(h.db, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Risk check before creating
	if riskResult := CheckExchangeRisk(h.db, userID); !riskResult.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": riskResult.Reason, "risk_level": riskResult.RiskLevel})
		return
	}

	var body struct {
		FromCurrency  string   `json:"from_currency"   binding:"required"`
		ToCurrency    string   `json:"to_currency"     binding:"required"`
		Amount        float64  `json:"amount"          binding:"required,gt=0"`
		PreferredRate *float64 `json:"preferred_rate"`
		PaymentMethod string   `json:"payment_method"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.PaymentMethod != "" {
		if err := ValidatePaymentMethod(body.PaymentMethod); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	req := ExchangeRequest{
		ID:            uuid.New(),
		UserID:        userID,
		FromCurrency:  body.FromCurrency,
		ToCurrency:    body.ToCurrency,
		Amount:        body.Amount,
		PreferredRate: body.PreferredRate,
		PaymentMethod: body.PaymentMethod,
		Status:        StatusOpen,
	}
	if err := h.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// PRO tier: attempt auto-match immediately
	if IsProAutoMatch(h.db, userID) {
		if result, err := TryMatch(h.db, &req); err == nil {
			c.JSON(http.StatusCreated, gin.H{
				"request":    req,
				"auto_match": result.Match,
				"settlement": result.Settlement,
				"fees":       result.Fees,
				"tier":       TierPro,
			})
			return
		}
	}

	tier := GetUserTier(h.db, userID)
	c.JSON(http.StatusCreated, gin.H{"request": req, "tier": tier})
}

// ─── ListRequests — GET /exchange/requests ──────────────────────────────────

func (h *Handler) ListRequests(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	from := c.Query("from")
	to := c.Query("to")

	q := h.db.Where("status = ?", StatusOpen)
	if from != "" {
		q = q.Where("from_currency = ?", from)
	}
	if to != "" {
		q = q.Where("to_currency = ?", to)
	}
	var requests []ExchangeRequest
	if err := q.Order("created_at ASC").Limit(100).Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests, "count": len(requests)})
}

// ─── MatchRequest — POST /exchange/requests/:id/match ───────────────────────

func (h *Handler) MatchRequest(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}
	if !h.trustCheck(c, userID) {
		return
	}

	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	var req ExchangeRequest
	if err := h.db.First(&req, "id = ? AND user_id = ? AND status = ?", reqID, userID, StatusOpen).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "open request not found"})
		return
	}

	result, err := TryMatch(h.db, &req)
	if err == ErrNoCounterparty {
		userRate := 0.0
		if req.PreferredRate != nil {
			userRate = *req.PreferredRate
		}
		feedback := NoMatchFeedback(h.db, h.rdb, req.FromCurrency, req.ToCurrency, userRate)
		c.JSON(http.StatusAccepted, gin.H{
			"message":  "no match found yet, request remains open",
			"feedback": feedback,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "matching failed"})
		return
	}

	// Retrieve counterparty to build safety context
	var counterReq ExchangeRequest
	h.db.First(&counterReq, "id=?", result.Match.RequestBID)
	if counterReq.UserID == userID {
		h.db.First(&counterReq, "id=?", result.Match.RequestAID)
	}

	// Match-level risk check
	riskResult := CheckMatchRisk(h.db, &req, &counterReq)

	c.JSON(http.StatusOK, gin.H{
		"match":      result.Match,
		"settlement": result.Settlement,
		"fees":       result.Fees,
		"safety":     h.buildSafetyContext(counterReq.UserID, counterReq.PaymentMethod),
		"match_risk": riskResult,
	})
}

// ─── CancelRequest — DELETE /exchange/requests/:id ──────────────────────────

func (h *Handler) CancelRequest(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}

	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}

	res := h.db.Model(&ExchangeRequest{}).
		Where("id = ? AND user_id = ? AND status = ?", reqID, userID, StatusOpen).
		Update("status", StatusCancelled)
	if res.Error != nil || res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "cancellable request not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "request cancelled"})
}

// ─── UploadProof — POST /exchange/:id/upload-proof ──────────────────────────

func (h *Handler) UploadProof(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}

	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}

	var body struct {
		ProofURL string   `json:"proof_url" binding:"required"`
		Amount   *float64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := UploadProof(h.db, matchID, userID, body.ProofURL, body.Amount)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrAlreadyVerified || err == ErrProofMissing {
			status = http.StatusBadRequest
		} else if err == ErrMatchNotFound || err == ErrSettlementNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settlement)
}

// ─── VerifyProof — POST /exchange/:id/verify ────────────────────────────────
// Intended for admin or automated trust checks.

func (h *Handler) VerifyProof(c *gin.Context) {
	if !featureCheck(c) {
		return
	}

	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}

	var body struct {
		Side string `json:"side" binding:"required"` // "a" or "b"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := VerifyProof(h.db, matchID, body.Side)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrSettlementNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settlement)
}

// ─── RaiseDispute — POST /exchange/:id/dispute ──────────────────────────────

func (h *Handler) RaiseDispute(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}

	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}

	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dispute, err := OpenDispute(h.db, matchID, userID, body.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open dispute"})
		return
	}
	c.JSON(http.StatusCreated, dispute)
}

// ─── FeeEstimate — GET /exchange/fee-estimate ───────────────────────────────

func (h *Handler) FeeEstimate(c *gin.Context) {
	var body struct {
		Amount        float64 `form:"amount"         binding:"required,gt=0"`
		Currency      string  `form:"currency"       binding:"required"`
		HasPriority   bool    `form:"has_priority"`
		HasProtection bool    `form:"has_protection"`
	}
	if err := c.ShouldBindQuery(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Apply tier discount when user is authenticated
	tier := TierFree
	if userID, ok := h.mustUserIDOpt(c); ok {
		tier = GetUserTier(h.db, userID)
	}
	bd := CalculateExchangeFeeForTier(body.Amount, body.Currency, body.HasPriority, body.HasProtection, tier)
	c.JSON(http.StatusOK, bd)
}

// ─── MyTier — GET /exchange/me/tier ─────────────────────────────────────────

func (h *Handler) MyTier(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}
	tier := GetUserTier(h.db, userID)
	caps := GetTierCapabilities(tier)
	c.JSON(http.StatusOK, gin.H{"tier": tier, "capabilities": caps})
}

// ─── AdminSetTier — POST /exchange/admin/users/:user_id/tier ────────────────

func (h *Handler) AdminSetTier(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	adminID, ok := h.mustUserID(c)
	if !ok {
		return
	}
	var body struct {
		Tier      string  `json:"tier"       binding:"required"`
		ExpiresAt *string `json:"expires_at"` // RFC3339 or omit for lifetime
		Note      string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var exp *time.Time
	if body.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be RFC3339"})
			return
		}
		exp = &t
	}
	record, err := SetUserTier(h.db, targetID, body.Tier, exp, adminID, body.Note)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, record)
}

// ─── TierInfo — GET /exchange/tiers ─────────────────────────────────────────

func (h *Handler) TierInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"tiers": []TierCapabilities{
			GetTierCapabilities(TierFree),
			GetTierCapabilities(TierVIP),
			GetTierCapabilities(TierPro),
		},
	})
}

// ─── LiquidityInsight — GET /exchange/liquidity ──────────────────────────────

func (h *Handler) LiquidityInsight(c *gin.Context) {
	if !config.GetFlags().EnableLiquidityEngine {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "liquidity engine is disabled"})
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params required"})
		return
	}
	insight := GetLiquidityInsight(h.db, from, to)
	// Attach rate hint with anchor for UX intelligence
	hint := GetRateHintWithAnchor(h.db, h.rdb, from, to)
	c.JSON(http.StatusOK, gin.H{"liquidity": insight, "rate_hint": hint})
}

// ─── RateHintEndpoint — GET /exchange/rate-hint ──────────────────────────────

func (h *Handler) RateHintEndpoint(c *gin.Context) {
	if !config.GetFlags().EnableRateHints {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "rate hints are disabled"})
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query params required"})
		return
	}
	hint := GetRateHintWithAnchor(h.db, h.rdb, from, to)
	c.JSON(http.StatusOK, hint)
}

// ─── AutoResolveHandler — POST /exchange/:id/auto-resolve ────────────────────
// Admin-only: trigger automated dispute resolution.

func (h *Handler) AutoResolveHandler(c *gin.Context) {
	if !featureCheck(c) {
		return
	}
	matchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid match id"})
		return
	}
	result := AutoResolveDispute(h.db, matchID)
	c.JSON(http.StatusOK, result)
}

// ─── RiskProfile — GET /exchange/risk/me ─────────────────────────────────────

func (h *Handler) RiskProfile(c *gin.Context) {
	if !config.GetFlags().EnableExchangeRisk {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "risk engine is disabled"})
		return
	}
	userID, ok := h.mustUserID(c)
	if !ok {
		return
	}
	var flags []ExchangeRiskFlag
	h.db.Where("user_id=? AND resolved=?", userID, false).Order("created_at DESC").Limit(20).Find(&flags)
	c.JSON(http.StatusOK, gin.H{
		"risk_level": RiskLevelForUser(h.db, userID),
		"flags":      flags,
	})
}

// ─── AdminSeedLiquidity — POST /exchange/admin/seed ──────────────────────────

func (h *Handler) AdminSeedLiquidity(c *gin.Context) {
	var body struct {
		From          string   `json:"from"           binding:"required"`
		To            string   `json:"to"             binding:"required"`
		Amount        float64  `json:"amount"         binding:"required,gt=0"`
		PreferredRate *float64 `json:"preferred_rate"` // set for influencer seed
		SeedType      string   `json:"seed_type"`      // "visibility" (default) | "influencer"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.SeedType == "influencer" && body.PreferredRate != nil && *body.PreferredRate > 0 {
		if err := SeedInfluencerRequest(h.db, body.From, body.To, body.Amount, *body.PreferredRate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "influencer seed created", "pair": body.From + "/" + body.To, "rate": *body.PreferredRate})
		return
	}
	if err := SeedSystemRequest(h.db, body.From, body.To, body.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "visibility seed created", "pair": body.From + "/" + body.To})
}

// ─── UX Intelligence Payload (Part 6) ────────────────────────────────────────

// UXIntelligencePayload assembles the full advisory payload for a pair.
func (h *Handler) UXIntelligencePayload(from, to string) gin.H {
	hint := GetRateHintWithAnchor(h.db, h.rdb, from, to)
	insight := GetLiquidityInsight(h.db, from, to)
	return gin.H{
		"rate_hint":         hint,
		"deal_quality":      hint.Quality,
		"spread_status":     hint.SpreadStatus,
		"action_suggestion": hint.ActionSuggestion,
		"match_probability": hint.ActionSuggestion.MatchProbability,
		"liquidity":         insight,
	}
}

// ─── optional auth helper ────────────────────────────────────────────────────

func (h *Handler) mustUserIDOpt(c *gin.Context) (uuid.UUID, bool) {
	raw, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw.(string))
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}
