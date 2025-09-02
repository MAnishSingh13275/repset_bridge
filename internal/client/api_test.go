package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
)

func TestHTTPClient_PairDevice(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("", "") // Not authenticated initially

	tests := []struct {
		name           string
		pairCode       string
		deviceInfo     *DeviceInfo
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantDeviceID   string
	}{
		{
			name:     "successful pairing",
			pairCode: "ABC123",
			deviceInfo: &DeviceInfo{
				Hostname: "test-host",
				Platform: "windows",
				Version:  "1.0.0",
				Tier:     "normal",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/devices/pair" {
					t.Errorf("Expected /api/v1/devices/pair, got %s", r.URL.Path)
				}

				// Verify request body
				var req PairRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}
				if req.PairCode != "ABC123" {
					t.Errorf("Expected pair code ABC123, got %s", req.PairCode)
				}

				// Return successful response
				resp := PairResponse{
					DeviceID:  "dev_abc123",
					DeviceKey: "secret_key",
					Config: &DeviceConfig{
						HeartbeatInterval: 60,
						QueueMaxSize:      10000,
						UnlockDuration:    3000,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(resp)
			},
			wantErr:      false,
			wantDeviceID: "dev_abc123",
		},
		{
			name:     "invalid pair code",
			pairCode: "INVALID",
			deviceInfo: &DeviceInfo{
				Hostname: "test-host",
				Platform: "windows",
				Version:  "1.0.0",
				Tier:     "normal",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "invalid pair code"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			resp, err := client.PairDevice(ctx, tt.pairCode, tt.deviceInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("PairDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if resp == nil {
					t.Error("Expected response, got nil")
					return
				}
				if resp.DeviceID != tt.wantDeviceID {
					t.Errorf("Expected device ID %s, got %s", tt.wantDeviceID, resp.DeviceID)
				}
			}
		})
	}
}

func TestHTTPClient_SubmitCheckinEvents(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		events         []CheckinEvent
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "successful submission",
			events: []CheckinEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_456",
					Timestamp:      "2024-01-01T10:00:00Z",
					EventType:      "entry",
					IsSimulated:    false,
					DeviceID:       "test-device",
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/checkin" {
					t.Errorf("Expected /api/v1/checkin, got %s", r.URL.Path)
				}

				// Verify authentication headers
				if r.Header.Get("X-Device-ID") != "test-device" {
					t.Errorf("Expected X-Device-ID test-device, got %s", r.Header.Get("X-Device-ID"))
				}
				if r.Header.Get("X-Signature") == "" {
					t.Error("Expected X-Signature header")
				}

				// Verify request body
				var req CheckinRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}
				if len(req.Events) != 1 {
					t.Errorf("Expected 1 event, got %d", len(req.Events))
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:   "empty events",
			events: []CheckinEvent{},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				t.Error("Should not make request for empty events")
			},
			wantErr: false, // Should not error, just not make request
		},
		{
			name: "server error",
			events: []CheckinEvent{
				{
					EventID:        "evt_123",
					ExternalUserID: "user_456",
					Timestamp:      "2024-01-01T10:00:00Z",
					EventType:      "entry",
					IsSimulated:    false,
					DeviceID:       "test-device",
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.SubmitCheckinEvents(ctx, tt.events)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitCheckinEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_SendHeartbeat(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		heartbeat      *HeartbeatRequest
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "successful heartbeat",
			heartbeat: &HeartbeatRequest{
				Status:        "healthy",
				Tier:          "normal",
				QueueDepth:    0,
				LastEventTime: "2024-01-01T10:00:00Z",
				SystemInfo: &SystemInfo{
					CPUUsage:    15.2,
					MemoryUsage: 45.8,
					DiskSpace:   85.1,
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/devices/heartbeat" {
					t.Errorf("Expected /api/v1/devices/heartbeat, got %s", r.URL.Path)
				}

				// Verify authentication headers
				if r.Header.Get("X-Device-ID") != "test-device" {
					t.Errorf("Expected X-Device-ID test-device, got %s", r.Header.Get("X-Device-ID"))
				}

				// Verify request body
				var req HeartbeatRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("Failed to decode request: %v", err)
				}
				if req.Status != "healthy" {
					t.Errorf("Expected status healthy, got %s", req.Status)
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "server error",
			heartbeat: &HeartbeatRequest{
				Status: "healthy",
				Tier:   "normal",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.SendHeartbeat(ctx, tt.heartbeat)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendHeartbeat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_OpenDoor(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "successful door open",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/open-door" {
					t.Errorf("Expected /open-door, got %s", r.URL.Path)
				}

				// Verify authentication headers
				if r.Header.Get("X-Device-ID") != "test-device" {
					t.Errorf("Expected X-Device-ID test-device, got %s", r.Header.Get("X-Device-ID"))
				}

				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.OpenDoor(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("OpenDoor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_GetDeviceConfig(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantConfig     *DeviceConfig
	}{
		{
			name: "successful config retrieval",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/devices/config" {
					t.Errorf("Expected /api/v1/devices/config, got %s", r.URL.Path)
				}

				config := DeviceConfig{
					HeartbeatInterval: 120,
					QueueMaxSize:      20000,
					UnlockDuration:    5000,
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(config)
			},
			wantErr: false,
			wantConfig: &DeviceConfig{
				HeartbeatInterval: 120,
				QueueMaxSize:      20000,
				UnlockDuration:    5000,
			},
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			config, err := client.GetDeviceConfig(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if config == nil {
					t.Error("Expected config, got nil")
					return
				}
				if config.HeartbeatInterval != tt.wantConfig.HeartbeatInterval {
					t.Errorf("Expected heartbeat interval %d, got %d", 
						tt.wantConfig.HeartbeatInterval, config.HeartbeatInterval)
				}
			}
		})
	}
}

func TestHTTPClient_CheckConnectivity(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "successful connectivity check",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/api/v1/health" {
					t.Errorf("Expected /api/v1/health, got %s", r.URL.Path)
				}

				// Should not require authentication
				if r.Header.Get("X-Device-ID") != "" {
					t.Error("Connectivity check should not require authentication")
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			},
			wantErr: false,
		},
		{
			name: "server unavailable",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.CheckConnectivity(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("CheckConnectivity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHTTPClient_ClockSkewTolerance(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that timestamp is within reasonable range
		timestamp := r.Header.Get("X-Timestamp")
		if timestamp == "" {
			t.Error("Expected X-Timestamp header")
		}
		
		// Just verify the header is present - actual validation would be done by server
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		ServerURL: server.URL,
	}
	client, err := NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	req := &Request{
		Method:      http.MethodPost,
		Path:        "/test",
		Body:        map[string]string{"test": "data"},
		RequireAuth: true,
	}

	ctx := context.Background()
	_, err = client.Do(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "connection refused",
			err:  &mockNetError{msg: "connection refused", temp: true},
			want: true,
		},
		{
			name: "timeout error",
			err:  &mockNetError{msg: "timeout", timeout: true},
			want: true,
		},
		{
			name: "temporary error",
			err:  &mockNetError{msg: "temporary failure", temp: true},
			want: true,
		},
		{
			name: "non-network error",
			err:  fmt.Errorf("not a network error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			if result != tt.want {
				t.Errorf("isNetworkError() = %v, want %v", result, tt.want)
			}
		})
	}
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	msg     string
	timeout bool
	temp    bool
}

func (e *mockNetError) Error() string   { return e.msg }
func (e *mockNetError) Timeout() bool   { return e.timeout }
func (e *mockNetError) Temporary() bool { return e.temp }