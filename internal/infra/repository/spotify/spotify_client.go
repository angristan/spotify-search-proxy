package spotify

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	spotifyLib "github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyClientConfig struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	tracer       trace.Tracer
}

func NewSpotifyClientConfig(
	clientID string,
	clientSecret string,
	httpClient *http.Client,
	tracer trace.Tracer,
) *SpotifyClientConfig {
	return &SpotifyClientConfig{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
		tracer:       tracer,
	}
}

type SpotifyClient struct {
	tracer    trace.Tracer
	apiClient *spotifyLib.Client
	config    clientcredentials.Config
}

func New(ctx context.Context, config *SpotifyClientConfig) *SpotifyClient {
	spotifyConfig := clientcredentials.Config{
		ClientID:     config.clientID,
		ClientSecret: config.clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := spotifyConfig.Token(ctx)
	if err != nil {
		panic(err) // TODO
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, config.httpClient)

	httpClient := spotifyauth.New().Client(ctx, token)
	APIClient := spotifyLib.New(httpClient)

	return &SpotifyClient{
		apiClient: APIClient,
		tracer:    config.tracer,
		config:    spotifyConfig,
	}
}

var (
	SpotifyClientLock   = sync.RWMutex{}
	InvalidQueryTypeErr = fmt.Errorf("Invalid type")
)

type SearchType int

const (
	SearchTypeAlbum  SearchType = 1 << iota
	SearchTypeArtist            = 1 << iota
	SearchTypeTrack             = 1 << iota
)

func (st SearchType) ToSpotifySearchType() spotifyLib.SearchType {
	switch st {
	case SearchTypeAlbum:
		return spotifyLib.SearchTypeAlbum
	case SearchTypeArtist:
		return spotifyLib.SearchTypeArtist
	case SearchTypeTrack:
		return spotifyLib.SearchTypeTrack
	}

	return spotifyLib.SearchTypeArtist // TODO
}

func (client *SpotifyClient) Search(ctx context.Context, query string, qType string) (interface{}, error) {
	ctx, span := client.tracer.Start(ctx, "SpotifyClient.Search")
	defer span.End()

	var spotifyQueryType SearchType
	switch qType {
	case "artist":
		spotifyQueryType = SearchTypeArtist
	case "album":
		spotifyQueryType = SearchTypeAlbum
	case "track":
		spotifyQueryType = SearchTypeTrack
	default:
		return nil, fmt.Errorf("%w: %s", InvalidQueryTypeErr, qType)
	}

	spotifyQueryType2 := spotifyQueryType.ToSpotifySearchType()

	err := client.RenewTokenIfNeeded(ctx)
	if err != nil {
		return nil, fmt.Errorf("client.RenewTokenIfNeeded: %w", err)
	}

	results, err := client.apiClient.Search(ctx, query, spotifyQueryType2)
	if err != nil {
		return nil, err
	}

	// TODO: better way to do it?
	var result interface{}

	switch spotifyQueryType2 {
	case spotifyLib.SearchTypeArtist:
		if results.Artists != nil {
			if len(results.Artists.Artists) > 0 {
				result = results.Artists.Artists[0]
			}
		}
	case spotifyLib.SearchTypeAlbum:
		if results.Albums != nil {
			if len(results.Albums.Albums) > 0 {
				result = results.Albums.Albums[0]
			}
		}
	case spotifyLib.SearchTypeTrack:
		if results.Tracks != nil {
			if len(results.Tracks.Tracks) > 0 {
				result = results.Tracks.Tracks[0]
			}
		}
	}

	return result, nil
}

// Check if the token expires soon, and if so recreates an API client with a new token
func (client *SpotifyClient) RenewTokenIfNeeded(ctx context.Context) error {
	ctx, span := client.tracer.Start(ctx, "SpotifyClient.RenewTokenIfNeeded")
	defer span.End()

	span.AddEvent("Checking if Spotify token needs to be renewed")

	spotifyToken, err := client.apiClient.Token()
	if err != nil {
		return fmt.Errorf("client.apiClient.Token: %w", err)
	}
	if time.Until(spotifyToken.Expiry) > time.Minute*5 {
		span.AddEvent("Token is still valid, no need to refresh", trace.WithAttributes(
			attribute.Float64("minutes_until_expiry", time.Until(spotifyToken.Expiry).Minutes()),
		))
		return nil
	}

	token, err := client.config.Token(ctx)
	if err != nil {
		return fmt.Errorf("client.config.Token: %w", err)
	}

	httpClient := spotifyauth.New().Client(ctx, token)
	client.apiClient = spotifyLib.New(httpClient)

	span.AddEvent("Token refreshed")

	return nil
}
