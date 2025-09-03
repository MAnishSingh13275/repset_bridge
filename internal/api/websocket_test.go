package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebSocketManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	
	wsm := NewWebSocketManager(logger)
	
	assert.NotNil(t, wsm)
	assert.NotNil(t, wsm.connections)
	assert.NotNil(t, wsm.upgrader)
	assert.NotNil(t, wsm.logger)
	assert.NotNil(t, wsm.broadcast)
	assert.NotNil(t, wsm.register)
	assert.NotNil(t, wsm.unregister)
	assert.NotNil(t, wsm.done)
	assert.Equal(t, 30*time.Second, wsm.pingInterval)
	assert.Equal(t, 60*time.Second, wsm.pongTimeout)
	assert.Equal(t, 10*time.Second, wsm.writeTimeout)
	assert.Equal(t, 60*time.Second, wsm.readTimeout)
	assert.Equal(t, int64(512), wsm.maxMessageSize)
	assert.Equal(t, 100, wsm.maxConnections)
}

func TestWebSocketManager_StartStop(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start the manager
	wsm.Start(ctx)
	
	// Give it a moment to start
	time.Sleep(10 * time.Millisecond)
	
	// Stop the manager
	cancel()
	wsm.Stop()
	
	// Give it a moment to stop
	time.Sleep(10 * time.Millisecond)
}

func TestWebSocketManager_BroadcastEvent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	wsm.Start(ctx)
	defer wsm.Stop()
	
	// Test broadcasting an event
	eventData := map[string]interface{}{
		"deviceId": "test-device",
		"message":  "test message",
	}
	
	wsm.BroadcastEvent("test_event", eventData)
	
	// Since there are no connections, this should not cause any issues
	assert.Equal(t, 0, wsm.GetConnectionCount())
}

func TestWebSocketManager_ConnectionCount(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	
	// Initially should be 0
	assert.Equal(t, 0, wsm.GetConnectionCount())
	
	// Add a mock connection
	conn := &WebSocketConnection{
		ID:         "test-conn-1",
		Send:       make(chan WebSocketMessage, 256),
		Filters:    WebSocketFilters{},
		LastPing:   time.Now(),
		RemoteAddr: "127.0.0.1:12345",
		UserAgent:  "test-agent",
	}
	
	wsm.mutex.Lock()
	wsm.connections[conn.ID] = conn
	wsm.mutex.Unlock()
	
	assert.Equal(t, 1, wsm.GetConnectionCount())
	
	// Remove the connection
	wsm.mutex.Lock()
	delete(wsm.connections, conn.ID)
	wsm.mutex.Unlock()
	
	assert.Equal(t, 0, wsm.GetConnectionCount())
}

func TestWebSocketManager_GetConnectionInfo(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	
	// Initially should be empty
	info := wsm.GetConnectionInfo()
	assert.Empty(t, info)
	
	// Add a mock connection with auth info
	authInfo := &AuthenticationInfo{
		UserID:    "test-user",
		Method:    "api_key",
		ExpiresAt: nil,
	}
	
	conn := &WebSocketConnection{
		ID:         "test-conn-1",
		Send:       make(chan WebSocketMessage, 256),
		Filters:    WebSocketFilters{EventTypes: []string{"door_unlock"}},
		LastPing:   time.Now(),
		RemoteAddr: "127.0.0.1:12345",
		UserAgent:  "test-agent",
		AuthInfo:   authInfo,
	}
	
	wsm.mutex.Lock()
	wsm.connections[conn.ID] = conn
	wsm.mutex.Unlock()
	
	info = wsm.GetConnectionInfo()
	require.Len(t, info, 1)
	
	connInfo := info[0]
	assert.Equal(t, "test-conn-1", connInfo["id"])
	assert.Equal(t, "127.0.0.1:12345", connInfo["remoteAddr"])
	assert.Equal(t, "test-agent", connInfo["userAgent"])
	
	// Check auth info
	authData, ok := connInfo["auth"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-user", authData["userId"])
	assert.Equal(t, "api_key", authData["method"])
}

func TestWebSocketFilters_ShouldSendMessage(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	
	tests := []struct {
		name     string
		filters  WebSocketFilters
		message  WebSocketMessage
		expected bool
	}{
		{
			name:    "no filters - should send",
			filters: WebSocketFilters{},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"deviceId": "device1"},
			},
			expected: true,
		},
		{
			name:    "event type filter - matching",
			filters: WebSocketFilters{EventTypes: []string{"door_unlock", "door_lock"}},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"deviceId": "device1"},
			},
			expected: true,
		},
		{
			name:    "event type filter - not matching",
			filters: WebSocketFilters{EventTypes: []string{"door_lock"}},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"deviceId": "device1"},
			},
			expected: false,
		},
		{
			name:    "device ID filter - matching",
			filters: WebSocketFilters{DeviceID: "device1"},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"deviceId": "device1"},
			},
			expected: true,
		},
		{
			name:    "device ID filter - not matching",
			filters: WebSocketFilters{DeviceID: "device2"},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"deviceId": "device1"},
			},
			expected: false,
		},
		{
			name:    "user ID filter - matching",
			filters: WebSocketFilters{UserID: "user1"},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"userId": "user1"},
			},
			expected: true,
		},
		{
			name:    "user ID filter - not matching",
			filters: WebSocketFilters{UserID: "user2"},
			message: WebSocketMessage{
				Type: "door_unlock",
				Data: map[string]interface{}{"userId": "user1"},
			},
			expected: false,
		},
		{
			name:    "system events excluded",
			filters: WebSocketFilters{IncludeSystem: false},
			message: WebSocketMessage{
				Type: "system_status",
				Data: map[string]interface{}{},
			},
			expected: false,
		},
		{
			name:    "system events included",
			filters: WebSocketFilters{IncludeSystem: true},
			message: WebSocketMessage{
				Type: "system_status",
				Data: map[string]interface{}{},
			},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &WebSocketConnection{
				ID:      "test-conn",
				Filters: tt.filters,
			}
			
			result := wsm.shouldSendMessage(conn, tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWebSocketMessage_JSON(t *testing.T) {
	message := WebSocketMessage{
		Type:      "door_unlock",
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Data: map[string]interface{}{
			"deviceId": "device1",
			"duration": 3000,
		},
		EventID: "evt_123",
	}
	
	// Test JSON marshaling
	jsonData, err := json.Marshal(message)
	require.NoError(t, err)
	
	// Test JSON unmarshaling
	var unmarshaled WebSocketMessage
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, message.Type, unmarshaled.Type)
	assert.Equal(t, message.EventID, unmarshaled.EventID)
	assert.True(t, message.Timestamp.Equal(unmarshaled.Timestamp))
	
	// Check data
	data, ok := unmarshaled.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "device1", data["deviceId"])
	assert.Equal(t, float64(3000), data["duration"]) // JSON numbers are float64
}

func TestWebSocketFilters_JSON(t *testing.T) {
	filters := WebSocketFilters{
		EventTypes:    []string{"door_unlock", "door_lock"},
		DeviceID:      "device1",
		UserID:        "user1",
		MinSeverity:   "warning",
		IncludeSystem: true,
	}
	
	// Test JSON marshaling
	jsonData, err := json.Marshal(filters)
	require.NoError(t, err)
	
	// Test JSON unmarshaling
	var unmarshaled WebSocketFilters
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, filters.EventTypes, unmarshaled.EventTypes)
	assert.Equal(t, filters.DeviceID, unmarshaled.DeviceID)
	assert.Equal(t, filters.UserID, unmarshaled.UserID)
	assert.Equal(t, filters.MinSeverity, unmarshaled.MinSeverity)
	assert.Equal(t, filters.IncludeSystem, unmarshaled.IncludeSystem)
}

// Integration test with actual WebSocket connection
func TestWebSocketManager_Integration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	wsm.Start(ctx)
	defer wsm.Stop()
	
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := wsm.HandleWebSocketConnection(w, r, &AuthenticationInfo{
			UserID: "test-user",
			Method: "test",
		})
		if err != nil {
			t.Errorf("Failed to handle WebSocket connection: %v", err)
		}
	}))
	defer server.Close()
	
	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	
	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give the connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Check that connection is registered
	assert.Equal(t, 1, wsm.GetConnectionCount())
	
	// Read the welcome message first
	var response map[string]interface{}
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "welcome", response["type"])
	
	// Send a message to set filters
	filterMsg := map[string]interface{}{
		"type": "set_filters",
		"filters": map[string]interface{}{
			"eventTypes": []string{"door_unlock"},
		},
	}
	
	err = conn.WriteJSON(filterMsg)
	require.NoError(t, err)
	
	// Read the filters confirmation
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "filters_updated", response["type"])
	
	// Broadcast an event
	wsm.BroadcastEvent("door_unlock", map[string]interface{}{
		"deviceId": "test-device",
		"duration": 3000,
	})
	
	// Read the broadcasted event
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "door_unlock", response["type"])
	
	data, ok := response["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-device", data["deviceId"])
	assert.Equal(t, float64(3000), data["duration"])
	
	// Close connection
	conn.Close()
	
	// Give the connection time to unregister
	time.Sleep(50 * time.Millisecond)
	
	// Check that connection is unregistered
	assert.Equal(t, 0, wsm.GetConnectionCount())
}

func TestWebSocketManager_MaxConnections(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	wsm.maxConnections = 2 // Set low limit for testing
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	wsm.Start(ctx)
	defer wsm.Stop()
	
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := wsm.HandleWebSocketConnection(w, r, nil)
		if err != nil {
			t.Logf("WebSocket connection failed (expected for max connections test): %v", err)
		}
	}))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	
	// Connect first two clients (should succeed)
	conn1, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()
	
	conn2, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()
	
	// Give connections time to register
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, wsm.GetConnectionCount())
	
	// Third connection should be rejected (connection limit reached)
	conn3, _, err := dialer.Dial(wsURL, nil)
	if err == nil {
		// Connection might succeed but should be closed immediately
		conn3.Close()
		time.Sleep(50 * time.Millisecond)
		// Should still be only 2 connections
		assert.Equal(t, 2, wsm.GetConnectionCount())
	}
}

func TestWebSocketManager_PingPong(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	wsm.pingInterval = 100 * time.Millisecond // Fast ping for testing
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	wsm.Start(ctx)
	defer wsm.Stop()
	
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := wsm.HandleWebSocketConnection(w, r, nil)
		if err != nil {
			t.Errorf("Failed to handle WebSocket connection: %v", err)
		}
	}))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Set up pong handler
	conn.SetPongHandler(func(appData string) error {
		return nil
	})
	
	// Give connection time to register and receive pings
	time.Sleep(200 * time.Millisecond)
	
	assert.Equal(t, 1, wsm.GetConnectionCount())
}

func TestWebSocketManager_MessageHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	
	wsm := NewWebSocketManager(logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	wsm.Start(ctx)
	defer wsm.Stop()
	
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := wsm.HandleWebSocketConnection(w, r, nil)
		if err != nil {
			t.Errorf("Failed to handle WebSocket connection: %v", err)
		}
	}))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}
	
	conn, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()
	
	// Give connection time to register
	time.Sleep(50 * time.Millisecond)
	
	// Read the welcome message first
	var response map[string]interface{}
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "welcome", response["type"])
	
	// Test ping message
	pingMsg := map[string]interface{}{
		"type": "ping",
	}
	
	err = conn.WriteJSON(pingMsg)
	require.NoError(t, err)
	
	// Read pong response
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "pong", response["type"])
	
	// Test subscribe message
	subscribeMsg := map[string]interface{}{
		"type":       "subscribe",
		"eventTypes": []interface{}{"door_unlock", "door_lock"},
	}
	
	err = conn.WriteJSON(subscribeMsg)
	require.NoError(t, err)
	
	// Read subscription confirmation
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "subscribed", response["type"])
	
	// Test unsubscribe message
	unsubscribeMsg := map[string]interface{}{
		"type":       "unsubscribe",
		"eventTypes": []interface{}{"door_lock"},
	}
	
	err = conn.WriteJSON(unsubscribeMsg)
	require.NoError(t, err)
	
	// Read unsubscription confirmation
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "unsubscribed", response["type"])
}