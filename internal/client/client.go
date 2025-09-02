package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/config"
	"github.com/sirupsen/logrus"
)

// AuthManager interface for authentication operations
type AuthManager interface {
	IsAuthenticated() bool
	GetDeviceID() string
	SignRequest(body []byte) (signature string, timestamp int64, err error)
}

// HTTPClient provides authenticated HTTP communication with the cloud API
type HTTPClient struct {
	httpClient    *http.Client
	authManager   AuthManager
	baseURL       string
	logger        *logrus.Logger
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	jitterFactor  float64
}

// ClientConfig holds configuration for the HTTP client
type ClientConfig struct {
	BaseURL       string
	Timeout       time.Duration
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	JitterFactor  float64
}

// DefaultClientConfig returns a client configuration with sensible defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:      30 * time.Second,
		MaxRetries:   5,
		BaseDelay:    1 * time.Second,
		MaxDelay:     30 * time.Second,
		JitterFactor: 0.1,
	}
}

// NewHTTPClient creates a new authenticated HTTP client
func NewHTTPClient(cfg *config.Config, authManager AuthManager, logger *logrus.Logger) (*HTTPClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if authManager == nil {
		return nil, fmt.Errorf("auth manager is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	clientCfg := DefaultClientConfig()
	clientCfg.BaseURL = cfg.ServerURL

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: clientCfg.Timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       90 * time.Second,
		},
	}

	return &HTTPClient{
		httpClient:    httpClient,
		authManager:   authManager,
		baseURL:       strings.TrimSuffix(clientCfg.BaseURL, "/"),
		logger:        logger,
		maxRetries:    clientCfg.MaxRetries,
		baseDelay:     clientCfg.BaseDelay,
		maxDelay:      clientCfg.MaxDelay,
		jitterFactor:  clientCfg.JitterFactor,
	}, nil
}

// NewHTTPClientWithAuthManager creates a new HTTP client with concrete auth manager
func NewHTTPClientWithAuthManager(cfg *config.Config, authManager *auth.AuthManager, logger *logrus.Logger) (*HTTPClient, error) {
	return NewHTTPClient(cfg, authManager, logger)
}

// Request represents an HTTP request to be made
type Request struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	RequireAuth bool
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an HTTP request with authentication and retry logic
func (c *HTTPClient) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	// Check network connectivity first
	if !c.IsConnected() {
		return nil, fmt.Errorf("no network connectivity")
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate delay with exponential backoff and jitter
			delay := c.calculateDelay(attempt)
			c.logger.Debug("Retrying request", "attempt", attempt, "delay", delay)
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		resp, err := c.doRequest(ctx, req)
		if err != nil {
			lastErr = err
			
			// Check if we should retry
			if !c.shouldRetry(err, resp) {
				return resp, err
			}
			
			c.logger.Warn("Request failed, will retry", "error", err, "attempt", attempt+1)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// doRequest performs a single HTTP request
func (c *HTTPClient) doRequest(ctx context.Context, req *Request) (*Response, error) {
	// Build URL
	fullURL := c.baseURL + req.Path

	// Marshal request body
	var bodyReader io.Reader
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for JSON requests
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add authentication headers if required
	if req.RequireAuth {
		if !c.authManager.IsAuthenticated() {
			return nil, fmt.Errorf("authentication required but device not authenticated")
		}

		// Sign the request
		signature, timestamp, err := c.authManager.SignRequest(bodyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}

		// Add authentication headers
		httpReq.Header.Set("X-Device-ID", c.authManager.GetDeviceID())
		httpReq.Header.Set("X-Signature", signature)
		httpReq.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	}

	// Log request details
	c.logger.Debug("Making HTTP request", 
		"method", req.Method, 
		"url", fullURL, 
		"authenticated", req.RequireAuth,
		"device_id", c.authManager.GetDeviceID())

	// Execute request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Body:       respBody,
		Headers:    httpResp.Header,
	}

	// Log response details
	c.logger.Debug("HTTP response received", 
		"status_code", httpResp.StatusCode, 
		"body_length", len(respBody))

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return resp, fmt.Errorf("HTTP error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return resp, nil
}

// shouldRetry determines if a request should be retried based on the error and response
func (c *HTTPClient) shouldRetry(err error, resp *Response) bool {
	// Don't retry context cancellation or timeout
	if err != nil {
		// Check if it's a context error (we don't have access to ctx here)
		if strings.Contains(err.Error(), "context") {
			return false
		}
		
		// Retry network errors
		if isNetworkError(err) {
			return true
		}
	}

	// Retry based on status code
	if resp != nil {
		switch resp.StatusCode {
		case 429: // Too Many Requests
			return true
		case 500, 502, 503, 504: // Server errors
			return true
		case 401, 403: // Authentication errors - don't retry
			return false
		case 400: // Bad request - don't retry
			return false
		}
	}

	return false
}

// calculateDelay calculates the delay for exponential backoff with jitter
func (c *HTTPClient) calculateDelay(attempt int) time.Duration {
	// Exponential backoff: baseDelay * 2^attempt
	delay := float64(c.baseDelay) * math.Pow(2, float64(attempt-1))
	
	// Cap at max delay
	if delay > float64(c.maxDelay) {
		delay = float64(c.maxDelay)
	}
	
	// Add jitter to avoid thundering herd
	jitter := delay * c.jitterFactor * (rand.Float64()*2 - 1) // Random between -jitterFactor and +jitterFactor
	delay += jitter
	
	// Ensure minimum delay
	if delay < float64(c.baseDelay) {
		delay = float64(c.baseDelay)
	}
	
	return time.Duration(delay)
}

// IsConnected checks if the client has network connectivity
func (c *HTTPClient) IsConnected() bool {
	// Parse base URL to get host
	u, err := url.Parse(c.baseURL)
	if err != nil {
		c.logger.Error("Failed to parse base URL for connectivity check", "error", err)
		return false
	}

	// For test servers (localhost), always return true
	if strings.Contains(u.Host, "127.0.0.1") || strings.Contains(u.Host, "localhost") {
		return true
	}

	// Try to establish a connection with a short timeout
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	
	conn, err := net.DialTimeout("tcp", u.Hostname()+":"+port, 5*time.Second)
	if err != nil {
		return false
	}
	
	conn.Close()
	return true
}

// Close closes the HTTP client and cleans up resources
func (c *HTTPClient) Close() error {
	// Close idle connections
	c.httpClient.CloseIdleConnections()
	return nil
}

// isNetworkError checks if an error is a network-related error that should be retried
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common network errors
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}

	// Check for DNS errors
	if dnsErr, ok := err.(*net.DNSError); ok {
		return dnsErr.Temporary()
	}

	// Check for connection refused, connection reset, etc.
	errStr := err.Error()
	networkErrors := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"no such host",
		"network is unreachable",
		"i/o timeout",
	}

	for _, netErr := range networkErrors {
		if strings.Contains(strings.ToLower(errStr), netErr) {
			return true
		}
	}

	return false
}

// parseJSONResponse parses a JSON response into the provided interface
func parseJSONResponse(resp *Response, v interface{}) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}

	if len(resp.Body) == 0 {
		return fmt.Errorf("response body is empty")
	}

	if err := json.Unmarshal(resp.Body, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	return nil
}