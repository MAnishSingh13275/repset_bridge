package simulator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"gym-door-bridge/internal/types"
)

func TestNewSimulatorAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)

	if adapter.Name() != "simulator" {
		t.Errorf("Expected adapter name 'simulator', got '%s'", adapter.Name())
	}

	status := adapter.GetStatus()
	if status.Status != types.StatusDisabled {
		t.Errorf("Expected initial status 'disabled', got '%s'", status.Status)
	}

	if !adapter.IsHealthy() == false { // Should be false when disabled
		t.Error("Expected adapter to be unhealthy when disabled")
	}
}

func TestSimulatorAdapter_Initialize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 5.0, // 5 seconds
			"simulatedUsers": []interface{}{
				"test_user_001",
				"test_user_002",
			},
		},
	}

	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	status := adapter.GetStatus()
	if status.Status != types.StatusActive {
		t.Errorf("Expected status 'active' after initialization, got '%s'", status.Status)
	}

	if !adapter.IsHealthy() {
		t.Error("Expected adapter to be healthy after initialization")
	}

	// Check if configuration was applied
	if adapter.eventInterval != 5*time.Second {
		t.Errorf("Expected event interval 5s, got %v", adapter.eventInterval)
	}

	if len(adapter.simulatedUsers) != 2 {
		t.Errorf("Expected 2 simulated users, got %d", len(adapter.simulatedUsers))
	}
}

func TestSimulatorAdapter_StartStopListening(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	// Initialize first
	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 1.0, // 1 second for faster testing
		},
	}
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Test starting without callback should fail
	err = adapter.StartListening(ctx)
	if err == nil {
		t.Error("Expected error when starting without event callback")
	}

	// Register callback and start
	var receivedEvents []types.RawHardwareEvent
	var eventMutex sync.Mutex

	adapter.OnEvent(func(event types.RawHardwareEvent) {
		eventMutex.Lock()
		receivedEvents = append(receivedEvents, event)
		eventMutex.Unlock()
	})

	err = adapter.StartListening(ctx)
	if err != nil {
		t.Fatalf("Failed to start listening: %v", err)
	}

	// Test starting again should fail
	err = adapter.StartListening(ctx)
	if err == nil {
		t.Error("Expected error when starting already listening adapter")
	}

	// Wait for at least one event
	time.Sleep(1500 * time.Millisecond)

	// Stop listening
	err = adapter.StopListening(ctx)
	if err != nil {
		t.Fatalf("Failed to stop listening: %v", err)
	}

	// Check that we received events
	eventMutex.Lock()
	eventCount := len(receivedEvents)
	eventMutex.Unlock()

	if eventCount == 0 {
		t.Error("Expected to receive at least one event")
	}

	// Test stopping again should not error
	err = adapter.StopListening(ctx)
	if err != nil {
		t.Errorf("Unexpected error when stopping already stopped adapter: %v", err)
	}
}

func TestSimulatorAdapter_UnlockDoor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	// Test unlock without initialization should fail
	err := adapter.UnlockDoor(ctx, 3000)
	if err == nil {
		t.Error("Expected error when unlocking door without initialization")
	}

	// Initialize adapter
	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
	}
	err = adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Test successful unlock
	start := time.Now()
	err = adapter.UnlockDoor(ctx, 1000) // 1 second
	if err != nil {
		t.Fatalf("Failed to unlock door: %v", err)
	}

	// Should complete quickly (simulation)
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("Unlock operation took too long: %v", elapsed)
	}

	// Test with cancelled context
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	err = adapter.UnlockDoor(cancelCtx, 1000)
	if err == nil {
		t.Error("Expected error when unlocking with cancelled context")
	}
}

func TestSimulatorAdapter_TriggerEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	// Initialize adapter
	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
	}
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Test trigger without callback should fail
	err = adapter.TriggerEvent("test_user", types.EventTypeEntry)
	if err == nil {
		t.Error("Expected error when triggering event without callback")
	}

	// Register callback
	var receivedEvent *types.RawHardwareEvent
	var eventMutex sync.Mutex

	adapter.OnEvent(func(event types.RawHardwareEvent) {
		eventMutex.Lock()
		receivedEvent = &event
		eventMutex.Unlock()
	})

	// Test successful trigger
	testUserID := "test_user_123"
	err = adapter.TriggerEvent(testUserID, types.EventTypeEntry)
	if err != nil {
		t.Fatalf("Failed to trigger event: %v", err)
	}

	// Wait a moment for event processing
	time.Sleep(100 * time.Millisecond)

	// Verify event was received
	eventMutex.Lock()
	event := receivedEvent
	eventMutex.Unlock()

	if event == nil {
		t.Fatal("Expected to receive triggered event")
	}

	if event.ExternalUserID != testUserID {
		t.Errorf("Expected external user ID '%s', got '%s'", testUserID, event.ExternalUserID)
	}

	if event.EventType != types.EventTypeEntry {
		t.Errorf("Expected event type '%s', got '%s'", types.EventTypeEntry, event.EventType)
	}

	if event.RawData["simulator"] != true {
		t.Error("Expected simulator flag to be true in raw data")
	}

	if event.RawData["method"] != "manual_trigger" {
		t.Error("Expected method to be 'manual_trigger' in raw data")
	}

	// Test invalid event type
	err = adapter.TriggerEvent("test_user", "invalid_type")
	if err == nil {
		t.Error("Expected error when triggering event with invalid type")
	}
}

func TestSimulatorAdapter_EventGeneration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	// Initialize with fast event generation
	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 0.5, // 500ms for fast testing
			"simulatedUsers": []interface{}{
				"user_001",
				"user_002",
			},
		},
	}
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Collect events
	var receivedEvents []types.RawHardwareEvent
	var eventMutex sync.Mutex

	adapter.OnEvent(func(event types.RawHardwareEvent) {
		eventMutex.Lock()
		receivedEvents = append(receivedEvents, event)
		eventMutex.Unlock()
	})

	// Start listening
	err = adapter.StartListening(ctx)
	if err != nil {
		t.Fatalf("Failed to start listening: %v", err)
	}

	// Wait for multiple events
	time.Sleep(1200 * time.Millisecond)

	// Stop listening
	err = adapter.StopListening(ctx)
	if err != nil {
		t.Fatalf("Failed to stop listening: %v", err)
	}

	// Verify events
	eventMutex.Lock()
	events := make([]types.RawHardwareEvent, len(receivedEvents))
	copy(events, receivedEvents)
	eventMutex.Unlock()

	if len(events) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}

	// Verify event properties
	for i, event := range events {
		if event.ExternalUserID == "" {
			t.Errorf("Event %d: empty external user ID", i)
		}

		if !types.IsValidEventType(event.EventType) {
			t.Errorf("Event %d: invalid event type '%s'", i, event.EventType)
		}

		if event.Timestamp.IsZero() {
			t.Errorf("Event %d: zero timestamp", i)
		}

		if event.RawData["simulator"] != true {
			t.Errorf("Event %d: missing simulator flag", i)
		}

		if event.RawData["method"] != "auto_generated" {
			t.Errorf("Event %d: expected method 'auto_generated', got '%v'", i, event.RawData["method"])
		}
	}

	// Verify status was updated with last event
	status := adapter.GetStatus()
	if status.LastEvent.IsZero() {
		t.Error("Expected last event time to be set in status")
	}
}

func TestSimulatorAdapter_ConcurrentOperations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	adapter := NewSimulatorAdapter(logger)
	ctx := context.Background()

	// Initialize adapter
	config := types.AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 0.1, // Very fast for stress testing
		},
	}
	err := adapter.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize adapter: %v", err)
	}

	// Register callback
	var eventCount int
	var eventMutex sync.Mutex

	adapter.OnEvent(func(event types.RawHardwareEvent) {
		eventMutex.Lock()
		eventCount++
		eventMutex.Unlock()
	})

	// Start listening
	err = adapter.StartListening(ctx)
	if err != nil {
		t.Fatalf("Failed to start listening: %v", err)
	}

	// Perform concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent status checks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				status := adapter.GetStatus()
				if status.Name != "simulator" {
					t.Errorf("Unexpected adapter name: %s", status.Name)
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Concurrent door unlocks
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				err := adapter.UnlockDoor(ctx, 100)
				if err != nil {
					t.Errorf("Unexpected error during concurrent unlock: %v", err)
				}
				time.Sleep(20 * time.Millisecond)
			}
		}()
	}

	// Concurrent manual triggers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				err := adapter.TriggerEvent(fmt.Sprintf("concurrent_user_%d_%d", id, j), types.EventTypeEntry)
				if err != nil {
					t.Errorf("Unexpected error during concurrent trigger: %v", err)
				}
				time.Sleep(15 * time.Millisecond)
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Stop listening
	err = adapter.StopListening(ctx)
	if err != nil {
		t.Fatalf("Failed to stop listening: %v", err)
	}

	// Verify we received events (both auto-generated and manually triggered)
	eventMutex.Lock()
	finalEventCount := eventCount
	eventMutex.Unlock()

	if finalEventCount == 0 {
		t.Error("Expected to receive events during concurrent operations")
	}

	// Verify adapter is still healthy
	if !adapter.IsHealthy() {
		t.Error("Expected adapter to remain healthy after concurrent operations")
	}
}