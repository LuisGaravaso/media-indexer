package server_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LuisGaravaso/media-indexer/internal/server"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthEndpoints_HealthyDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pinger := &mockPinger{err: nil}

	r.GET("/health", server.HealthHandler(pinger))
	r.GET("/healthz", server.HealthHandler(pinger))

	for _, path := range []string{"/health", "/healthz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ok"`)
		assert.Contains(t, w.Body.String(), `"db":"up"`)
		assert.Contains(t, w.Body.String(), `"service":"media-indexer"`)
	}
}

func TestHealthEndpoints_UnhealthyDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	pinger := &mockPinger{err: errors.New("db connection failure")}

	r.GET("/health", server.HealthHandler(pinger))
	r.GET("/healthz", server.HealthHandler(pinger))

	for _, path := range []string{"/health", "/healthz"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"degraded"`)
		assert.Contains(t, w.Body.String(), `"db":"down"`)
	}
}

func TestHealthEndpoints_NilDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/health", server.HealthHandler(nil))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
	assert.Contains(t, w.Body.String(), `"db":"disabled"`)
}

func TestPingEndpoint_Liveness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ping", server.PingHandler())

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}
