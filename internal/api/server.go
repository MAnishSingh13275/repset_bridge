package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)



// Server represents the HTTP API server
type Server struct {
	config         *config.Config
	logger         *logrus.Logger
	router         *mux.Router
	httpServer     *http.Server
	handlers       *Handlers
	rateLimiter    *rateLimiter
	errorHandler   *ErrorHandler
	requestLogger  *RequestLogger
}

// ServerConfig holds API server specific configuration
type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	TLSEnabled   bool   `mapstructure:"tls_enabled"`
	TLSCertFile  string `mapstructure:"tls_cert_file"`
	TLSKeyFile   string `mapstructure:"tls_key_file"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

// DefaultServerConfig returns default API server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Port:         8081,
		Host:         "0.0.0.0",
		TLSEnabled:   false,
		TLSCertFile:  "",
		TLSKeyFile:   "",
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  120,
	}
}

// NewServer creates a new API server instance
func NewServer(cfg *config.Config, serverCfg *ServerConfig, adapterRegistry AdapterRegistry, doorController DoorController, healthMonitor HealthMonitor, queueManager QueueManager, tierDetector TierDetector, configManager ConfigManager, version, deviceID string) *Server {
	logger := logging.Initialize(cfg.LogLevel)
	
	server := &Server{
		config:        cfg,
		logger:        logger,
		router:        mux.NewRouter(),
		errorHandler:  NewErrorHandler(logger),
		requestLogger: NewRequestLogger(logger),
	}
	
	// Initialize handlers with dependencies
	server.handlers = NewHandlers(cfg, logger, adapterRegistry, doorController, healthMonitor, queueManager, tierDetector, configManager, version, deviceID)
	
	// Set up middleware
	server.setupMiddleware()
	
	// Set up routes
	server.setupRoutes()
	
	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port),
		Handler:      server.router,
		ReadTimeout:  time.Duration(serverCfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(serverCfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(serverCfg.IdleTimeout) * time.Second,
	}
	
	// Configure TLS if enabled
	if serverCfg.TLSEnabled {
		if serverCfg.TLSCertFile == "" || serverCfg.TLSKeyFile == "" {
			logger.Fatal("TLS enabled but cert or key file not specified")
		}
		
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		
		server.httpServer.TLSConfig = tlsConfig
	}
	
	return server
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context, serverCfg *ServerConfig) error {
	s.logger.WithFields(logrus.Fields{
		"addr":        s.httpServer.Addr,
		"tls_enabled": serverCfg.TLSEnabled,
	}).Info("Starting API server")
	
	// Start WebSocket manager
	if s.handlers.wsManager != nil {
		s.handlers.wsManager.Start(ctx)
	}
	
	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		var err error
		if serverCfg.TLSEnabled {
			err = s.httpServer.ListenAndServeTLS(serverCfg.TLSCertFile, serverCfg.TLSKeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		
		if err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	
	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("API server shutting down")
		return s.Shutdown()
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	s.logger.Info("Gracefully shutting down API server")
	
	// Stop WebSocket manager first
	if s.handlers.wsManager != nil {
		s.handlers.wsManager.Stop()
	}
	
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("Error during server shutdown")
		return err
	}
	
	s.logger.Info("API server shutdown complete")
	return nil
}

// setupMiddleware configures middleware for the router
func (s *Server) setupMiddleware() {
	// Enhanced request logging middleware (replaces basic logging)
	s.router.Use(s.requestLogger.StructuredLoggingMiddleware)
	
	// Enhanced recovery middleware with structured error handling
	s.router.Use(s.errorHandler.RecoveryMiddleware)
	
	// Error recovery middleware with circuit breaker protection
	s.router.Use(s.errorHandler.ErrorRecoveryMiddleware)
	
	// Rate limiting middleware
	s.router.Use(s.rateLimitMiddleware)
	
	// CORS middleware
	s.router.Use(s.corsMiddleware)
	
	// Security headers middleware
	s.router.Use(s.securityHeadersMiddleware)
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Handle OPTIONS requests globally
	s.router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers are already set by middleware
		w.WriteHeader(http.StatusOK)
	})
	
	// API version prefix
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Health endpoint (no auth required)
	api.HandleFunc("/health", s.handlers.HealthCheck).Methods("GET")
	
	// Protected endpoints (require authentication)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.authenticationMiddleware)
	
	// Door control endpoints
	protected.HandleFunc("/door/unlock", s.handlers.UnlockDoor).Methods("POST")
	protected.HandleFunc("/door/lock", s.handlers.LockDoor).Methods("POST")
	protected.HandleFunc("/door/status", s.handlers.DoorStatus).Methods("GET")
	
	// Device status endpoints
	protected.HandleFunc("/status", s.handlers.DeviceStatus).Methods("GET")
	protected.HandleFunc("/metrics", s.handlers.DeviceMetrics).Methods("GET")
	
	// Configuration endpoints
	protected.HandleFunc("/config", s.handlers.GetConfig).Methods("GET")
	protected.HandleFunc("/config", s.handlers.UpdateConfig).Methods("PUT")
	protected.HandleFunc("/config/reload", s.handlers.ReloadConfig).Methods("POST")
	
	// Events endpoints
	protected.HandleFunc("/events", s.handlers.GetEvents).Methods("GET")
	protected.HandleFunc("/events/stats", s.handlers.GetEventStats).Methods("GET")
	protected.HandleFunc("/events", s.handlers.ClearEvents).Methods("DELETE")
	
	// Adapters endpoints
	protected.HandleFunc("/adapters", s.handlers.GetAdapters).Methods("GET")
	protected.HandleFunc("/adapters/{name}", s.handlers.GetAdapter).Methods("GET")
	protected.HandleFunc("/adapters/{name}/enable", s.handlers.EnableAdapter).Methods("POST")
	protected.HandleFunc("/adapters/{name}/disable", s.handlers.DisableAdapter).Methods("POST")
	protected.HandleFunc("/adapters/{name}/config", s.handlers.UpdateAdapterConfig).Methods("PUT")
	
	// WebSocket endpoints
	protected.HandleFunc("/ws", s.handlers.WebSocketHandler).Methods("GET")
	protected.HandleFunc("/ws/status", s.handlers.WebSocketStatus).Methods("GET")
	protected.HandleFunc("/ws/broadcast", s.handlers.WebSocketBroadcast).Methods("POST")
}