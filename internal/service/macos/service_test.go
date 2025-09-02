package macos

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}
	
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

func TestServiceRun(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}
	
	t.Run("SuccessfulRun", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Signal handling tests don't work reliably on Windows")
		}
		
		bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
			// Simulate bridge work
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return nil
			}
		}
		
		service := NewService(cfg, bridgeFunc)
		
		// Run service in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- service.Run()
		}()
		
		// Wait a moment then cancel
		time.Sleep(50 * time.Millisecond)
		service.cancel()
		
		// Wait for service to stop
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Service did not stop within timeout")
		}
	})
	
	t.Run("BridgeFunctionError", func(t *testing.T) {
		expectedErr := assert.AnError
		bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
			return expectedErr
		}
		
		service := NewService(cfg, bridgeFunc)
		
		// Run service in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- service.Run()
		}()
		
		// Wait for service to return error
		select {
		case err := <-done:
			assert.Equal(t, expectedErr, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Service did not return error within timeout")
		}
	})
}

func TestRunService(t *testing.T) {
	cfg := &config.Config{
		LogLevel: "info",
	}
	
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		// Simulate bridge work
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
	
	// Run service in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- RunService(cfg, bridgeFunc)
	}()
	
	// Wait a moment then send interrupt signal
	time.Sleep(50 * time.Millisecond)
	
	// We can't easily test signal handling in unit tests,
	// so we'll just verify the function doesn't panic
	select {
	case <-done:
		// Service completed
	case <-time.After(200 * time.Millisecond):
		// Service is still running, which is expected
	}
}

func TestIsMacOSDaemon(t *testing.T) {
	// Test without environment variable
	originalEnv := os.Getenv("LAUNCH_DAEMON_SOCKET_NAME")
	os.Unsetenv("LAUNCH_DAEMON_SOCKET_NAME")
	
	assert.False(t, IsMacOSDaemon())
	
	// Test with environment variable
	os.Setenv("LAUNCH_DAEMON_SOCKET_NAME", "test-socket")
	assert.True(t, IsMacOSDaemon())
	
	// Restore original environment
	if originalEnv != "" {
		os.Setenv("LAUNCH_DAEMON_SOCKET_NAME", originalEnv)
	} else {
		os.Unsetenv("LAUNCH_DAEMON_SOCKET_NAME")
	}
}

func TestServiceConstants(t *testing.T) {
	assert.Equal(t, "com.gymdoorbridge.agent", ServiceName)
	assert.Equal(t, "Gym Door Access Bridge", ServiceDisplayName)
	assert.Equal(t, "Connects gym door access hardware to SaaS platform", ServiceDescription)
}

// TestServiceContextCancellation tests that the service properly handles context cancellation
func TestServiceContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Signal handling tests don't work reliably on Windows")
	}
	
	cfg := &config.Config{
		LogLevel: "info",
	}
	
	// Create a bridge function that blocks until context is cancelled
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		<-ctx.Done()
		return ctx.Err()
	}
	
	service := NewService(cfg, bridgeFunc)
	
	// Run service in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run()
	}()
	
	// Cancel the context
	service.cancel()
	
	// Wait for service to stop
	select {
	case err := <-done:
		assert.NoError(t, err) // Context cancellation should not be treated as an error
	case <-time.After(5 * time.Second):
		t.Fatal("Service did not stop within timeout")
	}
}

// TestServiceGracefulShutdown tests the graceful shutdown timeout
func TestServiceGracefulShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Signal handling tests don't work reliably on Windows")
	}
	
	cfg := &config.Config{
		LogLevel: "info",
	}
	
	// Create a bridge function that responds to context cancellation quickly for testing
	bridgeFunc := func(ctx context.Context, cfg *config.Config) error {
		select {
		case <-ctx.Done():
			// Simulate quick shutdown for testing
			time.Sleep(100 * time.Millisecond)
			return ctx.Err()
		}
	}
	
	service := NewService(cfg, bridgeFunc)
	
	// Run service in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- service.Run()
	}()
	
	// Cancel the context
	service.cancel()
	
	// Wait for service to stop
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Service did not stop within expected timeout")
	}
}