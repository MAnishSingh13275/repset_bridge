package adapters

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"gym-door-bridge/internal/types"
)

func TestAdapterManager_LoadAdapters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	configs := []AdapterConfig{
		{
			Name:    "simulator",
			Enabled: true,
			Settings: map[string]interface{}{
				"eventInterval": 1.0,
			},
		},
		{
			Name:    "webhook",
			Enabled: true,
			Settings: map[string]interface{}{
				"port": 8090.0,
				"path": "/test",
			},
		},
		{
			Name:    "disabled_adapter",
			Enabled: false,
			Settings: map[string]interface{}{},
		},
	}

	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	// Check that enabled adapters were loaded
	adapters := manager.GetAllAdapters()
	if len(adapters) != 2 {
		t.Errorf("expected 2 adapters, got %d", len(adapters))
	}

	if _, exists := adapters["simulator"]; !exists {
		t.Error("expected simulator adapter to be loaded")
	}

	if _, exists := adapters["webhook"]; !exists {
		t.Error("expected webhook adapter to be loaded")
	}

	if _, exists := adapters["disabled_adapter"]; exists {
		t.Error("disabled adapter should not be loaded")
	}
}

func TestAdapterManager_StartStopAll(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	configs := []AdapterConfig{
		{
			Name:    "simulator",
			Enabled: true,
			Settings: map[string]interface{}{
				"eventInterval": 10.0, // Long interval to avoid events during test
			},
		},
	}

	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	// Register event callback
	var receivedEvents []types.RawHardwareEvent
	manager.OnEvent(func(event types.RawHardwareEvent) {
		receivedEvents = append(receivedEvents, event)
	})

	// Start all adapters
	err = manager.StartAll()
	if err != nil {
		t.Fatalf("failed to start adapters: %v", err)
	}

	// Check adapter status
	status := manager.GetAdapterStatus()
	if len(status) != 1 {
		t.Errorf("expected 1 adapter status, got %d", len(status))
	}

	if status["simulator"].Status != StatusActive {
		t.Errorf("expected simulator status 'active', got '%s'", status["simulator"].Status)
	}

	// Check healthy adapters
	healthy := manager.GetHealthyAdapters()
	if len(healthy) != 1 || healthy[0] != "simulator" {
		t.Errorf("expected 1 healthy adapter 'simulator', got %v", healthy)
	}

	// Stop all adapters
	err = manager.StopAll()
	if err != nil {
		t.Fatalf("failed to stop adapters: %v", err)
	}
}

func TestAdapterManager_GetAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	configs := []AdapterConfig{
		{
			Name:     "simulator",
			Enabled:  true,
			Settings: map[string]interface{}{},
		},
	}

	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	// Test getting existing adapter
	adapter, exists := manager.GetAdapter("simulator")
	if !exists {
		t.Error("expected to find simulator adapter")
	}
	if adapter.Name() != "simulator" {
		t.Errorf("expected adapter name 'simulator', got '%s'", adapter.Name())
	}

	// Test getting non-existent adapter
	_, exists = manager.GetAdapter("nonexistent")
	if exists {
		t.Error("expected not to find nonexistent adapter")
	}
}

func TestAdapterManager_UnlockDoor(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	configs := []AdapterConfig{
		{
			Name:     "simulator",
			Enabled:  true,
			Settings: map[string]interface{}{},
		},
	}

	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	manager.OnEvent(func(event types.RawHardwareEvent) {})

	err = manager.StartAll()
	if err != nil {
		t.Fatalf("failed to start adapters: %v", err)
	}

	// Test door unlock
	err = manager.UnlockDoor(3000)
	if err != nil {
		t.Errorf("unexpected error unlocking door: %v", err)
	}

	// Stop adapters and test unlock with no healthy adapters
	err = manager.StopAll()
	if err != nil {
		t.Fatalf("failed to stop adapters: %v", err)
	}

	err = manager.UnlockDoor(3000)
	if err == nil {
		t.Error("expected error when no healthy adapters available")
	}
}

func TestAdapterManager_ReloadAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	// Load initial adapter
	initialConfig := AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 5.0,
		},
	}

	err := manager.LoadAdapters([]AdapterConfig{initialConfig})
	if err != nil {
		t.Fatalf("failed to load initial adapter: %v", err)
	}

	// Reload with new configuration
	newConfig := AdapterConfig{
		Name:    "simulator",
		Enabled: true,
		Settings: map[string]interface{}{
			"eventInterval": 10.0,
		},
	}

	err = manager.ReloadAdapter(newConfig)
	if err != nil {
		t.Fatalf("failed to reload adapter: %v", err)
	}

	// Verify adapter still exists
	adapter, exists := manager.GetAdapter("simulator")
	if !exists {
		t.Error("expected adapter to exist after reload")
	}
	if adapter.Name() != "simulator" {
		t.Errorf("expected adapter name 'simulator', got '%s'", adapter.Name())
	}
}

func TestAdapterManager_EventCallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	var receivedEvents []types.RawHardwareEvent
	manager.OnEvent(func(event types.RawHardwareEvent) {
		receivedEvents = append(receivedEvents, event)
	})

	configs := []AdapterConfig{
		{
			Name:    "simulator",
			Enabled: true,
			Settings: map[string]interface{}{
				"eventInterval": 0.1, // Very short interval for testing
			},
		},
	}

	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	err = manager.StartAll()
	if err != nil {
		t.Fatalf("failed to start adapters: %v", err)
	}

	// Wait for some events
	time.Sleep(500 * time.Millisecond)

	err = manager.StopAll()
	if err != nil {
		t.Fatalf("failed to stop adapters: %v", err)
	}

	// Should have received at least one event
	if len(receivedEvents) == 0 {
		t.Error("expected to receive at least one event")
	}

	// Verify event structure
	if len(receivedEvents) > 0 {
		event := receivedEvents[0]
		if event.ExternalUserID == "" {
			t.Error("expected event to have external user ID")
		}
		if !types.IsValidEventType(event.EventType) {
			t.Errorf("expected valid event type, got '%s'", event.EventType)
		}
	}
}

func TestAdapterManager_UnknownAdapterType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	manager := NewAdapterManager(logger)
	defer manager.Shutdown()

	configs := []AdapterConfig{
		{
			Name:     "unknown_adapter",
			Enabled:  true,
			Settings: map[string]interface{}{},
		},
	}

	// Should not fail completely, just skip unknown adapter
	err := manager.LoadAdapters(configs)
	if err != nil {
		t.Fatalf("failed to load adapters: %v", err)
	}

	// Should have no adapters loaded
	adapters := manager.GetAllAdapters()
	if len(adapters) != 0 {
		t.Errorf("expected 0 adapters, got %d", len(adapters))
	}
}

func TestGetRegisteredAdapterTypes(t *testing.T) {
	types := GetRegisteredAdapterTypes()
	
	expectedTypes := []string{"simulator", "webhook", "fingerprint", "rfid"}
	if len(types) != len(expectedTypes) {
		t.Errorf("expected %d adapter types, got %d", len(expectedTypes), len(types))
	}

	typeMap := make(map[string]bool)
	for _, adapterType := range types {
		typeMap[adapterType] = true
	}

	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("expected adapter type '%s' to be registered", expected)
		}
	}
}

func TestRegisterAdapter(t *testing.T) {
	// Save original registry
	originalRegistry := make(map[string]AdapterFactory)
	for name, factory := range registeredAdapters {
		originalRegistry[name] = factory
	}

	// Restore original registry after test
	defer func() {
		registeredAdapters = originalRegistry
	}()

	// Register a custom adapter
	RegisterAdapter("custom", func(logger *slog.Logger) HardwareAdapter {
		return nil // Mock factory
	})

	types := GetRegisteredAdapterTypes()
	found := false
	for _, adapterType := range types {
		if adapterType == "custom" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected custom adapter to be registered")
	}
}