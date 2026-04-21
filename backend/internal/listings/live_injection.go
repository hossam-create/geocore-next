package listings

// Sprint 18 — Live Session Injection into Search Results.
// Cross-cuts livestream.Session to surface active live rooms matching the query.
// Read-only; no writes. Fails silently if livestream_sessions table is empty.

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/google/uuid"
)

// LiveResult is a minimal live-session card surfaced in search results.
type LiveResult struct {
	SessionID    uuid.UUID `json:"session_id"`
	Title        string    `json:"title"`
	HostID       uuid.UUID `json:"host_id"`
	Viewers      int       `json:"viewers"`
	BoostScore   int       `json:"boost_score,omitempty"`
	IsHot        bool      `json:"is_hot,omitempty"`
	IsPremium    bool      `json:"is_premium,omitempty"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	Urgency      string    `json:"urgency,omitempty"` // normal | hot | very_hot
}

// findMatchingLiveSessions returns up to `limit` active live sessions whose title/description
// matches the query (case-insensitive). If query is empty, returns top hot/boosted sessions.
// Ranking: is_premium DESC, is_hot DESC, boost_score DESC, viewer_count DESC.
func (h *Handler) findMatchingLiveSessions(ctx context.Context, query string, limit int) []LiveResult {
	if !config.GetFlags().EnableLiveInSearch {
		return nil
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	// Lightweight anonymous scan struct — avoid importing livestream package (avoids cycles).
	type row struct {
		ID           uuid.UUID  `gorm:"column:id"`
		Title        string     `gorm:"column:title"`
		HostID       uuid.UUID  `gorm:"column:host_id"`
		ViewerCount  int        `gorm:"column:viewer_count"`
		BoostScore   int        `gorm:"column:boost_score"`
		IsHot        bool       `gorm:"column:is_hot"`
		IsPremium    bool       `gorm:"column:is_premium"`
		ThumbnailURL string     `gorm:"column:thumbnail_url"`
		StartedAt    *time.Time `gorm:"column:started_at"`
	}

	// Short DB timeout so a slow query can never drag down /search.
	cctx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	defer cancel()

	db := h.readDB().WithContext(cctx).
		Table("livestream_sessions").
		Where("status = ? AND deleted_at IS NULL", "live")

	q := strings.TrimSpace(query)
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("title ILIKE ? OR description ILIKE ?", like, like)
	}

	var rows []row
	if err := db.
		Order("is_premium DESC, is_hot DESC, boost_score DESC, viewer_count DESC, started_at DESC NULLS LAST").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		slog.Debug("live injection skipped", "err", err)
		return nil
	}

	out := make([]LiveResult, 0, len(rows))
	for _, r := range rows {
		out = append(out, LiveResult{
			SessionID:    r.ID,
			Title:        r.Title,
			HostID:       r.HostID,
			Viewers:      r.ViewerCount,
			BoostScore:   r.BoostScore,
			IsHot:        r.IsHot,
			IsPremium:    r.IsPremium,
			ThumbnailURL: r.ThumbnailURL,
			StartedAt:    r.StartedAt,
			Urgency:      urgencyLabel(r.IsHot, r.BoostScore),
		})
	}
	return out
}

func urgencyLabel(isHot bool, boost int) string {
	switch {
	case isHot && boost >= 500:
		return "very_hot"
	case isHot, boost >= 200:
		return "hot"
	default:
		return "normal"
	}
}
