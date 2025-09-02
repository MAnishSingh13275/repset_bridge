package health

import (
	"fmt"
	"sync"

	"gym-door-bridge/internal/adapters"
	"gym-door-bridge/internal/types"
)

// SimpleAdapterRegistry is a simple implementation of AdapterRegistry
type SimpleAdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]adapters.HardwareAdapter
}

// NewSimpleAdapterRegistry creates a new simple adapter registry
func NewSimpleAdapterRegistry() *SimpleAdapterRegistry {
	return &SimpleAdapterRegistry{
		adapters: make(map[string]adapters.HardwareAdapter),
	}
}

// RegisterAdapter registers a hardware adapter
func (r *SimpleAdapterRegistry) RegisterAdapter(adapter adapters.HardwareAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.adapters[adapter.Name()] = adapter
}

// UnregisterAdapter unregisters a hardware adapter
func (r *SimpleAdapterRegistry) UnregisterAdapter(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.adapters, name)
}

// GetAdapter retrieves a hardware adapter by name
func (r *SimpleAdapterRegistry) GetAdapter(name string) (adapters.HardwareAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	adapter, exists := r.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}
	
	return adapter, nil
}

// GetAllAdapters returns all registered adapters
func (r *SimpleAdapterRegistry) GetAllAdapters() []adapters.HardwareAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]adapters.HardwareAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		result = append(result, adapter)
	}
	
	return result
}

// GetAdapterStatus returns the status of a specific adapter
func (r *SimpleAdapterRegistry) GetAdapterStatus(name string) (types.AdapterStatus, error) {
	adapter, err := r.GetAdapter(name)
	if err != nil {
		return types.AdapterStatus{}, err
	}
	
	return adapter.GetStatus(), nil
}

// GetAllAdapterStatuses returns the status of all registered adapters
func (r *SimpleAdapterRegistry) GetAllAdapterStatuses() []types.AdapterStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]types.AdapterStatus, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		result = append(result, adapter.GetStatus())
	}
	
	return result
}

// Count returns the number of registered adapters
func (r *SimpleAdapterRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return len(r.adapters)
}