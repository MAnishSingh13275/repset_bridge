package macos

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"

	"github.com/sirupsen/logrus"
)

const (
	ServiceName        = "com.gymdoorbridge.agent"
	ServiceDisplayName = "Gym Door Access Bridge"
	ServiceDescription = "Connects gym door access hardware to SaaS platform"
)

// Service represents the macOS daemon wrapper
type Service struct {
	config     *config.Config
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	bridgeFunc func(ctx context.Context, cfg *config.Config) error
}

// NewService creates a new macOS service instance
func NewService(cfg *config.Config, bridgeFunc func(context.Context, *config.Config) error) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		config:     cfg,
		logger:     logging.Initialize("info"), // Will be reconfigured based on service config
		ctx:        ctx,
		cancel:     cancel,
		bridgeFunc: bridgeFunc,
	}
}

// Run executes the service with proper signal handling for macOS daemon
func (s *Service) Run() error {
	s.logger.WithField("service", ServiceDisplayName).Info("Starting macOS daemon")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start the bridge in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.bridgeFunc(s.ctx, s.config); err != nil {
			errChan <- err
		}
	}()

	s.logger.WithField("service", ServiceDisplayName).Info("macOS daemon started successfully")

	// Service control loop
	for {
		select {
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				s.logger.WithField("signal", sig.String()).Info("Received shutdown signal")
				s.cancel() // Cancel the context to stop bridge operations

				// Wait for graceful shutdown with timeout
				select {
				case <-time.After(30 * time.Second):
					s.logger.Warning("Service shutdown timeout reached")
				case err := <-errChan:
					if err != nil && err != context.Canceled {
						s.logger.WithError(err).Error("Bridge stopped with error")
						return err
					}
				}

				s.logger.WithField("service", ServiceDisplayName).Info("macOS daemon stopped")
				return nil

			case syscall.SIGHUP:
				s.logger.Info("Received SIGHUP - reloading configuration")
				// TODO: Implement configuration reload
				// For now, just log the signal
			}

		case err := <-errChan:
			if err != nil && err != context.Canceled {
				s.logger.WithError(err).Error("Bridge error")
				return err
			}
		}
	}
}

// RunService runs the application as a macOS daemon
func RunService(cfg *config.Config, bridgeFunc func(context.Context, *config.Config) error) error {
	service := NewService(cfg, bridgeFunc)
	return service.Run()
}

// IsMacOSDaemon checks if the application is running as a macOS daemon
// This is determined by checking if we're running under launchd
func IsMacOSDaemon() bool {
	// Check if LAUNCH_DAEMON_SOCKET_NAME environment variable is set
	// This is set by launchd when running as a daemon
	return os.Getenv("LAUNCH_DAEMON_SOCKET_NAME") != ""
}