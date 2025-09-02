package door

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"
)

// DoorControlConfig holds configuration for the door control system
type DoorControlConfig struct {
	Port                int           `json:"port"`                // Port for door control endpoint
	Path                string        `json:"path"`                // Path for door control endpoint
	DefaultUnlockDuration int         `json:"defaultUnlockDuration"` // Default unlock duration in milliseconds
	MaxUnlockDuration   int           `json:"maxUnlockDuration"`   // Maximum allowed unlock duration in milliseconds
	AuthRequired        bool          `json:"authRequired"`        // Whether authentication is required
}

// DefaultDoorControlConfig returns the default door control configuration
func DefaultDoorControlConfig() DoorControlConfig {
	return DoorControlConfig{
		Port:                8081,
		Path:                "/open-door",
		DefaultUnlockDuration: 3000, // 3 seconds
		MaxUnlockDuration:   30000,  // 30 seconds max
		AuthRequired:        false,  // For now, no auth required for local access
	}
}

// UnlockRequest represents a door unlock request
type UnlockRequest struct {
	DurationMs int    `json:"durationMs,omitempty"` // Duration in milliseconds (optional)
	Adapter    string `json:"adapter,omitempty"`    // Specific adapter to use (optional)
}

// UnlockResponse represents a door unlock response
type UnlockResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Duration  int    `json:"duration"`  // Actual duration used
	Adapter   string `json:"adapter"`   // Adapter that performed the unlock
	Timestamp string `json:"timestamp"` // When the unlock was performed
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// AdapterRegistry interface for getting adapters
type AdapterRegistry interface {
	GetAllAdapters() []adapters.HardwareAdapter
	GetAdapter(name string) (adapters.HardwareAdapter, error)
	GetActiveAdapters() []adapters.HardwareAdapter
}

// DoorController manages door control operations and HTTP endpoints
type DoorController struct {
	mu              sync.RWMutex
	config          DoorControlConfig
	globalConfig    *config.Config
	logger          *logrus.Logger
	adapterRegistry AdapterRegistry
	httpServer      *http.Server
	
	// Statistics
	unlockCount     int64
	lastUnlockTime  time.Time
	failureCount    int64
}

// DoorControllerOption is a functional option for configuring the DoorController
type DoorControllerOption func(*DoorController)

// WithLogger sets the logger for the door controller
func WithLogger(logger *logrus.Logger) DoorControllerOption {
	return func(d *DoorController) {
		d.logger = logger
	}
}

// NewDoorController creates a new door controller
func NewDoorController(
	config DoorControlConfig,
	globalConfig *config.Config,
	adapterRegistry AdapterRegistry,
	opts ...DoorControllerOption,
) *DoorController {
	d := &DoorController{
		config:          config,
		globalConfig:    globalConfig,
		logger:          logrus.New(),
		adapterRegistry: adapterRegistry,
	}
	
	// Apply options
	for _, opt := range opts {
		opt(d)
	}
	
	return d
}

// Start begins the door control HTTP server
func (d *DoorController) Start(ctx context.Context) error {
	d.logger.Info("Starting door controller", "port", d.config.Port, "path", d.config.Path)
	
	// Set up HTTP server for door control endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(d.config.Path, d.handleDoorUnlock)
	
	d.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", d.config.Port),
		Handler: mux,
	}
	
	// Start HTTP server in a goroutine
	go func() {
		if err := d.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			d.logger.WithError(err).Error("Door control HTTP server failed")
		}
	}()
	
	d.logger.Info("Door control endpoint started", "url", fmt.Sprintf("http://localhost:%d%s", d.config.Port, d.config.Path))
	
	return nil
}

// Stop gracefully shuts down the door control system
func (d *DoorController) Stop(ctx context.Context) error {
	d.logger.Info("Stopping door controller")
	
	// Stop HTTP server
	if d.httpServer != nil {
		if err := d.httpServer.Shutdown(ctx); err != nil {
			d.logger.WithError(err).Error("Failed to shutdown door control HTTP server")
			return err
		}
	}
	
	return nil
}

// UnlockDoor unlocks the door using the specified adapter or the first available adapter
func (d *DoorController) UnlockDoor(ctx context.Context, adapterName string, durationMs int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Validate duration
	if durationMs <= 0 {
		durationMs = d.config.DefaultUnlockDuration
	}
	if durationMs > d.config.MaxUnlockDuration {
		d.failureCount++
		return fmt.Errorf("unlock duration %d exceeds maximum allowed %d", durationMs, d.config.MaxUnlockDuration)
	}
	
	// Get the adapter to use
	var adapter adapters.HardwareAdapter
	var err error
	
	if adapterName != "" {
		// Use specific adapter
		adapter, err = d.adapterRegistry.GetAdapter(adapterName)
		if err != nil {
			d.failureCount++
			return fmt.Errorf("failed to get adapter %s: %w", adapterName, err)
		}
	} else {
		// Use first active adapter
		activeAdapters := d.adapterRegistry.GetActiveAdapters()
		if len(activeAdapters) == 0 {
			d.failureCount++
			return fmt.Errorf("no active adapters available")
		}
		adapter = activeAdapters[0]
	}
	
	// Check if adapter is healthy
	if !adapter.IsHealthy() {
		d.failureCount++
		return fmt.Errorf("adapter %s is not healthy", adapter.Name())
	}
	
	// Perform unlock
	d.logger.Info("Unlocking door",
		"adapter", adapter.Name(),
		"durationMs", durationMs)
	
	if err := adapter.UnlockDoor(ctx, durationMs); err != nil {
		d.failureCount++
		d.logger.WithError(err).Error("Failed to unlock door",
			"adapter", adapter.Name(),
			"durationMs", durationMs)
		return fmt.Errorf("failed to unlock door with adapter %s: %w", adapter.Name(), err)
	}
	
	// Update statistics
	d.unlockCount++
	d.lastUnlockTime = time.Now()
	
	d.logger.Info("Door unlocked successfully",
		"adapter", adapter.Name(),
		"durationMs", durationMs,
		"totalUnlocks", d.unlockCount)
	
	return nil
}

// GetStats returns door control statistics
func (d *DoorController) GetStats() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	return map[string]interface{}{
		"unlockCount":    d.unlockCount,
		"failureCount":   d.failureCount,
		"lastUnlockTime": d.lastUnlockTime,
	}
}

// handleDoorUnlock handles HTTP door unlock requests
func (d *DoorController) handleDoorUnlock(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Only allow POST requests
	if r.Method != http.MethodPost {
		d.writeErrorResponse(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST method is allowed")
		return
	}
	
	// Parse request
	var req UnlockRequest
	
	// Check if request has JSON body
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			d.writeErrorResponse(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON in request body")
			return
		}
	} else {
		// Parse query parameters as fallback
		if durationStr := r.URL.Query().Get("duration"); durationStr != "" {
			if duration, err := strconv.Atoi(durationStr); err == nil {
				req.DurationMs = duration
			}
		}
		if adapter := r.URL.Query().Get("adapter"); adapter != "" {
			req.Adapter = adapter
		}
	}
	
	// Use default duration if not specified
	if req.DurationMs <= 0 {
		req.DurationMs = d.globalConfig.UnlockDuration
		if req.DurationMs <= 0 {
			req.DurationMs = d.config.DefaultUnlockDuration
		}
	}
	
	// Perform unlock
	err := d.UnlockDoor(ctx, req.Adapter, req.DurationMs)
	if err != nil {
		d.logger.WithError(err).Error("Door unlock request failed")
		d.writeErrorResponse(w, http.StatusInternalServerError, "UNLOCK_FAILED", err.Error())
		return
	}
	
	// Determine which adapter was used
	adapterName := req.Adapter
	if adapterName == "" {
		if activeAdapters := d.adapterRegistry.GetActiveAdapters(); len(activeAdapters) > 0 {
			adapterName = activeAdapters[0].Name()
		}
	}
	
	// Return success response
	response := UnlockResponse{
		Success:   true,
		Message:   "Door unlocked successfully",
		Duration:  req.DurationMs,
		Adapter:   adapterName,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		d.logger.WithError(err).Error("Failed to encode unlock response")
	}
	
	d.logger.Info("Door unlock request completed successfully",
		"adapter", adapterName,
		"duration", req.DurationMs,
		"remoteAddr", r.RemoteAddr)
}

// writeErrorResponse writes an error response to the HTTP response writer
func (d *DoorController) writeErrorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := ErrorResponse{
		Error:   errorCode,
		Code:    statusCode,
		Message: message,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		d.logger.WithError(err).Error("Failed to encode error response")
	}
}