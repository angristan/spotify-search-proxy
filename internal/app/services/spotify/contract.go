package spotify

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type SpotifyClient interface {
	Search(ctx context.Context, query string, searchType string) (any, error)
}
