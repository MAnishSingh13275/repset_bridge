package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_WebSocketHandler_Success(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers with WebSocket manager
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
	
	// Start WebSocket manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handlers.wsManager.Start(ctx)
	defer handlers.wsManager.Stop()
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add authentication context
		authCtx := map[string]interface{}{
			"method":    "api_key",
			"userId":    "test-user",
			"expiresAt": time.Now().Add(time.Hour).Format(time.RFC3339),
		}
		ctx := context.WithValue(r.Context(), "auth", authCtx)
		r = r.WithContext(ctx)
		
		handlers.WebSocketHandler(w, r)
	}))
	defer server.Close()
	
	// Convert to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Verify connection is registered
	assert.Equal(t, 1, handlers.GetWebSocketConnectionCount())
	
	// Read welcome message
	var message map[string]interface{}
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "welcome", message["type"])
	
	// Verify connection info includes auth data
	connInfo := handlers.GetWebSocketConnectionInfo()
	require.Len(t, connInfo, 1)
	
	authData, ok := connInfo[0]["auth"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-user", authData["userId"])
	assert.Equal(t, "api_key", authData["method"])
}

func TestHandlers_WebSocketHandler_NoWebSocketManager(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers without WebSocket manager
	handlers := &Handlers{
		config:    cfg,
		logger:    logger,
		wsManager: nil, // Explicitly set to nil
		startTime: time.Now(),
		version:   "1.0.0",
		deviceID:  "test-device",
	}
	
	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	
	w := httptest.NewRecorder()
	
	// Call handler
	handlers.WebSocketHandler(w, req)
	
	// Should return service unavailable
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "WebSocket functionality not available", response["message"])
}

func TestHandlers_WebSocketHandler_NoAuthentication(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers with WebSocket manager
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
	
	// Start WebSocket manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handlers.wsManager.Start(ctx)
	defer handlers.wsManager.Stop()
	
	// Create test server without authentication context
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No authentication context added
		handlers.WebSocketHandler(w, r)
	}))
	defer server.Close()
	
	// Convert to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Connection should still work (auth is optional for WebSocket)
	assert.Equal(t, 1, handlers.GetWebSocketConnectionCount())
	
	// Verify connection info has no auth data
	connInfo := handlers.GetWebSocketConnectionInfo()
	require.Len(t, connInfo, 1)
	
	_, hasAuth := connInfo[0]["auth"]
	assert.False(t, hasAuth)
}

func TestHandlers_BroadcastEvent(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers with WebSocket manager
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
	
	// Start WebSocket manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handlers.wsManager.Start(ctx)
	defer handlers.wsManager.Stop()
	
	// Test broadcasting without connections (should not panic)
	handlers.BroadcastEvent("test_event", map[string]interface{}{
		"message": "test",
	})
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.WebSocketHandler(w, r)
	}))
	defer server.Close()
	
	// Convert to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Read welcome message
	var message map[string]interface{}
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "welcome", message["type"])
	
	// Broadcast an event
	eventData := map[string]interface{}{
		"deviceId": "test-device",
		"action":   "unlock",
		"duration": 3000,
	}
	
	handlers.BroadcastEvent("door_unlock", eventData)
	
	// Read the broadcasted event
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "door_unlock", message["type"])
	
	data, ok := message["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-device", data["deviceId"])
	assert.Equal(t, "unlock", data["action"])
	assert.Equal(t, float64(3000), data["duration"])
}

func TestHandlers_BroadcastEvent_NoWebSocketManager(t *testing.T) {
	// Create handlers without WebSocket manager
	handlers := &Handlers{
		wsManager: nil,
	}
	
	// Should not panic when WebSocket manager is nil
	handlers.BroadcastEvent("test_event", map[string]interface{}{
		"message": "test",
	})
	
	// Should return 0 connections
	assert.Equal(t, 0, handlers.GetWebSocketConnectionCount())
	
	// Should return empty connection info
	connInfo := handlers.GetWebSocketConnectionInfo()
	assert.Empty(t, connInfo)
}

func TestHandlers_WebSocketConnectionInfo(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers with WebSocket manager
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
	
	// Start WebSocket manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handlers.wsManager.Start(ctx)
	defer handlers.wsManager.Stop()
	
	// Initially should be empty
	assert.Equal(t, 0, handlers.GetWebSocketConnectionCount())
	assert.Empty(t, handlers.GetWebSocketConnectionInfo())
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add authentication context
		authCtx := map[string]interface{}{
			"method": "jwt",
			"userId": "user123",
		}
		ctx := context.WithValue(r.Context(), "auth", authCtx)
		r = r.WithContext(ctx)
		
		handlers.WebSocketHandler(w, r)
	}))
	defer server.Close()
	
	// Convert to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect multiple WebSocket clients
	dialer := websocket.Dialer{}
	
	conn1, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()
	
	conn2, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()
	
	// Give connections time to register
	time.Sleep(50 * time.Millisecond)
	
	// Check connection count
	assert.Equal(t, 2, handlers.GetWebSocketConnectionCount())
	
	// Check connection info
	connInfo := handlers.GetWebSocketConnectionInfo()
	assert.Len(t, connInfo, 2)
	
	// Verify each connection has the expected fields
	for _, info := range connInfo {
		assert.Contains(t, info, "id")
		assert.Contains(t, info, "remoteAddr")
		assert.Contains(t, info, "userAgent")
		assert.Contains(t, info, "lastPing")
		assert.Contains(t, info, "filters")
		assert.Contains(t, info, "auth")
		
		// Check auth info
		authData, ok := info["auth"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "user123", authData["userId"])
		assert.Equal(t, "jwt", authData["method"])
	}
}

func TestHandlers_WebSocketEventFiltering(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		LogLevel: "error",
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	// Create handlers with WebSocket manager
	handlers := NewHandlers(cfg, logger, nil, nil, nil, nil, nil, nil, "1.0.0", "test-device")
	
	// Start WebSocket manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handlers.wsManager.Start(ctx)
	defer handlers.wsManager.Stop()
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.WebSocketHandler(w, r)
	}))
	defer server.Close()
	
	// Convert to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Read welcome message
	var message map[string]interface{}
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "welcome", message["type"])
	
	// Set filters to only receive door_unlock events
	filterMsg := map[string]interface{}{
		"type": "set_filters",
		"filters": map[string]interface{}{
			"eventTypes": []string{"door_unlock"},
		},
	}
	
	err = conn.WriteJSON(filterMsg)
	require.NoError(t, err)
	
	// Read filter confirmation
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "filters_updated", message["type"])
	
	// Broadcast a door_lock event (should be filtered out)
	handlers.BroadcastEvent("door_lock", map[string]interface{}{
		"deviceId": "test-device",
	})
	
	// Broadcast a door_unlock event (should be received)
	handlers.BroadcastEvent("door_unlock", map[string]interface{}{
		"deviceId": "test-device",
		"duration": 3000,
	})
	
	// Should only receive the door_unlock event
	err = conn.ReadJSON(&message)
	require.NoError(t, err)
	assert.Equal(t, "door_unlock", message["type"])
	
	data, ok := message["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-device", data["deviceId"])
	assert.Equal(t, float64(3000), data["duration"])
	
	// Set a timeout to ensure no more messages are received
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	err = conn.ReadJSON(&message)
	
	// Should timeout (no more messages)
	if err == nil {
		t.Errorf("Expected timeout, but received message: %v", message)
	}
}