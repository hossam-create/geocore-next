package disputes

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDisputeSLADB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS disputes (
			id TEXT PRIMARY KEY,
			order_id TEXT,
			auction_id TEXT,
			escrow_id TEXT,
			buyer_id TEXT NOT NULL,
			seller_id TEXT NOT NULL,
			assigned_to TEXT,
			reason TEXT,
			description TEXT,
			amount REAL,
			currency TEXT,
			status TEXT,
			priority INTEGER,
			resolution TEXT,
			resolution_amount REAL,
			resolution_notes TEXT,
			resolved_by TEXT,
			resolved_at DATETIME,
			response_deadline DATETIME,
			resolution_deadline DATETIME,
			sla_breached BOOLEAN DEFAULT 0,
			escalation_date DATETIME,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS dispute_activities (
			id TEXT PRIMARY KEY,
			dispute_id TEXT,
			actor_id TEXT,
			action TEXT,
			details TEXT,
			created_at DATETIME
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			type TEXT,
			title TEXT,
			body TEXT,
			data TEXT,
			read BOOLEAN,
			read_at DATETIME,
			created_at DATETIME,
			deleted_at DATETIME
		);
	`).Error)
	return db
}

func TestSLAHoursForPriority(t *testing.T) {
	_, lowRes := slaHoursForPriority(5)
	_, medRes := slaHoursForPriority(3)
	_, highRes := slaHoursForPriority(1)
	require.Equal(t, 72, lowRes)
	require.Equal(t, 48, medRes)
	require.Equal(t, 24, highRes)
}

func TestMarkSLABreaches(t *testing.T) {
	db := setupDisputeSLADB(t)
	h := NewHandler(db)
	now := time.Now()
	d := Dispute{
		ID:                 uuid.New(),
		BuyerID:            uuid.New(),
		SellerID:           uuid.New(),
		Status:             StatusOpen,
		Priority:           5,
		ResolutionDeadline: ptrTime(now.Add(-2 * time.Hour)),
		SLABreached:        false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	require.NoError(t, db.Create(&d).Error)

	n, err := h.MarkSLABreaches()
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	var updated Dispute
	require.NoError(t, db.First(&updated, "id = ?", d.ID).Error)
	require.True(t, updated.SLABreached)
}

func ptrTime(t time.Time) *time.Time { return &t }
