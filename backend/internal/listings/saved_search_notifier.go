package listings

// Sprint 18.2 — Production-Grade Saved Search Notifier.
// Background ticker that matches newly-created listings against saved searches
// with NotifyOnMatch=true and dispatches in-app / push / email via a caller-provided
// NotifyFunc (so we don't import internal/notifications and introduce a cycle).
//
// Phases implemented:
//   Phase 1 — Matching engine (MatchSavedSearch pure fn + MatchSavedSearchDB query builder)
//   Phase 2 — Background job (7-min ticker, listing-centric batch processing)
//   Phase 3 — Redis deduplication (saved_notify:{user_id}:{listing_id}, TTL 24h, atomic SetNX)
//   Phase 4 — Notification dispatch (NotifyFunc injection)
//   Phase 5 — Batch notifications per user (group matches, single notification)
//   Phase 6 — Rate limiting (notify_limit:{user_id}:{date}, max 3/day, atomic Incr)
//   Phase 7 — Async goroutines + timeout safety per batch
//   Phase 8 — Feature flag (ENABLE_SAVED_SEARCH_NOTIFICATIONS)
//   Phase 9 — Failure handling with retry + failed_notifications logging
//   Phase 10 — pgvector similarity for fuzzy match (when available)

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NotifyFunc is the injection point for dispatching a notification.
// Wire in cmd/api/main.go against notifications.Service.Notify.
type NotifyFunc func(userID uuid.UUID, notifType, title, body string, data map[string]string)

// userMatch tracks one listing that matched a saved search for a specific user.
type userMatch struct {
	SavedSearchID uuid.UUID
	UserID        uuid.UUID // the user who owns the saved search (notification target)
	Query         string
	Label         string
	Listing       Listing
}

// SavedSearchNotifier polls saved_searches and dispatches notifications for matches.
type SavedSearchNotifier struct {
	db           *gorm.DB
	rdb          *redis.Client
	notify       NotifyFunc
	interval     time.Duration
	MaxBatch     int           // listings fetched per tick (Phase 7)
	MaxPerSearch int           // max results per saved search per tick
	MinGap       time.Duration // minimum time between two notifications for the same saved search
	MaxDaily     int           // max notifications per user per day (Phase 6)
	Workers      int           // concurrent goroutines for matching (Phase 7)
	MaxRetries   int           // notification send retries (Phase 9)
}

// NewSavedSearchNotifier builds the notifier with safe defaults.
func NewSavedSearchNotifier(db *gorm.DB, rdb *redis.Client, notify NotifyFunc) *SavedSearchNotifier {
	return &SavedSearchNotifier{
		db:           db,
		rdb:          rdb,
		notify:       notify,
		interval:     7 * time.Minute,
		MaxBatch:     200,
		MaxPerSearch: 50,
		MinGap:       30 * time.Minute,
		MaxDaily:     3,
		Workers:      10,
		MaxRetries:   3,
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Phase 1 — Matching Engine (pure + DB query builder)
// ════════════════════════════════════════════════════════════════════════════

// MatchSavedSearch checks whether a listing matches a saved search's filters.
// Pure function — no DB access, suitable for unit testing.
func MatchSavedSearch(ss SavedSearch, l Listing, categoryPath string) bool {
	filters := map[string]interface{}{}
	if ss.Filters != "" && ss.Filters != "{}" {
		_ = json.Unmarshal([]byte(ss.Filters), &filters)
	}
	if v, ok := filters["category_id"].(string); ok && v != "" {
		if l.CategoryID.String() != v {
			return false
		}
	} else if v, ok := filters["category_path"].(string); ok && v != "" {
		if categoryPath != v && !strings.HasPrefix(categoryPath, v+"/") {
			return false
		}
	}
	if v, ok := filters["min_price"].(float64); ok {
		if l.Price == nil || *l.Price < v {
			return false
		}
	}
	if v, ok := filters["max_price"].(float64); ok {
		if l.Price == nil || *l.Price > v {
			return false
		}
	}
	if v, ok := filters["condition"].(string); ok && v != "" {
		if l.Condition != v {
			return false
		}
	}
	if v, ok := filters["city"].(string); ok && v != "" {
		if !strings.EqualFold(l.City, v) {
			return false
		}
	}
	if v, ok := filters["country"].(string); ok && v != "" {
		if !strings.EqualFold(l.Country, v) {
			return false
		}
	}
	return true
}

// MatchSavedSearchDB builds a GORM query applying the saved search's
// text query (tsvector) + filters against the listings table.
func MatchSavedSearchDB(db *gorm.DB, ss SavedSearch) *gorm.DB {
	filters := map[string]interface{}{}
	if ss.Filters != "" && ss.Filters != "{}" {
		_ = json.Unmarshal([]byte(ss.Filters), &filters)
	}
	q := db.Model(&Listing{}).Where("status = ?", "active")
	if strings.TrimSpace(ss.Query) != "" {
		q = q.Where("to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', ?)", ss.Query)
	}
	if v, ok := filters["category_id"].(string); ok && v != "" {
		q = q.Where("category_id = ?", v)
	} else if v, ok := filters["category"].(string); ok && v != "" {
		q = q.Where("category_id = (SELECT id FROM categories WHERE slug = ? LIMIT 1)", v)
	} else if v, ok := filters["category_path"].(string); ok && v != "" {
		q = q.Where("category_id IN (SELECT id FROM categories WHERE path = ? OR path LIKE ?)", v, v+"/%")
	}
	if v, ok := filters["min_price"].(float64); ok {
		q = q.Where("price >= ?", v)
	}
	if v, ok := filters["max_price"].(float64); ok {
		q = q.Where("price <= ?", v)
	}
	if v, ok := filters["condition"].(string); ok && v != "" {
		q = q.Where("condition = ?", v)
	}
	if v, ok := filters["city"].(string); ok && v != "" {
		q = q.Where("LOWER(city) = LOWER(?)", v)
	}
	if v, ok := filters["country"].(string); ok && v != "" {
		q = q.Where("LOWER(country) = LOWER(?)", v)
	}
	return q
}

// ════════════════════════════════════════════════════════════════════════════
// Phase 10 — pgvector similarity (optional, graceful fallback)
// ════════════════════════════════════════════════════════════════════════════

// SimilarityMatch uses pgvector cosine similarity for fuzzy matching.
func SimilarityMatch(db *gorm.DB, query string, limit int, threshold float64) ([]uuid.UUID, error) {
	if query == "" || db == nil {
		return nil, nil
	}
	var ids []uuid.UUID
	err := db.Raw(`SELECT listing_id FROM listing_embeddings WHERE embedding <=> (SELECT embedding FROM query_embeddings WHERE query_text = ? LIMIT 1) < ? ORDER BY embedding <=> (SELECT embedding FROM query_embeddings WHERE query_text = ? LIMIT 1) LIMIT ?`, query, 1-threshold, query, limit).Scan(&ids).Error
	if err != nil {
		slog.Debug("similarity_match: pgvector not available", "err", err)
		return nil, nil
	}
	return ids, nil
}

// Start runs the ticker until ctx is cancelled.
// Safe to call `go n.Start(ctx)` from main.
func (n *SavedSearchNotifier) Start(ctx context.Context) {
	if !config.GetFlags().EnableSavedSearch {
		slog.Info("saved search notifier disabled via flag")
		return
	}
	if n.notify == nil {
		slog.Warn("saved search notifier: NotifyFunc is nil, will only log matches")
	}
	t := time.NewTicker(n.interval)
	defer t.Stop()
	slog.Info("saved search notifier started",
		"interval", n.interval,
		"max_daily", n.MaxDaily,
		"workers", n.Workers)
	// Run once immediately on startup so operators see feedback without waiting a full cycle.
	n.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("saved search notifier stopping")
			return
		case <-t.C:
			n.tick(ctx)
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 2 — Background tick
// ──────────────────────────────────────────────────────────────────────────────

// tick fetches new listings since last run, matches against saved searches,
// groups by user, deduplicates, rate-limits, and dispatches batched notifications.
func (n *SavedSearchNotifier) tick(ctx context.Context) {
	if n.db == nil {
		return
	}
	since := time.Now().Add(-2 * n.interval)

	// Phase 7 — Process listings in batches.
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var batch []Listing
		if err := n.db.WithContext(ctx).
			Where("status = ? AND created_at > ?", "active", since).
			Order("created_at ASC").
			Offset(offset).Limit(n.MaxBatch).
			Find(&batch).Error; err != nil {
			slog.Warn("saved search notifier: listing batch query failed", "err", err)
			return
		}
		if len(batch) == 0 {
			break
		}
		batchCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		n.processBatch(batchCtx, batch)
		cancel()
		offset += len(batch)
		if len(batch) < n.MaxBatch {
			break
		}
	}
}

// processBatch matches a batch of listings against eligible saved searches.
func (n *SavedSearchNotifier) processBatch(ctx context.Context, listings []Listing) {
	cutoff := time.Now().Add(-n.MinGap)
	var searches []SavedSearch
	if err := n.db.WithContext(ctx).
		Where("notify_on_match = ? AND (last_notified_at IS NULL OR last_notified_at < ?)", true, cutoff).
		Order("updated_at ASC").Limit(500).
		Find(&searches).Error; err != nil {
		slog.Warn("saved search notifier: search query failed", "err", err)
		return
	}
	if len(searches) == 0 {
		return
	}

	userMatches := make(map[uuid.UUID][]userMatch)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, n.Workers)

	for i := range searches {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ss := &searches[i]
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			matched := n.matchSearchToListings(ctx, ss, listings)
			if len(matched) == 0 {
				return
			}
			mu.Lock()
			for _, l := range matched {
				if !n.checkAndMarkDedup(ctx, ss.UserID, l.ID) {
					continue
				}
				userMatches[ss.UserID] = append(userMatches[ss.UserID], userMatch{
					SavedSearchID: ss.ID,
					UserID:        ss.UserID,
					Query:         ss.Query,
					Label:         ss.Label,
					Listing:       l,
				})
			}
			mu.Unlock()
			now := time.Now()
			if err := n.db.WithContext(ctx).Model(ss).Update("last_notified_at", now).Error; err != nil {
				slog.Warn("saved search notifier: update last_notified_at failed", "id", ss.ID, "err", err)
			}
		}()
	}
	wg.Wait()

	for userID, matches := range userMatches {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n.dispatchUser(ctx, userID, matches)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 1 — Matching engine
// ──────────────────────────────────────────────────────────────────────────────

// matchSearchToListings applies a saved search against candidate listings using DB query.
func (n *SavedSearchNotifier) matchSearchToListings(ctx context.Context, ss *SavedSearch, candidates []Listing) []Listing {
	q := MatchSavedSearchDB(n.db.WithContext(ctx), *ss)
	ids := make([]uuid.UUID, 0, len(candidates))
	for _, c := range candidates {
		ids = append(ids, c.ID)
	}
	q = q.Where("id IN ?", ids)

	var matched []Listing
	if err := q.Limit(n.MaxPerSearch).Find(&matched).Error; err != nil {
		slog.Warn("saved search notifier: match query failed", "id", ss.ID, "err", err)
		return nil
	}

	// Phase 10 — Boost with pgvector similarity if available.
	if config.GetFlags().EnableSavedSearchNotifications && strings.TrimSpace(ss.Query) != "" {
		similarIDs, err := SimilarityMatch(n.db, ss.Query, 20, 0.7)
		if err == nil && len(similarIDs) > 0 {
			existing := make(map[uuid.UUID]bool, len(matched))
			for _, l := range matched {
				existing[l.ID] = true
			}
			var extra []Listing
			n.db.WithContext(ctx).Where("id IN ? AND status = ?", similarIDs, "active").Limit(10).Find(&extra)
			for _, l := range extra {
				if !existing[l.ID] && MatchSavedSearch(*ss, l, "") {
					matched = append(matched, l)
				}
			}
		}
	}
	return matched
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 3+5+6 — Deduplication, batching, rate limiting
// ──────────────────────────────────────────────────────────────────────────────

// dispatchUser sends a single grouped notification for one user after
// deduplication, feature flag, and rate-limit checks.
func (n *SavedSearchNotifier) dispatchUser(ctx context.Context, userID uuid.UUID, matches []userMatch) {
	// Phase 8 — Feature flag gate
	if !config.GetFlags().EnableSavedSearchNotifications {
		return
	}

	// Phase 6 — Rate limit: max MaxDaily notifications per user per day.
	if !n.checkRateLimit(ctx, userID) {
		slog.Debug("saved search notifier: rate limited",
			"user_id", userID.String(), "matches_skipped", len(matches))
		return
	}

	// Phase 5 — Build grouped notification body.
	title, body, data := n.buildGroupedNotification(matches)

	// Phase 9 — Send with retry.
	if err := n.sendWithRetry(ctx, userID, title, body, data); err != nil {
		slog.Warn("saved search notifier: send failed after retries",
			"user_id", userID.String(), "err", err)
		notifications.LogFailedNotification(n.db, userID, "in_app", "saved_search_match",
			map[string]interface{}{
				"match_count": len(matches),
				"listing_ids": data["listing_ids"],
				"title":       title,
				"body":        body,
			}, err.Error())
		return
	}

	slog.Info("saved search notified",
		"user_id", userID.String(),
		"matches", len(matches))
}

// checkAndMarkDedup returns true if this user+listing pair has NOT been notified
// within the dedup window, and atomically marks it. Uses SetNX for atomicity.
func (n *SavedSearchNotifier) checkAndMarkDedup(ctx context.Context, userID, listingID uuid.UUID) bool {
	if n.rdb == nil {
		return true // no Redis → no dedup (graceful degradation)
	}
	key := fmt.Sprintf("saved_notify:%s:%s", userID.String(), listingID.String())
	set, err := n.rdb.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		slog.Warn("saved search dedup: Redis error, allowing notification", "err", err)
		return true // fail open
	}
	return set
}

// checkRateLimit returns true if the user is under the daily notification limit.
// Atomically increments the counter using Incr.
func (n *SavedSearchNotifier) checkRateLimit(ctx context.Context, userID uuid.UUID) bool {
	if n.rdb == nil {
		return true
	}
	today := time.Now().Format("2006-01-02")
	key := fmt.Sprintf("notify_limit:%s:%s", userID.String(), today)
	count, err := n.rdb.Incr(ctx, key).Result()
	if err != nil {
		slog.Warn("saved search rate limit: Redis error, allowing", "err", err)
		return true
	}
	if count == 1 {
		n.rdb.Expire(ctx, key, 25*time.Hour)
	}
	return count <= int64(n.MaxDaily)
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 5 — Build grouped notification
// ──────────────────────────────────────────────────────────────────────────────

// buildGroupedNotification creates a single notification summarizing all matches
// for a user across multiple saved searches.
func (n *SavedSearchNotifier) buildGroupedNotification(matches []userMatch) (string, string, map[string]string) {
	total := len(matches)

	// Collect unique search labels.
	searchLabels := map[string]bool{}
	for _, m := range matches {
		lbl := m.Query
		if lbl == "" && m.Label != "" {
			lbl = m.Label
		}
		if lbl != "" {
			searchLabels[lbl] = true
		}
	}

	var labelStr string
	switch len(searchLabels) {
	case 0:
		labelStr = "your saved searches"
	case 1:
		for l := range searchLabels {
			labelStr = fmt.Sprintf("%q", l)
		}
	default:
		labelStr = fmt.Sprintf("%d saved searches", len(searchLabels))
	}

	title := fmt.Sprintf("%d new listing%s for %s", total, pluralS(total), labelStr)

	// Body: first 3 listing titles.
	firstTitles := make([]string, 0, 3)
	for i := 0; i < total && i < 3; i++ {
		firstTitles = append(firstTitles, matches[i].Listing.Title)
	}
	body := strings.Join(firstTitles, " • ")
	if total > 3 {
		body += fmt.Sprintf(" + %d more", total-3)
	}

	// Data payload.
	ids := make([]string, 0, total)
	for _, m := range matches {
		ids = append(ids, m.Listing.ID.String())
	}
	searchIDs := make([]string, 0, len(searchLabels))
	seenSearch := map[uuid.UUID]bool{}
	for _, m := range matches {
		if !seenSearch[m.SavedSearchID] {
			seenSearch[m.SavedSearchID] = true
			searchIDs = append(searchIDs, m.SavedSearchID.String())
		}
	}

	data := map[string]string{
		"type":             "saved_search_match",
		"count":            fmt.Sprintf("%d", total),
		"listing_ids":      strings.Join(ids, ","),
		"saved_search_ids": strings.Join(searchIDs, ","),
	}

	return title, body, data
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 9 — Failure handling with retry
// ──────────────────────────────────────────────────────────────────────────────

// sendWithRetry attempts to send a notification up to MaxRetries times.
func (n *SavedSearchNotifier) sendWithRetry(ctx context.Context, userID uuid.UUID, title, body string, data map[string]string) error {
	if n.notify == nil {
		return nil
	}
	var lastErr error
	for attempt := 1; attempt <= n.MaxRetries; attempt++ {
		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("panic in notify: %v", r)
				}
			}()
			n.notify(userID, "saved_search_match", title, body, data)
			lastErr = nil
		}()
		if lastErr == nil {
			return nil
		}
		slog.Warn("saved search notifier: send attempt failed",
			"user_id", userID.String(),
			"attempt", attempt,
			"err", lastErr)
		if attempt < n.MaxRetries {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond) // linear backoff
		}
	}
	return lastErr
}
