package geoscore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/cache"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	cacheTTL = 5 * time.Minute
	cacheKey = "geoscore:user:%s"
)

// Service handles GeoScore computation and caching.
type Service struct {
	repo  *Repository
	cache *cache.Cache
}

// NewService creates a GeoScore service.
func NewService(db *gorm.DB, rdb *redis.Client) *Service {
	return &Service{
		repo:  NewRepository(db),
		cache: cache.New(rdb),
	}
}

// Get returns the GeoScore for a user, using Redis cache with stampede protection.
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*GeoScore, error) {
	key := fmt.Sprintf(cacheKey, userID.String())

	var gs GeoScore
	if hit, locked := s.cache.GetWithStampedeProtection(ctx, key, &gs); hit {
		return &gs, nil
	} else if !locked {
		// Another goroutine is rebuilding — fall through to DB
		if cached, err := s.repo.Get(userID); err == nil {
			return cached, nil
		}
		return defaultScore(userID), nil
	}

	// Cache miss — compute and populate
	updated, err := s.compute(userID)
	if err != nil {
		slog.Warn("geoscore: compute failed, returning default", "user_id", userID, "error", err)
		return defaultScore(userID), nil
	}
	s.cache.SetWithStampede(ctx, key, updated, cacheTTL)
	return updated, nil
}

// UpdateAsync enqueues an async geoscore.update job. Safe to call from hot paths.
func UpdateAsync(userID string) {
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:     jobs.JobTypeGeoScoreUpdate,
		Priority: 7, // low priority
		Payload:  map[string]interface{}{"user_id": userID},
	})
}

// TrackEventAsync enqueues a behavior tracking job. Non-blocking.
func TrackEventAsync(userID, eventType string, metadata map[string]interface{}) {
	payload := map[string]interface{}{
		"user_id":    userID,
		"event_type": eventType,
	}
	if metadata != nil {
		payload["metadata"] = metadata
	}
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:     jobs.JobTypeBehaviorTrack,
		Priority: 10, // lowest priority
		Payload:  payload,
	})
}

// compute gathers signals and returns an updated GeoScore (does NOT cache).
func (s *Service) compute(userID uuid.UUID) (*GeoScore, error) {
	in := s.repo.GatherSignals(userID)
	score := Calculate(in)
	gs := &GeoScore{
		UserID:        userID,
		Score:         score,
		SuccessRate:   in.SuccessRate,
		DisputeRate:   in.DisputeRate,
		KYCScore:      in.KYCScore,
		DeliveryScore: in.DeliveryScore,
		FraudScore:    in.FraudScore,
		UpdatedAt:     time.Now(),
	}
	if err := s.repo.Save(gs); err != nil {
		return gs, fmt.Errorf("geoscore save: %w", err)
	}
	slog.Info("geoscore: updated", "user_id", userID, "score", score)
	return gs, nil
}

// Invalidate removes the Redis cache entry, forcing recompute on next Get.
func (s *Service) Invalidate(ctx context.Context, userID uuid.UUID) {
	key := fmt.Sprintf(cacheKey, userID.String())
	s.cache.Del(ctx, key, key+":stale")
}

func defaultScore(userID uuid.UUID) *GeoScore {
	return &GeoScore{
		UserID:    userID,
		Score:     50.0, // neutral default for new users
		UpdatedAt: time.Now(),
	}
}
