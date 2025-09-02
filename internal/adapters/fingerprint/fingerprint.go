package fingerprint

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gym-door-bridge/internal/types"
)



// FingerprintAdapter implements the HardwareAdapter interface for fingerprint scanners
type FingerprintAdapter struct {
	name          string
	config        types.AdapterConfig
	status        types.AdapterStatus
	eventCallback types.EventCallback
	isListening   bool
	mutex         sync.RWMutex
	logger        *slog.Logger
	devicePath    string
	baudRate      int
	protocol      string
}

// NewFingerprintAdapter creates a new fingerprint adapter instance
func NewFingerprintAdapter(logger *slog.Logger) *FingerprintAdapter {
	return &FingerprintAdapter{
		name:   "fingerprint",
		logger: logger,
		status: types.AdapterStatus{
			Name:      "fingerprint",
			Status:    types.StatusDisabled,
			UpdatedAt: time.Now(),
		},
		baudRate: 9600,
		protocol: "wiegand",
	}
}

// Name returns the adapter name
func (f *FingerprintAdapter) Name() string {
	return f.name
}

// Initialize sets up the fingerprint adapter with configuration
func (f *FingerprintAdapter) Initialize(ctx context.Context, config types.AdapterConfig) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.config = config
	f.status.Status = types.StatusInitializing
	f.status.UpdatedAt = time.Now()

	// Parse configuration settings
	if settings := config.Settings; settings != nil {
		if devicePath, ok := settings["devicePath"].(string); ok {
			f.devicePath = devicePath
		}
		if baudRate, ok := settings["baudRate"].(float64); ok {
			f.baudRate = int(baudRate)
		}
		if protocol, ok := settings["protocol"].(string); ok {
			f.protocol = protocol
		}
	}

	// Validate configuration
	if f.devicePath == "" {
		f.status.Status = types.StatusError
		f.status.ErrorMessage = "devicePath is required"
		f.status.UpdatedAt = time.Now()
		return fmt.Errorf("devicePath is required for fingerprint adapter")
	}

	// TODO: Implement actual hardware initialization
	// This would include:
	// - Opening serial/USB connection to fingerprint scanner
	// - Configuring communication parameters
	// - Testing device connectivity
	// - Loading device-specific drivers or libraries

	f.status.Status = types.StatusActive
	f.status.UpdatedAt = time.Now()
	f.status.ErrorMessage = ""

	f.logger.Info("Fingerprint adapter initialized",
		"name", f.name,
		"devicePath", f.devicePath,
		"baudRate", f.baudRate,
		"protocol", f.protocol)

	return nil
}

// StartListening begins listening for fingerprint scan events
func (f *FingerprintAdapter) StartListening(ctx context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.isListening {
		return fmt.Errorf("fingerprint adapter is already listening")
	}

	if f.eventCallback == nil {
		return fmt.Errorf("no event callback registered")
	}

	// TODO: Implement actual hardware communication
	// This would include:
	// - Starting communication with fingerprint scanner
	// - Setting up event handlers for scan results
	// - Implementing protocol-specific message parsing
	// - Handling device errors and reconnection

	f.isListening = true
	f.status.Status = types.StatusActive
	f.status.UpdatedAt = time.Now()

	f.logger.Info("Fingerprint adapter started listening", "name", f.name)
	
	// For now, return an error indicating this is a framework implementation
	return fmt.Errorf("fingerprint adapter is a framework implementation - actual hardware integration required")
}

// StopListening stops listening for fingerprint scan events
func (f *FingerprintAdapter) StopListening(ctx context.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if !f.isListening {
		return nil // Already stopped
	}

	// TODO: Implement actual hardware disconnection
	// This would include:
	// - Stopping communication with fingerprint scanner
	// - Closing serial/USB connections
	// - Cleaning up resources

	f.isListening = false
	f.status.UpdatedAt = time.Now()

	f.logger.Info("Fingerprint adapter stopped listening", "name", f.name)
	return nil
}

// UnlockDoor triggers door unlock via fingerprint scanner (if supported)
func (f *FingerprintAdapter) UnlockDoor(ctx context.Context, durationMs int) error {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	if f.status.Status != types.StatusActive {
		return fmt.Errorf("fingerprint adapter is not active")
	}

	// TODO: Implement actual door unlock command
	// This would include:
	// - Sending unlock command to fingerprint scanner
	// - Handling device-specific unlock protocols
	// - Managing unlock duration and automatic re-lock

	f.logger.Info("Door unlock requested via fingerprint adapter",
		"adapter", f.name,
		"durationMs", durationMs)

	// For now, return an error indicating this is a framework implementation
	return fmt.Errorf("door unlock not implemented - framework implementation only")
}

// GetStatus returns the current adapter status
func (f *FingerprintAdapter) GetStatus() types.AdapterStatus {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return f.status
}

// OnEvent registers a callback for hardware events
func (f *FingerprintAdapter) OnEvent(callback types.EventCallback) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.eventCallback = callback
}

// IsHealthy returns true if the fingerprint scanner is functioning properly
func (f *FingerprintAdapter) IsHealthy() bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	
	// TODO: Implement actual health check
	// This would include:
	// - Testing communication with fingerprint scanner
	// - Checking device status and error conditions
	// - Verifying hardware connectivity
	
	return f.status.Status == types.StatusActive
}

// processRawScanData converts raw fingerprint scan data to standardized event
// This is a helper method that would be called by the actual hardware integration
func (f *FingerprintAdapter) processRawScanData(rawData []byte) (*types.RawHardwareEvent, error) {
	// TODO: Implement protocol-specific data parsing
	// This would include:
	// - Parsing device-specific data formats
	// - Extracting user ID from fingerprint template
	// - Determining scan result (success/failure)
	// - Converting to standardized event format

	// Example implementation structure:
	event := &types.RawHardwareEvent{
		ExternalUserID: "fp_placeholder", // Would be extracted from rawData
		Timestamp:      time.Now(),
		EventType:      types.EventTypeEntry, // Would be determined from scan result
		RawData: map[string]interface{}{
			"fingerprint": true,
			"protocol":    f.protocol,
			"devicePath":  f.devicePath,
			"confidence":  0.95, // Would be extracted from scan data
			"templateId":  "template_placeholder",
		},
	}

	return event, nil
}

// GetSupportedProtocols returns a list of supported fingerprint scanner protocols
func GetSupportedProtocols() []string {
	return []string{
		"wiegand",
		"rs485",
		"tcp",
		"usb_hid",
		"serial",
	}
}

// ValidateProtocol checks if the specified protocol is supported
func ValidateProtocol(protocol string) bool {
	supported := GetSupportedProtocols()
	for _, p := range supported {
		if p == protocol {
			return true
		}
	}
	return false
}