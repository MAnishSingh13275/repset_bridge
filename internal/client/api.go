package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PairRequest represents a device pairing request
type PairRequest struct {
	PairCode   string      `json:"pairCode"`
	DeviceInfo *DeviceInfo `json:"deviceInfo"`
}

// DeviceInfo contains information about the device being paired
type DeviceInfo struct {
	Hostname string `json:"hostname"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
	Tier     string `json:"tier"`
}

// PairResponse represents the response from device pairing
type PairResponse struct {
	DeviceID string       `json:"deviceId"`
	DeviceKey string      `json:"deviceKey"`
	Config   *DeviceConfig `json:"config"`
}

// DeviceConfig contains configuration received from the cloud
type DeviceConfig struct {
	HeartbeatInterval int `json:"heartbeatInterval"`
	QueueMaxSize      int `json:"queueMaxSize"`
	UnlockDuration    int `json:"unlockDuration"`
}

// CheckinEvent represents a single check-in event
type CheckinEvent struct {
	EventID        string `json:"eventId"`
	ExternalUserID string `json:"externalUserId"`
	Timestamp      string `json:"timestamp"`
	EventType      string `json:"eventType"`
	IsSimulated    bool   `json:"isSimulated"`
	DeviceID       string `json:"deviceId"`
}

// CheckinRequest represents a batch of check-in events
type CheckinRequest struct {
	Events []CheckinEvent `json:"events"`
}

// HeartbeatRequest represents a device heartbeat
type HeartbeatRequest struct {
	Status        string      `json:"status"`
	Tier          string      `json:"tier"`
	QueueDepth    int         `json:"queueDepth"`
	LastEventTime string      `json:"lastEventTime,omitempty"`
	SystemInfo    *SystemInfo `json:"systemInfo,omitempty"`
}

// SystemInfo contains system resource information
type SystemInfo struct {
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	DiskSpace   float64 `json:"diskSpace"`
}

// PairDevice pairs the device with the cloud using a pair code
func (c *HTTPClient) PairDevice(ctx context.Context, pairCode string, deviceInfo *DeviceInfo) (*PairResponse, error) {
	req := &Request{
		Method: http.MethodPost,
		Path:   "/api/v1/devices/pair",
		Body: &PairRequest{
			PairCode:   pairCode,
			DeviceInfo: deviceInfo,
		},
		RequireAuth: false, // Pairing doesn't require authentication
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("pairing request failed: %w", err)
	}

	var pairResp PairResponse
	if err := json.Unmarshal(resp.Body, &pairResp); err != nil {
		return nil, fmt.Errorf("failed to parse pairing response: %w", err)
	}

	c.logger.Info("Device paired successfully", "device_id", pairResp.DeviceID)
	return &pairResp, nil
}

// SubmitCheckinEvents submits a batch of check-in events to the cloud
func (c *HTTPClient) SubmitCheckinEvents(ctx context.Context, events []CheckinEvent) error {
	if len(events) == 0 {
		return nil // Nothing to submit
	}

	req := &Request{
		Method: http.MethodPost,
		Path:   "/api/v1/checkin",
		Body: &CheckinRequest{
			Events: events,
		},
		RequireAuth: true,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("checkin submission failed: %w", err)
	}

	c.logger.Info("Check-in events submitted successfully", "count", len(events))
	return nil
}

// SendHeartbeat sends a heartbeat to the cloud
func (c *HTTPClient) SendHeartbeat(ctx context.Context, heartbeat *HeartbeatRequest) error {
	req := &Request{
		Method: http.MethodPost,
		Path:   "/api/v1/devices/heartbeat",
		Body:   heartbeat,
		RequireAuth: true,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat failed: %w", err)
	}

	c.logger.Debug("Heartbeat sent successfully")
	return nil
}

// OpenDoor sends a door open command (for remote door control)
func (c *HTTPClient) OpenDoor(ctx context.Context) error {
	req := &Request{
		Method:      http.MethodPost,
		Path:        "/open-door",
		RequireAuth: true,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("door open command failed: %w", err)
	}

	c.logger.Info("Door open command sent successfully")
	return nil
}

// GetDeviceConfig retrieves the current device configuration from the cloud
func (c *HTTPClient) GetDeviceConfig(ctx context.Context) (*DeviceConfig, error) {
	req := &Request{
		Method:      http.MethodGet,
		Path:        "/api/v1/devices/config",
		RequireAuth: true,
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("config retrieval failed: %w", err)
	}

	var config DeviceConfig
	if err := json.Unmarshal(resp.Body, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config response: %w", err)
	}

	return &config, nil
}

// CheckConnectivity performs a simple connectivity check
func (c *HTTPClient) CheckConnectivity(ctx context.Context) error {
	req := &Request{
		Method:      http.MethodGet,
		Path:        "/api/v1/health",
		RequireAuth: false,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("connectivity check failed: %w", err)
	}

	return nil
}

// DeviceStatusRequest represents a device status check request
type DeviceStatusRequest struct {
	RequestID string `json:"requestId,omitempty"`
}

// DeviceStatusResponse represents the response from device status check
type DeviceStatusResponse struct {
	Status        string      `json:"status"`
	LastSeen      string      `json:"lastSeen"`
	QueueDepth    int         `json:"queueDepth"`
	SystemInfo    *SystemInfo `json:"systemInfo,omitempty"`
	ConnectedDevices []string `json:"connectedDevices,omitempty"`
}

// SendDeviceStatus sends device status to the cloud
func (c *HTTPClient) SendDeviceStatus(ctx context.Context, statusReq *DeviceStatusRequest) (*DeviceStatusResponse, error) {
	req := &Request{
		Method:      http.MethodPost,
		Path:        "/api/v1/devices/status",
		Body:        statusReq,
		RequireAuth: true,
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("device status request failed: %w", err)
	}

	var statusResp DeviceStatusResponse
	if err := json.Unmarshal(resp.Body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	c.logger.Debug("Device status sent successfully")
	return &statusResp, nil
}

// TriggerHeartbeat manually triggers a heartbeat
func (c *HTTPClient) TriggerHeartbeat(ctx context.Context) error {
	req := &Request{
		Method:      http.MethodPost,
		Path:        "/api/v1/devices/heartbeat/trigger",
		RequireAuth: true,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("heartbeat trigger failed: %w", err)
	}

	c.logger.Info("Heartbeat triggered successfully")
	return nil
}

// SubmitEvents submits events to the general events endpoint
func (c *HTTPClient) SubmitEvents(ctx context.Context, events []CheckinEvent) error {
	if len(events) == 0 {
		return nil // Nothing to submit
	}

	req := &Request{
		Method: http.MethodPost,
		Path:   "/api/v1/events",
		Body: &CheckinRequest{
			Events: events,
		},
		RequireAuth: true,
	}

	_, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("events submission failed: %w", err)
	}

	c.logger.Info("Events submitted successfully", "count", len(events))
	return nil
}