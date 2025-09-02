package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	"github.com/sirupsen/logrus"
)

// mockAuthManager implements auth.AuthManager interface for testing
type mockAuthManager struct {
	deviceID      string
	deviceKey     string
	authenticated bool
	signError     error
}

func newMockAuthManager(deviceID, deviceKey string) *mockAuthManager {
	return &mockAuthManager{
		deviceID:      deviceID,
		deviceKey:     deviceKey,
		authenticated: deviceID != "" && deviceKey != "",
	}
}

func (m *mockAuthManager) IsAuthenticated() bool {
	return m.authenticated
}

func (m *mockAuthManager) GetDeviceID() string {
	return m.deviceID
}

func (m *mockAuthManager) SignRequest(body []byte) (string, int64, error) {
	if m.signError != nil {
		return "", 0, m.signError
	}
	timestamp := time.Now().Unix()
	signature := fmt.Sprintf("mock_signature_%d", timestamp)
	return signature, timestamp, nil
}

func (m *mockAuthManager) SetCredentials(deviceID, deviceKey string) error {
	m.deviceID = deviceID
	m.deviceKey = deviceKey
	m.authenticated = true
	return nil
}

func TestNewHTTPClient(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	cfg := &config.Config{
		ServerURL: "https://api.example.com",
	}

	tests := []struct {
		name        string
		cfg         *config.Config
		authManager *mockAuthManager
		logger      *logrus.Logger
		wantErr     bool
	}{
		{
			name:        "valid configuration",
			cfg:         cfg,
			authManager: authManager,
			logger:      logger,
			wantErr:     false,
		},
		{
			name:        "nil config",
			cfg:         nil,
			authManager: authManager,
			logger:      logger,
			wantErr:     true,
		},
		{
			name:        "nil auth manager",
			cfg:         cfg,
			authManager: nil,
			logger:      logger,
			wantErr:     true,
		},
		{
			name:        "nil logger",
			cfg:         cfg,
			authManager: authManager,
			logger:      nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHTTPClient(tt.cfg, tt.authManager, tt.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewHTTPClient() returned nil client")
			}
		})
	}
}

func TestHTTPClient_Do(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")

	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		request        *Request
		wantErr        bool
		wantStatusCode int
	}{
		{
			name: "successful GET request",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			},
			request: &Request{
				Method:      http.MethodGet,
				Path:        "/test",
				RequireAuth: false,
			},
			wantErr:        false,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "successful POST request with body",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"id": "123"}`))
			},
			request: &Request{
				Method: http.MethodPost,
				Path:   "/test",
				Body:   map[string]string{"test": "data"},
				RequireAuth: false,
			},
			wantErr:        false,
			wantStatusCode: http.StatusCreated,
		},
		{
			name: "authenticated request",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Check authentication headers
				if r.Header.Get("X-Device-ID") != "test-device" {
					t.Errorf("Expected X-Device-ID test-device, got %s", r.Header.Get("X-Device-ID"))
				}
				if r.Header.Get("X-Signature") == "" {
					t.Error("Expected X-Signature header")
				}
				if r.Header.Get("X-Timestamp") == "" {
					t.Error("Expected X-Timestamp header")
				}
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"authenticated": true}`))
			},
			request: &Request{
				Method:      http.MethodPost,
				Path:        "/authenticated",
				Body:        map[string]string{"data": "test"},
				RequireAuth: true,
			},
			wantErr:        false,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "server error with retry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "internal server error"}`))
			},
			request: &Request{
				Method:      http.MethodGet,
				Path:        "/error",
				RequireAuth: false,
			},
			wantErr:        true,
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "client error no retry",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "bad request"}`))
			},
			request: &Request{
				Method:      http.MethodPost,
				Path:        "/bad",
				RequireAuth: false,
			},
			wantErr:        true,
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			// Create client with test server URL
			cfg := &config.Config{
				ServerURL: server.URL,
			}
			client, err := NewHTTPClient(cfg, authManager, logger)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Override retry settings for faster tests
			client.maxRetries = 1
			client.baseDelay = 10 * time.Millisecond

			ctx := context.Background()
			resp, err := client.Do(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPClient.Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if resp != nil && resp.StatusCode != tt.wantStatusCode {
				t.Errorf("HTTPClient.Do() statusCode = %v, want %v", resp.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestHTTPClient_AuthenticationRequired(t *testing.T) {
	logger := logging.Initialize("debug")
	
	// Create unauthenticated auth manager
	authManager := newMockAuthManager("", "")
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		Method:      http.MethodGet,
		Path:        "/test",
		RequireAuth: true,
	}

	ctx := context.Background()
	_, err = client.Do(ctx, req)

	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}
	if !strings.Contains(err.Error(), "authentication required") {
		t.Errorf("Expected authentication error, got: %v", err)
	}
}

func TestHTTPClient_SigningError(t *testing.T) {
	logger := logging.Initialize("debug")
	
	// Create auth manager that fails to sign
	authManager := newMockAuthManager("test-device", "test-key")
	authManager.signError = fmt.Errorf("signing failed")
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if err == nil {
		t.Error("Expected error for signing failure")
	}
	if !strings.Contains(err.Error(), "failed to sign request") {
		t.Errorf("Expected signing error, got: %v", err)
	}
}

func TestHTTPClient_ContextCancellation(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
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
		Method:      http.MethodGet,
		Path:        "/test",
		RequireAuth: false,
	}

	// Create context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = client.Do(ctx, req)

	if err == nil {
		t.Error("Expected error for context cancellation")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline error, got: %v", err)
	}
}

func TestHTTPClient_RetryLogic(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		ServerURL: server.URL,
	}
	client, err := NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set fast retry for testing
	client.maxRetries = 5
	client.baseDelay = 1 * time.Millisecond

	req := &Request{
		Method:      http.MethodGet,
		Path:        "/test",
		RequireAuth: false,
	}

	ctx := context.Background()
	resp, err := client.Do(ctx, req)

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

func TestHTTPClient_CustomHeaders(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Expected X-Custom-Header custom-value, got %s", r.Header.Get("X-Custom-Header"))
		}
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
		Method:      http.MethodGet,
		Path:        "/test",
		RequireAuth: false,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	ctx := context.Background()
	_, err = client.Do(ctx, req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCalculateDelay(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	cfg := &config.Config{
		ServerURL: "https://api.example.com",
	}
	
	client, err := NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client.baseDelay = 1 * time.Second
	client.maxDelay = 30 * time.Second
	client.jitterFactor = 0.1

	tests := []struct {
		attempt int
		minDelay time.Duration
		maxDelay time.Duration
	}{
		{1, 900 * time.Millisecond, 1100 * time.Millisecond}, // 1s ± 10%
		{2, 1800 * time.Millisecond, 2200 * time.Millisecond}, // 2s ± 10%
		{3, 3600 * time.Millisecond, 4400 * time.Millisecond}, // 4s ± 10%
		{10, 27 * time.Second, 33 * time.Second}, // Capped at 30s ± 10%
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := client.calculateDelay(tt.attempt)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("calculateDelay(%d) = %v, want between %v and %v", 
					tt.attempt, delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

func TestShouldRetry(t *testing.T) {
	logger := logging.Initialize("debug")
	authManager := newMockAuthManager("test-device", "test-key")
	cfg := &config.Config{
		ServerURL: "https://api.example.com",
	}
	
	client, err := NewHTTPClient(cfg, authManager, logger)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name       string
		err        error
		resp       *Response
		shouldRetry bool
	}{
		{
			name:        "500 server error",
			resp:        &Response{StatusCode: 500},
			shouldRetry: true,
		},
		{
			name:        "502 bad gateway",
			resp:        &Response{StatusCode: 502},
			shouldRetry: true,
		},
		{
			name:        "503 service unavailable",
			resp:        &Response{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "429 too many requests",
			resp:        &Response{StatusCode: 429},
			shouldRetry: true,
		},
		{
			name:        "400 bad request",
			resp:        &Response{StatusCode: 400},
			shouldRetry: false,
		},
		{
			name:        "401 unauthorized",
			resp:        &Response{StatusCode: 401},
			shouldRetry: false,
		},
		{
			name:        "403 forbidden",
			resp:        &Response{StatusCode: 403},
			shouldRetry: false,
		},
		{
			name:        "200 success",
			resp:        &Response{StatusCode: 200},
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.shouldRetry(tt.err, tt.resp)
			if result != tt.shouldRetry {
				t.Errorf("shouldRetry() = %v, want %v", result, tt.shouldRetry)
			}
		})
	}
}