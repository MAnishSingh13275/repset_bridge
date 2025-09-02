package simulator

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"gym-door-bridge/internal/types"
)

// SimulatorAdapter implements the HardwareAdapter interface for testing and simulation
type SimulatorAdapter struct {
	name           string
	config         types.AdapterConfig
	status         types.AdapterStatus
	eventCallback  types.EventCallback
	stopChan       chan struct{}
	isListening    bool
	mutex          sync.RWMutex
	logger         *slog.Logger
	eventInterval  time.Duration
	simulatedUsers []string
}

// NewSimulatorAdapter creates a new simulator adapter instance
func NewSimulatorAdapter(logger *slog.Logger) *SimulatorAdapter {
	return &SimulatorAdapter{
		name:   "simulator",
		logger: logger,
		status: types.AdapterStatus{
			Name:      "simulator",
			Status:    types.StatusDisabled,
			UpdatedAt: time.Now(),
		},
		eventInterval: 30 * time.Second, // Default: generate event every 30 seconds
		simulatedUsers: []string{
			"sim_user_001",
			"sim_user_002", 
			"sim_user_003",
			"sim_user_004",
			"sim_user_005",
		},
	}
}

// Name returns the adapter name
func (s *SimulatorAdapter) Name() string {
	return s.name
}

// Initialize sets up the simulator with configuration
func (s *SimulatorAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.config = config
	s.status.Status = types.StatusInitializing
	s.status.UpdatedAt = time.Now()

	// Parse configuration settings
	if settings := config.Settings; settings != nil {
		if interval, ok := settings["eventInterval"].(float64); ok {
			s.eventInterval = time.Duration(interval) * time.Second
			// Ensure minimum interval to avoid ticker issues
			if s.eventInterval < 100*time.Millisecond {
				s.eventInterval = 100 * time.Millisecond
			}
		}
		if users, ok := settings["simulatedUsers"].([]interface{}); ok {
			s.simulatedUsers = make([]string, len(users))
			for i, user := range users {
				if userStr, ok := user.(string); ok {
					s.simulatedUsers[i] = userStr
				}
			}
		}
	}

	s.status.Status = types.StatusActive
	s.status.UpdatedAt = time.Now()
	s.status.ErrorMessage = ""

	s.logger.Info("Simulator adapter initialized",
		"name", s.name,
		"eventInterval", s.eventInterval,
		"userCount", len(s.simulatedUsers))

	return nil
}

// StartListening begins generating simulated events
func (s *SimulatorAdapter) StartListening(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isListening {
		return fmt.Errorf("simulator adapter is already listening")
	}

	if s.eventCallback == nil {
		return fmt.Errorf("no event callback registered")
	}

	s.stopChan = make(chan struct{})
	s.isListening = true
	s.status.Status = types.StatusActive
	s.status.UpdatedAt = time.Now()

	// Start event generation goroutine
	go s.generateEvents(ctx)

	s.logger.Info("Simulator adapter started listening", "name", s.name)
	return nil
}

// StopListening stops generating simulated events
func (s *SimulatorAdapter) StopListening(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isListening {
		return nil // Already stopped
	}

	close(s.stopChan)
	s.isListening = false
	s.status.UpdatedAt = time.Now()

	s.logger.Info("Simulator adapter stopped listening", "name", s.name)
	return nil
}

// UnlockDoor simulates door unlock operation
func (s *SimulatorAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.status.Status != types.StatusActive {
		return fmt.Errorf("simulator adapter is not active")
	}

	duration := time.Duration(durationMs) * time.Millisecond

	s.logger.Info("Simulating door unlock",
		"adapter", s.name,
		"durationMs", durationMs,
		"duration", duration)

	// Simulate unlock delay
	select {
	case <-time.After(100 * time.Millisecond):
		s.logger.Info("Door unlocked (simulated)",
			"adapter", s.name,
			"unlockDuration", duration)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Simulate automatic re-lock after duration
	go func() {
		select {
		case <-time.After(duration):
			s.logger.Info("Door automatically re-locked (simulated)",
				"adapter", s.name)
		case <-ctx.Done():
			return
		}
	}()

	return nil
}

// GetStatus returns the current adapter status
func (s *SimulatorAdapter) GetStatus() types.AdapterStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status
}

// OnEvent registers a callback for hardware events
func (s *SimulatorAdapter) OnEvent(callback types.EventCallback) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.eventCallback = callback
}

// IsHealthy returns true if the simulator is functioning properly
func (s *SimulatorAdapter) IsHealthy() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.status.Status == types.StatusActive
}

// generateEvents runs in a goroutine to generate simulated events
func (s *SimulatorAdapter) generateEvents(ctx context.Context) {
	ticker := time.NewTicker(s.eventInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.generateRandomEvent()
		case <-s.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// generateRandomEvent creates and sends a random simulated event
func (s *SimulatorAdapter) generateRandomEvent() {
	s.mutex.RLock()
	callback := s.eventCallback
	users := s.simulatedUsers
	s.mutex.RUnlock()

	if callback == nil || len(users) == 0 {
		return
	}

	// Generate random event
	eventTypes := []string{types.EventTypeEntry, types.EventTypeExit, types.EventTypeDenied}
	randomUser := users[rand.Intn(len(users))]
	randomEventType := eventTypes[rand.Intn(len(eventTypes))]

	event := types.RawHardwareEvent{
		ExternalUserID: randomUser,
		Timestamp:      time.Now(),
		EventType:      randomEventType,
		RawData: map[string]interface{}{
			"simulator":   true,
			"method":      "auto_generated",
			"confidence":  1.0,
			"deviceInfo":  "Simulator Hardware Adapter v1.0",
		},
	}

	// Update status with last event time
	s.mutex.Lock()
	s.status.LastEvent = event.Timestamp
	s.status.UpdatedAt = time.Now()
	s.mutex.Unlock()

	s.logger.Debug("Generated simulated event",
		"externalUserId", event.ExternalUserID,
		"eventType", event.EventType,
		"timestamp", event.Timestamp)

	// Send event to callback
	callback(event)
}

// TriggerEvent manually triggers a specific event (useful for testing)
func (s *SimulatorAdapter) TriggerEvent(externalUserID, eventType string) error {
	s.mutex.RLock()
	callback := s.eventCallback
	s.mutex.RUnlock()

	if callback == nil {
		return fmt.Errorf("no event callback registered")
	}

	if !types.IsValidEventType(eventType) {
		return fmt.Errorf("invalid event type: %s", eventType)
	}

	event := types.RawHardwareEvent{
		ExternalUserID: externalUserID,
		Timestamp:      time.Now(),
		EventType:      eventType,
		RawData: map[string]interface{}{
			"simulator":  true,
			"method":     "manual_trigger",
			"confidence": 1.0,
			"deviceInfo": "Simulator Hardware Adapter v1.0",
		},
	}

	// Update status
	s.mutex.Lock()
	s.status.LastEvent = event.Timestamp
	s.status.UpdatedAt = time.Now()
	s.mutex.Unlock()

	s.logger.Info("Manually triggered simulated event",
		"externalUserId", event.ExternalUserID,
		"eventType", event.EventType)

	callback(event)
	return nil
}