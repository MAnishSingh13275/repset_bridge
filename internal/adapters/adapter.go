package adapters

import (
	"context"

	"gym-door-bridge/internal/types"
)

// HardwareAdapter defines the interface that all hardware adapters must implement
type HardwareAdapter interface {
	// Name returns the unique name of this adapter
	Name() string

	// Initialize sets up the adapter with the provided configuration
	Initialize(ctx context.Context, config types.AdapterConfig) error

	// StartListening begins listening for hardware events
	StartListening(ctx context.Context) error

	// StopListening stops listening for hardware events
	StopListening(ctx context.Context) error

	// UnlockDoor triggers door unlock for the specified duration
	UnlockDoor(ctx context.Context, durationMs int) error

	// GetStatus returns the current status of the adapter
	GetStatus() types.AdapterStatus

	// OnEvent registers a callback function to handle hardware events
	OnEvent(callback types.EventCallback)

	// IsHealthy returns true if the adapter is functioning properly
	IsHealthy() bool
}