package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	EventID   string      `json:"eventId,omitempty"`
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID         string
	Conn       *websocket.Conn
	Send       chan WebSocketMessage
	Filters    WebSocketFilters
	LastPing   time.Time
	RemoteAddr string
	UserAgent  string
	AuthInfo   *AuthenticationInfo
}

// WebSocketFilters represents filtering options for WebSocket messages
type WebSocketFilters struct {
	EventTypes    []string `json:"eventTypes,omitempty"`
	DeviceID      string   `json:"deviceId,omitempty"`
	UserID        string   `json:"userId,omitempty"`
	MinSeverity   string   `json:"minSeverity,omitempty"`
	IncludeSystem bool     `json:"includeSystem,omitempty"`
}

// AuthenticationInfo represents authentication information for WebSocket connections
type AuthenticationInfo struct {
	UserID    string
	Method    string // "api_key", "hmac", "jwt"
	ExpiresAt *time.Time
}

// WebSocketManager manages WebSocket connections and message broadcasting
type WebSocketManager struct {
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
	upgrader    websocket.Upgrader
	logger      *logrus.Logger
	broadcast   chan WebSocketMessage
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	done        chan struct{}
	
	// Configuration
	pingInterval    time.Duration
	pongTimeout     time.Duration
	writeTimeout    time.Duration
	readTimeout     time.Duration
	maxMessageSize  int64
	maxConnections  int
}

// NewWebSocketManager creates a new WebSocket manager
func NewWebSocketManager(logger *logrus.Logger) *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[string]*WebSocketConnection),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// TODO: Implement proper origin checking based on CORS configuration
				return true
			},
		},
		logger:         logger,
		broadcast:      make(chan WebSocketMessage, 256),
		register:       make(chan *WebSocketConnection),
		unregister:     make(chan *WebSocketConnection),
		done:           make(chan struct{}),
		pingInterval:   30 * time.Second,
		pongTimeout:    60 * time.Second,
		writeTimeout:   10 * time.Second,
		readTimeout:    60 * time.Second,
		maxMessageSize: 512,
		maxConnections: 100,
	}
}

// Start starts the WebSocket manager
func (wsm *WebSocketManager) Start(ctx context.Context) {
	wsm.logger.Info("Starting WebSocket manager")
	
	go wsm.run(ctx)
}

// Stop stops the WebSocket manager
func (wsm *WebSocketManager) Stop() {
	wsm.logger.Info("Stopping WebSocket manager")
	close(wsm.done)
}

// run is the main loop for the WebSocket manager
func (wsm *WebSocketManager) run(ctx context.Context) {
	ticker := time.NewTicker(wsm.pingInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			wsm.logger.Info("WebSocket manager context cancelled")
			return
		case <-wsm.done:
			wsm.logger.Info("WebSocket manager stopped")
			return
		case conn := <-wsm.register:
			wsm.registerConnection(conn)
		case conn := <-wsm.unregister:
			wsm.unregisterConnection(conn)
		case message := <-wsm.broadcast:
			wsm.broadcastMessage(message)
		case <-ticker.C:
			wsm.pingConnections()
		}
	}
}

// registerConnection registers a new WebSocket connection
func (wsm *WebSocketManager) registerConnection(conn *WebSocketConnection) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	// Check connection limit
	if len(wsm.connections) >= wsm.maxConnections {
		wsm.logger.WithField("connectionId", conn.ID).Warn("Maximum WebSocket connections reached")
		conn.Conn.Close()
		return
	}
	
	wsm.connections[conn.ID] = conn
	wsm.logger.WithFields(logrus.Fields{
		"connectionId": conn.ID,
		"remoteAddr":   conn.RemoteAddr,
		"userAgent":    conn.UserAgent,
		"totalConns":   len(wsm.connections),
	}).Info("WebSocket connection registered")
	
	// Send welcome message
	welcomeMsg := WebSocketMessage{
		Type:      "welcome",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"connectionId": conn.ID,
			"serverTime":   time.Now().UTC(),
			"version":      "1.0",
		},
	}
	
	select {
	case conn.Send <- welcomeMsg:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send welcome message")
	}
}

// unregisterConnection unregisters a WebSocket connection
func (wsm *WebSocketManager) unregisterConnection(conn *WebSocketConnection) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()
	
	if _, exists := wsm.connections[conn.ID]; exists {
		delete(wsm.connections, conn.ID)
		close(conn.Send)
		
		wsm.logger.WithFields(logrus.Fields{
			"connectionId": conn.ID,
			"remoteAddr":   conn.RemoteAddr,
			"totalConns":   len(wsm.connections),
		}).Info("WebSocket connection unregistered")
	}
}

// broadcastMessage broadcasts a message to all matching connections
func (wsm *WebSocketManager) broadcastMessage(message WebSocketMessage) {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	
	sentCount := 0
	for _, conn := range wsm.connections {
		if wsm.shouldSendMessage(conn, message) {
			select {
			case conn.Send <- message:
				sentCount++
			default:
				wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send message, connection buffer full")
				// Connection is blocked, close it
				go func(c *WebSocketConnection) {
					wsm.unregister <- c
				}(conn)
			}
		}
	}
	
	if sentCount > 0 {
		wsm.logger.WithFields(logrus.Fields{
			"messageType": message.Type,
			"sentCount":   sentCount,
			"totalConns":  len(wsm.connections),
		}).Debug("Message broadcasted to WebSocket connections")
	}
}

// shouldSendMessage determines if a message should be sent to a connection based on filters
func (wsm *WebSocketManager) shouldSendMessage(conn *WebSocketConnection, message WebSocketMessage) bool {
	// Check event type filter
	if len(conn.Filters.EventTypes) > 0 {
		found := false
		for _, eventType := range conn.Filters.EventTypes {
			if eventType == message.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check device ID filter
	if conn.Filters.DeviceID != "" {
		if data, ok := message.Data.(map[string]interface{}); ok {
			if deviceID, exists := data["deviceId"]; exists {
				if deviceIDStr, ok := deviceID.(string); ok && deviceIDStr != conn.Filters.DeviceID {
					return false
				}
			}
		}
	}
	
	// Check user ID filter
	if conn.Filters.UserID != "" {
		if data, ok := message.Data.(map[string]interface{}); ok {
			if userID, exists := data["userId"]; exists {
				if userIDStr, ok := userID.(string); ok && userIDStr != conn.Filters.UserID {
					return false
				}
			}
		}
	}
	
	// Check system events filter
	if !conn.Filters.IncludeSystem {
		systemEventTypes := map[string]bool{
			"system_status": true,
			"health_check":  true,
			"config_change": true,
		}
		if systemEventTypes[message.Type] {
			return false
		}
	}
	
	return true
}

// pingConnections sends ping messages to all connections
func (wsm *WebSocketManager) pingConnections() {
	wsm.mutex.RLock()
	connections := make([]*WebSocketConnection, 0, len(wsm.connections))
	for _, conn := range wsm.connections {
		connections = append(connections, conn)
	}
	wsm.mutex.RUnlock()
	
	for _, conn := range connections {
		// Check if connection is stale
		if time.Since(conn.LastPing) > wsm.pongTimeout {
			wsm.logger.WithField("connectionId", conn.ID).Warn("WebSocket connection timed out")
			wsm.unregister <- conn
			continue
		}
		
		// Send ping
		conn.Conn.SetWriteDeadline(time.Now().Add(wsm.writeTimeout))
		if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			wsm.logger.WithError(err).WithField("connectionId", conn.ID).Warn("Failed to send ping")
			wsm.unregister <- conn
		}
	}
}

// BroadcastEvent broadcasts an event to all connected clients
func (wsm *WebSocketManager) BroadcastEvent(eventType string, data interface{}) {
	message := WebSocketMessage{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
		EventID:   wsm.generateEventID(),
	}
	
	select {
	case wsm.broadcast <- message:
	default:
		wsm.logger.WithField("eventType", eventType).Warn("Broadcast channel full, dropping message")
	}
}

// GetConnectionCount returns the current number of WebSocket connections
func (wsm *WebSocketManager) GetConnectionCount() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.connections)
}

// GetConnectionInfo returns information about all connections
func (wsm *WebSocketManager) GetConnectionInfo() []map[string]interface{} {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	
	info := make([]map[string]interface{}, 0, len(wsm.connections))
	for _, conn := range wsm.connections {
		connInfo := map[string]interface{}{
			"id":         conn.ID,
			"remoteAddr": conn.RemoteAddr,
			"userAgent":  conn.UserAgent,
			"lastPing":   conn.LastPing,
			"filters":    conn.Filters,
		}
		
		if conn.AuthInfo != nil {
			connInfo["auth"] = map[string]interface{}{
				"userId":    conn.AuthInfo.UserID,
				"method":    conn.AuthInfo.Method,
				"expiresAt": conn.AuthInfo.ExpiresAt,
			}
		}
		
		info = append(info, connInfo)
	}
	
	return info
}

// generateEventID generates a unique event ID
func (wsm *WebSocketManager) generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// HandleWebSocketConnection handles a new WebSocket connection
func (wsm *WebSocketManager) HandleWebSocketConnection(w http.ResponseWriter, r *http.Request, authInfo *AuthenticationInfo) error {
	// Upgrade HTTP connection to WebSocket
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wsm.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return err
	}
	
	// Create connection object
	wsConn := &WebSocketConnection{
		ID:         wsm.generateConnectionID(),
		Conn:       conn,
		Send:       make(chan WebSocketMessage, 256),
		Filters:    WebSocketFilters{IncludeSystem: true}, // Default filters
		LastPing:   time.Now(),
		RemoteAddr: r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		AuthInfo:   authInfo,
	}
	
	// Set connection options
	conn.SetReadLimit(wsm.maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(wsm.readTimeout))
	conn.SetPongHandler(func(string) error {
		wsConn.LastPing = time.Now()
		conn.SetReadDeadline(time.Now().Add(wsm.readTimeout))
		return nil
	})
	
	// Register connection
	wsm.register <- wsConn
	
	// Start goroutines for reading and writing
	go wsm.writePump(wsConn)
	go wsm.readPump(wsConn)
	
	return nil
}

// generateConnectionID generates a unique connection ID
func (wsm *WebSocketManager) generateConnectionID() string {
	return fmt.Sprintf("conn_%d", time.Now().UnixNano())
}

// writePump handles writing messages to the WebSocket connection
func (wsm *WebSocketManager) writePump(conn *WebSocketConnection) {
	defer func() {
		conn.Conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(wsm.writeTimeout))
			if !ok {
				// Channel closed
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			// Send message as JSON
			if err := conn.Conn.WriteJSON(message); err != nil {
				wsm.logger.WithError(err).WithField("connectionId", conn.ID).Error("Failed to write WebSocket message")
				return
			}
		}
	}
}

// readPump handles reading messages from the WebSocket connection
func (wsm *WebSocketManager) readPump(conn *WebSocketConnection) {
	defer func() {
		wsm.unregister <- conn
		conn.Conn.Close()
	}()
	
	for {
		messageType, data, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wsm.logger.WithError(err).WithField("connectionId", conn.ID).Error("WebSocket connection error")
			}
			break
		}
		
		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			wsm.handleTextMessage(conn, data)
		case websocket.BinaryMessage:
			wsm.logger.WithField("connectionId", conn.ID).Warn("Binary messages not supported")
		}
		
		// Update read deadline
		conn.Conn.SetReadDeadline(time.Now().Add(wsm.readTimeout))
	}
}

// handleTextMessage handles text messages from WebSocket clients
func (wsm *WebSocketManager) handleTextMessage(conn *WebSocketConnection, data []byte) {
	var message map[string]interface{}
	if err := json.Unmarshal(data, &message); err != nil {
		wsm.logger.WithError(err).WithField("connectionId", conn.ID).Error("Failed to parse WebSocket message")
		return
	}
	
	messageType, ok := message["type"].(string)
	if !ok {
		wsm.logger.WithField("connectionId", conn.ID).Error("WebSocket message missing type field")
		return
	}
	
	wsm.logger.WithFields(logrus.Fields{
		"connectionId": conn.ID,
		"messageType":  messageType,
	}).Debug("Received WebSocket message")
	
	switch messageType {
	case "set_filters":
		wsm.handleSetFilters(conn, message)
	case "ping":
		wsm.handlePing(conn, message)
	case "subscribe":
		wsm.handleSubscribe(conn, message)
	case "unsubscribe":
		wsm.handleUnsubscribe(conn, message)
	default:
		wsm.logger.WithFields(logrus.Fields{
			"connectionId": conn.ID,
			"messageType":  messageType,
		}).Warn("Unknown WebSocket message type")
	}
}

// handleSetFilters handles filter update messages
func (wsm *WebSocketManager) handleSetFilters(conn *WebSocketConnection, message map[string]interface{}) {
	filtersData, ok := message["filters"]
	if !ok {
		wsm.sendError(conn, "Missing filters in set_filters message")
		return
	}
	
	// Parse filters
	filtersJSON, err := json.Marshal(filtersData)
	if err != nil {
		wsm.sendError(conn, "Invalid filters format")
		return
	}
	
	var filters WebSocketFilters
	if err := json.Unmarshal(filtersJSON, &filters); err != nil {
		wsm.sendError(conn, "Failed to parse filters")
		return
	}
	
	// Update connection filters
	conn.Filters = filters
	
	wsm.logger.WithFields(logrus.Fields{
		"connectionId": conn.ID,
		"filters":      filters,
	}).Info("WebSocket filters updated")
	
	// Send confirmation
	response := WebSocketMessage{
		Type:      "filters_updated",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"filters": filters,
		},
	}
	
	select {
	case conn.Send <- response:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send filters confirmation")
	}
}

// handlePing handles ping messages from clients
func (wsm *WebSocketManager) handlePing(conn *WebSocketConnection, message map[string]interface{}) {
	response := WebSocketMessage{
		Type:      "pong",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"serverTime": time.Now().UTC(),
		},
	}
	
	select {
	case conn.Send <- response:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send pong response")
	}
}

// handleSubscribe handles subscription messages
func (wsm *WebSocketManager) handleSubscribe(conn *WebSocketConnection, message map[string]interface{}) {
	eventTypes, ok := message["eventTypes"].([]interface{})
	if !ok {
		wsm.sendError(conn, "Missing or invalid eventTypes in subscribe message")
		return
	}
	
	// Convert to string slice
	var eventTypeStrings []string
	for _, et := range eventTypes {
		if etStr, ok := et.(string); ok {
			eventTypeStrings = append(eventTypeStrings, etStr)
		}
	}
	
	// Add to existing filters
	existingTypes := make(map[string]bool)
	for _, et := range conn.Filters.EventTypes {
		existingTypes[et] = true
	}
	
	for _, et := range eventTypeStrings {
		if !existingTypes[et] {
			conn.Filters.EventTypes = append(conn.Filters.EventTypes, et)
		}
	}
	
	wsm.logger.WithFields(logrus.Fields{
		"connectionId": conn.ID,
		"eventTypes":   eventTypeStrings,
	}).Info("WebSocket subscribed to event types")
	
	// Send confirmation
	response := WebSocketMessage{
		Type:      "subscribed",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"eventTypes": eventTypeStrings,
			"allFilters": conn.Filters,
		},
	}
	
	select {
	case conn.Send <- response:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send subscription confirmation")
	}
}

// handleUnsubscribe handles unsubscription messages
func (wsm *WebSocketManager) handleUnsubscribe(conn *WebSocketConnection, message map[string]interface{}) {
	eventTypes, ok := message["eventTypes"].([]interface{})
	if !ok {
		wsm.sendError(conn, "Missing or invalid eventTypes in unsubscribe message")
		return
	}
	
	// Convert to string slice
	var eventTypeStrings []string
	for _, et := range eventTypes {
		if etStr, ok := et.(string); ok {
			eventTypeStrings = append(eventTypeStrings, etStr)
		}
	}
	
	// Remove from existing filters
	var newEventTypes []string
	removeTypes := make(map[string]bool)
	for _, et := range eventTypeStrings {
		removeTypes[et] = true
	}
	
	for _, et := range conn.Filters.EventTypes {
		if !removeTypes[et] {
			newEventTypes = append(newEventTypes, et)
		}
	}
	
	conn.Filters.EventTypes = newEventTypes
	
	wsm.logger.WithFields(logrus.Fields{
		"connectionId": conn.ID,
		"eventTypes":   eventTypeStrings,
	}).Info("WebSocket unsubscribed from event types")
	
	// Send confirmation
	response := WebSocketMessage{
		Type:      "unsubscribed",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"eventTypes": eventTypeStrings,
			"allFilters": conn.Filters,
		},
	}
	
	select {
	case conn.Send <- response:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send unsubscription confirmation")
	}
}

// sendError sends an error message to a WebSocket connection
func (wsm *WebSocketManager) sendError(conn *WebSocketConnection, errorMsg string) {
	response := WebSocketMessage{
		Type:      "error",
		Timestamp: time.Now().UTC(),
		Data: map[string]interface{}{
			"error": errorMsg,
		},
	}
	
	select {
	case conn.Send <- response:
	default:
		wsm.logger.WithField("connectionId", conn.ID).Warn("Failed to send error message")
	}
}