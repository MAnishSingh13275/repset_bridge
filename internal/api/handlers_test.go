package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)



func TestNewHandlers(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	assert.NotNil(t, handlers)
	assert.Equal(t, cfg, handlers.config)
	assert.Equal(t, logger, handlers.logger)
	assert.Equal(t, mockAdapterRegistry, handlers.adapterRegistry)
	assert.Equal(t, mockDoorController, handlers.doorController)
}

func TestHealthCheckHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	
	handlers.HealthCheck(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	assert.Contains(t, body, "status")
	assert.Contains(t, body, "healthy")
	assert.Contains(t, body, "timestamp")
	assert.Contains(t, body, "version")
}

func TestNotImplementedHandlers(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	tests := []struct {
		name    string
		handler http.HandlerFunc
		method  string
		path    string
	}{
		{"WebSocketHandler", handlers.WebSocketHandler, "GET", "/api/v1/ws"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			
			tt.handler(w, req)
			
			// WebSocket handler now tries to upgrade connection, which fails without proper headers
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			
			body := w.Body.String()
			assert.Contains(t, body, "Failed to establish WebSocket connection")
		})
	}
}

func TestGetAdaptersHandler(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockDoorController := &MockDoorController{}
	
	t.Run("GetAdapters with nil adapter registry", func(t *testing.T) {
		// Create handlers with nil adapter registry
		nilHandlers := NewHandlers(cfg, logger, nil, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
		
		req := httptest.NewRequest("GET", "/api/v1/adapters", nil)
		w := httptest.NewRecorder()
		
		nilHandlers.GetAdapters(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		// Should return empty adapters list but still be successful
		body := w.Body.String()
		assert.Contains(t, body, `"totalCount":0`)
		assert.Contains(t, body, `"activeCount":0`)
	})
}

func TestAdapterHandlersWithParams(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	
	tests := []struct {
		name           string
		handler        http.HandlerFunc
		method         string
		path           string
		adapterName    string
		requestBody    string
		expectedStatus int
		setupMocks     func()
	}{
		{
			name:           "GetAdapter - adapter registry unavailable",
			handler:        nil, // Will be set in test
			method:         "GET",
			path:           "/api/v1/adapters/{name}",
			adapterName:    "test-adapter",
			requestBody:    "",
			expectedStatus: http.StatusServiceUnavailable,
			setupMocks:     func() {
				// No mocks needed - adapter registry will be nil
			},
		},
		{
			name:           "EnableAdapter - config manager unavailable",
			handler:        nil, // Will be set in test
			method:         "POST",
			path:           "/api/v1/adapters/{name}/enable",
			adapterName:    "test-adapter",
			requestBody:    "",
			expectedStatus: http.StatusServiceUnavailable,
			setupMocks:     func() {
				// No mocks needed - config manager will be nil
			},
		},
		{
			name:           "DisableAdapter - config manager unavailable",
			handler:        nil, // Will be set in test
			method:         "POST",
			path:           "/api/v1/adapters/{name}/disable",
			adapterName:    "test-adapter",
			requestBody:    "",
			expectedStatus: http.StatusServiceUnavailable,
			setupMocks:     func() {
				// No mocks needed - config manager will be nil
			},
		},
		{
			name:           "UpdateAdapterConfig - config manager unavailable",
			handler:        nil, // Will be set in test
			method:         "PUT",
			path:           "/api/v1/adapters/{name}/config",
			adapterName:    "test-adapter",
			requestBody:    `{"config": {"enabled": true}}`,
			expectedStatus: http.StatusServiceUnavailable,
			setupMocks:     func() {
				// No mocks needed - config manager will be nil
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks for this test
			tt.setupMocks()
			
			// Create handlers with nil dependencies to test service unavailable cases
			var testHandlers *Handlers
			if strings.Contains(tt.name, "adapter registry unavailable") {
				testHandlers = NewHandlers(cfg, logger, nil, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
				tt.handler = testHandlers.GetAdapter
			} else if strings.Contains(tt.name, "config manager unavailable") {
				testHandlers = NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
				switch tt.name {
				case "EnableAdapter - config manager unavailable":
					tt.handler = testHandlers.EnableAdapter
				case "DisableAdapter - config manager unavailable":
					tt.handler = testHandlers.DisableAdapter
				case "UpdateAdapterConfig - config manager unavailable":
					tt.handler = testHandlers.UpdateAdapterConfig
				}
			}
			
			// Create a router to handle path parameters
			router := mux.NewRouter()
			router.HandleFunc(tt.path, tt.handler).Methods(tt.method)
			
			actualPath := "/api/v1/adapters/" + tt.adapterName
			if tt.path != "/api/v1/adapters/{name}" {
				actualPath += tt.path[len("/api/v1/adapters/{name}"):]
			}
			
			req := httptest.NewRequest(tt.method, actualPath, strings.NewReader(tt.requestBody))
			if tt.requestBody != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	w := httptest.NewRecorder()
	data := map[string]string{"test": "value"}
	
	handlers.writeJSONResponse(w, data, http.StatusOK)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	assert.Contains(t, body, "test")
	assert.Contains(t, body, "value")
}

func TestWriteErrorResponse(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := logging.Initialize("info")
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	handlers := NewHandlers(cfg, logger, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
	
	w := httptest.NewRecorder()
	message := "Test error message"
	requestID := "test-request-id"
	
	handlers.writeErrorResponseLegacy(w, message, http.StatusBadRequest, "TEST_ERROR", requestID)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	assert.Contains(t, body, "error")
	assert.Contains(t, body, "true")
	assert.Contains(t, body, message)
	assert.Contains(t, body, "timestamp")
	assert.Contains(t, body, requestID)
}