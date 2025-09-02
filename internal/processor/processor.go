package processor

import (
	"context"

	"gym-door-bridge/internal/types"
)

// ProcessorConfig holds configuration for the event processor
type ProcessorConfig struct {
	DeviceID           string `json:"deviceId"`
	EnableDeduplication bool   `json:"enableDeduplication"`
	DeduplicationWindow int    `json:"deduplicationWindow"` // seconds
}

// ProcessingResult contains the result of event processing
type ProcessingResult struct {
	Event     types.StandardEvent `json:"event"`
	Processed bool                `json:"processed"`
	Reason    string              `json:"reason,omitempty"` // reason if not processed (e.g., "duplicate", "invalid")
}

// EventProcessor defines the interface for processing raw hardware events
type EventProcessor interface {
	// Initialize sets up the processor with the provided configuration
	Initialize(ctx context.Context, config ProcessorConfig) error

	// ProcessEvent converts a raw hardware event to a standard event
	// Returns the processed event and whether it should be queued
	ProcessEvent(ctx context.Context, rawEvent types.RawHardwareEvent) (ProcessingResult, error)

	// ValidateEvent checks if a raw event is valid for processing
	ValidateEvent(rawEvent types.RawHardwareEvent) error

	// GenerateEventID creates a unique event ID for the processed event
	GenerateEventID(rawEvent types.RawHardwareEvent) string

	// IsEventDuplicate checks if an event is a duplicate within the deduplication window
	IsEventDuplicate(ctx context.Context, rawEvent types.RawHardwareEvent) (bool, error)

	// GetStats returns processing statistics
	GetStats() ProcessorStats
}

// ProcessorStats contains statistics about event processing
type ProcessorStats struct {
	TotalProcessed   int64 `json:"totalProcessed"`
	TotalDuplicates  int64 `json:"totalDuplicates"`
	TotalInvalid     int64 `json:"totalInvalid"`
	LastProcessedAt  int64 `json:"lastProcessedAt"` // Unix timestamp
}

// ValidationError represents an event validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}