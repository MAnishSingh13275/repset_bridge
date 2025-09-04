package biometric

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// ESSLDevice handles ESSL biometric devices
type ESSLDevice struct {
	config     map[string]string
	logger     *logrus.Logger
	conn       net.Conn
	connected  bool
	ipAddress  string
	port       int
	deviceID   int
	password   string
	sessionID  int
	replyID    int
}

// ESSL Protocol Constants
const (
	CMD_CONNECT    = 1000
	CMD_EXIT       = 1001
	CMD_ENABLEDEVICE = 1002
	CMD_DISABLEDEVICE = 1003
	CMD_ACK_OK     = 2000
	CMD_ACK_ERROR  = 2001
	CMD_ACK_DATA   = 2002
	CMD_ACK_RETRY  = 2003
	CMD_ACK_REPEAT = 2004
	CMD_ACK_UNAUTH = 2005
	
	CMD_USER_WRQ      = 8
	CMD_USERTEMP_RRQ  = 9
	CMD_USERTEMP_WRQ  = 10
	CMD_OPTIONS_RRQ   = 11
	CMD_ATTLOG_RRQ    = 13
	CMD_CLEAR_DATA    = 14
	CMD_CLEAR_ATTLOG  = 15
	CMD_DELETE_USER   = 18
	CMD_DELETE_USERTEMP = 19
	CMD_CLEAR_ADMIN   = 20
	CMD_USERGRP_RRQ   = 21
	CMD_USERGRP_WRQ   = 22
	CMD_USERTZ_RRQ    = 23
	CMD_USERTZ_WRQ    = 24
	CMD_GRPTZ_RRQ     = 25
	CMD_GRPTZ_WRQ     = 26
	CMD_TZ_RRQ        = 27
	CMD_TZ_WRQ        = 28
	CMD_ULG_RRQ       = 29
	CMD_ULG_WRQ       = 30
	CMD_UNLOCK        = 31
	CMD_CLEAR_ACC     = 32
	CMD_CLEAR_OPLOG   = 33
	CMD_OPLOG_RRQ     = 34
	CMD_GET_FREE_SIZES = 50
	CMD_ENABLE_CLOCK  = 57
	CMD_STARTVERIFY   = 60
	CMD_STARTENROLL   = 61
	CMD_CANCELCAPTURE = 62
	CMD_STATE_RRQ     = 64
	CMD_WRITE_LCD     = 66
	CMD_CLEAR_LCD     = 67
	CMD_GET_PINWIDTH  = 69
	CMD_SMS_WRQ       = 70
	CMD_SMS_RRQ       = 71
	CMD_DELETE_SMS    = 72
	CMD_UDATA_WRQ     = 73
	CMD_DELETE_UDATA  = 74
	CMD_DOORSTATE_RRQ = 75
	CMD_WRITE_MIFARE  = 76
	CMD_EMPTY_MIFARE  = 78
	CMD_VERIFY_WRQ    = 79
	CMD_VERIFY_RRQ    = 80
	CMD_TMP_WRITE     = 87
	CMD_CHECKSUM_BUFFER = 119
	CMD_DEL_FPTMP     = 134
	CMD_GET_TIME      = 201
	CMD_SET_TIME      = 202
	CMD_REG_EVENT     = 500
)

// NewESSLDevice creates a new ESSL device
func NewESSLDevice(config map[string]string, logger *logrus.Logger) *ESSLDevice {
	ipAddress := config["ip_address"]
	if ipAddress == "" {
		ipAddress = "192.168.1.100"
	}

	port, _ := strconv.Atoi(config["port"])
	if port == 0 {
		port = 4370 // ESSL default port
	}

	deviceID, _ := strconv.Atoi(config["device_id"])
	if deviceID == 0 {
		deviceID = 1
	}

	password := config["password"]
	if password == "" {
		password = "0"
	}

	return &ESSLDevice{
		config:    config,
		logger:    logger,
		connected: false,
		ipAddress: ipAddress,
		port:      port,
		deviceID:  deviceID,
		password:  password,
		sessionID: 0,
		replyID:   0,
	}
}

// Connect connects to the ESSL device
func (e *ESSLDevice) Connect() error {
	e.logger.WithFields(logrus.Fields{
		"ip":        e.ipAddress,
		"port":      e.port,
		"device_id": e.deviceID,
	}).Info("Connecting to ESSL device")

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", e.ipAddress, e.port), 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to ESSL device: %w", err)
	}

	e.conn = conn
	e.connected = true

	// Send connect command
	if err := e.sendCommand(CMD_CONNECT, []byte{}); err != nil {
		e.Disconnect()
		return fmt.Errorf("failed to send connect command: %w", err)
	}

	// Read response
	response, err := e.readResponse()
	if err != nil {
		e.Disconnect()
		return fmt.Errorf("failed to read connect response: %w", err)
	}

	if response.Command != CMD_ACK_OK {
		e.Disconnect()
		return fmt.Errorf("connect command failed with response: %d", response.Command)
	}

	e.sessionID = int(response.SessionID)
	e.logger.WithField("session_id", e.sessionID).Info("Connected to ESSL device successfully")

	return nil
}

// Disconnect disconnects from the ESSL device
func (e *ESSLDevice) Disconnect() error {
	if !e.connected {
		return nil
	}

	if e.conn != nil {
		// Send exit command
		e.sendCommand(CMD_EXIT, []byte{})
		e.conn.Close()
	}

	e.connected = false
	e.sessionID = 0
	e.logger.Info("Disconnected from ESSL device")

	return nil
}

// IsConnected returns whether the device is connected
func (e *ESSLDevice) IsConnected() bool {
	return e.connected
}

// GetStatus returns the device status
func (e *ESSLDevice) GetStatus() string {
	if e.connected {
		return fmt.Sprintf("Connected to %s:%d (Session: %d)", e.ipAddress, e.port, e.sessionID)
	}
	return "Disconnected"
}

// EnrollUser enrolls a user on the ESSL device
func (e *ESSLDevice) EnrollUser(platformUserID string, deviceUserID int, name string) error {
	if !e.connected {
		return fmt.Errorf("device not connected")
	}

	e.logger.WithFields(logrus.Fields{
		"platform_user_id": platformUserID,
		"device_user_id":   deviceUserID,
		"name":             name,
	}).Info("Enrolling user on ESSL device")

	// TODO: Implement ESSL user enrollment protocol
	// This involves sending user data and fingerprint templates
	// The exact protocol depends on the ESSL device model

	return fmt.Errorf("ESSL user enrollment not yet implemented")
}

// DeleteUser deletes a user from the ESSL device
func (e *ESSLDevice) DeleteUser(deviceUserID int) error {
	if !e.connected {
		return fmt.Errorf("device not connected")
	}

	// Create delete user command data
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, uint16(deviceUserID))

	if err := e.sendCommand(CMD_DELETE_USER, data); err != nil {
		return fmt.Errorf("failed to send delete user command: %w", err)
	}

	response, err := e.readResponse()
	if err != nil {
		return fmt.Errorf("failed to read delete user response: %w", err)
	}

	if response.Command != CMD_ACK_OK {
		return fmt.Errorf("delete user command failed with response: %d", response.Command)
	}

	e.logger.WithField("device_user_id", deviceUserID).Info("User deleted from ESSL device")
	return nil
}

// GetUsers gets all users from the ESSL device
func (e *ESSLDevice) GetUsers() ([]DeviceUser, error) {
	if !e.connected {
		return nil, fmt.Errorf("device not connected")
	}

	// TODO: Implement ESSL get users protocol
	// This involves reading user data from the device

	return []DeviceUser{}, fmt.Errorf("ESSL get users not yet implemented")
}

// GetNewAttendanceRecords gets new attendance records from the ESSL device
func (e *ESSLDevice) GetNewAttendanceRecords() ([]AttendanceRecord, error) {
	if !e.connected {
		return nil, fmt.Errorf("device not connected")
	}

	// Send attendance log read request
	if err := e.sendCommand(CMD_ATTLOG_RRQ, []byte{}); err != nil {
		return nil, fmt.Errorf("failed to send attendance log request: %w", err)
	}

	response, err := e.readResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to read attendance log response: %w", err)
	}

	if response.Command == CMD_ACK_OK && len(response.Data) == 0 {
		// No new records
		return []AttendanceRecord{}, nil
	}

	if response.Command != CMD_ACK_DATA {
		return nil, fmt.Errorf("attendance log request failed with response: %d", response.Command)
	}

	// Parse attendance records from response data
	records := e.parseAttendanceRecords(response.Data)
	
	e.logger.WithField("count", len(records)).Info("Retrieved attendance records from ESSL device")
	return records, nil
}

// ClearAttendanceRecords clears attendance records from the ESSL device
func (e *ESSLDevice) ClearAttendanceRecords() error {
	if !e.connected {
		return fmt.Errorf("device not connected")
	}

	if err := e.sendCommand(CMD_CLEAR_ATTLOG, []byte{}); err != nil {
		return fmt.Errorf("failed to send clear attendance log command: %w", err)
	}

	response, err := e.readResponse()
	if err != nil {
		return fmt.Errorf("failed to read clear attendance log response: %w", err)
	}

	if response.Command != CMD_ACK_OK {
		return fmt.Errorf("clear attendance log command failed with response: %d", response.Command)
	}

	e.logger.Info("Cleared attendance records from ESSL device")
	return nil
}

// GetDeviceInfo gets device information
func (e *ESSLDevice) GetDeviceInfo() (*DeviceInfo, error) {
	if !e.connected {
		return nil, fmt.Errorf("device not connected")
	}

	// TODO: Implement ESSL device info retrieval
	// This involves reading device status and capabilities

	return &DeviceInfo{
		SerialNumber:    "ESSL-" + e.ipAddress,
		DeviceModel:     "ESSL Device",
		FirmwareVer:     "Unknown",
		UserCount:       0,
		AttendanceCount: 0,
		FingerCapacity:  1000,
	}, nil
}

// GetDeviceTime gets the device time
func (e *ESSLDevice) GetDeviceTime() (time.Time, error) {
	if !e.connected {
		return time.Time{}, fmt.Errorf("device not connected")
	}

	if err := e.sendCommand(CMD_GET_TIME, []byte{}); err != nil {
		return time.Time{}, fmt.Errorf("failed to send get time command: %w", err)
	}

	response, err := e.readResponse()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read get time response: %w", err)
	}

	if response.Command != CMD_ACK_OK || len(response.Data) < 4 {
		return time.Time{}, fmt.Errorf("get time command failed")
	}

	// Parse time from response data (Unix timestamp)
	timestamp := binary.LittleEndian.Uint32(response.Data[:4])
	deviceTime := time.Unix(int64(timestamp), 0)

	return deviceTime, nil
}

// SetDeviceTime sets the device time
func (e *ESSLDevice) SetDeviceTime(t time.Time) error {
	if !e.connected {
		return fmt.Errorf("device not connected")
	}

	// Create time data (Unix timestamp)
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(t.Unix()))

	if err := e.sendCommand(CMD_SET_TIME, data); err != nil {
		return fmt.Errorf("failed to send set time command: %w", err)
	}

	response, err := e.readResponse()
	if err != nil {
		return fmt.Errorf("failed to read set time response: %w", err)
	}

	if response.Command != CMD_ACK_OK {
		return fmt.Errorf("set time command failed with response: %d", response.Command)
	}

	e.logger.WithField("time", t).Info("Device time updated")
	return nil
}

// ESSLResponse represents a response from the ESSL device
type ESSLResponse struct {
	Command   uint16
	Checksum  uint16
	SessionID uint16
	ReplyID   uint16
	Data      []byte
}

// sendCommand sends a command to the ESSL device
func (e *ESSLDevice) sendCommand(command uint16, data []byte) error {
	if e.conn == nil {
		return fmt.Errorf("connection not established")
	}

	e.replyID++

	// Build command packet
	packet := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint16(packet[0:2], command)
	binary.LittleEndian.PutUint16(packet[2:4], 0) // Checksum (calculated later)
	binary.LittleEndian.PutUint16(packet[4:6], uint16(e.sessionID))
	binary.LittleEndian.PutUint16(packet[6:8], uint16(e.replyID))
	
	if len(data) > 0 {
		copy(packet[8:], data)
	}

	// Calculate checksum
	checksum := e.calculateChecksum(packet)
	binary.LittleEndian.PutUint16(packet[2:4], checksum)

	// Send packet
	_, err := e.conn.Write(packet)
	return err
}

// readResponse reads a response from the ESSL device
func (e *ESSLDevice) readResponse() (*ESSLResponse, error) {
	if e.conn == nil {
		return nil, fmt.Errorf("connection not established")
	}

	// Set read timeout
	e.conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read header (8 bytes)
	header := make([]byte, 8)
	_, err := e.conn.Read(header)
	if err != nil {
		return nil, fmt.Errorf("failed to read response header: %w", err)
	}

	response := &ESSLResponse{
		Command:   binary.LittleEndian.Uint16(header[0:2]),
		Checksum:  binary.LittleEndian.Uint16(header[2:4]),
		SessionID: binary.LittleEndian.Uint16(header[4:6]),
		ReplyID:   binary.LittleEndian.Uint16(header[6:8]),
	}

	// Read data if present (for data responses)
	if response.Command == CMD_ACK_DATA {
		// Read data length (next 4 bytes)
		lengthBytes := make([]byte, 4)
		_, err := e.conn.Read(lengthBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to read data length: %w", err)
		}

		dataLength := binary.LittleEndian.Uint32(lengthBytes)
		if dataLength > 0 {
			response.Data = make([]byte, dataLength)
			_, err := e.conn.Read(response.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to read response data: %w", err)
			}
		}
	}

	return response, nil
}

// calculateChecksum calculates the checksum for a packet
func (e *ESSLDevice) calculateChecksum(packet []byte) uint16 {
	// Simple checksum calculation (sum of all bytes except checksum field)
	var sum uint16
	for i := 0; i < len(packet); i++ {
		if i < 2 || i >= 4 { // Skip checksum field
			sum += uint16(packet[i])
		}
	}
	return sum
}

// parseAttendanceRecords parses attendance records from raw data
func (e *ESSLDevice) parseAttendanceRecords(data []byte) []AttendanceRecord {
	var records []AttendanceRecord

	// Each attendance record is typically 16 bytes in ESSL format
	recordSize := 16
	recordCount := len(data) / recordSize

	for i := 0; i < recordCount; i++ {
		offset := i * recordSize
		recordData := data[offset : offset+recordSize]

		// Parse record fields (this is device-specific)
		deviceUserID := int(binary.LittleEndian.Uint16(recordData[0:2]))
		timestamp := time.Unix(int64(binary.LittleEndian.Uint32(recordData[4:8])), 0)
		status := int(recordData[8])
		verifyMode := int(recordData[9])
		workCode := int(recordData[10])

		record := AttendanceRecord{
			DeviceUserID: deviceUserID,
			Timestamp:    timestamp,
			Status:       status,
			VerifyMode:   verifyMode,
			WorkCode:     workCode,
		}

		records = append(records, record)
	}

	return records
}