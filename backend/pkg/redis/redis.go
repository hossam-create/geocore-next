package redis

import (
	"context"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()

func Connect() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
}

func Set(c *redis.Client, key string, val interface{}, ttl time.Duration) error {
	return c.Set(Ctx, key, val, ttl).Err()
}
func Get(c *redis.Client, key string) (string, error) {
	return c.Get(Ctx, key).Result()
}
func Del(c *redis.Client, keys ...string) error {
	return c.Del(Ctx, keys...).Err()
}
func Publish(c *redis.Client, channel string, msg interface{}) error {
	return c.Publish(Ctx, channel, msg).Err()
}
func Subscribe(c *redis.Client, channels ...string) *redis.PubSub {
	return c.Subscribe(Ctx, channels...)
}
