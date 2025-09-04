package biometric

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

// SimulatorDevice simulates a biometric device for testing
type SimulatorDevice struct {
	config           map[string]string
	logger           *logrus.Logger
	connected        bool
	users            []DeviceUser
	attendanceRecords []AttendanceRecord
	lastPoll         time.Time
}

// NewSimulatorDevice creates a new simulator device
func NewSimulatorDevice(config map[string]string, logger *logrus.Logger) *SimulatorDevice {
	return &SimulatorDevice{
		config:    config,
		logger:    logger,
		connected: false,
		users:     []DeviceUser{},
		attendanceRecords: []AttendanceRecord{},
		lastPoll:  time.Now(),
	}
}

// Connect connects to the simulator device
func (s *SimulatorDevice) Connect() error {
	s.connected = true
	s.logger.Info("Biometric simulator connected")
	
	// Pre-populate with some test users
	s.users = []DeviceUser{
		{
			DeviceUserID:   1,
			PlatformUserID: "user123",
			Name:           "John Doe",
			Privilege:      0,
		},
		{
			DeviceUserID:   2,
			PlatformUserID: "user456",
			Name:           "Jane Smith",
			Privilege:      0,
		},
	}
	
	// Start generating random attendance records
	go s.generateRandomAttendance()
	
	return nil
}

// Disconnect disconnects from the simulator device
func (s *SimulatorDevice) Disconnect() error {
	s.connected = false
	s.logger.Info("Biometric simulator disconnected")
	return nil
}

// IsConnected returns whether the device is connected
func (s *SimulatorDevice) IsConnected() bool {
	return s.connected
}

// GetStatus returns the device status
func (s *SimulatorDevice) GetStatus() string {
	if s.connected {
		return fmt.Sprintf("Simulator Connected - %d users, %d pending records", 
			len(s.users), len(s.attendanceRecords))
	}
	return "Simulator Disconnected"
}

// EnrollUser enrolls a user on the simulator device
func (s *SimulatorDevice) EnrollUser(platformUserID string, deviceUserID int, name string) error {
	if !s.connected {
		return fmt.Errorf("device not connected")
	}

	// Check if user already exists
	for _, user := range s.users {
		if user.DeviceUserID == deviceUserID {
			return fmt.Errorf("user with device ID %d already exists", deviceUserID)
		}
	}

	// Add user
	user := DeviceUser{
		DeviceUserID:   deviceUserID,
		PlatformUserID: platformUserID,
		Name:           name,
		Privilege:      0,
	}

	s.users = append(s.users, user)
	
	s.logger.WithFields(logrus.Fields{
		"platform_user_id": platformUserID,
		"device_user_id":   deviceUserID,
		"name":             name,
	}).Info("User enrolled on simulator device")

	return nil
}

// DeleteUser deletes a user from the simulator device
func (s *SimulatorDevice) DeleteUser(deviceUserID int) error {
	if !s.connected {
		return fmt.Errorf("device not connected")
	}

	// Find and remove user
	for i, user := range s.users {
		if user.DeviceUserID == deviceUserID {
			s.users = append(s.users[:i], s.users[i+1:]...)
			s.logger.WithField("device_user_id", deviceUserID).Info("User deleted from simulator device")
			return nil
		}
	}

	return fmt.Errorf("user with device ID %d not found", deviceUserID)
}

// GetUsers gets all users from the simulator device
func (s *SimulatorDevice) GetUsers() ([]DeviceUser, error) {
	if !s.connected {
		return nil, fmt.Errorf("device not connected")
	}

	return s.users, nil
}

// GetNewAttendanceRecords gets new attendance records from the simulator device
func (s *SimulatorDevice) GetNewAttendanceRecords() ([]AttendanceRecord, error) {
	if !s.connected {
		return nil, fmt.Errorf("device not connected")
	}

	// Return and clear pending records
	records := make([]AttendanceRecord, len(s.attendanceRecords))
	copy(records, s.attendanceRecords)
	s.attendanceRecords = []AttendanceRecord{}

	if len(records) > 0 {
		s.logger.WithField("count", len(records)).Info("Retrieved attendance records from simulator")
	}

	return records, nil
}

// ClearAttendanceRecords clears attendance records from the simulator device
func (s *SimulatorDevice) ClearAttendanceRecords() error {
	if !s.connected {
		return fmt.Errorf("device not connected")
	}

	s.attendanceRecords = []AttendanceRecord{}
	s.logger.Info("Cleared attendance records from simulator")
	return nil
}

// GetDeviceInfo gets device information
func (s *SimulatorDevice) GetDeviceInfo() (*DeviceInfo, error) {
	if !s.connected {
		return nil, fmt.Errorf("device not connected")
	}

	return &DeviceInfo{
		SerialNumber:    "SIM-001",
		DeviceModel:     "Biometric Simulator",
		FirmwareVer:     "1.0.0",
		UserCount:       len(s.users),
		AttendanceCount: len(s.attendanceRecords),
		FingerCapacity:  1000,
	}, nil
}

// GetDeviceTime gets the device time
func (s *SimulatorDevice) GetDeviceTime() (time.Time, error) {
	if !s.connected {
		return time.Time{}, fmt.Errorf("device not connected")
	}

	return time.Now(), nil
}

// SetDeviceTime sets the device time
func (s *SimulatorDevice) SetDeviceTime(t time.Time) error {
	if !s.connected {
		return fmt.Errorf("device not connected")
	}

	s.logger.WithField("time", t).Info("Simulator device time updated")
	return nil
}

// generateRandomAttendance generates random attendance records for testing
func (s *SimulatorDevice) generateRandomAttendance() {
	ticker := time.NewTicker(30 * time.Second) // Generate record every 30 seconds
	defer ticker.Stop()

	for {
		if !s.connected {
			return
		}

		select {
		case <-ticker.C:
			if len(s.users) == 0 {
				continue
			}

			// 20% chance to generate an attendance record
			if rand.Float32() > 0.2 {
				continue
			}

			// Pick random user
			user := s.users[rand.Intn(len(s.users))]

			// Generate random attendance record
			record := AttendanceRecord{
				DeviceUserID: user.DeviceUserID,
				Timestamp:    time.Now(),
				Status:       rand.Intn(2), // 0=check-in, 1=check-out
				VerifyMode:   1,            // 1=fingerprint
				WorkCode:     0,
			}

			s.attendanceRecords = append(s.attendanceRecords, record)

			s.logger.WithFields(logrus.Fields{
				"device_user_id": record.DeviceUserID,
				"user_name":      user.Name,
				"status":         record.Status,
				"timestamp":      record.Timestamp,
			}).Info("Generated simulated attendance record")
		}
	}
}