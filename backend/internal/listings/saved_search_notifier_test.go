package listings

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// ──────────────────────────────────────────────────────────────────────────────
// Phase 1 — MatchSavedSearch unit tests (pure function, no DB)
// ──────────────────────────────────────────────────────────────────────────────

func TestMatchSavedSearch_CategoryID(t *testing.T) {
	catID := uuid.New()
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"category_id": catID.String()}),
	}
	l := Listing{CategoryID: catID}
	if !MatchSavedSearch(ss, l, "") {
		t.Error("expected match for same category_id")
	}

	otherCat := uuid.New()
	l.CategoryID = otherCat
	if MatchSavedSearch(ss, l, "") {
		t.Error("expected no match for different category_id")
	}
}

func TestMatchSavedSearch_CategoryPath(t *testing.T) {
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"category_path": "electronics"}),
	}

	// Exact path match
	if !MatchSavedSearch(ss, Listing{}, "electronics") {
		t.Error("expected match for exact category_path")
	}

	// Descendant path match
	if !MatchSavedSearch(ss, Listing{}, "electronics/phones") {
		t.Error("expected match for descendant category_path")
	}

	// Unrelated path
	if MatchSavedSearch(ss, Listing{}, "clothing") {
		t.Error("expected no match for unrelated category_path")
	}
}

func TestMatchSavedSearch_PriceRange(t *testing.T) {
	minPrice := 100.0
	maxPrice := 500.0
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{
			"min_price": minPrice,
			"max_price": maxPrice,
		}),
	}

	price := 250.0
	l := Listing{Price: &price}
	if !MatchSavedSearch(ss, l, "") {
		t.Error("expected match for price in range")
	}

	lowPrice := 50.0
	l.Price = &lowPrice
	if MatchSavedSearch(ss, l, "") {
		t.Error("expected no match for price below min")
	}

	highPrice := 600.0
	l.Price = &highPrice
	if MatchSavedSearch(ss, l, "") {
		t.Error("expected no match for price above max")
	}
}

func TestMatchSavedSearch_Condition(t *testing.T) {
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"condition": "new"}),
	}

	if !MatchSavedSearch(ss, Listing{Condition: "new"}, "") {
		t.Error("expected match for same condition")
	}
	if MatchSavedSearch(ss, Listing{Condition: "used"}, "") {
		t.Error("expected no match for different condition")
	}
}

func TestMatchSavedSearch_City(t *testing.T) {
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"city": "Dubai"}),
	}

	if !MatchSavedSearch(ss, Listing{City: "Dubai"}, "") {
		t.Error("expected match for same city")
	}
	if !MatchSavedSearch(ss, Listing{City: "dubai"}, "") {
		t.Error("expected case-insensitive city match")
	}
	if MatchSavedSearch(ss, Listing{City: "Abu Dhabi"}, "") {
		t.Error("expected no match for different city")
	}
}

func TestMatchSavedSearch_Country(t *testing.T) {
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"country": "AE"}),
	}

	if !MatchSavedSearch(ss, Listing{Country: "AE"}, "") {
		t.Error("expected match for same country")
	}
	if MatchSavedSearch(ss, Listing{Country: "US"}, "") {
		t.Error("expected no match for different country")
	}
}

func TestMatchSavedSearch_EmptyFilters(t *testing.T) {
	ss := SavedSearch{Filters: "{}"}
	l := Listing{Price: floatPtr(100)}
	if !MatchSavedSearch(ss, l, "") {
		t.Error("expected match with empty filters (everything passes)")
	}
}

func TestMatchSavedSearch_NilPrice(t *testing.T) {
	ss := SavedSearch{
		Filters: mustJSON(map[string]interface{}{"min_price": 50.0}),
	}
	l := Listing{Price: nil} // nil price
	if MatchSavedSearch(ss, l, "") {
		t.Error("expected no match when listing price is nil and min_price is set")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 5 — buildGroupedNotification tests
// ──────────────────────────────────────────────────────────────────────────────

func TestBuildGroupedNotification_SingleMatch(t *testing.T) {
	n := &SavedSearchNotifier{}
	lid := uuid.New()
	sid := uuid.New()
	uid := uuid.New()
	matches := []userMatch{{
		SavedSearchID: sid,
		UserID:        uid,
		Query:         "iPhone",
		Listing:       Listing{ID: lid, Title: "iPhone 15 Pro"},
	}}

	title, body, data := n.buildGroupedNotification(matches)
	if title != `1 new listing for "iPhone"` {
		t.Errorf("unexpected title: %s", title)
	}
	if body != "iPhone 15 Pro" {
		t.Errorf("unexpected body: %s", body)
	}
	if data["count"] != "1" {
		t.Errorf("unexpected count: %s", data["count"])
	}
	if data["listing_ids"] != lid.String() {
		t.Errorf("unexpected listing_ids: %s", data["listing_ids"])
	}
}

func TestBuildGroupedNotification_MultipleMatches(t *testing.T) {
	n := &SavedSearchNotifier{}
	uid := uuid.New()
	matches := []userMatch{
		{SavedSearchID: uuid.New(), UserID: uid, Query: "iPhone", Listing: Listing{ID: uuid.New(), Title: "iPhone 15"}},
		{SavedSearchID: uuid.New(), UserID: uid, Query: "iPhone", Listing: Listing{ID: uuid.New(), Title: "iPhone 14"}},
		{SavedSearchID: uuid.New(), UserID: uid, Query: "iPhone", Listing: Listing{ID: uuid.New(), Title: "iPhone 13"}},
		{SavedSearchID: uuid.New(), UserID: uid, Query: "iPhone", Listing: Listing{ID: uuid.New(), Title: "iPhone 12"}},
	}

	title, body, data := n.buildGroupedNotification(matches)
	if title != `4 new listings for "iPhone"` {
		t.Errorf("unexpected title: %s", title)
	}
	if body != "iPhone 15 • iPhone 14 • iPhone 13 + 1 more" {
		t.Errorf("unexpected body: %s", body)
	}
	if data["count"] != "4" {
		t.Errorf("unexpected count: %s", data["count"])
	}
}

func TestBuildGroupedNotification_NoQueryUsesLabel(t *testing.T) {
	n := &SavedSearchNotifier{}
	uid := uuid.New()
	matches := []userMatch{{
		SavedSearchID: uuid.New(),
		UserID:        uid,
		Query:         "",
		Label:         "My Search",
		Listing:       Listing{ID: uuid.New(), Title: "Item"},
	}}

	title, _, _ := n.buildGroupedNotification(matches)
	if title != `1 new listing for "My Search"` {
		t.Errorf("unexpected title: %s", title)
	}
}

func TestBuildGroupedNotification_MultipleSearches(t *testing.T) {
	n := &SavedSearchNotifier{}
	uid := uuid.New()
	matches := []userMatch{
		{SavedSearchID: uuid.New(), UserID: uid, Query: "iPhone", Listing: Listing{ID: uuid.New(), Title: "iPhone"}},
		{SavedSearchID: uuid.New(), UserID: uid, Query: "Samsung", Listing: Listing{ID: uuid.New(), Title: "Samsung"}},
	}

	title, _, _ := n.buildGroupedNotification(matches)
	if title != "2 new listings for 2 saved searches" {
		t.Errorf("unexpected title: %s", title)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Phase 9 — sendWithRetry tests
// ──────────────────────────────────────────────────────────────────────────────

func TestSendWithRetry_Success(t *testing.T) {
	called := 0
	n := &SavedSearchNotifier{
		notify:     func(uuid.UUID, string, string, string, map[string]string) { called++ },
		MaxRetries: 3,
	}
	err := n.sendWithRetry(context.TODO(), uuid.New(), "t", "b", nil)
	if err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}
	if called != 1 {
		t.Errorf("expected 1 call, got %d", called)
	}
}

func TestSendWithRetry_PanicRecovery(t *testing.T) {
	attempt := 0
	n := &SavedSearchNotifier{
		notify: func(uuid.UUID, string, string, string, map[string]string) {
			attempt++
			if attempt < 2 {
				panic("boom")
			}
		},
		MaxRetries: 3,
	}
	err := n.sendWithRetry(context.TODO(), uuid.New(), "t", "b", nil)
	if err != nil {
		t.Errorf("expected nil error after recovery, got: %v", err)
	}
	if attempt != 2 {
		t.Errorf("expected 2 attempts, got %d", attempt)
	}
}

func TestSendWithRetry_NilNotify(t *testing.T) {
	n := &SavedSearchNotifier{notify: nil, MaxRetries: 3}
	err := n.sendWithRetry(context.TODO(), uuid.New(), "t", "b", nil)
	if err != nil {
		t.Errorf("expected nil error with nil notify, got: %v", err)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// pluralS tests
// ──────────────────────────────────────────────────────────────────────────────

func TestPluralS(t *testing.T) {
	if pluralS(1) != "" {
		t.Error("expected empty string for singular")
	}
	if pluralS(2) != "s" {
		t.Error("expected 's' for plural")
	}
	if pluralS(0) != "s" {
		t.Error("expected 's' for zero")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

func mustJSON(v map[string]interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func floatPtr(f float64) *float64 {
	return &f
}
