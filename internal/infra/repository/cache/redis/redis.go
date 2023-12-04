package redis

import (
	"context"
	"errors"
	"time"

	"github.com/angristan/spotify-search-proxy/internal/infra/repository/cache"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrCacheMiss = errors.New("cache: key not found")
)

type RedisCache struct {
	tracer      trace.Tracer
	redisClient *redis.Client
	defaultTTL  time.Duration
}

func NewCache(
	tracer trace.Tracer,
	redisClient *redis.Client,
	defaultTTL time.Duration,
) cache.Cache {
	return &RedisCache{
		tracer:      tracer,
		redisClient: redisClient,
		defaultTTL:  defaultTTL,
	}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	ctx, span := c.tracer.Start(ctx, "RedisCache.Get")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	value, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrCacheMiss
		}

		return "", err
	}

	return value, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "RedisCache.Set")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	return c.redisClient.Set(ctx, key, value, ttl).Err()
}
