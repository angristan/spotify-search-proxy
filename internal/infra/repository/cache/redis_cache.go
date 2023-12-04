package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	redisClient *redis.Client
	defaultTTL  time.Duration
}

func NewCache(
	redisClient *redis.Client,
	defaultTTL time.Duration,
) *Cache {
	return &Cache{
		redisClient: redisClient,
		defaultTTL:  defaultTTL,
	}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	value, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}

		return "", err
	}

	return value, nil
}

func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	return c.redisClient.Set(ctx, key, value, ttl).Err()
}
