package spotify_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/angristan/spotify-search-proxy/internal/app/services/spotify"
	"github.com/angristan/spotify-search-proxy/internal/app/services/spotify/mocks"
	"github.com/angristan/spotify-search-proxy/internal/infra/repository/cache/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel"
)

func TestSpotifySearchService_Search(t *testing.T) {
	mockedSpotifyClient := &mocks.MockSpotifyClient{}
	mockedCache := &mocks.MockCache{}

	s := spotify.New(
		otel.Tracer("test"),
		mockedSpotifyClient,
		mockedCache,
	)

	t.Run("invalid query type", func(t *testing.T) {
		_, err := s.Search(context.TODO(), "test", "invalid")
		assert.ErrorIs(t, err, spotify.InvalidQueryTypeErr)
	})

	t.Run("no results found", func(t *testing.T) {
		mockedCache.On("Get",
			mock.Anything,
			"spotify:artist:TWICE",
		).
			Return("", redis.ErrCacheMiss).
			Once()
		mockedSpotifyClient.On("Search",
			mock.Anything, "TWICE", "artist",
		).
			Return(nil, nil).
			Once()

		_, err := s.Search(context.Background(), "TWICE", "artist")
		assert.ErrorIs(t, err, spotify.NoResultsFoundErr)
	})

	t.Run("spotify client error", func(t *testing.T) {
		mockedCache.On("Get",
			mock.Anything,
			"spotify:artist:TWICE",
		).
			Return("", redis.ErrCacheMiss).
			Once()
		mockedSpotifyClient.On("Search",
			mock.Anything, "TWICE", "artist",
		).
			Return(nil, spotify.SpotifyClientErr).
			Once()

		_, err := s.Search(context.Background(), "TWICE", "artist")
		assert.ErrorIs(t, err, spotify.SpotifyClientErr)
	})

	t.Run("cache miss", func(t *testing.T) {
		mockedCache.On("Get",
			mock.Anything,
			"spotify:artist:TWICE",
		).
			Return("", redis.ErrCacheMiss).
			Once()
		mockedSpotifyClient.On("Search",
			mock.Anything, "TWICE", "artist",
		).
			Return("data", nil).
			Once()
		mockedCache.On("Set",
			mock.Anything,
			"spotify:artist:TWICE",
			mock.Anything,
			time.Hour*24,
		).
			Return(nil).
			Once()

		_, err := s.Search(context.Background(), "TWICE", "artist")
		assert.NoError(t, err)
	})

	t.Run("cache hit", func(t *testing.T) {
		mockedCache.On("Get",
			mock.Anything,
			"spotify:artist:TWICE",
		).
			Return(`{"data": "TODO"}`, nil).
			Once()

		_, err := s.Search(context.Background(), "TWICE", "artist")
		assert.NoError(t, err)
	})

	t.Run("cache set error", func(t *testing.T) {
		mockedCache.On("Get", mock.Anything, "spotify:artist:TWICE").
			Return("", redis.ErrCacheMiss).
			Once()
		mockedSpotifyClient.On("Search", mock.Anything, "TWICE", "artist").
			Return("data", nil).
			Once()
		mockedCache.On("Set", mock.Anything, "spotify:artist:TWICE", mock.Anything, time.Hour*24).
			Return(errors.New("TODO")).
			Once()

		_, err := s.Search(context.Background(), "TWICE", "artist")
		assert.NoError(t, err)
	})
}
