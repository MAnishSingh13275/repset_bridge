package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gym-door-bridge/internal/types"
)





// WebhookAdapter implements the HardwareAdapter interface for HTTP-based integrations
type WebhookAdapter struct {
	name          string
	config        types.AdapterConfig
	status        types.AdapterStatus
	eventCallback types.EventCallback
	server        *http.Server
	isListening   bool
	mutex         sync.RWMutex
	logger        *slog.Logger
	port          int
	path          string
	authToken     string
}

// WebhookEvent represents the expected webhook payload structure
type WebhookEvent struct {
	ExternalUserID string                 `json:"externalUserId"`
	EventType      string                 `json:"eventType"`
	Timestamp      *time.Time             `json:"timestamp,omitempty"`
	RawData        map[string]interface{} `json:"rawData,omitempty"`
}

// NewWebhookAdapter creates a new webhook adapter instance
func NewWebhookAdapter(logger *slog.Logger) *WebhookAdapter {
	return &WebhookAdapter{
		name:   "webhook",
		logger: logger,
		status: types.AdapterStatus{
			Name:      "webhook",
			Status:    types.StatusDisabled,
			UpdatedAt: time.Now(),
		},
		port: 8080,
		path: "/webhook",
	}
}

// Name returns the adapter name
func (w *WebhookAdapter) Name() string {
	return w.name
}

// Initialize sets up the webhook adapter with configuration
func (w *WebhookAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.config = config
	w.status.Status = types.StatusInitializing
	w.status.UpdatedAt = time.Now()

	// Parse configuration settings
	if settings := config.Settings; settings != nil {
		if port, ok := settings["port"].(float64); ok {
			w.port = int(port)
		}
		if path, ok := settings["path"].(string); ok {
			w.path = path
		}
		if token, ok := settings["authToken"].(string); ok {
			w.authToken = token
		}
	}

	// Validate configuration
	if w.port <= 0 || w.port > 65535 {
		w.status.Status = types.StatusError
		w.status.ErrorMessage = "invalid port number"
		w.status.UpdatedAt = time.Now()
		return fmt.Errorf("invalid port number: %d", w.port)
	}

	if w.path == "" {
		w.path = "/webhook"
	}

	w.status.Status = types.StatusActive
	w.status.UpdatedAt = time.Now()
	w.status.ErrorMessage = ""

	w.logger.Info("Webhook adapter initialized",
		"name", w.name,
		"port", w.port,
		"path", w.path,
		"authRequired", w.authToken != "")

	return nil
}

// StartListening starts the HTTP server to receive webhook events
func (w *WebhookAdapter) StartListening(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.isListening {
		return fmt.Errorf("webhook adapter is already listening")
	}

	if w.eventCallback == nil {
		return fmt.Errorf("no event callback registered")
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc(w.path, w.handleWebhook)
	mux.HandleFunc("/health", w.handleHealth)

	w.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", w.port),
		Handler: mux,
	}

	w.isListening = true
	w.status.Status = types.StatusActive
	w.status.UpdatedAt = time.Now()

	// Start server in goroutine
	go func() {
		w.logger.Info("Starting webhook server",
			"name", w.name,
			"addr", w.server.Addr,
			"path", w.path)

		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			w.mutex.Lock()
			w.status.Status = types.StatusError
			w.status.ErrorMessage = fmt.Sprintf("server error: %v", err)
			w.status.UpdatedAt = time.Now()
			w.isListening = false
			w.mutex.Unlock()

			w.logger.Error("Webhook server error",
				"name", w.name,
				"error", err)
		}
	}()

	w.logger.Info("Webhook adapter started listening",
		"name", w.name,
		"port", w.port,
		"path", w.path)

	return nil
}

// StopListening stops the HTTP server
func (w *WebhookAdapter) StopListening(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.isListening || w.server == nil {
		return nil // Already stopped
	}

	// Shutdown server with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := w.server.Shutdown(shutdownCtx); err != nil {
		w.logger.Error("Error shutting down webhook server",
			"name", w.name,
			"error", err)
		return err
	}

	w.isListening = false
	w.server = nil
	w.status.UpdatedAt = time.Now()

	w.logger.Info("Webhook adapter stopped listening", "name", w.name)
	return nil
}

// UnlockDoor is not supported by webhook adapter
func (w *WebhookAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	return fmt.Errorf("door unlock not supported by webhook adapter")
}

// GetStatus returns the current adapter status
func (w *WebhookAdapter) GetStatus() types.AdapterStatus {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.status
}

// OnEvent registers a callback for hardware events
func (w *WebhookAdapter) OnEvent(callback types.EventCallback) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.eventCallback = callback
}

// IsHealthy returns true if the webhook server is running
func (w *WebhookAdapter) IsHealthy() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.status.Status == types.StatusActive && w.isListening
}

// handleWebhook processes incoming webhook requests
func (w *WebhookAdapter) handleWebhook(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication if required
	if w.authToken != "" {
		authHeader := req.Header.Get("Authorization")
		expectedAuth := "Bearer " + w.authToken
		if authHeader != expectedAuth {
			w.logger.Warn("Webhook authentication failed",
				"name", w.name,
				"remoteAddr", req.RemoteAddr)
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse webhook payload
	var webhookEvent WebhookEvent
	if err := json.NewDecoder(req.Body).Decode(&webhookEvent); err != nil {
		w.logger.Error("Failed to parse webhook payload",
			"name", w.name,
			"error", err,
			"remoteAddr", req.RemoteAddr)
		http.Error(rw, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if webhookEvent.ExternalUserID == "" {
		http.Error(rw, "externalUserId is required", http.StatusBadRequest)
		return
	}

	if !types.IsValidEventType(webhookEvent.EventType) {
		http.Error(rw, "invalid eventType", http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	timestamp := time.Now()
	if webhookEvent.Timestamp != nil {
		timestamp = *webhookEvent.Timestamp
	}

	// Create raw hardware event
	event := types.RawHardwareEvent{
		ExternalUserID: webhookEvent.ExternalUserID,
		Timestamp:      timestamp,
		EventType:      webhookEvent.EventType,
		RawData:        webhookEvent.RawData,
	}

	// Add webhook metadata
	if event.RawData == nil {
		event.RawData = make(map[string]interface{})
	}
	event.RawData["webhook"] = true
	event.RawData["remoteAddr"] = req.RemoteAddr
	event.RawData["userAgent"] = req.Header.Get("User-Agent")
	event.RawData["receivedAt"] = time.Now()

	// Update status with last event time
	w.mutex.Lock()
	w.status.LastEvent = event.Timestamp
	w.status.UpdatedAt = time.Now()
	callback := w.eventCallback
	w.mutex.Unlock()

	w.logger.Info("Received webhook event",
		"name", w.name,
		"externalUserId", event.ExternalUserID,
		"eventType", event.EventType,
		"remoteAddr", req.RemoteAddr)

	// Send event to callback
	if callback != nil {
		callback(event)
	}

	// Send success response
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"success":   true,
		"timestamp": time.Now(),
		"eventId":   fmt.Sprintf("webhook_%d", time.Now().UnixNano()),
	})
}

// handleHealth provides a health check endpoint
func (w *WebhookAdapter) handleHealth(rw http.ResponseWriter, req *http.Request) {
	status := w.GetStatus()
	
	rw.Header().Set("Content-Type", "application/json")
	if status.Status == types.StatusActive {
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(rw).Encode(map[string]interface{}{
		"name":         status.Name,
		"status":       status.Status,
		"lastEvent":    status.LastEvent,
		"errorMessage": status.ErrorMessage,
		"updatedAt":    status.UpdatedAt,
	})
}