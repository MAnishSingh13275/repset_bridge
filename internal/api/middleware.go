package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Process request
		next.ServeHTTP(wrapped, r)
		
		// Log request
		duration := time.Since(start)
		s.logger.WithFields(logrus.Fields{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP request")
	})
}

// recoveryMiddleware recovers from panics and returns 500 error
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.WithFields(logrus.Fields{
					"error": err,
					"stack": string(debug.Stack()),
					"path":  r.URL.Path,
				}).Error("Panic recovered in HTTP handler")
				
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware handles CORS headers with configurable origins
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.APIServer.CORS.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowedOrigin := ""
		for _, allowed := range s.config.APIServer.CORS.AllowedOrigins {
			if allowed == "*" || allowed == origin {
				allowedOrigin = allowed
				break
			}
		}
		
		// Set CORS headers if origin is allowed
		if allowedOrigin != "" {
			if allowedOrigin == "*" && !s.config.APIServer.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.APIServer.CORS.AllowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.APIServer.CORS.AllowedHeaders, ", "))
			
			if len(s.config.APIServer.CORS.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(s.config.APIServer.CORS.ExposedHeaders, ", "))
			}
			
			if s.config.APIServer.CORS.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			
			w.Header().Set("Access-Control-Max-Age", strconv.Itoa(s.config.APIServer.CORS.MaxAge))
		}
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			if allowedOrigin != "" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// securityHeadersMiddleware adds comprehensive security headers
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		security := s.config.APIServer.Security
		
		// Content-Type Options
		if security.ContentTypeOptions {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}
		
		// Frame Options
		if security.FrameOptions != "" {
			w.Header().Set("X-Frame-Options", security.FrameOptions)
		}
		
		// XSS Protection
		if security.XSSProtection {
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}
		
		// Referrer Policy
		if security.ReferrerPolicy != "" {
			w.Header().Set("Referrer-Policy", security.ReferrerPolicy)
		}
		
		// Content Security Policy
		if security.CSPEnabled && security.CSPDirective != "" {
			w.Header().Set("Content-Security-Policy", security.CSPDirective)
		}
		
		// HTTPS Strict Transport Security
		if security.HSTSEnabled && (r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https") {
			hstsValue := fmt.Sprintf("max-age=%d", security.HSTSMaxAge)
			if security.HSTSIncludeSubdomains {
				hstsValue += "; includeSubDomains"
			}
			w.Header().Set("Strict-Transport-Security", hstsValue)
		}
		
		next.ServeHTTP(w, r)
	})
}

// authenticationMiddleware validates API authentication using HMAC, API keys, or JWT
func (s *Server) authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication if disabled
		if !s.config.APIServer.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check IP allowlist if configured
		if len(s.config.APIServer.Auth.AllowedIPs) > 0 {
			clientIP := getClientIP(r)
			if !isIPAllowed(clientIP, s.config.APIServer.Auth.AllowedIPs) {
				s.logSecurityEvent("ip_blocked", clientIP, r)
				s.writeErrorResponse(w, "Access denied from this IP address", http.StatusForbidden)
				return
			}
		}
		
		// Try different authentication methods
		authenticated := false
		authMethod := ""
		
		// 1. Try HMAC authentication
		if s.config.APIServer.Auth.HMACSecret != "" {
			if s.validateHMACAuth(r) {
				authenticated = true
				authMethod = "hmac"
			}
		}
		
		// 2. Try API key authentication
		if !authenticated && len(s.config.APIServer.Auth.APIKeys) > 0 {
			if s.validateAPIKeyAuth(r) {
				authenticated = true
				authMethod = "api_key"
			}
		}
		
		// 3. Try JWT authentication
		if !authenticated && s.config.APIServer.Auth.JWTSecret != "" {
			if s.validateJWTAuth(r) {
				authenticated = true
				authMethod = "jwt"
			}
		}
		
		if !authenticated {
			s.logSecurityEvent("auth_failed", getClientIP(r), r)
			s.writeErrorResponse(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		
		// Log successful authentication
		s.logger.WithFields(logrus.Fields{
			"method":     authMethod,
			"path":       r.URL.Path,
			"client_ip":  getClientIP(r),
			"user_agent": r.UserAgent(),
		}).Debug("Authentication successful")
		
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// rateLimitEntry represents a rate limit entry for sliding window
type rateLimitEntry struct {
	requests  []time.Time
	mutex     sync.RWMutex
	lastClean time.Time
}

// rateLimiter implements sliding window rate limiting
type rateLimiter struct {
	entries         map[string]*rateLimitEntry
	mutex           sync.RWMutex
	requestsPerMin  int
	burstSize       int
	windowSize      time.Duration
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(requestsPerMin, burstSize int, windowSize, cleanupInterval time.Duration) *rateLimiter {
	return &rateLimiter{
		entries:         make(map[string]*rateLimitEntry),
		requestsPerMin:  requestsPerMin,
		burstSize:       burstSize,
		windowSize:      windowSize,
		cleanupInterval: cleanupInterval,
		lastCleanup:     time.Now(),
	}
}

// isAllowed checks if a request is allowed under rate limiting
func (rl *rateLimiter) isAllowed(key string) (bool, int, time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// Cleanup old entries periodically
	if now.Sub(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup(now)
		rl.lastCleanup = now
	}
	
	// Get or create entry for this key
	entry, exists := rl.entries[key]
	if !exists {
		entry = &rateLimitEntry{
			requests:  make([]time.Time, 0),
			lastClean: now,
		}
		rl.entries[key] = entry
	}
	
	entry.mutex.Lock()
	defer entry.mutex.Unlock()
	
	// Clean old requests from this entry
	cutoff := now.Add(-rl.windowSize)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range entry.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	entry.requests = validRequests
	
	// Check if request is allowed
	currentCount := len(entry.requests)
	if currentCount >= rl.requestsPerMin {
		// Calculate reset time (when oldest request expires)
		resetTime := entry.requests[0].Add(rl.windowSize)
		return false, rl.requestsPerMin - currentCount, resetTime
	}
	
	// Add current request
	entry.requests = append(entry.requests, now)
	remaining := rl.requestsPerMin - (currentCount + 1)
	
	return true, remaining, time.Time{}
}

// cleanup removes old entries
func (rl *rateLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-rl.windowSize * 2) // Keep entries a bit longer for safety
	
	for key, entry := range rl.entries {
		entry.mutex.RLock()
		shouldDelete := len(entry.requests) == 0 || (len(entry.requests) > 0 && entry.requests[len(entry.requests)-1].Before(cutoff))
		entry.mutex.RUnlock()
		
		if shouldDelete {
			delete(rl.entries, key)
		}
	}
}

// rateLimitMiddleware implements sliding window rate limiting
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	if !s.config.APIServer.RateLimit.Enabled {
		return next
	}
	
	// Initialize rate limiter if not already done
	if s.rateLimiter == nil {
		s.rateLimiter = newRateLimiter(
			s.config.APIServer.RateLimit.RequestsPerMin,
			s.config.APIServer.RateLimit.BurstSize,
			time.Duration(s.config.APIServer.RateLimit.WindowSize)*time.Second,
			time.Duration(s.config.APIServer.RateLimit.CleanupInterval)*time.Second,
		)
	}
	
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use client IP as rate limit key
		key := getClientIP(r)
		
		// Check if request is allowed
		allowed, remaining, resetTime := s.rateLimiter.isAllowed(key)
		
		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(s.config.APIServer.RateLimit.RequestsPerMin))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		
		if !resetTime.IsZero() {
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		}
		
		if !allowed {
			s.logSecurityEvent("rate_limit_exceeded", key, r)
			s.writeErrorResponse(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Authentication helper functions

// validateHMACAuth validates HMAC-SHA256 authentication
func (s *Server) validateHMACAuth(r *http.Request) bool {
	signature := r.Header.Get("X-Signature")
	timestamp := r.Header.Get("X-Timestamp")
	
	if signature == "" || timestamp == "" {
		return false
	}
	
	// Parse timestamp
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	
	// Check timestamp is within acceptable range (5 minutes)
	now := time.Now().Unix()
	if abs(now-ts) > 300 {
		return false
	}
	
	// Create message to sign: METHOD + PATH + TIMESTAMP + BODY
	body := ""
	if r.Body != nil {
		// Note: In production, you'd want to read and restore the body
		// For now, we'll skip body signing for simplicity
	}
	
	message := r.Method + r.URL.Path + timestamp + body
	
	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(s.config.APIServer.Auth.HMACSecret))
	mac.Write([]byte(message))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	
	// Compare signatures using constant time comparison
	return subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) == 1
}

// validateAPIKeyAuth validates API key authentication
func (s *Server) validateAPIKeyAuth(r *http.Request) bool {
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		// Also check Authorization header with Bearer scheme
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			apiKey = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	
	if apiKey == "" {
		return false
	}
	
	// Check if API key is in the allowed list
	for _, allowedKey := range s.config.APIServer.Auth.APIKeys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(allowedKey)) == 1 {
			return true
		}
	}
	
	return false
}

// validateJWTAuth validates JWT token authentication
func (s *Server) validateJWTAuth(r *http.Request) bool {
	tokenString := ""
	
	// Check Authorization header
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		tokenString = strings.TrimPrefix(auth, "Bearer ")
	}
	
	if tokenString == "" {
		return false
	}
	
	// Parse and validate JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.APIServer.Auth.JWTSecret), nil
	})
	
	if err != nil {
		return false
	}
	
	// Check if token is valid and not expired
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Check expiration
		if exp, ok := claims["exp"].(float64); ok {
			if time.Now().Unix() > int64(exp) {
				return false
			}
		}
		return true
	}
	
	return false
}

// Helper functions

// isIPAllowed checks if an IP is in the allowed list
func isIPAllowed(clientIP string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return true
	}
	
	clientIPAddr := net.ParseIP(clientIP)
	if clientIPAddr == nil {
		return false
	}
	
	for _, allowed := range allowedIPs {
		// Check if it's a CIDR range
		if strings.Contains(allowed, "/") {
			_, network, err := net.ParseCIDR(allowed)
			if err == nil && network.Contains(clientIPAddr) {
				return true
			}
		} else {
			// Direct IP comparison
			allowedIPAddr := net.ParseIP(allowed)
			if allowedIPAddr != nil && allowedIPAddr.Equal(clientIPAddr) {
				return true
			}
		}
	}
	
	return false
}

// logSecurityEvent logs security-related events
func (s *Server) logSecurityEvent(event, clientIP string, r *http.Request) {
	s.logger.WithFields(logrus.Fields{
		"event":      event,
		"client_ip":  clientIP,
		"path":       r.URL.Path,
		"method":     r.Method,
		"user_agent": r.UserAgent(),
		"timestamp":  time.Now().Unix(),
	}).Warn("Security event")
}

// abs returns the absolute value of an integer
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// writeErrorResponse writes a JSON error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	errorResponse := map[string]interface{}{
		"error":     true,
		"message":   message,
		"timestamp": time.Now().Unix(),
	}
	
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		s.logger.WithError(err).Error("Failed to encode error response")
	}
}