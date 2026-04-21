package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/pkg/cache"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// kycWithdrawThreshold: withdrawals above this amount require an approved KYC profile.
var kycWithdrawThreshold = decimal.NewFromFloat(2000)

const idempotencyNoExpiry = -1

// ── Idempotency helpers ──────────────────────────────────────────────────────

// loadIdempotentResponse returns a previously stored response for this user+key combo.
// Returns (record, true) on hit; (nil, false) on miss.
func (h *Handler) loadIdempotentResponse(userID uuid.UUID, key string) (*IdempotentRequest, bool) {
	var rec IdempotentRequest
	err := h.db.Where("user_id = ? AND idempotency_key = ? AND expires_at > ?", userID, key, time.Now()).First(&rec).Error
	if err != nil {
		return nil, false
	}
	return &rec, true
}

// beginIdempotentRequest reserves an idempotency key before execution.
// Returns:
// - "none": key is empty
// - "new": freshly reserved, execute request
// - "cached": already executed, return stored response
// - "in_flight": duplicate request currently executing, reject
// - "path_conflict": same key reused for different endpoint, reject
func (h *Handler) beginIdempotentRequest(userID uuid.UUID, key, path string) (status string, rec *IdempotentRequest) {
	if key == "" {
		return "none", nil
	}
	if existing, ok := h.loadIdempotentResponse(userID, key); ok {
		if existing.Path != path {
			return "path_conflict", existing
		}
		if existing.ResponseCode > 0 {
			return "cached", existing
		}
		return "in_flight", existing
	}

	placeholder := IdempotentRequest{
		UserID:         userID,
		IdempotencyKey: key,
		Path:           path,
		ResponseCode:   0,
		ResponseBody:   "",
		CreatedAt:      time.Now(),
		ExpiresAt:      h.idempotencyExpiryForPath(path),
	}
	if err := h.db.Create(&placeholder).Error; err != nil {
		if existing, ok := h.loadIdempotentResponse(userID, key); ok {
			if existing.Path != path {
				return "path_conflict", existing
			}
			if existing.ResponseCode > 0 {
				return "cached", existing
			}
			return "in_flight", existing
		}
		return "in_flight", nil
	}
	return "new", &placeholder
}

// saveIdempotentResponse persists the response so retries get the same result.
func (h *Handler) saveIdempotentResponse(userID uuid.UUID, key, path string, code int, body any) {
	if key == "" {
		return
	}
	b, _ := json.Marshal(body)
	_ = h.db.Model(&IdempotentRequest{}).
		Where("user_id = ? AND idempotency_key = ? AND path = ?", userID, key, path).
		Updates(map[string]any{
			"response_code": code,
			"response_body": string(b),
			"expires_at":    h.idempotencyExpiryForPath(path),
		}).Error
}

func (h *Handler) idempotencyExpiryForPath(path string) time.Time {
	now := time.Now()
	switch {
	case strings.Contains(path, "/escrow"):
		return now.Add(7 * 24 * time.Hour)
	case strings.Contains(path, "/wallet/"):
		return now.Add(24 * time.Hour)
	case strings.Contains(path, "/webhooks/"):
		return now.AddDate(100, 0, 0)
	default:
		return now.Add(24 * time.Hour)
	}
}

func respondIdempotencyConflict(c *gin.Context, status string) {
	switch status {
	case "in_flight":
		response.Conflict(c, "Request with this idempotency key is already in progress")
	case "path_conflict":
		response.Conflict(c, "Idempotency key already used for a different endpoint")
	}
}

func logFinancialAudit(event, actor string, transactionID *uuid.UUID, escrowID *uuid.UUID, orderID, idempotencyKey string) {
	txID := ""
	if transactionID != nil {
		txID = transactionID.String()
	}
	escID := ""
	if escrowID != nil {
		escID = escrowID.String()
	}
	slog.Info("financial_audit",
		"event", event,
		"transaction_id", txID,
		"escrow_id", escID,
		"order_id", orderID,
		"idempotency_key", idempotencyKey,
		"actor", actor,
	)
}

func (h *Handler) buildReconcileReport() (mismatchCount int, report []gin.H, err error) {
	type currencySum struct {
		Currency string
		Total    decimal.Decimal
	}

	var txSums []currencySum
	if err = h.db.Model(&WalletTransaction{}).
		Select("currency, COALESCE(SUM(amount), 0) AS total").
		Where("status = ?", StatusCompleted).
		Group("currency").
		Scan(&txSums).Error; err != nil {
		return 0, nil, err
	}

	var balSums []currencySum
	if err = h.db.Model(&WalletBalance{}).
		Select("currency, COALESCE(SUM(balance), 0) AS total").
		Group("currency").
		Scan(&balSums).Error; err != nil {
		return 0, nil, err
	}

	txByCur := make(map[string]decimal.Decimal, len(txSums))
	for _, row := range txSums {
		txByCur[row.Currency] = row.Total
	}
	balByCur := make(map[string]decimal.Decimal, len(balSums))
	for _, row := range balSums {
		balByCur[row.Currency] = row.Total
	}

	allCurrencies := map[string]struct{}{}
	for k := range txByCur {
		allCurrencies[k] = struct{}{}
	}
	for k := range balByCur {
		allCurrencies[k] = struct{}{}
	}

	report = make([]gin.H, 0, len(allCurrencies))
	mismatchCount = 0
	for cur := range allCurrencies {
		txTotal := txByCur[cur]
		balTotal := balByCur[cur]
		delta := txTotal.Sub(balTotal)
		mismatch := !delta.Equal(decimal.Zero)
		if mismatch {
			mismatchCount++
		}
		report = append(report, gin.H{
			"currency":            cur,
			"transactions_sum":    txTotal,
			"wallet_balances_sum": balTotal,
			"delta":               delta,
			"mismatch":            mismatch,
		})
	}

	if mismatchCount > 0 {
		metrics.IncReconcileMismatch()
		slog.Error("wallet reconciliation mismatch detected",
			"severity", "CRITICAL",
			"mismatch_count", mismatchCount,
		)
	}

	return mismatchCount, report, nil
}

type Handler struct {
	db    *gorm.DB
	rdb   *redis.Client
	cache *cache.Cache
}

const walletSnapshotTTL = 30 * time.Second

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb, cache: cache.New(rdb)}
}

func RunReconcileOnce(db *gorm.DB) error {
	h := NewHandler(db, nil)
	_, _, err := h.buildReconcileReport()
	return err
}

func StartReconcileJob(db *gorm.DB, interval time.Duration) func() {
	stop := make(chan struct{})
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := RunReconcileOnce(db); err != nil {
					slog.Error("wallet reconciliation job failed",
						"severity", "CRITICAL",
						"error", err.Error(),
					)
				}
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}

// CreateWallet creates a new wallet for user
func (h *Handler) CreateWallet(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	// Check if wallet exists
	var existing Wallet
	if h.db.Where("user_id = ?", uid).First(&existing).Error == nil {
		response.Conflict(c, "Wallet already exists")
		return
	}

	// Create wallet with balances for all currencies
	wallet := Wallet{
		UserID:          uid,
		PrimaryCurrency: USD,
		DailyLimit:      decimal.NewFromInt(10000),
		MonthlyLimit:    decimal.NewFromInt(100000),
		IsActive:        true,
	}

	if err := h.db.Create(&wallet).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Create balances for all supported currencies
	for _, currency := range SupportedCurrencies {
		balance := WalletBalance{
			WalletID:         wallet.ID,
			Currency:         currency,
			Balance:          decimal.Zero,
			AvailableBalance: decimal.Zero,
			PendingBalance:   decimal.Zero,
		}
		h.db.Create(&balance)
	}

	// Reload with balances
	h.db.Preload("Balances").First(&wallet, wallet.ID)

	response.Created(c, wallet)
}

// GetWallet returns user's wallet with balances (cached snapshot 30s)
func (h *Handler) GetWallet(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	// Try cache first
	cacheKey := fmt.Sprintf("wallet:snapshot:%s", userID)
	var cachedWallet Wallet
	if h.cache != nil && h.cache.Get(c.Request.Context(), cacheKey, &cachedWallet) {
		response.OK(c, cachedWallet)
		return
	}

	var wallet Wallet
	if err := h.db.Preload("Balances").Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	// Store in cache
	if h.cache != nil {
		h.cache.Set(c.Request.Context(), cacheKey, &wallet, walletSnapshotTTL)
	}
	response.OK(c, wallet)
}

func (h *Handler) invalidateWalletCache(userID string) {
	if h.cache == nil {
		return
	}
	ctx := context.Background()
	h.cache.Del(ctx,
		fmt.Sprintf("wallet:snapshot:%s", userID),
		fmt.Sprintf("wallet:balance:%s:AED", userID),
		fmt.Sprintf("wallet:balance:%s:USD", userID),
		fmt.Sprintf("wallet:balance:%s:EUR", userID),
		fmt.Sprintf("wallet:balance:%s:SAR", userID),
		fmt.Sprintf("wallet:balance:%s:EGP", userID),
	)
}

// GetBalance returns balance for specific currency (cached snapshot 30s)
func (h *Handler) GetBalance(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	currency := Currency(c.Param("currency"))

	// Try cache first
	cacheKey := fmt.Sprintf("wallet:balance:%s:%s", userID, currency)
	var cachedBalance WalletBalance
	if h.cache != nil && h.cache.Get(c.Request.Context(), cacheKey, &cachedBalance) {
		response.OK(c, cachedBalance)
		return
	}

	var wallet Wallet
	if err := h.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	var balance WalletBalance
	if err := h.db.Where("wallet_id = ? AND currency = ?", wallet.ID, currency).First(&balance).Error; err != nil {
		response.NotFound(c, "Balance")
		return
	}

	// Store in cache
	if h.cache != nil {
		h.cache.Set(c.Request.Context(), cacheKey, &balance, walletSnapshotTTL)
	}
	response.OK(c, balance)
}

type DepositReq struct {
	Currency    Currency        `json:"currency" binding:"required"`
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	ReferenceID string          `json:"reference_id"`
}

// Deposit adds funds to wallet
func (h *Handler) Deposit(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req DepositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "Amount must be positive")
		return
	}

	// ── Idempotency check ───────────────────────────────────────────────────
	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(uid, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	var result gin.H
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var wallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}
		if !wallet.IsActive {
			return fmt.Errorf("wallet_inactive")
		}
		// Lock the balance row to prevent concurrent modifications
		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", wallet.ID, req.Currency).First(&balance).Error; err != nil {
			return err
		}
		// FIX-1: use applyDeposit helper — preserves PendingBalance invariant
		balBefore, availBefore := applyDeposit(&balance, req.Amount)
		if err := checkInvariant(balance); err != nil {
			return err
		}
		if err := tx.Save(&balance).Error; err != nil {
			return err
		}
		refID := req.ReferenceID
		refType := "deposit"
		now := time.Now()
		wt := WalletTransaction{
			WalletID: wallet.ID, Type: TransactionDeposit, Currency: req.Currency,
			Amount: req.Amount, BalanceBefore: balBefore, BalanceAfter: balance.Balance,
			Status: StatusCompleted, ReferenceID: &refID, ReferenceType: &refType,
			Description: "Deposit of " + req.Amount.String() + " " + string(req.Currency) +
				" | avail: " + availBefore.String() + "→" + balance.AvailableBalance.String(),
			CompletedAt: &now,
		}
		if err := tx.Create(&wt).Error; err != nil {
			return err
		}
		result = gin.H{"transaction_id": wt.ID, "currency": req.Currency, "amount": req.Amount, "new_balance": balance.Balance, "available_balance": balance.AvailableBalance, "status": StatusCompleted}

		// ── Transactional outbox: wallet.deposited event ──────────────────────
		_ = kafka.WriteOutbox(tx, kafka.TopicWallet, kafka.New(
			"wallet.deposited",
			wallet.ID.String(),
			"wallet",
			kafka.Actor{Type: "user", ID: userID},
			map[string]interface{}{
				"user_id":   userID,
				"wallet_id": wallet.ID.String(),
				"amount":    req.Amount.String(),
				"currency":  string(req.Currency),
				"tx_id":     wt.ID.String(),
			},
			kafka.EventMeta{Source: "api-service"},
		))

		return nil
	})
	if dbErr != nil {
		if dbErr.Error() == "wallet_inactive" {
			response.BadRequest(c, "Wallet is inactive")
		} else {
			response.NotFound(c, "Wallet or currency")
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(uid, idempKey, c.FullPath(), http.StatusOK, result)
	}
	if txID, ok := result["transaction_id"].(uuid.UUID); ok {
		logFinancialAudit("wallet_deposit", userID, &txID, nil, req.ReferenceID, idempKey)
	} else {
		logFinancialAudit("wallet_deposit", userID, nil, nil, req.ReferenceID, idempKey)
	}
	// Invalidate wallet snapshot cache on balance change
	h.invalidateWalletCache(userID)
	metrics.IncWalletOp("deposit", "success")
	metrics.IncWalletTransaction()
	response.OK(c, result)
}

type WithdrawReq struct {
	Currency    Currency        `json:"currency" binding:"required"`
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	ReferenceID string          `json:"reference_id"`
}

// Withdraw removes funds from wallet
func (h *Handler) Withdraw(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	uid, err := uuid.Parse(userID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req WithdrawReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "Amount must be positive")
		return
	}

	// ── Idempotency check ───────────────────────────────────────────────────
	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(uid, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	var result gin.H
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var wallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}
		if !wallet.IsActive {
			return fmt.Errorf("wallet_inactive")
		}
		// Lock balance row — prevents TOCTOU race between concurrent withdrawals
		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", wallet.ID, req.Currency).First(&balance).Error; err != nil {
			return err
		}
		if balance.AvailableBalance.LessThan(req.Amount) {
			return fmt.Errorf("insufficient_balance")
		}

		// FIX-4: KYC + fraud checks run INSIDE the transaction, after FOR UPDATE lock.
		// Pre-fix: checks ran outside the transaction — a revoked KYC or spiked fraud
		// score between the check and the debit was invisible (TOCTOU window).
		if req.Amount.GreaterThan(kycWithdrawThreshold) {
			var kycStatus struct{ Status string }
			tx.Table("kyc_profiles").Select("status").Where("user_id = ?", uid).Scan(&kycStatus)
			if kycStatus.Status != "approved" {
				return fmt.Errorf("kyc_required")
			}
		}
		{
			var userCreatedAt time.Time
			tx.Table("users").Select("created_at").Where("id = ?", uid).Scan(&userCreatedAt)
			var profile fraud.UserRiskProfile
			tx.Where("user_id = ?", uid).First(&profile)
			acctAgeHours := time.Since(userCreatedAt).Hours()
			risk := fraud.AnalyzeTransaction(req.Amount.InexactFloat64(), profile.TotalOrders, profile.AvgOrderValue, acctAgeHours)
			if risk.RiskScore >= 80 {
				slog.Warn("wallet withdrawal declined by fraud",
					"user_id", uid, "amount", req.Amount, "risk_score", risk.RiskScore)
				return fmt.Errorf("fraud_declined")
			}
		}
		today := time.Now().Truncate(24 * time.Hour)
		var todayTotal decimal.Decimal
		tx.Model(&WalletTransaction{}).Where("wallet_id = ? AND type = ? AND created_at >= ? AND status = ?",
			wallet.ID, TransactionWithdrawal, today, StatusCompleted).
			Select("COALESCE(SUM(ABS(amount)), 0)").Scan(&todayTotal)
		if todayTotal.Add(req.Amount).GreaterThan(wallet.DailyLimit) {
			return fmt.Errorf("daily_limit_exceeded")
		}
		// FIX-2: use applyWithdrawal helper — preserves PendingBalance invariant;
		// balBefore captured PRE-mutation (old code captured post-mutation = wrong).
		balBefore, availBefore := applyWithdrawal(&balance, req.Amount)
		if err := checkInvariant(balance); err != nil {
			return err
		}
		if err := tx.Save(&balance).Error; err != nil {
			return err
		}
		refID := req.ReferenceID
		refType := "withdrawal"
		now := time.Now()
		wt := WalletTransaction{
			WalletID: wallet.ID, Type: TransactionWithdrawal, Currency: req.Currency,
			Amount: req.Amount.Neg(), BalanceBefore: balBefore, BalanceAfter: balance.Balance,
			Status: StatusCompleted, ReferenceID: &refID, ReferenceType: &refType,
			Description: "Withdrawal of " + req.Amount.String() + " " + string(req.Currency) +
				" | avail: " + availBefore.String() + "→" + balance.AvailableBalance.String(),
			CompletedAt: &now,
		}
		if err := tx.Create(&wt).Error; err != nil {
			return err
		}
		result = gin.H{"transaction_id": wt.ID, "currency": req.Currency, "amount": req.Amount, "new_balance": balance.Balance, "available_balance": balance.AvailableBalance, "status": StatusCompleted}
		return nil
	})
	if dbErr != nil {
		msg := dbErr.Error()
		switch msg {
		case "wallet_inactive":
			response.BadRequest(c, "Wallet is inactive")
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance")
		case "kyc_required":
			response.BadRequest(c, fmt.Sprintf("KYC verification required for withdrawals over %.0f", kycWithdrawThreshold.InexactFloat64()))
		case "fraud_declined":
			response.BadRequest(c, "Withdrawal declined by fraud prevention")
		case "daily_limit_exceeded":
			response.BadRequest(c, "Daily withdrawal limit exceeded")
		default:
			response.NotFound(c, "Wallet or currency")
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(uid, idempKey, c.FullPath(), http.StatusOK, result)
	}
	if txID, ok := result["transaction_id"].(uuid.UUID); ok {
		logFinancialAudit("wallet_withdraw", userID, &txID, nil, req.ReferenceID, idempKey)
	} else {
		logFinancialAudit("wallet_withdraw", userID, nil, nil, req.ReferenceID, idempKey)
	}
	// Invalidate wallet snapshot cache on balance change
	h.invalidateWalletCache(userID)
	metrics.IncWalletOp("withdraw", "success")
	response.OK(c, result)
}

// GetTransactions returns transaction history
func (h *Handler) GetTransactions(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var wallet Wallet
	if err := h.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	var transactions []WalletTransaction
	query := h.db.Where("wallet_id = ?", wallet.ID).Order("created_at DESC").Limit(50)

	if currency := c.Query("currency"); currency != "" {
		query = query.Where("currency = ?", currency)
	}
	if txType := c.Query("type"); txType != "" {
		query = query.Where("type = ?", txType)
	}

	query.Find(&transactions)

	response.OK(c, transactions)
}

// ============ ESCROW ============

type CreateEscrowReq struct {
	SellerID    string          `json:"seller_id" binding:"required"`
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	Currency    Currency        `json:"currency" binding:"required"`
	ReferenceID string          `json:"reference_id" binding:"required"`
	Type        string          `json:"type" binding:"required"` // AUCTION, ORDER
}

// CreateEscrow creates escrow for auction/order
func (h *Handler) CreateEscrow(c *gin.Context) {
	buyerID := c.MustGet("user_id").(string)
	buyerUUID, err := uuid.Parse(buyerID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req CreateEscrowReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	sellerUUID, err := uuid.Parse(req.SellerID)
	if err != nil {
		response.BadRequest(c, "Invalid seller ID")
		return
	}

	// ── Idempotency check ───────────────────────────────────────────────────
	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(buyerUUID, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	var escrow Escrow
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var buyerWallet Wallet
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", buyerID).First(&buyerWallet).Error; err != nil {
			return err
		}
		// Lock balance row to prevent concurrent escrow creation draining the same funds
		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", buyerWallet.ID, req.Currency).First(&balance).Error; err != nil {
			return fmt.Errorf("currency_not_supported")
		}
		if balance.AvailableBalance.LessThan(req.Amount) {
			return fmt.Errorf("insufficient_balance")
		}
		fee := req.Amount.Mul(decimal.NewFromFloat(0.025))
		escrow = Escrow{
			BuyerID: buyerUUID, SellerID: sellerUUID, Amount: req.Amount,
			Currency: req.Currency, Fee: fee, Status: StatusPending,
			ReferenceID: req.ReferenceID, Type: req.Type,
		}
		if err := tx.Create(&escrow).Error; err != nil {
			return err
		}
		availBefore := applyEscrowHold(&balance, req.Amount)
		if err := checkInvariant(balance); err != nil {
			return err
		}
		if err := tx.Save(&balance).Error; err != nil {
			return err
		}
		refType := "escrow"
		refID := req.ReferenceID
		if err := tx.Create(&WalletTransaction{
			WalletID: buyerWallet.ID, Type: TransactionEscrow, Currency: req.Currency,
			Amount: req.Amount.Neg(), BalanceBefore: availBefore, BalanceAfter: balance.AvailableBalance, Status: StatusPending,
			ReferenceID: &refID, ReferenceType: &refType,
			Description: "Escrow for " + req.Type + " #" + req.ReferenceID,
		}).Error; err != nil {
			return err
		}

		// ── Transactional outbox: escrow.created event ────────────────────────
		_ = kafka.WriteOutbox(tx, kafka.TopicEscrow, kafka.New(
			"escrow.created",
			escrow.ID.String(),
			"escrow",
			kafka.Actor{Type: "user", ID: buyerID},
			map[string]interface{}{
				"escrow_id": escrow.ID.String(),
				"order_id":  req.ReferenceID,
				"buyer_id":  buyerID,
				"seller_id": sellerUUID.String(),
				"amount":    req.Amount.String(),
				"currency":  string(req.Currency),
			},
			kafka.EventMeta{Source: "api-service"},
		))

		// ── Transactional outbox: wallet.debited event ────────────────────────
		_ = kafka.WriteOutbox(tx, kafka.TopicWallet, kafka.New(
			"wallet.debited",
			buyerWallet.ID.String(),
			"wallet",
			kafka.Actor{Type: "user", ID: buyerID},
			map[string]interface{}{
				"user_id":   buyerID,
				"wallet_id": buyerWallet.ID.String(),
				"amount":    req.Amount.String(),
				"currency":  string(req.Currency),
				"reason":    "escrow_hold",
			},
			kafka.EventMeta{Source: "api-service"},
		))

		return nil
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "currency_not_supported":
			response.BadRequest(c, "Currency not supported")
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(buyerUUID, idempKey, c.FullPath(), http.StatusCreated, escrow)
	}
	logFinancialAudit("escrow_create", buyerID, nil, &escrow.ID, req.ReferenceID, idempKey)
	response.Created(c, escrow)
}

// ReleaseEscrow releases funds to seller — admin only (IDOR + state machine guard).
func (h *Handler) ReleaseEscrow(c *gin.Context) {
	escrowID := c.Param("id")
	adminIDStr := c.GetString("user_id")
	adminID, parseErr := uuid.Parse(adminIDStr)
	if parseErr != nil {
		response.BadRequest(c, "Invalid admin user")
		return
	}

	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(adminID, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	var result gin.H
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// Lock escrow row to prevent double-release race condition (STEP 7)
		var escrow Escrow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&escrow, "id = ?", escrowID).Error; err != nil {
			return fmt.Errorf("not_found")
		}
		// ── State machine: only PENDING → COMPLETED is valid ────────────────
		if escrow.Status != StatusPending {
			return fmt.Errorf("already_processed")
		}
		readyToRelease, approvalErr := applyEscrowReleaseApproval(&escrow, adminID, time.Now())
		if approvalErr != nil {
			return approvalErr
		}
		if err := tx.Save(&escrow).Error; err != nil {
			return err
		}
		if !readyToRelease {
			result = gin.H{"escrow_id": escrow.ID, "status": "AWAITING_SECOND_APPROVAL"}
			return nil
		}
		// ── Lock wallets in deterministic order to prevent deadlocks ────────
		// Always lock the lower user_id first. Two concurrent escrow releases
		// that touch overlapping users will acquire locks in the same order.
		buyerIDStr := escrow.BuyerID.String()
		sellerIDStr := escrow.SellerID.String()
		var buyerWallet, sellerWallet Wallet
		if buyerIDStr < sellerIDStr {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", buyerIDStr).First(&buyerWallet).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", sellerIDStr).First(&sellerWallet).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", sellerIDStr).First(&sellerWallet).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", buyerIDStr).First(&buyerWallet).Error; err != nil {
				return err
			}
		}
		// ── Lock balances in deterministic wallet_id order ──────────────────
		var buyerBalance, sellerBalance WalletBalance
		if buyerWallet.ID.String() < sellerWallet.ID.String() {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("wallet_id = ? AND currency = ?", buyerWallet.ID, escrow.Currency).First(&buyerBalance).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("wallet_id = ? AND currency = ?", sellerWallet.ID, escrow.Currency).First(&sellerBalance).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("wallet_id = ? AND currency = ?", sellerWallet.ID, escrow.Currency).First(&sellerBalance).Error; err != nil {
				return err
			}
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("wallet_id = ? AND currency = ?", buyerWallet.ID, escrow.Currency).First(&buyerBalance).Error; err != nil {
				return err
			}
		}
		buyerBalBefore := buyerBalance.Balance
		buyerAvailBefore := buyerBalance.AvailableBalance
		applyEscrowRelease(&buyerBalance, escrow.Amount)
		if err := checkInvariant(buyerBalance); err != nil {
			return err
		}
		if err := tx.Save(&buyerBalance).Error; err != nil {
			return err
		}
		// Preserve total system balance: released amount is fully credited to seller.
		// Fee remains explicit in metadata/reporting and can be settled via separate flow.
		sellerAmount := escrow.Amount
		sellerBalBefore, sellerAvailBefore := applyDeposit(&sellerBalance, sellerAmount)
		if err := checkInvariant(sellerBalance); err != nil {
			return err
		}
		if err := tx.Save(&sellerBalance).Error; err != nil {
			return err
		}
		now := time.Now()
		escrow.Status = StatusCompleted
		escrow.ReleasedAt = &now
		if err := tx.Save(&escrow).Error; err != nil {
			return err
		}

		// ── Transactional outbox: escrow.released event ───────────────────────
		_ = kafka.WriteOutbox(tx, kafka.TopicEscrow, kafka.New(
			"escrow.released",
			escrow.ID.String(),
			"escrow",
			kafka.Actor{Type: "admin", ID: adminIDStr},
			map[string]interface{}{
				"escrow_id":   escrow.ID.String(),
				"order_id":    escrow.ReferenceID,
				"buyer_id":    escrow.BuyerID.String(),
				"seller_id":   escrow.SellerID.String(),
				"amount":      escrow.Amount.String(),
				"currency":    string(escrow.Currency),
				"seller_paid": true,
			},
			kafka.EventMeta{Source: "api-service"},
		))
		buyerRefType := "escrow_release_buyer"
		if err := tx.Create(&WalletTransaction{
			WalletID: buyerWallet.ID, Type: TransactionPayment, Currency: escrow.Currency,
			Amount: escrow.Amount.Neg(), BalanceBefore: buyerBalBefore, BalanceAfter: buyerBalance.Balance,
			Status: StatusCompleted, ReferenceID: &escrow.ReferenceID, ReferenceType: &buyerRefType,
			Description: "Escrow released (buyer debit) for " + escrow.Type + " #" + escrow.ReferenceID +
				" | buyer avail: " + buyerAvailBefore.String() + "→" + buyerBalance.AvailableBalance.String(),
			CompletedAt: &now,
		}).Error; err != nil {
			return err
		}
		refType := "escrow_release"
		if err := tx.Create(&WalletTransaction{
			WalletID: sellerWallet.ID, Type: TransactionRelease, Currency: escrow.Currency,
			Amount: sellerAmount, BalanceBefore: sellerBalBefore, BalanceAfter: sellerBalance.Balance,
			Fee: escrow.Fee, Status: StatusCompleted, ReferenceID: &escrow.ReferenceID, ReferenceType: &refType,
			Description: "Escrow release for " + escrow.Type + " #" + escrow.ReferenceID +
				" | seller avail: " + sellerAvailBefore.String() + "→" + sellerBalance.AvailableBalance.String(),
			CompletedAt: &now,
		}).Error; err != nil {
			return err
		}
		result = gin.H{"escrow_id": escrow.ID, "seller_amount": sellerAmount, "fee": escrow.Fee, "status": StatusCompleted}
		return nil
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "not_found":
			response.NotFound(c, "Escrow")
		case "already_processed":
			response.BadRequest(c, "Escrow already processed")
		case "second_approval_must_be_distinct_admin":
			response.BadRequest(c, "Second approval must be from a different admin")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(adminID, idempKey, c.FullPath(), http.StatusOK, result)
	}
	var escID uuid.UUID
	if v, ok := result["escrow_id"].(uuid.UUID); ok {
		escID = v
		logFinancialAudit("escrow_release", adminIDStr, nil, &escID, "", idempKey)
		// Sprint 8.5: Centralized audit log for financial action
		freeze.LogAudit(h.db, "escrow_release", adminID, escID, fmt.Sprintf("escrow_id=%s", escID))
	} else {
		logFinancialAudit("escrow_release", adminIDStr, nil, nil, "", idempKey)
	}
	response.OK(c, result)
}

// CancelEscrow cancels a PENDING escrow and refunds buyer — admin only.
// Fills the missing refund path that the OLD project handled via disputes.
func (h *Handler) CancelEscrow(c *gin.Context) {
	escrowID := c.Param("id")
	adminIDStr := c.GetString("user_id")
	adminID, parseErr := uuid.Parse(adminIDStr)
	if parseErr != nil {
		response.BadRequest(c, "Invalid admin user")
		return
	}

	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(adminID, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	var result gin.H
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var escrow Escrow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&escrow, "id = ?", escrowID).Error; err != nil {
			return fmt.Errorf("not_found")
		}
		if escrow.Status != StatusPending {
			return fmt.Errorf("not_cancellable")
		}

		// Refund buyer: move funds from pending back to available
		var buyerWallet Wallet
		if err := tx.Where("user_id = ?", escrow.BuyerID).First(&buyerWallet).Error; err != nil {
			return err
		}
		var buyerBalance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", buyerWallet.ID, escrow.Currency).First(&buyerBalance).Error; err != nil {
			return err
		}

		availBefore := applyEscrowCancel(&buyerBalance, escrow.Amount)
		if err := checkInvariant(buyerBalance); err != nil {
			return err
		}
		if err := tx.Save(&buyerBalance).Error; err != nil {
			return err
		}

		now := time.Now()
		escrow.Status = StatusCancelled
		escrow.ReleasedAt = &now
		if err := tx.Save(&escrow).Error; err != nil {
			return err
		}

		refType := "escrow_cancel"
		if err := tx.Create(&WalletTransaction{
			WalletID: buyerWallet.ID, Type: TransactionRefund, Currency: escrow.Currency,
			Amount: escrow.Amount, BalanceBefore: availBefore,
			BalanceAfter: buyerBalance.AvailableBalance, Status: StatusCompleted,
			ReferenceID: &escrow.ReferenceID, ReferenceType: &refType,
			Description: "Escrow cancelled — refund for " + escrow.Type + " #" + escrow.ReferenceID, CompletedAt: &now,
		}).Error; err != nil {
			return err
		}

		result = gin.H{"escrow_id": escrow.ID, "refunded_amount": escrow.Amount, "status": StatusCancelled}
		return nil
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "not_found":
			response.NotFound(c, "Escrow")
		case "not_cancellable":
			response.BadRequest(c, "Only PENDING escrows can be cancelled")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(adminID, idempKey, c.FullPath(), http.StatusOK, result)
	}
	if escID, ok := result["escrow_id"].(uuid.UUID); ok {
		logFinancialAudit("escrow_cancel", adminIDStr, nil, &escID, "", idempKey)
	} else {
		logFinancialAudit("escrow_cancel", adminIDStr, nil, nil, "", idempKey)
	}
	response.OK(c, result)
}

// ============ PRICE PLANS ============

// GetPricePlans returns all active price plans
func (h *Handler) GetPricePlans(c *gin.Context) {
	var plans []PricePlan
	h.db.Where("is_active = ?", true).Order("price ASC").Find(&plans)
	response.OK(c, plans)
}

type SubscribeReq struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// Subscribe subscribes user to a plan
func (h *Handler) Subscribe(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	var req SubscribeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get plan (read-only, no lock needed)
	var plan PricePlan
	if err := h.db.First(&plan, "id = ?", req.PlanID).Error; err != nil {
		response.NotFound(c, "Plan")
		return
	}

	// ── Atomic: check existing sub + balance + deduct + create sub in ONE tx ──
	var subscription UserSubscription
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// Check if user has active subscription (inside tx for consistency)
		var existing UserSubscription
		if tx.Where("user_id = ? AND is_active = ? AND end_date > ?", userID, true, time.Now()).First(&existing).Error == nil {
			return fmt.Errorf("active_subscription_exists")
		}

		var wallet Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return fmt.Errorf("wallet_not_found")
		}
		if !wallet.IsActive {
			return fmt.Errorf("wallet_inactive")
		}

		// Lock balance row — prevents concurrent Subscribe calls from both passing the check
		var balance WalletBalance
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("wallet_id = ? AND currency = ?", wallet.ID, plan.Currency).First(&balance).Error; err != nil {
			return fmt.Errorf("currency_not_supported")
		}

		if balance.AvailableBalance.LessThan(plan.Price) {
			return fmt.Errorf("insufficient_balance")
		}

		// Deduct payment
		balance.Balance = balance.Balance.Sub(plan.Price)
		balance.AvailableBalance = balance.AvailableBalance.Sub(plan.Price)
		if err := tx.Save(&balance).Error; err != nil {
			return err
		}

		// Create subscription
		now := time.Now()
		subscription = UserSubscription{
			UserID:    userUUID,
			PlanID:    plan.ID,
			StartDate: now,
			EndDate:   now.AddDate(0, 0, plan.DurationDays),
			IsActive:  true,
			AutoRenew: false,
		}
		if err := tx.Create(&subscription).Error; err != nil {
			return err
		}

		// Create transaction record
		planID := plan.ID.String()
		refType := "subscription"
		return tx.Create(&WalletTransaction{
			WalletID: wallet.ID, Type: TransactionPayment, Currency: plan.Currency,
			Amount: plan.Price.Neg(), BalanceBefore: balance.Balance.Add(plan.Price), BalanceAfter: balance.Balance,
			Status: StatusCompleted, ReferenceID: &planID, ReferenceType: &refType,
			Description: "Subscription to " + plan.Name, CompletedAt: &now,
		}).Error
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "active_subscription_exists":
			response.Conflict(c, "Active subscription exists")
		case "wallet_not_found":
			response.BadRequest(c, "Wallet not found")
		case "wallet_inactive":
			response.BadRequest(c, "Wallet is inactive")
		case "currency_not_supported":
			response.BadRequest(c, "Currency not supported")
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	subscription.Plan = plan
	response.Created(c, subscription)
}

// ============ P2P TRANSFER ============

type TransferReq struct {
	ToUserID string          `json:"to_user_id" binding:"required"`
	Currency Currency        `json:"currency" binding:"required"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Note     string          `json:"note"`
}

// Transfer executes an atomic wallet-to-wallet transfer with double-entry bookkeeping.
// Adapted from OLD project's TransferService — deadlock-safe via consistent lock order.
func (h *Handler) Transfer(c *gin.Context) {
	senderID := c.MustGet("user_id").(string)
	senderUUID, err := uuid.Parse(senderID)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req TransferReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "Amount must be positive")
		return
	}
	receiverUUID, err := uuid.Parse(req.ToUserID)
	if err != nil {
		response.BadRequest(c, "Invalid recipient user ID")
		return
	}
	if senderUUID == receiverUUID {
		response.BadRequest(c, "Cannot transfer to yourself")
		return
	}

	// ── Idempotency check ───────────────────────────────────────────────────
	idempKey := c.GetHeader("X-Idempotency-Key")
	if status, rec := h.beginIdempotentRequest(senderUUID, idempKey, c.FullPath()); status != "none" {
		switch status {
		case "cached":
			c.Data(rec.ResponseCode, "application/json", []byte(rec.ResponseBody))
			return
		case "new":
			// continue
		default:
			respondIdempotencyConflict(c, status)
			return
		}
	}

	// ── Fraud check on sender ───────────────────────────────────────────────
	{
		var userCreatedAt time.Time
		h.db.Table("users").Select("created_at").Where("id = ?", senderUUID).Scan(&userCreatedAt)
		var profile fraud.UserRiskProfile
		h.db.Where("user_id = ?", senderUUID).First(&profile)
		acctAgeHours := time.Since(userCreatedAt).Hours()
		risk := fraud.AnalyzeTransaction(req.Amount.InexactFloat64(), profile.TotalOrders, profile.AvgOrderValue, acctAgeHours)
		if risk.RiskScore >= 80 {
			response.BadRequest(c, "transfer declined by fraud prevention")
			return
		}
	}

	var result gin.H
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// ── Load both wallets ────────────────────────────────────────────────
		var senderWallet, receiverWallet Wallet
		if err := tx.Where("user_id = ?", senderUUID).First(&senderWallet).Error; err != nil {
			return fmt.Errorf("sender_wallet_not_found")
		}
		if !senderWallet.IsActive {
			return fmt.Errorf("sender_wallet_inactive")
		}
		if err := tx.Where("user_id = ?", receiverUUID).First(&receiverWallet).Error; err != nil {
			return fmt.Errorf("receiver_wallet_not_found")
		}
		if !receiverWallet.IsActive {
			return fmt.Errorf("receiver_wallet_inactive")
		}

		// ── Lock balance rows in consistent order to prevent deadlocks ──────
		// Always lock the lower wallet ID first (same pattern as OLD TransferService).
		type balRef struct {
			walletID uuid.UUID
			balance  *WalletBalance
		}
		refs := []balRef{
			{walletID: senderWallet.ID},
			{walletID: receiverWallet.ID},
		}
		if refs[0].walletID.String() > refs[1].walletID.String() {
			refs[0], refs[1] = refs[1], refs[0]
		}
		for i := range refs {
			var b WalletBalance
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("wallet_id = ? AND currency = ?", refs[i].walletID, req.Currency).First(&b).Error; err != nil {
				return fmt.Errorf("currency_not_supported")
			}
			refs[i].balance = &b
		}

		// Resolve which locked balance belongs to sender vs receiver
		var senderBal, receiverBal *WalletBalance
		for _, r := range refs {
			if r.walletID == senderWallet.ID {
				senderBal = r.balance
			} else {
				receiverBal = r.balance
			}
		}

		// ── Validate balance ─────────────────────────────────────────────────
		if senderBal.AvailableBalance.LessThan(req.Amount) {
			return fmt.Errorf("insufficient_balance")
		}

		// ── Apply double-entry bookkeeping ───────────────────────────────────
		senderBalBefore := senderBal.Balance
		senderBal.Balance = senderBal.Balance.Sub(req.Amount)
		senderBal.AvailableBalance = senderBal.AvailableBalance.Sub(req.Amount)
		if err := tx.Save(senderBal).Error; err != nil {
			return err
		}

		receiverBalBefore := receiverBal.Balance
		receiverBal.Balance = receiverBal.Balance.Add(req.Amount)
		receiverBal.AvailableBalance = receiverBal.AvailableBalance.Add(req.Amount)
		if err := tx.Save(receiverBal).Error; err != nil {
			return err
		}

		now := time.Now()
		refID := receiverUUID.String()
		debitRefType := "transfer_out"
		creditRefType := "transfer_in"
		desc := "Transfer"
		if req.Note != "" {
			desc = "Transfer: " + req.Note
		}

		// DEBIT entry (sender)
		if err := tx.Create(&WalletTransaction{
			WalletID: senderWallet.ID, Type: TransactionTransfer, Currency: req.Currency,
			Amount: req.Amount.Neg(), BalanceBefore: senderBalBefore, BalanceAfter: senderBal.Balance,
			Status: StatusCompleted, ReferenceID: &refID, ReferenceType: &debitRefType,
			Description: desc + " to " + receiverUUID.String(), CompletedAt: &now,
		}).Error; err != nil {
			return err
		}

		// CREDIT entry (receiver)
		senderRef := senderUUID.String()
		if err := tx.Create(&WalletTransaction{
			WalletID: receiverWallet.ID, Type: TransactionTransfer, Currency: req.Currency,
			Amount: req.Amount, BalanceBefore: receiverBalBefore, BalanceAfter: receiverBal.Balance,
			Status: StatusCompleted, ReferenceID: &senderRef, ReferenceType: &creditRefType,
			Description: desc + " from " + senderUUID.String(), CompletedAt: &now,
		}).Error; err != nil {
			return err
		}

		result = gin.H{
			"sender_new_balance":   senderBal.Balance,
			"receiver_new_balance": receiverBal.Balance,
			"amount":               req.Amount,
			"currency":             req.Currency,
			"status":               StatusCompleted,
		}
		return nil
	})
	if dbErr != nil {
		switch dbErr.Error() {
		case "sender_wallet_not_found":
			response.BadRequest(c, "Sender wallet not found")
		case "sender_wallet_inactive":
			response.BadRequest(c, "Sender wallet is inactive")
		case "receiver_wallet_not_found":
			response.BadRequest(c, "Recipient wallet not found")
		case "receiver_wallet_inactive":
			response.BadRequest(c, "Recipient wallet is inactive")
		case "currency_not_supported":
			response.BadRequest(c, "Currency not supported for one or both wallets")
		case "insufficient_balance":
			response.BadRequest(c, "Insufficient balance")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	if idempKey != "" {
		h.saveIdempotentResponse(senderUUID, idempKey, c.FullPath(), http.StatusOK, result)
	}
	if txID, ok := result["transaction_id"].(uuid.UUID); ok {
		logFinancialAudit("wallet_transfer", senderID, &txID, nil, "", idempKey)
	} else {
		logFinancialAudit("wallet_transfer", senderID, nil, nil, "", idempKey)
	}
	// Invalidate wallet snapshot caches for both sender and receiver
	h.invalidateWalletCache(senderID)
	h.invalidateWalletCache(req.ToUserID)
	metrics.IncWalletOp("transfer", "success")
	response.OK(c, result)
}

// Reconcile compares wallet transaction net amounts against wallet balances.
// Admin-only endpoint for financial integrity checks.
func (h *Handler) Reconcile(c *gin.Context) {
	mismatchCount, report, err := h.buildReconcileReport()
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"mismatch_count": mismatchCount,
		"report":         report,
	})
}

// GetSubscription returns user's active subscription
func (h *Handler) GetSubscription(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var subscription UserSubscription
	if err := h.db.Preload("Plan").Where("user_id = ? AND is_active = ? AND end_date > ?",
		userID, true, time.Now()).First(&subscription).Error; err != nil {
		response.NotFound(c, "Subscription")
		return
	}

	response.OK(c, subscription)
}
