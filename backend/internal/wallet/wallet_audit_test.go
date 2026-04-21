package wallet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/users"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type kycProfileStub struct {
	UserID string `gorm:"column:user_id"`
	Status string `gorm:"column:status"`
}

func (kycProfileStub) TableName() string { return "kyc_profiles" }

func setupWalletAuditDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Skip("wallet audit integration tests require PostgreSQL-compatible schema defaults")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.AutoMigrate(
		&users.User{},
		&Wallet{},
		&WalletBalance{},
		&WalletTransaction{},
		&Escrow{},
		&IdempotentRequest{},
		&kycProfileStub{},
	))
	// Create stub user_risk_profiles table for SQLite (fraud model uses PostgreSQL-specific uuid_generate_v4)
	db.Exec(`CREATE TABLE IF NOT EXISTS user_risk_profiles (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		risk_score REAL DEFAULT 0,
		risk_level TEXT DEFAULT 'low',
		total_orders INTEGER DEFAULT 0,
		total_spent REAL DEFAULT 0,
		avg_order_value REAL DEFAULT 0,
		flags TEXT DEFAULT '[]',
		last_assessed DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	return db
}

func createUserWalletWithBalance(t *testing.T, db *gorm.DB, userID uuid.UUID, balance, available, pending decimal.Decimal) Wallet {
	t.Helper()

	u := users.User{
		ID:        userID,
		Name:      "User " + userID.String()[:8],
		Email:     userID.String()[:8] + "@example.com",
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now().Add(-48 * time.Hour),
	}
	require.NoError(t, db.Create(&u).Error)

	w := Wallet{
		ID:              uuid.New(),
		UserID:          userID,
		PrimaryCurrency: USD,
		DailyLimit:      decimal.NewFromInt(100000),
		MonthlyLimit:    decimal.NewFromInt(1000000),
		IsActive:        true,
	}
	require.NoError(t, db.Create(&w).Error)

	wb := WalletBalance{
		ID:               uuid.New(),
		WalletID:         w.ID,
		Currency:         USD,
		Balance:          balance,
		AvailableBalance: available,
		PendingBalance:   pending,
	}
	require.NoError(t, db.Create(&wb).Error)
	require.NoError(t, checkInvariant(wb))

	// Insert stub risk profile via raw SQL for SQLite compatibility
	db.Exec(`INSERT INTO user_risk_profiles (id, user_id, risk_score, risk_level, total_orders, avg_order_value, created_at, updated_at)
		VALUES (?, ?, 0, 'low', 0, 0, ?, ?)`,
		uuid.New().String(), userID.String(), time.Now(), time.Now())

	return w
}

func callJSONHandler(t *testing.T, h gin.HandlerFunc, method, path string, body any, userID uuid.UUID, params gin.Params) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(method, path, bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("user_id", userID.String())
	ctx.Params = params

	h(ctx)
	return rec
}

func TestWithdrawRejectsWhenAmountExceedsAvailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupWalletAuditDB(t)
	h := NewHandler(db, nil)

	userID := uuid.New()
	createUserWalletWithBalance(t, db, userID, decimal.NewFromInt(60), decimal.NewFromInt(20), decimal.NewFromInt(40))

	rec := callJSONHandler(t, h.Withdraw, http.MethodPost, "/wallet/withdraw", WithdrawReq{
		Currency: USD,
		Amount:   decimal.NewFromInt(30),
	}, userID, nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Insufficient balance")

	var wb WalletBalance
	require.NoError(t, db.Where("currency = ?", USD).First(&wb).Error)
	assert.True(t, wb.Balance.Equal(decimal.NewFromInt(60)))
	assert.True(t, wb.AvailableBalance.Equal(decimal.NewFromInt(20)))
	assert.True(t, wb.PendingBalance.Equal(decimal.NewFromInt(40)))
	require.NoError(t, checkInvariant(wb))
}

func TestEscrowReleaseFullFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupWalletAuditDB(t)
	h := NewHandler(db, nil)

	buyerID := uuid.New()
	sellerID := uuid.New()
	buyerWallet := createUserWalletWithBalance(t, db, buyerID, decimal.NewFromInt(100), decimal.NewFromInt(60), decimal.NewFromInt(40))
	sellerWallet := createUserWalletWithBalance(t, db, sellerID, decimal.NewFromInt(10), decimal.NewFromInt(10), decimal.Zero)

	escrow := Escrow{
		ID:          uuid.New(),
		BuyerID:     buyerID,
		SellerID:    sellerID,
		Amount:      decimal.NewFromInt(40),
		Currency:    USD,
		Fee:         decimal.Zero,
		Status:      StatusPending,
		ReferenceID: "ORDER-123",
		Type:        "ORDER",
	}
	require.NoError(t, db.Create(&escrow).Error)

	totalBefore := decimal.NewFromInt(110)

	rec := callJSONHandler(t, h.ReleaseEscrow, http.MethodPost, "/escrow/"+escrow.ID.String()+"/release", map[string]any{}, buyerID,
		gin.Params{{Key: "id", Value: escrow.ID.String()}})
	require.Equal(t, http.StatusOK, rec.Code)

	var buyerBal, sellerBal WalletBalance
	require.NoError(t, db.Where("wallet_id = ? AND currency = ?", buyerWallet.ID, USD).First(&buyerBal).Error)
	require.NoError(t, db.Where("wallet_id = ? AND currency = ?", sellerWallet.ID, USD).First(&sellerBal).Error)

	assert.True(t, buyerBal.PendingBalance.Equal(decimal.Zero), "buyer pending should decrease to zero")
	assert.True(t, buyerBal.AvailableBalance.Equal(decimal.NewFromInt(60)), "buyer available unchanged")
	assert.True(t, buyerBal.Balance.Equal(decimal.NewFromInt(60)), "buyer total should decrease by escrow amount")
	assert.True(t, sellerBal.AvailableBalance.Equal(decimal.NewFromInt(50)), "seller available should increase")
	assert.True(t, sellerBal.Balance.Equal(decimal.NewFromInt(50)), "seller total should increase")

	totalAfter := buyerBal.Balance.Add(sellerBal.Balance)
	assert.True(t, totalAfter.Equal(totalBefore), "system total must remain unchanged (fee=0 path)")

	require.NoError(t, checkInvariant(buyerBal))
	require.NoError(t, checkInvariant(sellerBal))

	var txs []WalletTransaction
	require.NoError(t, db.Where("reference_id = ?", escrow.ReferenceID).Order("created_at asc").Find(&txs).Error)
	require.GreaterOrEqual(t, len(txs), 2, "must create at least buyer debit + seller credit")

	buyerFound := false
	sellerFound := false
	net := decimal.Zero
	for _, tx := range txs {
		net = net.Add(tx.Amount)
		if tx.WalletID == buyerWallet.ID && tx.Type == TransactionPayment {
			buyerFound = true
			assert.True(t, tx.Amount.Equal(decimal.NewFromInt(-40)))
		}
		if tx.WalletID == sellerWallet.ID && tx.Type == TransactionRelease {
			sellerFound = true
			assert.True(t, tx.Amount.Equal(decimal.NewFromInt(40)))
		}
	}

	assert.True(t, buyerFound, "buyer debit transaction must exist")
	assert.True(t, sellerFound, "seller credit transaction must exist")
	assert.True(t, net.Equal(decimal.Zero), "double-entry net should be zero for escrow release (fee=0)")
}

func TestConcurrentWithdrawOnlyOneSucceeds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupWalletAuditDB(t)
	h := NewHandler(db, nil)

	userID := uuid.New()
	wallet := createUserWalletWithBalance(t, db, userID, decimal.NewFromInt(100), decimal.NewFromInt(100), decimal.Zero)

	var wg sync.WaitGroup
	results := make(chan int, 2)

	withdraw := func() {
		defer wg.Done()
		rec := callJSONHandler(t, h.Withdraw, http.MethodPost, "/wallet/withdraw", WithdrawReq{
			Currency: USD,
			Amount:   decimal.NewFromInt(80),
		}, userID, nil)
		results <- rec.Code
	}

	wg.Add(2)
	go withdraw()
	go withdraw()
	wg.Wait()
	close(results)

	success := 0
	failed := 0
	for code := range results {
		if code == http.StatusOK {
			success++
		} else {
			failed++
		}
	}

	assert.Equal(t, 1, success, "only one concurrent withdraw should succeed")
	assert.Equal(t, 1, failed, "one concurrent withdraw should fail")

	var wb WalletBalance
	require.NoError(t, db.Where("wallet_id = ? AND currency = ?", wallet.ID, USD).First(&wb).Error)
	assert.True(t, wb.Balance.Equal(decimal.NewFromInt(20)))
	assert.True(t, wb.AvailableBalance.Equal(decimal.NewFromInt(20)))
	assert.True(t, wb.PendingBalance.Equal(decimal.Zero))
	require.NoError(t, checkInvariant(wb))
}
