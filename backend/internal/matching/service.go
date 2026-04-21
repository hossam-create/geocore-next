package matching

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/geoscore"
	"github.com/geocore-next/backend/pkg/cache"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const matchCacheTTL = 1 * time.Minute

// Service fetches order + trip data, enriches with GeoScore, and ranks.
type Service struct {
	db    *gorm.DB
	cache *cache.Cache
	geo   *geoscore.Service
}

// NewService creates a matching service.
func NewService(db *gorm.DB, rdb *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache.New(rdb),
		geo:   geoscore.NewService(db, rdb),
	}
}

// GetMatches handles GET /api/v1/orders/:id/matches.
func (s *Service) GetMatches(c *gin.Context) {
	orderID := c.Param("id")
	if _, err := uuid.Parse(orderID); err != nil {
		response.BadRequest(c, "invalid order id")
		return
	}

	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("matches:order:%s", orderID)

	var results []MatchResult
	if hit, locked := s.cache.GetWithStampedeProtection(ctx, cacheKey, &results); hit {
		response.OK(c, results)
		return
	} else if !locked {
		// Serve degraded from DB
		results, _ = s.computeMatches(ctx, orderID)
		response.OK(c, results)
		return
	}

	var err error
	results, err = s.computeMatches(ctx, orderID)
	if err != nil {
		slog.Warn("matching: compute failed", "order_id", orderID, "error", err)
		response.OK(c, []MatchResult{})
		return
	}

	s.cache.SetWithStampede(ctx, cacheKey, results, matchCacheTTL)
	response.OK(c, results)
}

// rawOrderRow is used to scan the order for matching context.
type rawOrderRow struct {
	ID          string
	Origin      string
	Destination string
	WeightKg    float64
	Budget      float64
	Deadline    *time.Time
	BuyerID     string
}

func (s *Service) computeMatches(ctx context.Context, orderID string) ([]MatchResult, error) {
	// ── 1. Fetch order ────────────────────────────────────────────────────────
	var ord rawOrderRow
	if err := s.db.Raw(`
		SELECT o.id,
		       COALESCE(a->>'pickup_city', '') AS origin,
		       COALESCE(a->>'delivery_city', '') AS destination,
		       COALESCE((o.total / NULLIF(oi_sum.total_weight, 0)), 0) AS weight_kg,
		       o.total AS budget,
		       NULL::timestamptz AS deadline,
		       o.buyer_id
		FROM orders o
		LEFT JOIN LATERAL (
		  SELECT SUM(COALESCE((attributes->>'weight')::float, 1)) AS total_weight
		  FROM order_items WHERE order_id = o.id
		) oi_sum ON true
		LEFT JOIN LATERAL (
		  SELECT (shipping_address)::text AS a
		) addr ON false
		WHERE o.id = ? AND o.deleted_at IS NULL`, orderID).
		Scan(&ord).Error; err != nil || ord.ID == "" {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	// ── 2. Fetch candidate trips (crowdshipping) ──────────────────────────────
	type rawTrip struct {
		ID           string
		TravelerID   string
		TravelerName string
		OriginCity   string
		DestCity     string
		DepartureDate time.Time
		ArrivalDate  time.Time
		AvailableWeight float64
		PricePerKg   float64
		BasePrice    float64
	}

	var rawTrips []rawTrip
	s.db.Raw(`
		SELECT t.id, t.traveler_id, u.name AS traveler_name,
		       t.origin_city, t.dest_city, t.departure_date, t.arrival_date,
		       t.available_weight, t.price_per_kg, t.base_price
		FROM trips t
		JOIN users u ON u.id = t.traveler_id
		WHERE t.status = 'active'
		  AND t.available_weight > 0
		  AND t.departure_date > NOW()
		ORDER BY t.departure_date ASC
		LIMIT 50`).Scan(&rawTrips)

	if len(rawTrips) == 0 {
		return []MatchResult{}, nil
	}

	// ── 3. Enrich candidates with GeoScore ────────────────────────────────────
	candidates := make([]TripCandidate, 0, len(rawTrips))
	for _, rt := range rawTrips {
		geoScore := 50.0
		if uid, err := uuid.Parse(rt.TravelerID); err == nil {
			if gs, err := s.geo.Get(ctx, uid); err == nil {
				geoScore = gs.Score
			}
		}
		candidates = append(candidates, TripCandidate{
			TripID:           rt.ID,
			TravelerID:       rt.TravelerID,
			TravelerName:     rt.TravelerName,
			Origin:           rt.OriginCity,
			Destination:      rt.DestCity,
			DepartureAt:      rt.DepartureDate,
			ArrivalAt:        rt.ArrivalDate,
			AvailableKg:      rt.AvailableWeight,
			PricePerKg:       rt.PricePerKg,
			BasePrice:        rt.BasePrice,
			TravelerGeoScore: geoScore,
		})
	}

	// ── 4. Build order context ────────────────────────────────────────────────
	orderCtx := OrderContext{
		Origin:      ord.Origin,
		Destination: ord.Destination,
		WeightKg:    ord.WeightKg,
		MaxBudget:   ord.Budget,
	}

	// ── 5. Rank ───────────────────────────────────────────────────────────────
	ranked := RankTrips(orderCtx, candidates)
	slog.Info("matching: ranked trips", "order_id", orderID, "candidates", len(candidates))
	return ranked, nil
}
