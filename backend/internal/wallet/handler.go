package wallet

import (
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
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

// GetWallet returns user's wallet with balances
func (h *Handler) GetWallet(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var wallet Wallet
	if err := h.db.Preload("Balances").Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	response.OK(c, wallet)
}

// GetBalance returns balance for specific currency
func (h *Handler) GetBalance(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	currency := Currency(c.Param("currency"))

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

	var req DepositReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "Amount must be positive")
		return
	}

	var wallet Wallet
	if err := h.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	var balance WalletBalance
	if err := h.db.Where("wallet_id = ? AND currency = ?", wallet.ID, req.Currency).First(&balance).Error; err != nil {
		response.BadRequest(c, "Currency not supported")
		return
	}

	// Update balance
	newBalance := balance.Balance.Add(req.Amount)
	balance.Balance = newBalance
	balance.AvailableBalance = newBalance
	balance.UpdatedAt = time.Now()

	if err := h.db.Save(&balance).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Create transaction record
	refID := req.ReferenceID
	refType := "deposit"
	now := time.Now()
	tx := WalletTransaction{
		WalletID:      wallet.ID,
		Type:          TransactionDeposit,
		Currency:      req.Currency,
		Amount:        req.Amount,
		BalanceAfter:  newBalance,
		Status:        StatusCompleted,
		ReferenceID:   &refID,
		ReferenceType: &refType,
		Description:   "Deposit of " + req.Amount.String() + " " + string(req.Currency),
		CompletedAt:   &now,
	}
	h.db.Create(&tx)

	response.OK(c, gin.H{
		"transaction_id": tx.ID,
		"currency":       req.Currency,
		"amount":         req.Amount,
		"new_balance":    newBalance,
		"status":         StatusCompleted,
	})
}

type WithdrawReq struct {
	Currency    Currency        `json:"currency" binding:"required"`
	Amount      decimal.Decimal `json:"amount" binding:"required"`
	ReferenceID string          `json:"reference_id"`
}

// Withdraw removes funds from wallet
func (h *Handler) Withdraw(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var req WithdrawReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		response.BadRequest(c, "Amount must be positive")
		return
	}

	var wallet Wallet
	if err := h.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.NotFound(c, "Wallet")
		return
	}

	var balance WalletBalance
	if err := h.db.Where("wallet_id = ? AND currency = ?", wallet.ID, req.Currency).First(&balance).Error; err != nil {
		response.BadRequest(c, "Currency not supported")
		return
	}

	// Check sufficient balance
	if balance.AvailableBalance.LessThan(req.Amount) {
		response.BadRequest(c, "Insufficient balance")
		return
	}

	// Check daily limit
	today := time.Now().Truncate(24 * time.Hour)
	var todayTotal decimal.Decimal
	h.db.Model(&WalletTransaction{}).
		Where("wallet_id = ? AND type = ? AND created_at >= ? AND status = ?",
			wallet.ID, TransactionWithdrawal, today, StatusCompleted).
		Select("COALESCE(SUM(ABS(amount)), 0)").Scan(&todayTotal)

	if todayTotal.Add(req.Amount).GreaterThan(wallet.DailyLimit) {
		response.BadRequest(c, "Daily withdrawal limit exceeded")
		return
	}

	// Update balance
	newBalance := balance.Balance.Sub(req.Amount)
	balance.Balance = newBalance
	balance.AvailableBalance = newBalance
	balance.UpdatedAt = time.Now()

	if err := h.db.Save(&balance).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Create transaction record
	refID := req.ReferenceID
	refType := "withdrawal"
	now := time.Now()
	tx := WalletTransaction{
		WalletID:      wallet.ID,
		Type:          TransactionWithdrawal,
		Currency:      req.Currency,
		Amount:        req.Amount.Neg(),
		BalanceAfter:  newBalance,
		Status:        StatusCompleted,
		ReferenceID:   &refID,
		ReferenceType: &refType,
		Description:   "Withdrawal of " + req.Amount.String() + " " + string(req.Currency),
		CompletedAt:   &now,
	}
	h.db.Create(&tx)

	response.OK(c, gin.H{
		"transaction_id": tx.ID,
		"currency":       req.Currency,
		"amount":         req.Amount,
		"new_balance":    newBalance,
		"status":         StatusCompleted,
	})
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
	buyerUUID, _ := uuid.Parse(buyerID)

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

	// Get buyer wallet
	var buyerWallet Wallet
	if err := h.db.Where("user_id = ?", buyerID).First(&buyerWallet).Error; err != nil {
		response.BadRequest(c, "Buyer wallet not found")
		return
	}

	// Check buyer balance
	var balance WalletBalance
	if err := h.db.Where("wallet_id = ? AND currency = ?", buyerWallet.ID, req.Currency).First(&balance).Error; err != nil {
		response.BadRequest(c, "Currency not supported")
		return
	}

	if balance.AvailableBalance.LessThan(req.Amount) {
		response.BadRequest(c, "Insufficient balance")
		return
	}

	// Calculate fee (2.5%)
	fee := req.Amount.Mul(decimal.NewFromFloat(0.025))

	// Create escrow
	escrow := Escrow{
		BuyerID:     buyerUUID,
		SellerID:    sellerUUID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Fee:         fee,
		Status:      StatusPending,
		ReferenceID: req.ReferenceID,
		Type:        req.Type,
	}

	if err := h.db.Create(&escrow).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Lock buyer funds (move to pending)
	balance.AvailableBalance = balance.AvailableBalance.Sub(req.Amount)
	balance.PendingBalance = balance.PendingBalance.Add(req.Amount)
	h.db.Save(&balance)

	// Create transaction
	refType := "escrow"
	tx := WalletTransaction{
		WalletID:      buyerWallet.ID,
		Type:          TransactionEscrow,
		Currency:      req.Currency,
		Amount:        req.Amount.Neg(),
		BalanceAfter:  balance.Balance,
		Status:        StatusPending,
		ReferenceID:   &req.ReferenceID,
		ReferenceType: &refType,
		Description:   "Escrow for " + req.Type + " #" + req.ReferenceID,
	}
	h.db.Create(&tx)

	response.Created(c, escrow)
}

// ReleaseEscrow releases funds to seller
func (h *Handler) ReleaseEscrow(c *gin.Context) {
	escrowID := c.Param("id")

	var escrow Escrow
	if err := h.db.First(&escrow, "id = ?", escrowID).Error; err != nil {
		response.NotFound(c, "Escrow")
		return
	}

	if escrow.Status != StatusPending {
		response.BadRequest(c, "Escrow already processed")
		return
	}

	// Get buyer and seller wallets
	var buyerWallet, sellerWallet Wallet
	h.db.Where("user_id = ?", escrow.BuyerID).First(&buyerWallet)
	h.db.Where("user_id = ?", escrow.SellerID).First(&sellerWallet)

	// Get balances
	var buyerBalance, sellerBalance WalletBalance
	h.db.Where("wallet_id = ? AND currency = ?", buyerWallet.ID, escrow.Currency).First(&buyerBalance)
	h.db.Where("wallet_id = ? AND currency = ?", sellerWallet.ID, escrow.Currency).First(&sellerBalance)

	// Release funds: deduct from buyer pending, add to seller (minus fee)
	buyerBalance.PendingBalance = buyerBalance.PendingBalance.Sub(escrow.Amount)
	buyerBalance.Balance = buyerBalance.Balance.Sub(escrow.Amount)
	h.db.Save(&buyerBalance)

	sellerAmount := escrow.Amount.Sub(escrow.Fee)
	sellerBalance.Balance = sellerBalance.Balance.Add(sellerAmount)
	sellerBalance.AvailableBalance = sellerBalance.AvailableBalance.Add(sellerAmount)
	h.db.Save(&sellerBalance)

	// Update escrow
	now := time.Now()
	escrow.Status = StatusCompleted
	escrow.ReleasedAt = &now
	h.db.Save(&escrow)

	// Create transactions
	refType := "escrow_release"
	h.db.Create(&WalletTransaction{
		WalletID:      sellerWallet.ID,
		Type:          TransactionRelease,
		Currency:      escrow.Currency,
		Amount:        sellerAmount,
		BalanceAfter:  sellerBalance.Balance,
		Fee:           escrow.Fee,
		Status:        StatusCompleted,
		ReferenceID:   &escrow.ReferenceID,
		ReferenceType: &refType,
		Description:   "Escrow release for " + escrow.Type + " #" + escrow.ReferenceID,
		CompletedAt:   &now,
	})

	response.OK(c, gin.H{
		"escrow_id":     escrow.ID,
		"seller_amount": sellerAmount,
		"fee":           escrow.Fee,
		"status":        StatusCompleted,
	})
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

	// Get plan
	var plan PricePlan
	if err := h.db.First(&plan, "id = ?", req.PlanID).Error; err != nil {
		response.NotFound(c, "Plan")
		return
	}

	// Check if user has active subscription
	var existing UserSubscription
	if h.db.Where("user_id = ? AND is_active = ? AND end_date > ?", userID, true, time.Now()).First(&existing).Error == nil {
		response.Conflict(c, "Active subscription exists")
		return
	}

	// Get wallet and check balance
	var wallet Wallet
	if err := h.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		response.BadRequest(c, "Wallet not found")
		return
	}

	var balance WalletBalance
	if err := h.db.Where("wallet_id = ? AND currency = ?", wallet.ID, plan.Currency).First(&balance).Error; err != nil {
		response.BadRequest(c, "Currency not supported")
		return
	}

	if balance.AvailableBalance.LessThan(plan.Price) {
		response.BadRequest(c, "Insufficient balance")
		return
	}

	// Deduct payment
	balance.Balance = balance.Balance.Sub(plan.Price)
	balance.AvailableBalance = balance.AvailableBalance.Sub(plan.Price)
	h.db.Save(&balance)

	// Create subscription
	now := time.Now()
	subscription := UserSubscription{
		UserID:    userUUID,
		PlanID:    plan.ID,
		StartDate: now,
		EndDate:   now.AddDate(0, 0, plan.DurationDays),
		IsActive:  true,
		AutoRenew: false,
	}
	h.db.Create(&subscription)

	// Create transaction
	planID := plan.ID.String()
	refType := "subscription"
	h.db.Create(&WalletTransaction{
		WalletID:      wallet.ID,
		Type:          TransactionPayment,
		Currency:      plan.Currency,
		Amount:        plan.Price.Neg(),
		BalanceAfter:  balance.Balance,
		Status:        StatusCompleted,
		ReferenceID:   &planID,
		ReferenceType: &refType,
		Description:   "Subscription to " + plan.Name,
		CompletedAt:   &now,
	})

	subscription.Plan = plan
	response.Created(c, subscription)
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
