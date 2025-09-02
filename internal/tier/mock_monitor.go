package tier

import "time"

// MockResourceMonitor is a mock implementation of ResourceMonitor for testing
type MockResourceMonitor struct {
	resources SystemResources
	err       error
}

// NewMockResourceMonitor creates a new mock resource monitor
func NewMockResourceMonitor(resources SystemResources) *MockResourceMonitor {
	return &MockResourceMonitor{
		resources: resources,
	}
}

// SetResources sets the resources that will be returned by GetSystemResources
func (m *MockResourceMonitor) SetResources(resources SystemResources) {
	m.resources = resources
}

// SetError sets an error that will be returned by GetSystemResources
func (m *MockResourceMonitor) SetError(err error) {
	m.err = err
}

// GetSystemResources returns the mock system resources
func (m *MockResourceMonitor) GetSystemResources() (SystemResources, error) {
	if m.err != nil {
		return SystemResources{}, m.err
	}
	
	// Update timestamp to current time
	m.resources.LastUpdated = time.Now()
	return m.resources, nil
}

// CreateLiteSystemResources creates system resources that should result in Lite tier
func CreateLiteSystemResources() SystemResources {
	return SystemResources{
		CPUCores:    1,
		MemoryGB:    1.5,
		CPUUsage:    50.0,
		MemoryUsage: 60.0,
		DiskUsage:   30.0,
		LastUpdated: time.Now(),
	}
}

// CreateNormalSystemResources creates system resources that should result in Normal tier
func CreateNormalSystemResources() SystemResources {
	return SystemResources{
		CPUCores:    4,
		MemoryGB:    4.0,
		CPUUsage:    30.0,
		MemoryUsage: 40.0,
		DiskUsage:   25.0,
		LastUpdated: time.Now(),
	}
}

// CreateFullSystemResources creates system resources that should result in Full tier
func CreateFullSystemResources() SystemResources {
	return SystemResources{
		CPUCores:    8,
		MemoryGB:    16.0,
		CPUUsage:    20.0,
		MemoryUsage: 30.0,
		DiskUsage:   20.0,
		LastUpdated: time.Now(),
	}
}