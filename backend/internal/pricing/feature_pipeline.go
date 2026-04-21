package pricing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Feature Pipeline ────────────────────────────────────────────────────────────────
//
// Integration layer: wires Feature Store + Embeddings + Retrieval into Cross-System RL.
//
// Full Flow:
//
//	User opens app →
//	  Feature Store → user/session features
//	  Embedding Service → user vector
//	  Retrieval → top 200 items
//	  Ranking + RL → top 20
//	  Pricing Engine → price
//	  Response (< 100ms target)

// ── Pipeline Models ──────────────────────────────────────────────────────────────────

// PipelineRequest is the input for the full feature pipeline.
type PipelineRequest struct {
	UserID    uuid.UUID `json:"user_id"`
	OrderID   uuid.UUID `json:"order_id"`
	ItemID    uuid.UUID `json:"item_id"`     // optional: seed item for similar items
	SessionID string    `json:"session_id"`
	Geo       string    `json:"geo"`
}

// PipelineResponse is the enriched output ready for RL/ranking.
type PipelineResponse struct {
	// ── Features ──────────────────────────────────────────────────────────────
	UserFeatures    *UserFeatures    `json:"user_features"`
	ItemFeatures    *ItemFeatures    `json:"item_features"`
	SessionFeatures *SessionFeatures `json:"session_features"`

	// ── Embeddings ────────────────────────────────────────────────────────────
	UserEmbedding   []float32        `json:"user_embedding"`
	ItemEmbedding   []float32        `json:"item_embedding"`

	// ── Retrieval ──────────────────────────────────────────────────────────────
	Candidates      []RetrievalCandidate `json:"candidates"`

	// ── Cross State (pre-built for RL) ────────────────────────────────────────
	CrossState      *CrossState      `json:"cross_state"`

	// ── Performance ────────────────────────────────────────────────────────────
	FeatureLatencyMs  int64 `json:"feature_latency_ms"`
	EmbeddingLatencyMs int64 `json:"embedding_latency_ms"`
	RetrievalLatencyMs int64 `json:"retrieval_latency_ms"`
	TotalLatencyMs     int64 `json:"total_latency_ms"`
}

// PipelineLatencyLog records pipeline performance for monitoring.
type PipelineLatencyLog struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	FeatureLatencyMs   int64     `gorm:"not null" json:"feature_latency_ms"`
	EmbeddingLatencyMs int64     `gorm:"not null" json:"embedding_latency_ms"`
	RetrievalLatencyMs int64     `gorm:"not null" json:"retrieval_latency_ms"`
	TotalLatencyMs     int64     `gorm:"not null" json:"total_latency_ms"`
	CandidateCount     int       `gorm:"not null" json:"candidate_count"`
	CreatedAt          time.Time `json:"created_at"`
}

func (PipelineLatencyLog) TableName() string { return "pipeline_latency_logs" }

// ── Pipeline Service ────────────────────────────────────────────────────────────────

type PipelineService struct {
	db           *gorm.DB
	featureStore *RedisFeatureStore
	embeddingSvc *RedisEmbeddingService
	retrievalSvc *RetrievalService
}

func NewPipelineService(db *gorm.DB, rdb *redis.Client) *PipelineService {
	fs := NewFeatureStore(db, rdb)
	emb := NewEmbeddingService(db, rdb)
	ret := NewRetrievalService(db, emb, fs)

	return &PipelineService{
		db:           db,
		featureStore: fs,
		embeddingSvc: emb,
		retrievalSvc: ret,
	}
}

// Enrich runs the full feature pipeline: features → embeddings → retrieval → cross state.
func (s *PipelineService) Enrich(req PipelineRequest) (*PipelineResponse, error) {
	totalStart := time.Now()
	resp := &PipelineResponse{}

	// ── Step 1: Feature Store (5-15ms target) ──────────────────────────────────
	featStart := time.Now()
	userFeat, err := s.featureStore.GetUserFeatures(req.UserID)
	if err != nil {
		// Create default features on cold start
		userFeat = &UserFeatures{
			UserID:   req.UserID,
			TrustScore: 50,
			Segment:  "new",
		}
	}
	resp.UserFeatures = userFeat

	if req.ItemID != uuid.Nil {
		itemFeat, err := s.featureStore.GetItemFeatures(req.ItemID)
		if err != nil {
			itemFeat = &ItemFeatures{ItemID: req.ItemID}
		}
		resp.ItemFeatures = itemFeat
	}

	if req.SessionID != "" {
		sessFeat, _ := s.featureStore.GetSessionFeatures(req.SessionID)
		resp.SessionFeatures = sessFeat
	}
	resp.FeatureLatencyMs = time.Since(featStart).Milliseconds()

	// ── Step 2: Embeddings (1-5ms target) ──────────────────────────────────────
	embStart := time.Now()
	userEmb, _ := s.embeddingSvc.GetUserEmbedding(req.UserID)
	resp.UserEmbedding = userEmb

	if req.ItemID != uuid.Nil {
		itemEmb, _ := s.embeddingSvc.GetItemEmbedding(req.ItemID)
		resp.ItemEmbedding = itemEmb
	}
	resp.EmbeddingLatencyMs = time.Since(embStart).Milliseconds()

	// ── Step 3: Retrieval (10-30ms target) ──────────────────────────────────────
	retStart := time.Now()
	retReq := RetrievalRequest{
		UserID:       req.UserID,
		ItemID:       req.ItemID,
		TopK:         200,
		Geo:          req.Geo,
	}
	if resp.ItemFeatures != nil {
		retReq.CategoryPath = resp.ItemFeatures.CategoryPath
		retReq.PriceMax = resp.ItemFeatures.PriceCents * 3
		retReq.PriceMin = resp.ItemFeatures.PriceCents / 3
	}
	retResp, _ := s.retrievalSvc.Retrieve(retReq)
	if retResp != nil {
		resp.Candidates = retResp.Candidates
		resp.RetrievalLatencyMs = retResp.LatencyMs
	}
	resp.RetrievalLatencyMs = time.Since(retStart).Milliseconds()

	// ── Step 4: Build CrossState for RL ────────────────────────────────────────
	resp.CrossState = buildCrossStateFromPipeline(resp, req)

	// ── Total latency ──────────────────────────────────────────────────────────
	resp.TotalLatencyMs = time.Since(totalStart).Milliseconds()

	// ── Log latency ────────────────────────────────────────────────────────────
	go s.logLatency(req.UserID, resp)

	return resp, nil
}

// buildCrossStateFromPipeline constructs a CrossState from pipeline data.
func buildCrossStateFromPipeline(resp *PipelineResponse, req PipelineRequest) *CrossState {
	state := &CrossState{}

	if resp.UserFeatures != nil {
		state.UserTrust = resp.UserFeatures.TrustScore
		state.UserSegment = resp.UserFeatures.Segment
		state.CancelRate = resp.UserFeatures.CancelRate
		state.BuyRate = resp.UserFeatures.InsuranceBuyRate
		state.AccountAgeDays = resp.UserFeatures.AccountAgeDays
		state.RiskScore = 1.0 - resp.UserFeatures.TrustScore/100.0
	}

	if resp.SessionFeatures != nil {
		state.SessionStep = resp.SessionFeatures.SessionStep
		state.Device = resp.SessionFeatures.Device
		state.Geo = resp.SessionFeatures.Geo
		state.RefusalCount = resp.SessionFeatures.RefusalCount
		state.DemandScore = resp.SessionFeatures.DemandScore
		state.UrgencyScore = resp.SessionFeatures.UrgencyScore
	}

	if resp.ItemFeatures != nil {
		state.ItemPriceCents = resp.ItemFeatures.PriceCents
		state.CategoryPath = resp.ItemFeatures.CategoryPath
		state.DeliveryRisk = resp.ItemFeatures.DeliveryRisk
	}

	return state
}

// logLatency records pipeline performance for monitoring.
func (s *PipelineService) logLatency(userID uuid.UUID, resp *PipelineResponse) {
	log := PipelineLatencyLog{
		UserID:             userID,
		FeatureLatencyMs:   resp.FeatureLatencyMs,
		EmbeddingLatencyMs: resp.EmbeddingLatencyMs,
		RetrievalLatencyMs: resp.RetrievalLatencyMs,
		TotalLatencyMs:     resp.TotalLatencyMs,
		CandidateCount:     len(resp.Candidates),
	}
	s.db.Create(&log)
}

// ── Event Processing (real-time embedding updates) ──────────────────────────────────

// ProcessEvent updates embeddings based on user actions.
func (s *PipelineService) ProcessEvent(userID, itemID uuid.UUID, eventType string, trustWeight float64) error {
	// Get the delta for this event type
	var delta []float32
	switch eventType {
	case "purchase":
		delta = PurchaseDelta()
	case "view":
		delta = ViewDelta()
	case "click":
		delta = ClickDelta()
	case "cancel":
		delta = CancelDelta()
	case "claim":
		delta = ClaimDelta()
	default:
		delta = ViewDelta() // default mild positive
	}

	// Update user embedding with trust weighting
	if err := s.embeddingSvc.UpdateUserEmbedding(userID, eventType, delta, trustWeight); err != nil {
		return err
	}

	// Update item embedding (no trust weighting — items don't have trust)
	if itemID != uuid.Nil {
		if err := s.embeddingSvc.UpdateItemEmbedding(itemID, eventType, delta); err != nil {
			return err
		}
	}

	// Refresh features in the background
	go s.featureStore.RefreshUserFeatures(userID)
	if itemID != uuid.Nil {
		go s.featureStore.RefreshItemFeatures(itemID)
	}

	return nil
}

// ── Pipeline Dashboard ──────────────────────────────────────────────────────────────

type PipelineDashboard struct {
	TotalRequests      int64              `json:"total_requests"`
	AvgTotalLatencyMs  float64            `json:"avg_total_latency_ms"`
	AvgFeatureMs       float64            `json:"avg_feature_ms"`
	AvgEmbeddingMs     float64            `json:"avg_embedding_ms"`
	AvgRetrievalMs     float64            `json:"avg_retrieval_ms"`
	P95LatencyMs       float64            `json:"p95_latency_ms"`
	TotalEmbeddings    int64              `json:"total_embeddings"`
	TotalEvents        int64              `json:"total_events"`
	RetrievalMetrics   *RetrievalMetrics  `json:"retrieval_metrics"`
}

func GetPipelineDashboard(db *gorm.DB) *PipelineDashboard {
	var totalLogs int64
	db.Model(&PipelineLatencyLog{}).Count(&totalLogs)

	var avgLat struct {
		AvgTotal     float64 `json:"avg_total"`
		AvgFeature   float64 `json:"avg_feature"`
		AvgEmbedding float64 `json:"avg_embedding"`
		AvgRetrieval float64 `json:"avg_retrieval"`
	}
	db.Model(&PipelineLatencyLog{}).Select(
		"COALESCE(AVG(total_latency_ms), 0) as avg_total, "+
			"COALESCE(AVG(feature_latency_ms), 0) as avg_feature, "+
			"COALESCE(AVG(embedding_latency_ms), 0) as avg_embedding, "+
			"COALESCE(AVG(retrieval_latency_ms), 0) as avg_retrieval").Scan(&avgLat)

	var totalEmbs int64
	db.Model(&EmbeddingVector{}).Count(&totalEmbs)

	var totalEvents int64
	db.Model(&EmbeddingEvent{}).Count(&totalEvents)

	// P95 approximation
	var p95 struct{ Latency float64 }
	db.Raw("SELECT COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY total_latency_ms), 0) as latency FROM pipeline_latency_logs").Scan(&p95)

	return &PipelineDashboard{
		TotalRequests:     totalLogs,
		AvgTotalLatencyMs: avgLat.AvgTotal,
		AvgFeatureMs:      avgLat.AvgFeature,
		AvgEmbeddingMs:    avgLat.AvgEmbedding,
		AvgRetrievalMs:    avgLat.AvgRetrieval,
		P95LatencyMs:      p95.Latency,
		TotalEmbeddings:   totalEmbs,
		TotalEvents:       totalEvents,
		RetrievalMetrics:  GetRetrievalMetrics(db),
	}
}

// Ensure imports used
var _ = json.Marshal
var _ = fmt.Sprintf
