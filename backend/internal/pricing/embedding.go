package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Real-time Embedding Service ────────────────────────────────────────────────────
//
// Every entity (user, item, session) gets a vector embedding.
// Embeddings enable similarity search and are updated in real-time on events.
//
// Pipeline: User action → event → embedding delta → Redis cache
//
// Cold start: default embeddings (zeros or popularity-based)
// Drift: daily retrain from offline job
// Spam: trust-weighted (low-trust actions have less impact)

const EmbeddingDim = 32

// ── Embedding Models ────────────────────────────────────────────────────────────────

type EmbeddingVector struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EntityType string   `gorm:"size:20;not null;index" json:"entity_type"` // user, item, session
	EntityID   uuid.UUID `gorm:"type:uuid;not null;index" json:"entity_id"`
	Vector     string    `gorm:"type:text;not null" json:"vector"` // JSON array of float32
	Version    int       `gorm:"not null;default:1" json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (EmbeddingVector) TableName() string { return "embedding_vectors" }

// EmbeddingEvent records an event that triggered an embedding update.
type EmbeddingEvent struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EntityType  string    `gorm:"size:20;not null;index" json:"entity_type"`
	EntityID    uuid.UUID `gorm:"type:uuid;not null;index" json:"entity_id"`
	EventType   string    `gorm:"size:30;not null" json:"event_type"` // view, click, purchase, cancel, claim
	DeltaJSON   string    `gorm:"type:text" json:"delta_json"` // JSON array of delta values
	TrustWeight float64   `gorm:"type:numeric(5,4);not null;default:1.0" json:"trust_weight"` // low-trust = less impact
	CreatedAt   time.Time `json:"created_at"`
}

func (EmbeddingEvent) TableName() string { return "embedding_events" }

// ── Embedding Service Interface ────────────────────────────────────────────────────

type EmbeddingService interface {
	GetUserEmbedding(userID uuid.UUID) ([]float32, error)
	GetItemEmbedding(itemID uuid.UUID) ([]float32, error)
	UpdateUserEmbedding(userID uuid.UUID, eventType string, delta []float32, trustWeight float64) error
	UpdateItemEmbedding(itemID uuid.UUID, eventType string, delta []float32) error
	CosineSimilarity(a, b []float32) float32
	GetSimilarItems(itemID uuid.UUID, topK int) ([]uuid.UUID, error)
}

// ── Redis-backed Embedding Service ─────────────────────────────────────────────────

type RedisEmbeddingService struct {
	db  *gorm.DB
	rdb *redis.Client
	ttl time.Duration
}

func NewEmbeddingService(db *gorm.DB, rdb *redis.Client) *RedisEmbeddingService {
	return &RedisEmbeddingService{
		db:  db,
		rdb: rdb,
		ttl: 30 * time.Minute,
	}
}

// GetUserEmbedding fetches a user's embedding vector.
func (s *RedisEmbeddingService) GetUserEmbedding(userID uuid.UUID) ([]float32, error) {
	return s.getEmbedding("user", userID)
}

// GetItemEmbedding fetches an item's embedding vector.
func (s *RedisEmbeddingService) GetItemEmbedding(itemID uuid.UUID) ([]float32, error) {
	return s.getEmbedding("item", itemID)
}

// getEmbedding fetches an embedding: Redis → Postgres → default.
func (s *RedisEmbeddingService) getEmbedding(entityType string, entityID uuid.UUID) ([]float32, error) {
	ctx := context.Background()
	key := fmt.Sprintf("emb:%s:%s", entityType, entityID)

	// Try Redis
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var vec []float32
		if json.Unmarshal([]byte(val), &vec) == nil {
			return vec, nil
		}
	}

	// Fallback to Postgres
	var emb EmbeddingVector
	if err := s.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&emb).Error; err == nil {
		var vec []float32
		if json.Unmarshal([]byte(emb.Vector), &vec) == nil {
			// Cache in Redis
			s.rdb.Set(ctx, key, emb.Vector, s.ttl)
			return vec, nil
		}
	}

	// Cold start: return default embedding (zeros)
	return defaultEmbedding(), nil
}

// defaultEmbedding returns a zero vector for cold-start entities.
func defaultEmbedding() []float32 {
	vec := make([]float32, EmbeddingDim)
	// Small random initialization to avoid all-zeros
	for i := range vec {
		vec[i] = float32(0.01 * (float64(i%7) - 3.0) / 7.0)
	}
	return vec
}

// UpdateUserEmbedding applies a delta to a user's embedding in real-time.
// trustWeight scales the delta: low-trust users have less impact (anti-spam).
func (s *RedisEmbeddingService) UpdateUserEmbedding(userID uuid.UUID, eventType string, delta []float32, trustWeight float64) error {
	return s.updateEmbedding("user", userID, eventType, delta, trustWeight)
}

// UpdateItemEmbedding applies a delta to an item's embedding.
func (s *RedisEmbeddingService) UpdateItemEmbedding(itemID uuid.UUID, eventType string, delta []float32) error {
	return s.updateEmbedding("item", itemID, eventType, delta, 1.0)
}

// updateEmbedding applies a trust-weighted delta to an entity's embedding.
func (s *RedisEmbeddingService) updateEmbedding(entityType string, entityID uuid.UUID, eventType string, delta []float32, trustWeight float64) error {
	// Load current embedding
	current, _ := s.getEmbedding(entityType, entityID)

	// Apply delta with trust weighting
	updated := applyDelta(current, delta, trustWeight)

	// Normalize to unit vector (prevents drift)
	updated = normalizeVector(updated)

	// Save to Postgres
	vecJSON, _ := json.Marshal(updated)
	var emb EmbeddingVector
	if err := s.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		First(&emb).Error; err != nil {
		// Create new
		emb = EmbeddingVector{
			EntityType: entityType,
			EntityID:   entityID,
			Vector:     string(vecJSON),
			Version:    1,
		}
		s.db.Create(&emb)
	} else {
		s.db.Model(&emb).Updates(map[string]interface{}{
			"vector":     string(vecJSON),
			"version":    emb.Version + 1,
			"updated_at": time.Now(),
		})
	}

	// Record the event
	deltaJSON, _ := json.Marshal(delta)
	s.db.Create(&EmbeddingEvent{
		EntityType:  entityType,
		EntityID:    entityID,
		EventType:   eventType,
		DeltaJSON:   string(deltaJSON),
		TrustWeight: trustWeight,
	})

	// Update Redis cache
	ctx := context.Background()
	key := fmt.Sprintf("emb:%s:%s", entityType, entityID)
	s.rdb.Set(ctx, key, string(vecJSON), s.ttl)

	return nil
}

// applyDelta applies a weighted delta to a vector.
func applyDelta(vec, delta []float32, weight float64) []float32 {
	result := make([]float32, len(vec))
	for i := range vec {
		if i < len(delta) {
			result[i] = vec[i] + float32(weight)*delta[i]
		} else {
			result[i] = vec[i]
		}
	}
	return result
}

// normalizeVector normalizes a vector to unit length.
func normalizeVector(vec []float32) []float32 {
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm < 1e-8 {
		return vec
	}
	result := make([]float32, len(vec))
	for i, v := range vec {
		result[i] = v / norm
	}
	return result
}

// CosineSimilarity computes cosine similarity between two vectors.
func (s *RedisEmbeddingService) CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA < 1e-8 || normB < 1e-8 {
		return 0
	}
	return dot / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}

// GetSimilarItems finds items similar to a given item using pgvector.
func (s *RedisEmbeddingService) GetSimilarItems(itemID uuid.UUID, topK int) ([]uuid.UUID, error) {
	// Get the item's embedding
	itemEmb, err := s.GetItemEmbedding(itemID)
	if err != nil {
		return nil, err
	}
	embJSON, _ := json.Marshal(itemEmb)

	// Query using pgvector cosine distance
	var results []struct {
		EntityID uuid.UUID `json:"entity_id"`
		Distance float64   `json:"distance"`
	}

	// Use raw SQL with pgvector if available, otherwise fall back to in-memory
	err = s.db.Raw(`
		SELECT entity_id, vector <=> $1::vector AS distance
		FROM embedding_vectors
		WHERE entity_type = 'item' AND entity_id != $2
		ORDER BY distance ASC
		LIMIT $3
	`, string(embJSON), itemID, topK).Scan(&results).Error

	if err != nil || len(results) == 0 {
		// Fallback: popularity-based
		return s.popularityFallback(topK), nil
	}

	ids := make([]uuid.UUID, len(results))
	for i, r := range results {
		ids[i] = r.EntityID
	}
	return ids, nil
}

// popularityFallback returns popular item IDs when vector search fails.
func (s *RedisEmbeddingService) popularityFallback(limit int) []uuid.UUID {
	var features []ItemFeatures
	s.db.Order("popularity_score DESC").Limit(limit).Find(&features)

	ids := make([]uuid.UUID, len(features))
	for i, f := range features {
		ids[i] = f.ItemID
	}
	return ids
}

// ── Predefined Event Deltas ────────────────────────────────────────────────────────
// These are simplified delta vectors for common events.
// In production, these would come from a trained model.

// PurchaseDelta returns the embedding delta for a purchase event.
func PurchaseDelta() []float32 {
	delta := make([]float32, EmbeddingDim)
	delta[0] = 0.1  // positive signal
	delta[1] = 0.05
	delta[2] = 0.03
	return delta
}

// ViewDelta returns the embedding delta for a view event.
func ViewDelta() []float32 {
	delta := make([]float32, EmbeddingDim)
	delta[0] = 0.02  // mild positive
	delta[1] = 0.01
	return delta
}

// CancelDelta returns the embedding delta for a cancel event.
func CancelDelta() []float32 {
	delta := make([]float32, EmbeddingDim)
	delta[0] = -0.05 // negative signal
	delta[1] = -0.03
	delta[3] = 0.02  // slight risk increase
	return delta
}

// ClaimDelta returns the embedding delta for a claim event.
func ClaimDelta() []float32 {
	delta := make([]float32, EmbeddingDim)
	delta[0] = -0.08 // strong negative
	delta[3] = 0.05   // risk increase
	delta[4] = 0.03
	return delta
}

// ClickDelta returns the embedding delta for a click event.
func ClickDelta() []float32 {
	delta := make([]float32, EmbeddingDim)
	delta[0] = 0.03
	delta[1] = 0.02
	return delta
}
