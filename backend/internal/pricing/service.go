package pricing

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

const priceCacheTTL = 2 * time.Minute

// Service wraps pricing calculation with DB lookups and Redis caching.
type Service struct {
	db    *gorm.DB
	cache *cache.Cache
	geo   *geoscore.Service
}

// NewService creates a pricing service.
func NewService(db *gorm.DB, rdb *redis.Client) *Service {
	return &Service{
		db:    db,
		cache: cache.New(rdb),
		geo:   geoscore.NewService(db, rdb),
	}
}

// CalculateRequest is the JSON body for POST /pricing/calculate.
type CalculateRequest struct {
	Origin      string  `json:"origin" binding:"required,min=2,max=10"`
	Destination string  `json:"destination" binding:"required,min=2,max=10"`
	Weight      float64 `json:"weight_kg" binding:"required,gt=0"`
	UserID      string  `json:"user_id,omitempty"` // optional — for trust discount
}

// Calculate handles POST /api/v1/pricing/calculate.
func (s *Service) Calculate(c *gin.Context) {
	var req CalculateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()
	cacheKey := fmt.Sprintf("pricing:route:%s:%s:%.2f", req.Origin, req.Destination, req.Weight)

	// Try user-scoped cache key if user_id is provided (trust discount varies per user)
	if req.UserID != "" {
		cacheKey = fmt.Sprintf("pricing:route:%s:%s:%.2f:user:%s", req.Origin, req.Destination, req.Weight, req.UserID)
	}

	var result PricingResult
	if hit, locked := s.cache.GetWithStampedeProtection(ctx, cacheKey, &result); hit {
		response.OK(c, result)
		return
	} else if !locked {
		// Serve degraded — compute without cache
		result = s.compute(ctx, req)
		response.OK(c, result)
		return
	}

	result = s.compute(ctx, req)
	s.cache.SetWithStampede(ctx, cacheKey, result, priceCacheTTL)
	response.OK(c, result)
}

func (s *Service) compute(ctx context.Context, req CalculateRequest) PricingResult {
	// Gather route metrics from analytics table
	var routeRow struct {
		AvgPrice    float64
		DemandScore float64
		DisputeRate float64
	}
	s.db.Raw(`
		SELECT avg_price, demand_score, dispute_rate
		FROM route_metrics
		WHERE origin = ? AND destination = ?
		ORDER BY updated_at DESC LIMIT 1`,
		req.Origin, req.Destination).Scan(&routeRow)

	// Gather traveler GeoScore
	geoScore := 50.0 // neutral default
	if req.UserID != "" {
		if uid, err := uuid.Parse(req.UserID); err == nil {
			if gs, err := s.geo.Get(ctx, uid); err == nil {
				geoScore = gs.Score
			}
		}
	}

	in := PricingInput{
		BaseRate:         routeRow.AvgPrice,
		Weight:           req.Weight,
		DemandScore:      routeRow.DemandScore,
		DisputeRate:      routeRow.DisputeRate,
		TravelerGeoScore: geoScore,
	}

	result := Calculate(in)
	slog.Debug("pricing: computed",
		"origin", req.Origin, "dest", req.Destination,
		"weight", req.Weight, "price", result.FinalPrice)
	return result
}
