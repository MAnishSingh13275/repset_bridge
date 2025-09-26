package door

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/config"
)

func TestDefaultDoorControlConfig(t *testing.T) {
	doorConfig := DefaultDoorControlConfig()
	
	assert.Equal(t, 8081, doorConfig.Port)
	assert.Equal(t, "/open-door", doorConfig.Path)
	assert.Equal(t, 3000, doorConfig.DefaultUnlockDuration)
	assert.Equal(t, 30000, doorConfig.MaxUnlockDuration)
	assert.False(t, doorConfig.AuthRequired)
}

func TestNewDoorController(t *testing.T) {
	doorConfig := DefaultDoorControlConfig()
	globalConfig := &config.Config{UnlockDuration: 5000}
	
	// Create a simple mock registry for this test
	registry := &mockRegistry{}
	
	controller := NewDoorController(doorConfig, globalConfig, registry)
	
	assert.NotNil(t, controller)
	assert.Equal(t, doorConfig, controller.config)
	assert.Equal(t, globalConfig, controller.globalConfig)
	assert.Equal(t, registry, controller.adapterRegistry)
}

// Simple mock registry for basic testing
type mockRegistry struct{}

func (m *mockRegistry) GetAllAdapters() []adapters.HardwareAdapter { return nil }
func (m *mockRegistry) GetAdapter(name string) (adapters.HardwareAdapter, error) { return nil, nil }
func (m *mockRegistry) GetActiveAdapters() []adapters.HardwareAdapter { return nil }
