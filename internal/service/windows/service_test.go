package windows

import (
	"context"
	"runtime"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service tests only run on Windows")
	}
	
	cfg := &config.Config{}
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		return nil
	}
	
	service := NewService(cfg, bridgeFunc)
	
	assert.NotNil(t, service)
	assert.Equal(t, cfg, service.config)
	assert.NotNil(t, service.logger)
	assert.NotNil(t, service.ctx)
	assert.NotNil(t, service.cancel)
	assert.NotNil(t, service.bridgeFunc)
}

func TestIsWindowsService(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service tests only run on Windows")
	}
	
	// This should return false when running in test environment
	isService, err := IsWindowsService()
	require.NoError(t, err)
	assert.False(t, isService, "Should not be running as service in test environment")
}

func TestServiceConstants(t *testing.T) {
	assert.Equal(t, "GymDoorBridge", ServiceName)
	assert.Equal(t, "Gym Door Access Bridge", ServiceDisplayName)
	assert.Equal(t, "Connects gym door access hardware to SaaS platform", ServiceDescription)
}

// Mock bridge function for testing
func mockBridgeFunction(ctx context.Context, cfg *config.Config) error {
	// Simulate bridge operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

// Mock bridge function that returns an error
func mockBridgeFunctionWithError(ctx context.Context, cfg *config.Config) error {
	return assert.AnError
}

func TestServiceExecution(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service tests only run on Windows")
	}
	
	tests := []struct {
		name       string
		bridgeFunc func(context.Context, *config.Config) error
		expectErr  bool
	}{
		{
			name:       "successful bridge function",
			bridgeFunc: mockBridgeFunction,
			expectErr:  false,
		},
		{
			name:       "bridge function with error",
			bridgeFunc: mockBridgeFunctionWithError,
			expectErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			service := NewService(cfg, tt.bridgeFunc)
			
			// Test service creation
			assert.NotNil(t, service)
			
			// Test context cancellation
			service.cancel()
			select {
			case <-service.ctx.Done():
				// Context should be cancelled
			case <-time.After(1 * time.Second):
				t.Error("Context was not cancelled")
			}
		})
	}
}

func TestServiceLifecycle(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service tests only run on Windows")
	}
	
	cfg := &config.Config{}
	service := NewService(cfg, mockBridgeFunction)
	
	// Test initial state
	assert.NotNil(t, service.ctx)
	assert.NotNil(t, service.cancel)
	
	// Test context cancellation
	originalCtx := service.ctx
	service.cancel()
	
	select {
	case <-originalCtx.Done():
		// Expected - context should be cancelled
	case <-time.After(1 * time.Second):
		t.Error("Context cancellation timeout")
	}
}

// Integration test for service configuration
func TestServiceLifecycleIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service tests only run on Windows")
	}
	
	// Test service configuration
	cfg := &config.Config{}
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		// Simulate some work
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	
	service := NewService(cfg, bridgeFunc)
	require.NotNil(t, service)
	
	// Test that service can be created and cancelled
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Simulate service execution
		time.Sleep(10 * time.Millisecond)
		service.cancel()
	}()
	
	select {
	case <-done:
		// Expected completion
	case <-time.After(1 * time.Second):
		t.Error("Service integration test timeout")
	}
	
	// Verify context is cancelled
	select {
	case <-service.ctx.Done():
		// Expected
	default:
		t.Error("Service context should be cancelled")
	}
}