package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"gym-door-bridge/internal/types"
)

func TestWebhookAdapter_Initialize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	tests := []struct {
		name        string
		config      AdapterConfig
		expectError bool
	}{
		{
			name: "valid configuration",
			config: AdapterConfig{
				Name:    "webhook",
				Enabled: true,
				Settings: map[string]interface{}{
					"port":      8080.0,
					"path":      "/webhook",
					"authToken": "test-token",
				},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: AdapterConfig{
				Name:    "webhook",
				Enabled: true,
				Settings: map[string]interface{}{
					"port": -1.0,
				},
			},
			expectError: true,
		},
		{
			name: "default values",
			config: AdapterConfig{
				Name:     "webhook",
				Enabled:  true,
				Settings: map[string]interface{}{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adapter.Initialize(context.Background(), tt.config)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestWebhookAdapter_StartStopListening(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	config := AdapterConfig{
		Name:    "webhook",
		Enabled: true,
		Settings: map[string]interface{}{
			"port": 8081.0, // Use different port to avoid conflicts
			"path": "/test-webhook",
		},
	}

	// Initialize adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	// Test starting without callback
	err = adapter.StartListening(context.Background())
	if err == nil {
		t.Error("expected error when starting without callback")
	}

	// Register callback
	var receivedEvent *types.RawHardwareEvent
	adapter.OnEvent(func(event types.RawHardwareEvent) {
		receivedEvent = &event
	})

	// Start listening
	err = adapter.StartListening(context.Background())
	if err != nil {
		t.Fatalf("failed to start listening: %v", err)
	}

	// Test double start
	err = adapter.StartListening(context.Background())
	if err == nil {
		t.Error("expected error when starting already listening adapter")
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test webhook endpoint
	webhookEvent := WebhookEvent{
		ExternalUserID: "test-user-123",
		EventType:      types.EventTypeEntry,
		RawData: map[string]interface{}{
			"test": true,
		},
	}

	payload, _ := json.Marshal(webhookEvent)
	resp, err := http.Post("http://localhost:8081/test-webhook", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Give time for event processing
	time.Sleep(100 * time.Millisecond)

	// Verify event was received
	if receivedEvent == nil {
		t.Error("expected to receive event but got none")
	} else {
		if receivedEvent.ExternalUserID != "test-user-123" {
			t.Errorf("expected externalUserId 'test-user-123', got '%s'", receivedEvent.ExternalUserID)
		}
		if receivedEvent.EventType != types.EventTypeEntry {
			t.Errorf("expected eventType 'entry', got '%s'", receivedEvent.EventType)
		}
	}

	// Stop listening
	err = adapter.StopListening(context.Background())
	if err != nil {
		t.Fatalf("failed to stop listening: %v", err)
	}

	// Test double stop
	err = adapter.StopListening(context.Background())
	if err != nil {
		t.Errorf("unexpected error on double stop: %v", err)
	}
}

func TestWebhookAdapter_Authentication(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	config := AdapterConfig{
		Name:    "webhook",
		Enabled: true,
		Settings: map[string]interface{}{
			"port":      8082.0,
			"path":      "/secure-webhook",
			"authToken": "secret-token",
		},
	}

	// Initialize and start adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	adapter.OnEvent(func(event types.RawHardwareEvent) {})

	err = adapter.StartListening(context.Background())
	if err != nil {
		t.Fatalf("failed to start listening: %v", err)
	}
	defer adapter.StopListening(context.Background())

	time.Sleep(100 * time.Millisecond)

	webhookEvent := WebhookEvent{
		ExternalUserID: "test-user",
		EventType:      types.EventTypeEntry,
	}
	payload, _ := json.Marshal(webhookEvent)

	// Test without authentication
	resp, err := http.Post("http://localhost:8082/secure-webhook", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	// Test with correct authentication
	client := &http.Client{}
	req, _ := http.NewRequest("POST", "http://localhost:8082/secure-webhook", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("failed to send authenticated webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestWebhookAdapter_InvalidPayloads(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	config := AdapterConfig{
		Name:    "webhook",
		Enabled: true,
		Settings: map[string]interface{}{
			"port": 8083.0,
			"path": "/webhook",
		},
	}

	// Initialize and start adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	adapter.OnEvent(func(event types.RawHardwareEvent) {})

	err = adapter.StartListening(context.Background())
	if err != nil {
		t.Fatalf("failed to start listening: %v", err)
	}
	defer adapter.StopListening(context.Background())

	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		payload        string
		expectedStatus int
	}{
		{
			name:           "invalid JSON",
			payload:        `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing externalUserId",
			payload:        `{"eventType": "entry"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid eventType",
			payload:        `{"externalUserId": "test", "eventType": "invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid payload",
			payload:        `{"externalUserId": "test", "eventType": "entry"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post("http://localhost:8083/webhook", "application/json", bytes.NewBufferString(tt.payload))
			if err != nil {
				t.Fatalf("failed to send webhook: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestWebhookAdapter_HealthEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	config := AdapterConfig{
		Name:    "webhook",
		Enabled: true,
		Settings: map[string]interface{}{
			"port": 8084.0,
		},
	}

	// Initialize and start adapter
	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	adapter.OnEvent(func(event types.RawHardwareEvent) {})

	err = adapter.StartListening(context.Background())
	if err != nil {
		t.Fatalf("failed to start listening: %v", err)
	}
	defer adapter.StopListening(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get("http://localhost:8084/health")
	if err != nil {
		t.Fatalf("failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	if err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if healthResponse["name"] != "webhook" {
		t.Errorf("expected name 'webhook', got '%v'", healthResponse["name"])
	}

	if healthResponse["status"] != "active" {
		t.Errorf("expected status 'active', got '%v'", healthResponse["status"])
	}
}

func TestWebhookAdapter_UnlockDoor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	// UnlockDoor should not be supported
	err := adapter.UnlockDoor(context.Background(), 3000)
	if err == nil {
		t.Error("expected error for unsupported UnlockDoor operation")
	}
}

func TestWebhookAdapter_Status(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewWebhookAdapter(logger)

	// Test initial status
	status := adapter.GetStatus()
	if status.Name != "webhook" {
		t.Errorf("expected name 'webhook', got '%s'", status.Name)
	}
	if status.Status != "disabled" {
		t.Errorf("expected status 'disabled', got '%s'", status.Status)
	}
	if adapter.IsHealthy() {
		t.Error("expected adapter to not be healthy initially")
	}

	// Initialize adapter
	config := AdapterConfig{
		Name:     "webhook",
		Enabled:  true,
		Settings: map[string]interface{}{},
	}

	err := adapter.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("failed to initialize adapter: %v", err)
	}

	// Test status after initialization
	status = adapter.GetStatus()
	if status.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", status.Status)
	}
}