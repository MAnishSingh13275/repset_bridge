package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)





func TestNewServer(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	
	server := NewServer(cfg, serverCfg, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	assert.NotNil(t, server)
	assert.NotNil(t, server.config)
	assert.NotNil(t, server.logger)
	assert.NotNil(t, server.router)
	assert.NotNil(t, server.httpServer)
	assert.NotNil(t, server.handlers)
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	
	assert.Equal(t, 8081, cfg.Port)
	assert.Equal(t, "0.0.0.0", cfg.Host)
	assert.False(t, cfg.TLSEnabled)
	assert.Equal(t, 30, cfg.ReadTimeout)
	assert.Equal(t, 30, cfg.WriteTimeout)
	assert.Equal(t, 120, cfg.IdleTimeout)
}

func TestServerRoutes(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	server := createTestServer(cfg, serverCfg)
	
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "Health check endpoint",
			method:         "GET",
			path:           "/api/v1/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Door unlock endpoint (implemented - bad request)",
			method:         "POST",
			path:           "/api/v1/door/unlock",
			expectedStatus: http.StatusBadRequest, // Empty body causes JSON decode error
		},
		{
			name:           "Door lock endpoint (implemented - success)",
			method:         "POST",
			path:           "/api/v1/door/lock",
			expectedStatus: http.StatusOK, // Lock endpoint accepts empty body
		},
		{
			name:           "Door status endpoint (implemented - success)",
			method:         "GET",
			path:           "/api/v1/door/status",
			expectedStatus: http.StatusOK, // Status endpoint works without body
		},
		{
			name:           "Device status endpoint (implemented - success)",
			method:         "GET",
			path:           "/api/v1/status",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Device metrics endpoint (implemented - success)",
			method:         "GET",
			path:           "/api/v1/metrics",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get config endpoint (implemented - success)",
			method:         "GET",
			path:           "/api/v1/config",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update config endpoint (implemented - bad request)",
			method:         "PUT",
			path:           "/api/v1/config",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Reload config endpoint (implemented - service unavailable)",
			method:         "POST",
			path:           "/api/v1/config/reload",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Get events endpoint (service unavailable)",
			method:         "GET",
			path:           "/api/v1/events",
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "Get adapters endpoint (implemented)",
			method:         "GET",
			path:           "/api/v1/adapters",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Get specific adapter endpoint (server error due to mock)",
			method:         "GET",
			path:           "/api/v1/adapters/test",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "WebSocket endpoint (bad request - no upgrade headers)",
			method:         "GET",
			path:           "/api/v1/ws",
			expectedStatus: http.StatusBadRequest, // WebSocket upgrade fails without proper headers
		},
		{
			name:           "Non-existent endpoint",
			method:         "GET",
			path:           "/api/v1/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			server.router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestServerMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	server := createTestServer(cfg, serverCfg)
	
	t.Run("CORS headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	})
	
	t.Run("Security headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		
		server.router.ServeHTTP(w, req)
		
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	})
	
	t.Run("OPTIONS request handling", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		
		server.router.ServeHTTP(w, req)
		
		// The CORS middleware should handle OPTIONS requests and return 200
		// Even if the route doesn't exist, CORS middleware should still set headers
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	})
}

func TestServerShutdown(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	serverCfg.Port = 0 // Use random available port
	
	server := createTestServer(cfg, serverCfg)
	
	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx, serverCfg)
	}()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Cancel context to trigger shutdown
	cancel()
	
	// Wait for shutdown to complete
	select {
	case err := <-errChan:
		// Context cancellation should not be treated as an error
		if err != nil && err != context.Canceled {
			t.Errorf("Unexpected error during shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server shutdown timed out")
	}
}

func TestHealthCheckEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	server := createTestServer(cfg, serverCfg)
	
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	
	server.router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	// Check response body contains expected fields
	body := w.Body.String()
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "timestamp")
	assert.Contains(t, body, "version")
}

func TestTLSConfiguration(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	serverCfg.TLSEnabled = true
	serverCfg.TLSCertFile = "test.crt"
	serverCfg.TLSKeyFile = "test.key"
	
	// This should not panic even with invalid cert files
	// since we're not actually starting the server
	server := createTestServer(cfg, serverCfg)
	
	assert.NotNil(t, server)
	assert.NotNil(t, server.httpServer.TLSConfig)
	require.NotNil(t, server.httpServer.TLSConfig)
	assert.Equal(t, uint16(0x0303), server.httpServer.TLSConfig.MinVersion) // TLS 1.2
}
