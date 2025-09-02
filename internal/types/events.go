package types

import (
	"time"
)

// RawHardwareEvent represents the raw event data from hardware adapters
type RawHardwareEvent struct {
	ExternalUserID string                 `json:"externalUserId"`
	Timestamp      time.Time              `json:"timestamp"`
	EventType      string                 `json:"eventType"` // "entry", "exit", "denied"
	RawData        map[string]interface{} `json:"rawData,omitempty"`
}

// StandardEvent represents the normalized event format for cloud submission
type StandardEvent struct {
	EventID        string    `json:"eventId"`
	ExternalUserID string    `json:"externalUserId"`
	InternalUserID string    `json:"internalUserId,omitempty"` // Resolved from external user mapping
	Timestamp      time.Time `json:"timestamp"`
	EventType      string    `json:"eventType"` // "entry", "exit", "denied"
	IsSimulated    bool      `json:"isSimulated"`
	DeviceID       string    `json:"deviceId"`
	RawData        map[string]interface{} `json:"rawData,omitempty"`
}

// EventType constants for type safety
const (
	EventTypeEntry  = "entry"
	EventTypeExit   = "exit"
	EventTypeDenied = "denied"
)

// IsValidEventType checks if the provided event type is valid
func IsValidEventType(eventType string) bool {
	switch eventType {
	case EventTypeEntry, EventTypeExit, EventTypeDenied:
		return true
	default:
		return false
	}
}

// AdapterConfig holds configuration for hardware adapters
type AdapterConfig struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}

// AdapterStatus represents the current status of a hardware adapter
type AdapterStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"` // "active", "error", "disabled", "initializing"
	LastEvent    time.Time `json:"lastEvent,omitempty"`
	ErrorMessage string    `json:"errorMessage,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Status constants for adapter status
const (
	StatusActive       = "active"
	StatusError        = "error"
	StatusDisabled     = "disabled"
	StatusInitializing = "initializing"
)

// EventCallback defines the function signature for event callbacks
type EventCallback func(event RawHardwareEvent)