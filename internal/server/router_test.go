package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuisGaravaso/media-indexer/internal/config"
	"github.com/LuisGaravaso/media-indexer/internal/server"
	"github.com/stretchr/testify/assert"
)

func TestSetupRouter_RoutesConfigured(t *testing.T) {
	cfg := &config.Config{
		GinMode: "test",
	}

	pinger := &mockPinger{err: nil}
	router := server.SetupRouter(cfg, pinger)

	endpoints := []string{"/health", "/healthz", "/ping"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	}
}
