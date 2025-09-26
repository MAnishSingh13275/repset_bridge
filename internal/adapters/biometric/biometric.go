package biometric

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

// BiometricAdapter handles biometric device integration (ESSL, ZKTeco, etc.)
type BiometricAdapter struct {
	name           string
	logger         *logrus.Logger
	eventCallback  types.EventCallback
	config         *Config
	device         BiometricDevice
	platformClient *PlatformClient
	isRunning      bool
	stopChan       chan struct{}
}

// Config holds biometric device configuration
type Config struct {
	DeviceType     string            `json:"device_type"`     // essl, zkteco, realtime, anviz
	Connection     string            `json:"connection"`      // tcp, serial, usb
	DeviceConfig   map[string]string `json:"device_config"`   // device-specific config
	SyncInterval   int               `json:"sync_interval"`   // seconds between polling
	PlatformURL    string            `json:"platform_url"`    // platform API URL
	DeviceID       string            `json:"device_id"`       // bridge device ID
	DeviceKey      string            `json:"device_key"`      // bridge device key
}

// BiometricDevice interface for different biometric hardware
type BiometricDevice interface {
	Connect() error
	Disconnect() error
	IsConnected() bool
	GetStatus() string
	
	// User management
	EnrollUser(platformUserID string, deviceUserID int, name string) error
	DeleteUser(deviceUserID int) error
	GetUsers() ([]DeviceUser, error)
	
	// Attendance polling
	GetNewAttendanceRecords() ([]AttendanceRecord, error)
	ClearAttendanceRecords() error
	
	// Device info
	GetDeviceInfo() (*DeviceInfo, error)
	GetDeviceTime() (time.Time, error)
	SetDeviceTime(t time.Time) error
}

// DeviceUser represents a user stored on the biometric device
type DeviceUser struct {
	DeviceUserID   int    `json:"device_user_id"`
	PlatformUserID string `json:"platform_user_id"`
	Name           string `json:"name"`
	Privilege      int    `json:"privilege"`      // User privilege level
	Password       string `json:"password"`       // Optional password
	CardNumber     string `json:"card_number"`    // Optional card number
}

// AttendanceRecord represents an attendance record from the device
type AttendanceRecord struct {
	DeviceUserID int       `json:"device_user_id"`
	Timestamp    time.Time `json:"timestamp"`
	Status       int       `json:"status"`        // 0=check-in, 1=check-out, etc.
	VerifyMode   int       `json:"verify_mode"`   // 1=fingerprint, 2=password, 3=card
	WorkCode     int       `json:"work_code"`     // Optional work code
}

// DeviceInfo contains device information
type DeviceInfo struct {
	SerialNumber   string `json:"serial_number"`
	DeviceModel    string `json:"device_model"`
	FirmwareVer    string `json:"firmware_version"`
	UserCount      int    `json:"user_count"`
	AttendanceCount int   `json:"attendance_count"`
	FingerCapacity int    `json:"finger_capacity"`
}

// PlatformClient handles API calls to the platform
type PlatformClient struct {
	baseURL   string
	deviceID  string
	deviceKey string
	client    *http.Client
	logger    *logrus.Logger
}

// NewBiometricAdapter creates a new biometric adapter
func NewBiometricAdapter(name string, config map[string]interface{}, logger *logrus.Logger) (*BiometricAdapter, error) {
	// Parse configuration
	configBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var bioConfig Config
	if err := json.Unmarshal(configBytes, &bioConfig); err != nil {
		return nil, fmt.Errorf("failed to parse biometric config: %w", err)
	}

	// Create device based on type
	var device BiometricDevice
	switch bioConfig.DeviceType {
	case "essl":
		device = NewESSLDevice(bioConfig.DeviceConfig, logger)
	case "zkteco":
		device = NewZKTecoDevice(bioConfig.DeviceConfig, logger)
	case "realtime":
		device = NewRealtimeDevice(bioConfig.DeviceConfig, logger)
	case "simulator":
		device = NewSimulatorDevice(bioConfig.DeviceConfig, logger)
	default:
		return nil, fmt.Errorf("unsupported device type: %s", bioConfig.DeviceType)
	}

	// Create platform client
	platformClient := &PlatformClient{
		baseURL:   bioConfig.PlatformURL,
		deviceID:  bioConfig.DeviceID,
		deviceKey: bioConfig.DeviceKey,
		client:    &http.Client{Timeout: 30 * time.Second},
		logger:    logger,
	}

	return &BiometricAdapter{
		name:           name,
		logger:         logger,
		config:         &bioConfig,
		device:         device,
		platformClient: platformClient,
		stopChan:       make(chan struct{}),
	}, nil
}

// Name returns the adapter name
func (b *BiometricAdapter) Name() string {
	return b.name
}

// Start begins biometric device operations
func (b *BiometricAdapter) Start(ctx context.Context, eventCallback types.EventCallback) error {
	b.eventCallback = eventCallback
	b.isRunning = true

	// Connect to device
	if err := b.device.Connect(); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	b.logger.Info("Biometric adapter started")

	// Start attendance polling loop
	go b.attendancePollingLoop(ctx)

	return nil
}

// Stop stops the biometric adapter
func (b *BiometricAdapter) Stop() error {
	if !b.isRunning {
		return nil
	}

	b.isRunning = false
	close(b.stopChan)

	// Disconnect from device
	if err := b.device.Disconnect(); err != nil {
		b.logger.WithError(err).Error("Error disconnecting from device")
	}

	b.logger.Info("Biometric adapter stopped")
	return nil
}

// IsRunning returns whether the adapter is running
func (b *BiometricAdapter) IsRunning() bool {
	return b.isRunning
}

// GetStatus returns the current adapter status
func (b *BiometricAdapter) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"name":         b.name,
		"running":      b.isRunning,
		"device_type":  b.config.DeviceType,
		"connection":   b.config.Connection,
		"connected":    b.device.IsConnected(),
		"device_status": b.device.GetStatus(),
	}

	// Add device info if connected
	if b.device.IsConnected() {
		if info, err := b.device.GetDeviceInfo(); err == nil {
			status["device_info"] = info
		}
	}

	return status
}

// EnrollUser enrolls a user on the biometric device
func (b *BiometricAdapter) EnrollUser(platformUserID string, deviceUserID int, name string) error {
	b.logger.WithFields(logrus.Fields{
		"platform_user_id": platformUserID,
		"device_user_id":   deviceUserID,
		"name":             name,
	}).Info("Enrolling user on biometric device")

	if err := b.device.EnrollUser(platformUserID, deviceUserID, name); err != nil {
		return fmt.Errorf("device enrollment failed: %w", err)
	}

	b.logger.Info("User enrolled successfully on biometric device")
	return nil
}

// attendancePollingLoop continuously polls for new attendance records
func (b *BiometricAdapter) attendancePollingLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.SyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.stopChan:
			return
		case <-ticker.C:
			b.pollAttendanceRecords()
		}
	}
}

// pollAttendanceRecords polls the device for new attendance records
func (b *BiometricAdapter) pollAttendanceRecords() {
	if !b.device.IsConnected() {
		b.logger.Warn("Device not connected, skipping attendance poll")
		return
	}

	records, err := b.device.GetNewAttendanceRecords()
	if err != nil {
		b.logger.WithError(err).Error("Failed to get attendance records")
		return
	}

	if len(records) == 0 {
		return // No new records
	}

	b.logger.WithField("count", len(records)).Info("Processing new attendance records")

	for _, record := range records {
		b.processAttendanceRecord(record)
	}

	// Clear processed records from device
	if err := b.device.ClearAttendanceRecords(); err != nil {
		b.logger.WithError(err).Warn("Failed to clear attendance records from device")
	}
}

// processAttendanceRecord processes a single attendance record
func (b *BiometricAdapter) processAttendanceRecord(record AttendanceRecord) {
	// Convert device user ID to platform user ID
	platformUserID, err := b.resolvePlatformUserID(record.DeviceUserID)
	if err != nil {
		b.logger.WithError(err).WithField("device_user_id", record.DeviceUserID).Error("Failed to resolve platform user ID")
		return
	}

	// Determine event type based on status
	eventType := types.EventTypeEntry
	if record.Status == 1 {
		eventType = types.EventTypeExit
	}

	// Create hardware event
	event := types.RawHardwareEvent{
		ExternalUserID: fmt.Sprintf("device_%d", record.DeviceUserID),
		Timestamp:      record.Timestamp,
		EventType:      eventType,
		RawData: map[string]interface{}{
			"device_user_id":    record.DeviceUserID,
			"platform_user_id":  platformUserID,
			"verify_mode":       record.VerifyMode,
			"work_code":         record.WorkCode,
			"device_type":       b.config.DeviceType,
			"adapter_name":      b.name,
		},
	}

	// Send to platform via check-in API
	if err := b.platformClient.SubmitCheckin(platformUserID, eventType, record.Timestamp); err != nil {
		b.logger.WithError(err).Error("Failed to submit check-in to platform")
		return
	}

	// Send event to callback
	if b.eventCallback != nil {
		b.eventCallback(event)
	}

	b.logger.WithFields(logrus.Fields{
		"platform_user_id": platformUserID,
		"device_user_id":   record.DeviceUserID,
		"event_type":       eventType,
		"timestamp":        record.Timestamp,
	}).Info("Attendance record processed successfully")
}

// resolvePlatformUserID converts device user ID to platform user ID
func (b *BiometricAdapter) resolvePlatformUserID(deviceUserID int) (string, error) {
	// Get users from device
	users, err := b.device.GetUsers()
	if err != nil {
		return "", fmt.Errorf("failed to get users from device: %w", err)
	}

	// Find user by device user ID
	for _, user := range users {
		if user.DeviceUserID == deviceUserID {
			return user.PlatformUserID, nil
		}
	}

	return "", fmt.Errorf("platform user ID not found for device user ID %d", deviceUserID)
}

// SubmitCheckin submits a check-in event to the platform
func (pc *PlatformClient) SubmitCheckin(userID string, eventType string, timestamp time.Time) error {
	url := fmt.Sprintf("%s/api/v1/checkin", pc.baseURL)
	
	payload := map[string]interface{}{
		"memberId":   userID,
		"authMethod": "FINGERPRINT",
		"eventType":  eventType,
		"timestamp":  timestamp.Format(time.RFC3339),
	}

	return pc.makeRequest("POST", url, payload, nil)
}

// makeRequest makes an HTTP request to the platform
func (pc *PlatformClient) makeRequest(method, url string, payload interface{}, result interface{}) error {
	var body []byte
	var err error

	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	req.Header.Set("X-Device-ID", pc.deviceID)
	req.Header.Set("X-Device-Key", pc.deviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := pc.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error: %d %s", resp.StatusCode, resp.Status)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}