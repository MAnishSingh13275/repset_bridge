package windows

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/logging"
	
	"github.com/sirupsen/logrus"
)

const (
	ServiceName        = "GymDoorBridge"
	ServiceDisplayName = "Gym Door Access Bridge"
	ServiceDescription = "Connects gym door access hardware to SaaS platform"
)

// Service represents the Windows service wrapper
type Service struct {
	config     *config.Config
	logger     *logrus.Logger
	eventLog   *eventlog.Log
	ctx        context.Context
	cancel     context.CancelFunc
	bridgeFunc func(ctx context.Context, cfg *config.Config) error
}

// NewService creates a new Windows service instance
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

// Execute implements the svc.Handler interface for Windows service execution
func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	
	// Initialize event log
	var err error
	s.eventLog, err = eventlog.Open(ServiceName)
	if err != nil {
		s.logger.WithError(err).Error("Failed to open event log")
		return false, 1
	}
	defer s.eventLog.Close()

	s.eventLog.Info(1, fmt.Sprintf("%s service starting", ServiceDisplayName))
	
	changes <- svc.Status{State: svc.StartPending}
	
	// Start the bridge in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.bridgeFunc(s.ctx, s.config); err != nil {
			errChan <- err
		}
	}()
	
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	s.eventLog.Info(1, fmt.Sprintf("%s service started successfully", ServiceDisplayName))

	// Service control loop
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Add a small delay to prevent rapid interrogation
				time.Sleep(100 * time.Millisecond)
				
			case svc.Stop, svc.Shutdown:
				s.eventLog.Info(1, fmt.Sprintf("%s service stopping", ServiceDisplayName))
				changes <- svc.Status{State: svc.StopPending}
				s.cancel() // Cancel the context to stop bridge operations
				
				// Wait for graceful shutdown with timeout
				select {
				case <-time.After(30 * time.Second):
					s.eventLog.Warning(1, "Service shutdown timeout reached")
				case err := <-errChan:
					if err != nil {
						s.eventLog.Error(1, fmt.Sprintf("Bridge stopped with error: %v", err))
					}
				}
				
				s.eventLog.Info(1, fmt.Sprintf("%s service stopped", ServiceDisplayName))
				return false, 0
				
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				s.eventLog.Info(1, fmt.Sprintf("%s service paused", ServiceDisplayName))
				
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				s.eventLog.Info(1, fmt.Sprintf("%s service resumed", ServiceDisplayName))
				
			default:
				s.eventLog.Error(1, fmt.Sprintf("Unexpected service control request: %d", c.Cmd))
			}
			
		case err := <-errChan:
			if err != nil {
				s.eventLog.Error(1, fmt.Sprintf("Bridge error: %v", err))
				changes <- svc.Status{State: svc.Stopped}
				return false, 1
			}
		}
	}
}

// RunService runs the application as a Windows service or in debug mode
func RunService(cfg *config.Config, bridgeFunc func(context.Context, *config.Config) error, isDebug bool) error {
	service := NewService(cfg, bridgeFunc)
	
	if isDebug {
		// Run in debug mode (console application)
		return debug.Run(ServiceName, service)
	}
	
	// Run as Windows service
	return svc.Run(ServiceName, service)
}

// IsWindowsService checks if the application is running as a Windows service
func IsWindowsService() (bool, error) {
	return svc.IsWindowsService()
}