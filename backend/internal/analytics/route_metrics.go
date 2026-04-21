package analytics

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RouteMetrics aggregates order and shipping data per origin→destination pair.
// Table: route_metrics
type RouteMetrics struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Origin      string    `gorm:"size:10;not null;uniqueIndex:idx_route_metrics_od" json:"origin"`
	Destination string    `gorm:"size:10;not null;uniqueIndex:idx_route_metrics_od" json:"destination"`
	TotalOrders int64     `gorm:"default:0" json:"total_orders"`
	TotalWeight float64   `gorm:"type:decimal(15,2);default:0" json:"total_weight_kg"`
	AvgPrice    float64   `gorm:"type:decimal(10,2);default:0" json:"avg_price"`
	SuccessRate float64   `gorm:"type:decimal(5,4);default:0" json:"success_rate"`
	DisputeRate float64   `gorm:"type:decimal(5,4);default:0" json:"dispute_rate"`
	DemandScore float64   `gorm:"type:decimal(5,4);default:0" json:"demand_score"` // normalised 0-1
	UpdatedAt   time.Time `json:"updated_at"`
}

func (RouteMetrics) TableName() string { return "route_metrics" }

// RouteMetricsHandler adds route analytics endpoints to the existing analytics handler.
type RouteMetricsHandler struct {
	db *gorm.DB
}

// NewRouteMetricsHandler creates a handler.
func NewRouteMetricsHandler(db *gorm.DB) *RouteMetricsHandler {
	return &RouteMetricsHandler{db: db}
}

// RegisterRouteMetricsRoutes adds route intelligence endpoints under /analytics.
func RegisterRouteMetricsRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewRouteMetricsHandler(db)
	routes := r.Group("/analytics/routes")
	{
		routes.GET("/top", h.TopRoutes)
		routes.GET("/risky", h.RiskyRoutes)
		routes.GET("/profitable", h.ProfitableRoutes)
	}
}

// TopRoutes returns the most active routes by total_orders.
// GET /api/v1/analytics/routes/top
func (h *RouteMetricsHandler) TopRoutes(c *gin.Context) {
	var metrics []RouteMetrics
	h.db.Order("total_orders DESC").Limit(20).Find(&metrics)
	response.OK(c, metrics)
}

// RiskyRoutes returns routes with the highest dispute rates.
// GET /api/v1/analytics/routes/risky
func (h *RouteMetricsHandler) RiskyRoutes(c *gin.Context) {
	var metrics []RouteMetrics
	h.db.Where("total_orders > 5").Order("dispute_rate DESC").Limit(20).Find(&metrics)
	response.OK(c, metrics)
}

// ProfitableRoutes returns routes with the highest average price.
// GET /api/v1/analytics/routes/profitable
func (h *RouteMetricsHandler) ProfitableRoutes(c *gin.Context) {
	var metrics []RouteMetrics
	h.db.Where("total_orders > 3").Order("avg_price DESC").Limit(20).Find(&metrics)
	response.OK(c, metrics)
}

// UpdateRouteMetrics aggregates orders for a given origin/destination and upserts.
// Called by the route.update background job.
func UpdateRouteMetrics(db *gorm.DB, origin, destination string) error {
	var agg struct {
		TotalOrders int64
		TotalWeight float64
		AvgPrice    float64
		Completed   int64
		Disputed    int64
	}

	err := db.Raw(`
		SELECT
			COUNT(*) AS total_orders,
			SUM(COALESCE((SELECT SUM(COALESCE((attributes->>'weight')::float, 1))
			              FROM order_items WHERE order_id = o.id), 1)) AS total_weight,
			AVG(o.total / NULLIF(
			    (SELECT SUM(COALESCE((attributes->>'weight')::float, 1))
			     FROM order_items WHERE order_id = o.id), 0)) AS avg_price,
			COUNT(*) FILTER (WHERE o.status IN ('delivered','completed')) AS completed,
			COUNT(*) FILTER (WHERE o.status = 'disputed') AS disputed
		FROM orders o
		WHERE deleted_at IS NULL
		  AND (shipping_address->>'country' = ? OR shipping_address->>'city' ILIKE ?)
		`,
		origin, "%"+origin+"%").Scan(&agg).Error
	if err != nil {
		return err
	}

	successRate := 0.0
	if agg.TotalOrders > 0 {
		successRate = float64(agg.Completed) / float64(agg.TotalOrders)
	}
	disputeRate := 0.0
	if agg.TotalOrders > 0 {
		disputeRate = float64(agg.Disputed) / float64(agg.TotalOrders)
	}

	rm := RouteMetrics{
		Origin:      origin,
		Destination: destination,
		TotalOrders: agg.TotalOrders,
		TotalWeight: agg.TotalWeight,
		AvgPrice:    agg.AvgPrice,
		SuccessRate: successRate,
		DisputeRate: disputeRate,
		DemandScore: normaliseDemand(agg.TotalOrders),
		UpdatedAt:   time.Now(),
	}

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "origin"}, {Name: "destination"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"total_orders", "total_weight", "avg_price",
			"success_rate", "dispute_rate", "demand_score", "updated_at",
		}),
	}).Create(&rm).Error
}

// normaliseDemand maps raw order count to a 0-1 demand score using log10 scale.
// log10(1001) ≈ 3, so orders of 1–1000 map to roughly 0–1.
func normaliseDemand(orders int64) float64 {
	if orders <= 0 {
		return 0
	}
	v := math.Log10(float64(orders)+1) / math.Log10(1001)
	if v > 1 {
		return 1
	}
	return v
}

// EnqueueRouteUpdate enqueues an async route.update job.
func EnqueueRouteUpdate(origin, destination string) {
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:     jobs.JobTypeRouteUpdate,
		Priority: 8,
		Payload: map[string]interface{}{
			"origin":      origin,
			"destination": destination,
		},
	})
}

// RouteUpdateJobHandler is the background worker for route.update jobs.
func RouteUpdateJobHandler(db *gorm.DB) jobs.JobHandler {
	return func(ctx context.Context, job *jobs.Job) error {
		origin, _ := job.Payload["origin"].(string)
		dest, _ := job.Payload["destination"].(string)
		if origin == "" || dest == "" {
			return nil // skip malformed payloads
		}
		if err := UpdateRouteMetrics(db, origin, dest); err != nil {
			slog.Warn("route.update: aggregation failed",
				"origin", origin, "destination", dest, "error", err)
			return err
		}
		slog.Info("route.update: completed", "origin", origin, "destination", dest)
		return nil
	}
}
