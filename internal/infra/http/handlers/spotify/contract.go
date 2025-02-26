package spotify

import "context"

type SpotifyService interface {
	Search(ctx context.Context, query string, searchType string) (any, error)
}
