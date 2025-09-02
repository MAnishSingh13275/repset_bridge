package tier

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemResourceMonitor_GetSystemResources(t *testing.T) {
	monitor := NewSystemResourceMonitor()
	
	resources, err := monitor.GetSystemResources()
	require.NoError(t, err)
	
	// Verify basic properties
	assert.Greater(t, resources.CPUCores, 0)
	assert.Greater(t, resources.MemoryGB, 0.0)
	assert.GreaterOrEqual(t, resources.CPUUsage, 0.0)
	assert.LessOrEqual(t, resources.CPUUsage, 100.0)
	assert.GreaterOrEqual(t, resources.MemoryUsage, 0.0)
	assert.LessOrEqual(t, resources.MemoryUsage, 100.0)
	assert.GreaterOrEqual(t, resources.DiskUsage, 0.0)
	assert.LessOrEqual(t, resources.DiskUsage, 100.0)
	assert.WithinDuration(t, time.Now(), resources.LastUpdated, time.Second)
	
	// CPU cores should match runtime.NumCPU()
	assert.Equal(t, runtime.NumCPU(), resources.CPUCores)
}

func TestSystemResourceMonitor_GetSystemResources_Consistency(t *testing.T) {
	monitor := NewSystemResourceMonitor()
	
	// Get resources twice
	resources1, err1 := monitor.GetSystemResources()
	require.NoError(t, err1)
	
	time.Sleep(10 * time.Millisecond)
	
	resources2, err2 := monitor.GetSystemResources()
	require.NoError(t, err2)
	
	// CPU cores and memory should be consistent
	assert.Equal(t, resources1.CPUCores, resources2.CPUCores)
	assert.Equal(t, resources1.MemoryGB, resources2.MemoryGB)
	
	// Timestamps should be different
	assert.True(t, resources2.LastUpdated.After(resources1.LastUpdated))
}

func TestMockResourceMonitor(t *testing.T) {
	t.Run("returns set resources", func(t *testing.T) {
		expectedResources := CreateNormalSystemResources()
		monitor := NewMockResourceMonitor(expectedResources)
		
		resources, err := monitor.GetSystemResources()
		require.NoError(t, err)
		
		assert.Equal(t, expectedResources.CPUCores, resources.CPUCores)
		assert.Equal(t, expectedResources.MemoryGB, resources.MemoryGB)
		assert.Equal(t, expectedResources.CPUUsage, resources.CPUUsage)
		assert.Equal(t, expectedResources.MemoryUsage, resources.MemoryUsage)
		assert.Equal(t, expectedResources.DiskUsage, resources.DiskUsage)
		// LastUpdated should be updated to current time
		assert.WithinDuration(t, time.Now(), resources.LastUpdated, time.Second)
	})

	t.Run("returns set error", func(t *testing.T) {
		monitor := NewMockResourceMonitor(CreateNormalSystemResources())
		monitor.SetError(assert.AnError)
		
		_, err := monitor.GetSystemResources()
		assert.Equal(t, assert.AnError, err)
	})

	t.Run("can update resources", func(t *testing.T) {
		monitor := NewMockResourceMonitor(CreateLiteSystemResources())
		
		// Get initial resources
		resources1, err := monitor.GetSystemResources()
		require.NoError(t, err)
		assert.Equal(t, 1, resources1.CPUCores)
		
		// Update resources
		newResources := CreateFullSystemResources()
		monitor.SetResources(newResources)
		
		// Get updated resources
		resources2, err := monitor.GetSystemResources()
		require.NoError(t, err)
		assert.Equal(t, newResources.CPUCores, resources2.CPUCores)
		assert.Equal(t, newResources.MemoryGB, resources2.MemoryGB)
	})
}

func TestCreateSystemResources(t *testing.T) {
	t.Run("CreateLiteSystemResources", func(t *testing.T) {
		resources := CreateLiteSystemResources()
		
		assert.Equal(t, 1, resources.CPUCores)
		assert.Equal(t, 1.5, resources.MemoryGB)
		assert.Equal(t, 50.0, resources.CPUUsage)
		assert.Equal(t, 60.0, resources.MemoryUsage)
		assert.Equal(t, 30.0, resources.DiskUsage)
		assert.WithinDuration(t, time.Now(), resources.LastUpdated, time.Second)
	})

	t.Run("CreateNormalSystemResources", func(t *testing.T) {
		resources := CreateNormalSystemResources()
		
		assert.Equal(t, 4, resources.CPUCores)
		assert.Equal(t, 4.0, resources.MemoryGB)
		assert.Equal(t, 30.0, resources.CPUUsage)
		assert.Equal(t, 40.0, resources.MemoryUsage)
		assert.Equal(t, 25.0, resources.DiskUsage)
		assert.WithinDuration(t, time.Now(), resources.LastUpdated, time.Second)
	})

	t.Run("CreateFullSystemResources", func(t *testing.T) {
		resources := CreateFullSystemResources()
		
		assert.Equal(t, 8, resources.CPUCores)
		assert.Equal(t, 16.0, resources.MemoryGB)
		assert.Equal(t, 20.0, resources.CPUUsage)
		assert.Equal(t, 30.0, resources.MemoryUsage)
		assert.Equal(t, 20.0, resources.DiskUsage)
		assert.WithinDuration(t, time.Now(), resources.LastUpdated, time.Second)
	})
}

func TestSystemResourceMonitor_getCPUUsage(t *testing.T) {
	monitor := NewSystemResourceMonitor()
	
	// First call should return 0 (no previous measurement)
	usage1, err := monitor.getCPUUsage()
	require.NoError(t, err)
	assert.Equal(t, 0.0, usage1)
	
	// Second call should return some value
	time.Sleep(10 * time.Millisecond)
	usage2, err := monitor.getCPUUsage()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, usage2, 0.0)
	assert.LessOrEqual(t, usage2, 100.0)
}

func TestSystemResourceMonitor_getDiskUsage(t *testing.T) {
	monitor := NewSystemResourceMonitor()
	
	usage, err := monitor.getDiskUsage(".")
	require.NoError(t, err)
	
	assert.GreaterOrEqual(t, usage, 0.0)
	assert.LessOrEqual(t, usage, 100.0)
}

func TestSystemResourceMonitor_getDiskUsage_InvalidPath(t *testing.T) {
	monitor := NewSystemResourceMonitor()
	
	// Use a path that should definitely not exist on any platform
	usage, err := monitor.getDiskUsage("/this/path/definitely/does/not/exist/anywhere")
	
	// On some platforms, this might not error but return 0 usage
	// Either error or zero usage is acceptable for invalid paths
	if err == nil {
		assert.Equal(t, 0.0, usage)
	} else {
		assert.Error(t, err)
	}
}