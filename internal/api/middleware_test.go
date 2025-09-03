package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestServer creates a server with mock dependencies for testing
func createTestServer(cfg *config.Config, serverCfg *ServerConfig) *Server {
	mockAdapterRegistry := &MockAdapterRegistry{}
	mockDoorController := &MockDoorController{}
	
	// Set up basic mocks for door control endpoints
	mockAdapterRegistry.On("GetActiveAdapters").Return([]adapters.HardwareAdapter{}).Maybe()
	mockAdapterRegistry.On("GetAllAdapters").Return([]adapters.HardwareAdapter{}).Maybe()
	mockDoorController.On("GetStats").Return(map[string]interface{}{
		"unlockCount": int64(0),
	}).Maybe()
	
	return NewServer(cfg, serverCfg, mockAdapterRegistry, mockDoorController, nil, nil, nil, nil, "test-version", "test-device-id")
}

func TestLoggingMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	server := createTestServer(cfg, serverCfg)
	
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})
	
	// Wrap with logging middleware
	wrapped := server.loggingMiddleware(testHandler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	
	wrapped.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestRecoveryMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	serverCfg := DefaultServerConfig()
	server := createTestServer(cfg, serverCfg)
	
	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	
	// Wrap with recovery middleware
	wrapped := server.recoveryMiddleware(panicHandler)
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	// This should not panic
	wrapped.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Internal Server Error")
}

func TestCORSMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	t.Run("CORS disabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.Enabled = false
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("Wildcard origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"*"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})
	
	t.Run("Specific allowed origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"https://example.com", "https://app.example.com"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("Disallowed origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"https://example.com"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("OPTIONS preflight allowed origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"https://example.com"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})
	
	t.Run("OPTIONS preflight disallowed origin", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"https://example.com"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
	
	t.Run("Credentials allowed", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.CORS.AllowedOrigins = []string{"https://example.com"}
		cfg.APIServer.CORS.AllowCredentials = true
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.corsMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	})
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	t.Run("All security headers enabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.securityHeadersMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src 'self'")
		
		// HSTS header should not be set for HTTP requests
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
	})
	
	t.Run("HTTPS request with HSTS", func(t *testing.T) {
		cfg := config.DefaultConfig()
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.securityHeadersMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{} // Simulate HTTPS
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		// HSTS header should be set for HTTPS requests
		assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	})
	
	t.Run("X-Forwarded-Proto HTTPS", func(t *testing.T) {
		cfg := config.DefaultConfig()
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.securityHeadersMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		// HSTS header should be set for forwarded HTTPS requests
		assert.Equal(t, "max-age=31536000; includeSubDomains", w.Header().Get("Strict-Transport-Security"))
	})
	
	t.Run("Custom security configuration", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Security.FrameOptions = "SAMEORIGIN"
		cfg.APIServer.Security.HSTSMaxAge = 86400
		cfg.APIServer.Security.HSTSIncludeSubdomains = false
		cfg.APIServer.Security.CSPDirective = "default-src 'none'"
		cfg.APIServer.Security.ReferrerPolicy = "no-referrer"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.securityHeadersMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{}
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, "SAMEORIGIN", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "max-age=86400", w.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "default-src 'none'", w.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	})
	
	t.Run("Disabled security features", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Security.ContentTypeOptions = false
		cfg.APIServer.Security.XSSProtection = false
		cfg.APIServer.Security.CSPEnabled = false
		cfg.APIServer.Security.HSTSEnabled = false
		cfg.APIServer.Security.FrameOptions = ""
		cfg.APIServer.Security.ReferrerPolicy = ""
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.securityHeadersMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{}
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Empty(t, w.Header().Get("X-Content-Type-Options"))
		assert.Empty(t, w.Header().Get("X-XSS-Protection"))
		assert.Empty(t, w.Header().Get("Content-Security-Policy"))
		assert.Empty(t, w.Header().Get("Strict-Transport-Security"))
		assert.Empty(t, w.Header().Get("X-Frame-Options"))
		assert.Empty(t, w.Header().Get("Referrer-Policy"))
	})
}

func TestAuthenticationMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})
	
	t.Run("Authentication disabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = false
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("No authentication headers", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("Valid API key in X-API-Key header", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key", "another-key"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "valid-key")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("Valid API key in Authorization header", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-key")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("Invalid API key", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "invalid-key")
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("Valid JWT token", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.JWTSecret = "test-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		// Create a valid JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		tokenString, err := token.SignedString([]byte("test-secret"))
		require.NoError(t, err)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("Expired JWT token", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.JWTSecret = "test-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		// Create an expired JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		})
		tokenString, err := token.SignedString([]byte("test-secret"))
		require.NoError(t, err)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("Invalid JWT signature", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.JWTSecret = "test-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		// Create a JWT token with wrong secret
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		require.NoError(t, err)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("Valid HMAC signature", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.HMACSecret = "test-hmac-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		// Create HMAC signature
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		message := "GET/test" + timestamp
		mac := hmac.New(sha256.New, []byte("test-hmac-secret"))
		mac.Write([]byte(message))
		signature := hex.EncodeToString(mac.Sum(nil))
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Signature", signature)
		req.Header.Set("X-Timestamp", timestamp)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("Invalid HMAC signature", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.HMACSecret = "test-hmac-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Signature", "invalid-signature")
		req.Header.Set("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("Expired HMAC timestamp", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.HMACSecret = "test-hmac-secret"
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		// Create signature with old timestamp (10 minutes ago)
		oldTimestamp := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
		message := "GET/test" + oldTimestamp
		mac := hmac.New(sha256.New, []byte("test-hmac-secret"))
		mac.Write([]byte(message))
		signature := hex.EncodeToString(mac.Sum(nil))
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Signature", signature)
		req.Header.Set("X-Timestamp", oldTimestamp)
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Authentication required")
	})
	
	t.Run("IP allowlist - allowed IP", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key"}
		cfg.APIServer.Auth.AllowedIPs = []string{"127.0.0.1", "192.168.1.0/24"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "valid-key")
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})
	
	t.Run("IP allowlist - blocked IP", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.Auth.Enabled = true
		cfg.APIServer.Auth.APIKeys = []string{"valid-key"}
		cfg.APIServer.Auth.AllowedIPs = []string{"127.0.0.1"}
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.authenticationMiddleware(testHandler)
		
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "valid-key")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Access denied from this IP address")
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})
	
	t.Run("Rate limiting disabled", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.RateLimit.Enabled = false
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.rateLimitMiddleware(testHandler)
		
		// Make multiple requests
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			
			wrapped.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
		}
	})
	
	t.Run("Rate limiting enabled - within limits", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.RateLimit.Enabled = true
		cfg.APIServer.RateLimit.RequestsPerMin = 10
		cfg.APIServer.RateLimit.WindowSize = 60
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.rateLimitMiddleware(testHandler)
		
		// Make requests within limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			
			wrapped.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
			assert.Equal(t, strconv.Itoa(10-(i+1)), w.Header().Get("X-RateLimit-Remaining"))
		}
	})
	
	t.Run("Rate limiting enabled - exceeds limits", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.RateLimit.Enabled = true
		cfg.APIServer.RateLimit.RequestsPerMin = 3
		cfg.APIServer.RateLimit.WindowSize = 60
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.rateLimitMiddleware(testHandler)
		
		// Make requests up to limit
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			
			wrapped.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
		}
		
		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()
		
		wrapped.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "Rate limit exceeded")
		assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})
	
	t.Run("Different IPs have separate limits", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.RateLimit.Enabled = true
		cfg.APIServer.RateLimit.RequestsPerMin = 2
		cfg.APIServer.RateLimit.WindowSize = 60
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.rateLimitMiddleware(testHandler)
		
		// Exhaust limit for first IP
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			w := httptest.NewRecorder()
			
			wrapped.ServeHTTP(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}
		
		// First IP should be rate limited
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "127.0.0.1:12345"
		w1 := httptest.NewRecorder()
		wrapped.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusTooManyRequests, w1.Code)
		
		// Second IP should still work
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.100:12345"
		w2 := httptest.NewRecorder()
		wrapped.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})
	
	t.Run("X-Forwarded-For header", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.APIServer.RateLimit.Enabled = true
		cfg.APIServer.RateLimit.RequestsPerMin = 1
		cfg.APIServer.RateLimit.WindowSize = 60
		serverCfg := DefaultServerConfig()
		server := createTestServer(cfg, serverCfg)
		
		wrapped := server.rateLimitMiddleware(testHandler)
		
		// First request with X-Forwarded-For
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "10.0.0.1:12345"
		req1.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
		w1 := httptest.NewRecorder()
		wrapped.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		
		// Second request from same forwarded IP should be rate limited
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "10.0.0.2:12345" // Different proxy
		req2.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.2")
		w2 := httptest.NewRecorder()
		wrapped.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("Basic rate limiting", func(t *testing.T) {
		rl := newRateLimiter(5, 2, time.Minute, 5*time.Minute)
		
		// First 5 requests should be allowed
		for i := 0; i < 5; i++ {
			allowed, remaining, _ := rl.isAllowed("test-key")
			assert.True(t, allowed)
			assert.Equal(t, 5-(i+1), remaining)
		}
		
		// 6th request should be denied
		allowed, remaining, resetTime := rl.isAllowed("test-key")
		assert.False(t, allowed)
		assert.Equal(t, 0, remaining) // Should be 0, not -1
		assert.False(t, resetTime.IsZero())
	})
	
	t.Run("Sliding window", func(t *testing.T) {
		rl := newRateLimiter(2, 1, time.Second, 5*time.Second)
		
		// Make 2 requests
		allowed1, _, _ := rl.isAllowed("test-key")
		assert.True(t, allowed1)
		
		allowed2, _, _ := rl.isAllowed("test-key")
		assert.True(t, allowed2)
		
		// 3rd request should be denied
		allowed3, _, _ := rl.isAllowed("test-key")
		assert.False(t, allowed3)
		
		// Wait for window to slide
		time.Sleep(1100 * time.Millisecond)
		
		// Should be allowed again
		allowed4, _, _ := rl.isAllowed("test-key")
		assert.True(t, allowed4)
	})
	
	t.Run("Cleanup old entries", func(t *testing.T) {
		rl := newRateLimiter(10, 5, 100*time.Millisecond, 100*time.Millisecond)
		
		// Add entries for multiple keys
		rl.isAllowed("key1")
		rl.isAllowed("key2")
		rl.isAllowed("key3")
		
		assert.Equal(t, 3, len(rl.entries))
		
		// Wait for cleanup
		time.Sleep(300 * time.Millisecond)
		
		// Trigger cleanup by making a new request
		rl.isAllowed("key4")
		
		// Old entries should be cleaned up
		assert.True(t, len(rl.entries) <= 1) // Only key4 should remain
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("getClientIP", func(t *testing.T) {
		// Test X-Forwarded-For header
		req1 := httptest.NewRequest("GET", "/test", nil)
		req1.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
		req1.RemoteAddr = "127.0.0.1:12345"
		assert.Equal(t, "203.0.113.1", getClientIP(req1))
		
		// Test X-Real-IP header
		req2 := httptest.NewRequest("GET", "/test", nil)
		req2.Header.Set("X-Real-IP", "203.0.113.2")
		req2.RemoteAddr = "127.0.0.1:12345"
		assert.Equal(t, "203.0.113.2", getClientIP(req2))
		
		// Test RemoteAddr fallback
		req3 := httptest.NewRequest("GET", "/test", nil)
		req3.RemoteAddr = "127.0.0.1:12345"
		assert.Equal(t, "127.0.0.1", getClientIP(req3))
		
		// Test RemoteAddr without port
		req4 := httptest.NewRequest("GET", "/test", nil)
		req4.RemoteAddr = "127.0.0.1"
		assert.Equal(t, "127.0.0.1", getClientIP(req4))
	})
	
	t.Run("isIPAllowed", func(t *testing.T) {
		// Empty allowlist should allow all
		assert.True(t, isIPAllowed("127.0.0.1", []string{}))
		
		// Direct IP match
		allowedIPs := []string{"127.0.0.1", "192.168.1.100"}
		assert.True(t, isIPAllowed("127.0.0.1", allowedIPs))
		assert.True(t, isIPAllowed("192.168.1.100", allowedIPs))
		assert.False(t, isIPAllowed("10.0.0.1", allowedIPs))
		
		// CIDR range match
		cidrIPs := []string{"192.168.1.0/24", "10.0.0.0/8"}
		assert.True(t, isIPAllowed("192.168.1.50", cidrIPs))
		assert.True(t, isIPAllowed("10.5.10.20", cidrIPs))
		assert.False(t, isIPAllowed("172.16.0.1", cidrIPs))
		
		// Invalid IP
		assert.False(t, isIPAllowed("invalid-ip", allowedIPs))
		
		// Invalid CIDR in allowlist
		invalidCIDR := []string{"192.168.1.0/invalid"}
		assert.False(t, isIPAllowed("192.168.1.1", invalidCIDR))
	})
	
	t.Run("abs function", func(t *testing.T) {
		assert.Equal(t, int64(5), abs(5))
		assert.Equal(t, int64(5), abs(-5))
		assert.Equal(t, int64(0), abs(0))
	})
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	
	// Test default status code
	assert.Equal(t, http.StatusOK, rw.statusCode)
	
	// Test WriteHeader
	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rw.statusCode)
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	// Test Write
	rw.Write([]byte("test"))
	assert.Equal(t, "test", w.Body.String())
}
