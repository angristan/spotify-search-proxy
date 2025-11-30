package spotify_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appspotify "github.com/angristan/spotify-search-proxy/internal/app/services/spotify"
	handler "github.com/angristan/spotify-search-proxy/internal/infra/http/handlers/spotify"
	"github.com/angristan/spotify-search-proxy/internal/infra/http/handlers/spotify/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func TestSpotifyHandler_SearchStatuses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		pathType       string
		rawQuery       string
		expectedQuery  string
		serviceResult  any
		serviceErr     error
		expectedStatus int
		expectedBody   map[string]string
	}{
		{
			name:           "invalid type",
			pathType:       "invalid",
			rawQuery:       "artist",
			expectedQuery:  "artist",
			serviceErr:     appspotify.InvalidQueryTypeErr,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   map[string]string{"error": "invalid search type"},
		},
		{
			name:           "no results",
			pathType:       "artist",
			rawQuery:       "twice",
			expectedQuery:  "twice",
			serviceErr:     appspotify.NoResultsFoundErr,
			expectedStatus: http.StatusNotFound,
			expectedBody:   map[string]string{"error": "no results found"},
		},
		{
			name:           "spotify client error",
			pathType:       "artist",
			rawQuery:       "aespa",
			expectedQuery:  "aespa",
			serviceErr:     appspotify.SpotifyClientErr,
			expectedStatus: http.StatusBadGateway,
			expectedBody:   map[string]string{"error": "spotify client error"},
		},
		{
			name:           "unexpected error",
			pathType:       "artist",
			rawQuery:       "ive",
			expectedQuery:  "ive",
			serviceErr:     assert.AnError,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   map[string]string{"error": "internal server error"},
		},
		{
			name:           "success with slash",
			pathType:       "artist",
			rawQuery:       "/AC/DC",
			expectedQuery:  "AC/DC",
			serviceResult:  map[string]string{"name": "AC/DC"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			req := httptest.NewRequest(http.MethodGet, "/search/"+tt.pathType+"/"+tt.rawQuery, nil)
			ctx.Request = req
			ctx.Params = gin.Params{
				{Key: "type", Value: tt.pathType},
				{Key: "query", Value: tt.rawQuery},
			}

			mockService := &mocks.MockSpotifyService{}
			t.Cleanup(func() {
				mockService.AssertExpectations(t)
			})

			if tt.serviceErr != nil {
				mockService.On("Search", mock.Anything, tt.expectedQuery, tt.pathType).
					Return(nil, tt.serviceErr).
					Once()
			} else {
				mockService.On("Search", mock.Anything, tt.expectedQuery, tt.pathType).
					Return(tt.serviceResult, nil).
					Once()
			}

			h := handler.New(otel.Tracer("test"), mockService)
			h.Search(ctx)

			assert.Equal(t, tt.expectedStatus, recorder.Code)

			if tt.expectedStatus == http.StatusOK {
				var payload map[string]string
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
				assert.Equal(t, tt.serviceResult, payload)
				return
			}

			var payload map[string]string
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
			assert.Equal(t, tt.expectedBody, payload)
		})
	}
}
