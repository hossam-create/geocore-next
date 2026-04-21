package ops

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const configCacheTTL = 5 * time.Minute
const configCachePrefix = "ops:cfg:"

var (
	globalDB  *gorm.DB
	globalRDB *redis.Client
	storeMu   sync.RWMutex
)

// InitConfigStore wires the DB and Redis used by ConfigGet.
func InitConfigStore(db *gorm.DB, rdb *redis.Client) {
	storeMu.Lock()
	defer storeMu.Unlock()
	globalDB = db
	globalRDB = rdb
}

// ConfigGet retrieves a runtime config value.
// Priority: DB (ops_configs table) > environment variable.
// Results are cached in Redis for configCacheTTL.
func ConfigGet(key string) string {
	storeMu.RLock()
	db := globalDB
	rdb := globalRDB
	storeMu.RUnlock()

	if rdb != nil {
		if v, err := rdb.Get(context.Background(), configCachePrefix+key).Result(); err == nil && v != "" {
			return v
		}
	}

	if db != nil {
		var cfg OpsConfig
		if err := db.Where("key = ?", key).First(&cfg).Error; err == nil && cfg.Value != "" {
			if rdb != nil {
				rdb.Set(context.Background(), configCachePrefix+key, cfg.Value, configCacheTTL)
			}
			return cfg.Value
		}
	}

	return os.Getenv(key)
}

// ConfigSet stores or updates a runtime config value, invalidates cache.
func ConfigSet(db *gorm.DB, rdb *redis.Client, key, value, updatedBy string, isSecret bool) error {
	cfg := OpsConfig{Key: key, Value: value, IsSecret: isSecret, UpdatedBy: updatedBy, UpdatedAt: time.Now()}
	result := db.Where("key = ?", key).Assign(cfg).FirstOrCreate(&cfg)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		if err := db.Model(&cfg).Where("key = ?", key).Updates(map[string]interface{}{
			"value":      value,
			"is_secret":  isSecret,
			"updated_by": updatedBy,
			"updated_at": time.Now(),
		}).Error; err != nil {
			return err
		}
	}
	if rdb != nil {
		rdb.Del(context.Background(), configCachePrefix+key)
	}
	return nil
}

// ConfigGetAll returns all config entries (values of secrets masked).
func ConfigGetAll(db *gorm.DB) ([]map[string]interface{}, error) {
	var cfgs []OpsConfig
	if err := db.Order("key").Find(&cfgs).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(cfgs))
	for _, c := range cfgs {
		val := c.Value
		if c.IsSecret && val != "" {
			val = fmt.Sprintf("***%s", val[max(0, len(val)-4):])
		}
		out = append(out, map[string]interface{}{
			"id":         c.ID,
			"key":        c.Key,
			"value":      val,
			"is_secret":  c.IsSecret,
			"updated_at": c.UpdatedAt,
			"updated_by": c.UpdatedBy,
		})
	}
	return out, nil
}
