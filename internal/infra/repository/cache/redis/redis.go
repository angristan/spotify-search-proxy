package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var ErrCacheMiss = errors.New("cache: key not found")

type RedisCache struct {
	tracer      trace.Tracer
	redisClient *redis.Client
}

func New(
	tracer trace.Tracer,
	redisClient *redis.Client,
) *RedisCache {
	return &RedisCache{
		tracer:      tracer,
		redisClient: redisClient,
	}
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	ctx, span := c.tracer.Start(ctx, "RedisCache.Get")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	value, err := c.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			span.SetStatus(codes.Ok, "Cache miss")
			return "", ErrCacheMiss
		}

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	span.SetAttributes(attribute.Int("value_length", len(value)))
	span.SetStatus(codes.Ok, "Cache hit")
	return value, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "RedisCache.Set")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	span.SetAttributes(attribute.Int64("ttl", int64(ttl.Seconds())))

	return c.redisClient.Set(ctx, key, value, ttl).Err()
}
