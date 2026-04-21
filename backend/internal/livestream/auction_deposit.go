package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Auction Entry Deposit (Sprint 10)
//
// High-value auctions require a deposit to participate, preventing fake bidders.
// Feature-flagged via ENABLE_AUCTION_DEPOSIT env var (default: true).
//
// Rules:
//   - Items ≥ 5000 EGP: suggest deposit (5% of price)
//   - Items ≥ 20000 EGP: require deposit
//   - Deposit is reserved (not held) during auction
//   - Winner: deposit converts to part of escrow
//   - Loser: deposit released back to available
//   - Fraud/spam: deposit forfeited
// ════════════════════════════════════════════════════════════════════════════

const (
	depositSuggestThresholdCents int64 = 500_000   // 5000 EGP (in cents)
	depositRequireThresholdCents int64 = 2_000_000 // 20000 EGP (in cents)
	depositPercentage                  = 0.05      // 5% of item price
	depositBidCoverage                 = 0.20      // 20% of current highest bid
)

// AuctionDeposit tracks a user's deposit for a specific auction item.
type AuctionDeposit struct {
	ID             uuid.UUID            `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ItemID         uuid.UUID            `gorm:"type:uuid;not null;index"                        json:"item_id"`
	UserID         uuid.UUID            `gorm:"type:uuid;not null;index"                        json:"user_id"`
	DepositCents   int64                `gorm:"not null"                                        json:"deposit_cents"`
	Status         AuctionDepositStatus `gorm:"size:20;not null;default:'held'"               json:"status"`
	IdempotencyKey string               `gorm:"size:255;index"                                 json:"idempotency_key,omitempty"`
	ForfeitedAt    *time.Time           `json:"forfeited_at,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type AuctionDepositStatus string

const (
	DepositHeld      AuctionDepositStatus = "held"
	DepositReleased  AuctionDepositStatus = "released"
	DepositConverted AuctionDepositStatus = "converted"
	DepositForfeited AuctionDepositStatus = "forfeited"
)

func (AuctionDeposit) TableName() string { return "auction_deposits" }

// IsAuctionDepositEnabled returns true unless explicitly disabled via env.
func IsAuctionDepositEnabled() bool {
	val := os.Getenv("ENABLE_AUCTION_DEPOSIT")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// CalculateDepositAmount returns the deposit amount in cents for a given item price.
// Returns 0 if deposit is not applicable.
func CalculateDepositAmount(itemPriceCents int64) (amountCents int64, required bool) {
	if !IsAuctionDepositEnabled() {
		return 0, false
	}
	if itemPriceCents >= depositRequireThresholdCents {
		return int64(float64(itemPriceCents) * depositPercentage), true
	}
	if itemPriceCents >= depositSuggestThresholdCents {
		return int64(float64(itemPriceCents) * depositPercentage), false
	}
	return 0, false
}

// HasUserPaidDeposit checks if a user has paid the deposit for an item.
func HasUserPaidDeposit(db *gorm.DB, userID, itemID uuid.UUID) bool {
	var count int64
	db.Model(&AuctionDeposit{}).
		Where("user_id = ? AND item_id = ? AND status = ?", userID, itemID, DepositHeld).
		Count(&count)
	return count > 0
}

// GetUserDepositAmount returns the deposit amount a user has paid for an item.
func GetUserDepositAmount(db *gorm.DB, userID, itemID uuid.UUID) int64 {
	var deposit AuctionDeposit
	if err := db.Where("user_id = ? AND item_id = ? AND status = ?", userID, itemID, DepositHeld).
		First(&deposit).Error; err != nil {
		return 0
	}
	return deposit.DepositCents
}

// CalculateRequiredDeposit computes the dynamic deposit needed for an item.
// Uses max(5% of item price, 20% of current highest bid) to prevent
// users from entering with a small deposit and then placing huge bids.
func CalculateRequiredDeposit(itemPriceCents, currentBidCents int64) int64 {
	priceDeposit := int64(float64(itemPriceCents) * depositPercentage)
	bidDeposit := int64(float64(currentBidCents) * depositBidCoverage)

	required := priceDeposit
	if bidDeposit > required {
		required = bidDeposit
	}
	return required
}

// ValidateDepositCoverage checks if a user's deposit still covers the
// required amount given the current highest bid. Returns true if OK,
// false if the user needs to top up their deposit.
func ValidateDepositCoverage(db *gorm.DB, userID, itemID uuid.UUID, itemPriceCents, currentBidCents int64) (bool, int64) {
	userDeposit := GetUserDepositAmount(db, userID, itemID)
	required := CalculateRequiredDeposit(itemPriceCents, currentBidCents)
	return userDeposit >= required, required - userDeposit
}

// ── POST /api/v1/livestream/:id/items/:itemId/deposit — pay deposit ──────

func (h *LiveAuctionHandler) PayDeposit(c *gin.Context) {
	if !IsAuctionDepositEnabled() {
		response.BadRequest(c, "Auction deposit is not enabled")
		return
	}

	// ── Global panic check ──────────────────────────────────────────────────
	if IsLiveSystemDisabled() {
		response.BadRequest(c, "Live system temporarily disabled")
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// Idempotency
	idemKey := c.GetHeader("X-Idempotency-Key")

	var item LiveItem
	var depositCents int64

	err := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND session_id = ?", itemID, sessionID).
			First(&item).Error; err != nil {
			return fmt.Errorf("item_not_found")
		}

		if !item.RequiresEntryDeposit {
			return fmt.Errorf("deposit_not_required")
		}

		// Check if already paid
		var existing AuctionDeposit
		if err := tx.Where("user_id = ? AND item_id = ? AND status = ?",
			userID, itemID, DepositHeld).First(&existing).Error; err == nil {
			return fmt.Errorf("deposit_already_paid")
		}

		depositCents = item.EntryDepositCents
		if depositCents <= 0 {
			return fmt.Errorf("invalid_deposit_amount")
		}

		// Reserve funds
		if !wallet.HasSufficientBalance(h.db, userID, depositCents) {
			return fmt.Errorf("insufficient_balance")
		}
		if err := wallet.ReserveFunds(tx, userID, depositCents); err != nil {
			return fmt.Errorf("reserve_failed:%s", err.Error())
		}

		// Create deposit record
		deposit := AuctionDeposit{
			ItemID:         itemID,
			UserID:         userID,
			DepositCents:   depositCents,
			Status:         DepositHeld,
			IdempotencyKey: idemKey,
		}
		if err := tx.Create(&deposit).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		msg := err.Error()
		switch {
		case msg == "item_not_found":
			response.NotFound(c, "Item")
		case msg == "deposit_not_required":
			response.BadRequest(c, "Deposit not required for this item")
		case msg == "deposit_already_paid":
			response.OK(c, gin.H{"message": "Deposit already paid"})
		case msg == "insufficient_balance":
			response.BadRequest(c, "Insufficient balance for deposit")
		case len(msg) > 15 && msg[:15] == "reserve_failed:":
			response.BadRequest(c, "Insufficient balance: "+msg[15:])
		default:
			response.InternalError(c, err)
		}
		return
	}

	freeze.LogAudit(h.db, "auction_deposit_paid", userID, itemID,
		fmt.Sprintf("deposit_cents=%d", depositCents))
	response.OK(c, gin.H{
		"message":       "Deposit paid successfully",
		"deposit_cents": depositCents,
	})
}

// ── Deposit release/convert/forfeit helpers ──────────────────────────────

// ReleaseDepositForNonWinner releases a loser's deposit back to available.
func ReleaseDepositForNonWinner(db *gorm.DB, userID, itemID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var deposit AuctionDeposit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND item_id = ? AND status = ?", userID, itemID, DepositHeld).
			First(&deposit).Error; err != nil {
			return err // no deposit found — skip
		}

		if err := wallet.ReleaseReservedFunds(tx, userID, deposit.DepositCents); err != nil {
			return err
		}

		return tx.Model(&deposit).Update("status", DepositReleased).Error
	})
}

// ConvertDepositToEscrow converts winner's deposit to part of the escrow hold.
func ConvertDepositToEscrow(db *gorm.DB, userID, itemID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var deposit AuctionDeposit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND item_id = ? AND status = ?", userID, itemID, DepositHeld).
			First(&deposit).Error; err != nil {
			return err
		}

		// Release the reserve first, then the bid amount already covers escrow
		// The deposit reserve is merged into the bid's escrow hold
		if err := wallet.ReleaseReservedFunds(tx, userID, deposit.DepositCents); err != nil {
			return err
		}

		return tx.Model(&deposit).Update("status", DepositConverted).Error
	})
}

// ForfeitDeposit seizes a user's deposit (partial or full) for fraud/spam.
func ForfeitDeposit(db *gorm.DB, userID, itemID uuid.UUID, adminID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var deposit AuctionDeposit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND item_id = ? AND status = ?", userID, itemID, DepositHeld).
			First(&deposit).Error; err != nil {
			return err
		}

		// Release reserved funds (they go back to available, then we hold as platform fee)
		if err := wallet.ReleaseReservedFunds(tx, userID, deposit.DepositCents); err != nil {
			slog.Error("auction-deposit: failed to release forfeited deposit reserve",
				"user", userID, "amount", deposit.DepositCents, "error", err)
			return err
		}

		now := time.Now()
		return tx.Model(&deposit).Updates(map[string]interface{}{
			"status":       DepositForfeited,
			"forfeited_at": now,
		}).Error
	})
}

// ── Auto-set deposit requirement on AddItem ──────────────────────────────

// ApplyDepositRules sets the deposit fields on an item based on its price.
func ApplyDepositRules(item *LiveItem) {
	if !IsAuctionDepositEnabled() {
		return
	}

	price := item.StartPriceCents
	if item.BuyNowPriceCents != nil && *item.BuyNowPriceCents > price {
		price = *item.BuyNowPriceCents
	}

	amount, required := CalculateDepositAmount(price)
	if amount > 0 {
		item.RequiresEntryDeposit = required
		item.EntryDepositCents = amount
	}
}

// releaseLoserDeposits releases deposits for all non-winners of an item.
func (h *LiveAuctionHandler) releaseLoserDeposits(itemID, winnerID uuid.UUID) {
	var deposits []AuctionDeposit
	h.db.Where("item_id = ? AND status = ? AND user_id != ?", itemID, DepositHeld, winnerID).Find(&deposits)
	for _, dep := range deposits {
		if err := ReleaseDepositForNonWinner(h.db, dep.UserID, itemID); err != nil {
			slog.Error("live-auction: failed to release loser deposit",
				"user", dep.UserID, "item", itemID, "error", err)
		}
	}
	if len(deposits) > 0 {
		slog.Info("live-auction: released loser deposits",
			"item_id", itemID, "count", len(deposits))
	}
}
