package spotify

import (
	"context"
	"net/http"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type SpotifyClientConfig struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	tracer       trace.Tracer
}

func NewSpotifyClientConfig(
	clientID string,
	clientSecret string,
	httpClient *http.Client,
	tracer trace.Tracer,
) *SpotifyClientConfig {
	return &SpotifyClientConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		HTTPClient:   httpClient,
		tracer:       tracer,
	}
}

type SpotifyClient struct {
	tracer    trace.Tracer
	apiClient *spotify.Client
}

func NewSpotifyClient(ctx context.Context, config *SpotifyClientConfig) *SpotifyClient {
	spotifyConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := spotifyConfig.Token(ctx)
	if err != nil {
		panic(err) //TODO
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, config.HTTPClient)

	httpClient := spotifyauth.New().Client(ctx, token)
	APIClient := spotify.New(httpClient)

	return &SpotifyClient{
		apiClient: APIClient,
		tracer:    config.tracer,
	}
}

type SearchType int

const (
	SearchTypeAlbum  SearchType = 1 << iota
	SearchTypeArtist            = 1 << iota
	SearchTypeTrack             = 1 << iota
)

func (st SearchType) ToSpotifySearchType() spotify.SearchType {
	switch st {
	case SearchTypeAlbum:
		return spotify.SearchTypeAlbum
	case SearchTypeArtist:
		return spotify.SearchTypeArtist
	case SearchTypeTrack:
		return spotify.SearchTypeTrack
	}

	return spotify.SearchTypeArtist //TODO
}

func (client *SpotifyClient) Search(ctx context.Context, query string, qType SearchType) (interface{}, error) {
	ctx, span := client.tracer.Start(ctx, "SpotifyClient.Search")
	defer span.End()

	spotifyQueryType := qType.ToSpotifySearchType()

	// TODO: client.client...
	results, err := client.apiClient.Search(ctx, query, spotifyQueryType)
	if err != nil {
		return nil, err
	}

	// TODO: better way to do it?
	var result interface{}

	switch spotifyQueryType {
	case spotify.SearchTypeArtist:
		if results.Artists != nil {
			if len(results.Artists.Artists) > 0 {
				result = results.Artists.Artists[0]
			}
		}
	case spotify.SearchTypeAlbum:
		if results.Albums != nil {
			if len(results.Albums.Albums) > 0 {
				result = results.Albums.Albums[0]
			}
		}
	case spotify.SearchTypeTrack:
		if results.Tracks != nil {
			if len(results.Tracks.Tracks) > 0 {
				result = results.Tracks.Tracks[0]
			}
		}
	}

	return result, nil
}

// TODO: renew token
