package fraud

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PredictHandler serves Sprint 24 admin endpoints for the Predictor.
type PredictHandler struct {
	db *gorm.DB
	p  *Predictor
}

func NewPredictHandler(db *gorm.DB, rdb *redis.Client) *PredictHandler {
	return &PredictHandler{db: db, p: NewPredictor(db, rdb)}
}

// RiskUsersHandler — GET /admin/system/risk/users?min_score=60&limit=50
// Returns the top-N users by their latest snapshot score, joined with their
// current Sprint 23 security profile for cross-reference.
func (h *PredictHandler) RiskUsersHandler(c *gin.Context) {
	minScore, _ := strconv.Atoi(c.DefaultQuery("min_score", "40"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	type row struct {
		UserID        uuid.UUID `json:"user_id"`
		Score         int       `json:"score"`
		Decision      string    `json:"decision"`
		SnapshotAt    time.Time `json:"snapshot_at"`
		SecurityScore int       `json:"security_score"`
		Frozen        bool      `json:"frozen"`
	}
	var rows []row

	// Latest snapshot per user (postgres DISTINCT ON), joined with security profile.
	h.db.Raw(`
		SELECT DISTINCT ON (s.user_id)
			s.user_id,
			s.score,
			s.decision,
			s.created_at               AS snapshot_at,
			COALESCE(p.risk_score, 0)  AS security_score,
			COALESCE(p.frozen, false)  AS frozen
		FROM user_risk_snapshots s
		LEFT JOIN user_security_profiles p ON p.user_id = s.user_id
		WHERE s.score >= ?
		ORDER BY s.user_id, s.created_at DESC
		LIMIT ?
	`, minScore, limit).Scan(&rows)

	c.JSON(http.StatusOK, gin.H{
		"users":       rows,
		"count":       len(rows),
		"min_score":   minScore,
		"captured_at": time.Now().UTC(),
	})
}

// PredictUserHandler — GET /admin/system/risk/users/:userId
// On-demand re-evaluation of a specific user. Useful during investigation.
func (h *PredictHandler) PredictUserHandler(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	res := h.p.PredictRisk(c.Request.Context(), uid)

	// Recent snapshot history for trend visualisation.
	var history []UserRiskSnapshot
	h.db.Where("user_id = ?", uid).
		Order("created_at DESC").
		Limit(30).
		Find(&history)

	c.JSON(http.StatusOK, gin.H{
		"current": res,
		"history": history,
	})
}
