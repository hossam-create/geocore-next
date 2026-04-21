package waitlist

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"gorm.io/gorm"
)

// StartRecalcJob runs RecalculatePositions on the given interval.
// Returns a stop function that gracefully shuts the ticker down.
func StartRecalcJob(db *gorm.DB, interval time.Duration) func() {
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if config.GetFlags().EnablePriorityQueue {
					RecalculatePositions(db)
					slog.Info("waitlist: periodic recalc complete")
				}
			case <-stop:
				return
			}
		}
	}()
	return func() { close(stop) }
}
