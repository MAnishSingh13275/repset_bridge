package database

import (
	"time"
)

// EventQueue represents a queued event in the database
type EventQueue struct {
	ID             int64     `json:"id"`
	EventID        string    `json:"event_id"`
	ExternalUserID string    `json:"external_user_id"`
	Timestamp      time.Time `json:"timestamp"`
	EventType      string    `json:"event_type"`
	IsSimulated    bool      `json:"is_simulated"`
	DeviceID       string    `json:"device_id"`
	RawData        string    `json:"raw_data,omitempty"` // Encrypted JSON
	CreatedAt      time.Time `json:"created_at"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	RetryCount     int       `json:"retry_count"`
}

// DeviceConfig represents a configuration key-value pair
type DeviceConfig struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"` // May be encrypted
	UpdatedAt time.Time `json:"updated_at"`
}

// AdapterStatus represents the status of a hardware adapter
type AdapterStatus struct {
	AdapterName  string     `json:"adapter_name"`
	Status       string     `json:"status"`
	LastEvent    *time.Time `json:"last_event,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// EventType constants
const (
	EventTypeEntry  = "entry"
	EventTypeExit   = "exit"
	EventTypeDenied = "denied"
)

// AdapterStatusType constants
const (
	AdapterStatusActive   = "active"
	AdapterStatusError    = "error"
	AdapterStatusDisabled = "disabled"
)

// ExternalUserMapping represents a mapping between external user ID and internal user ID
type ExternalUserMapping struct {
	ID             int64     `json:"id"`
	ExternalUserID string    `json:"external_user_id"`
	InternalUserID string    `json:"internal_user_id"`
	UserName       string    `json:"user_name,omitempty"`       // Optional display name
	Notes          string    `json:"notes,omitempty"`           // Optional notes
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}