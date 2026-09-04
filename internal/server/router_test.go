package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/LuisGaravaso/media-indexer/internal/middleware"
	"github.com/LuisGaravaso/media-indexer/internal/search"
	"github.com/LuisGaravaso/media-indexer/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockSearchService struct {
	results []search.SemanticSearchResult
	err     error
}

func (m *mockSearchService) Search(ctx context.Context, userID uuid.UUID, req search.SemanticSearchRequest) ([]search.SemanticSearchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func TestSetupRouter_RoutesConfigured(t *testing.T) {
	cfg := &config.Config{
		GinMode: "test",
	}

	pinger := &mockPinger{err: nil}
	router := server.SetupRouter(cfg, pinger, nil, nil)

	endpoints := []string{"/health", "/healthz", "/ping"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestSearchHandler_Success(t *testing.T) {
	userID := uuid.New()
	mockSvc := &mockSearchService{
		results: []search.SemanticSearchResult{
			{
				MediaID:    uuid.New(),
				UserID:     userID,
				Summary:    "Tropical sunset",
				Similarity: 0.95,
			},
		},
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.ContextKeyUserID, userID.String())
		c.Next()
	})
	router.POST("/api/v1/search/semantic", server.SearchHandler(mockSvc))

	body, _ := json.Marshal(search.SemanticSearchRequest{
		Query: "tropical sunset",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/semantic", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Tropical sunset")
}
