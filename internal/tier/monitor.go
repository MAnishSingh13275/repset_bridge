package tier

import (
	"fmt"
	"runtime"
	"time"
)

// ResourceMonitor defines the interface for system resource monitoring
type ResourceMonitor interface {
	GetSystemResources() (SystemResources, error)
}

// SystemResourceMonitor implements ResourceMonitor using system calls
type SystemResourceMonitor struct {
	lastCPUTime time.Time
	lastCPUUsage float64
}

// NewSystemResourceMonitor creates a new system resource monitor
func NewSystemResourceMonitor() *SystemResourceMonitor {
	return &SystemResourceMonitor{}
}

// GetSystemResources retrieves current system resource information
func (m *SystemResourceMonitor) GetSystemResources() (SystemResources, error) {
	resources := SystemResources{
		LastUpdated: time.Now(),
	}
	
	// Get CPU cores
	resources.CPUCores = runtime.NumCPU()
	
	// Get memory information
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	
	// Get system memory
	totalMemory, err := m.getTotalSystemMemory()
	if err != nil {
		return resources, fmt.Errorf("failed to get total system memory: %w", err)
	}
	resources.MemoryGB = float64(totalMemory) / (1024 * 1024 * 1024)
	
	// Calculate memory usage percentage
	usedMemory := memStats.Sys
	resources.MemoryUsage = (float64(usedMemory) / float64(totalMemory)) * 100
	
	// Get CPU usage (simplified approach)
	cpuUsage, err := m.getCPUUsage()
	if err != nil {
		// If we can't get CPU usage, default to a reasonable value
		resources.CPUUsage = 0.0
	} else {
		resources.CPUUsage = cpuUsage
	}
	
	// Get disk usage for current directory
	diskUsage, err := m.getDiskUsage(".")
	if err != nil {
		// If we can't get disk usage, default to 0
		resources.DiskUsage = 0.0
	} else {
		resources.DiskUsage = diskUsage
	}
	
	return resources, nil
}

// getTotalSystemMemory gets the total system memory in bytes
func (m *SystemResourceMonitor) getTotalSystemMemory() (uint64, error) {
	switch runtime.GOOS {
	case "windows":
		return m.getTotalMemoryWindows()
	case "darwin":
		return m.getTotalMemoryDarwin()
	case "linux":
		return m.getTotalMemoryLinux()
	default:
		// Fallback: use a reasonable default based on Go's memory stats
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		// Estimate total memory as 4x the current heap size (very rough estimate)
		return memStats.Sys * 4, nil
	}
}

// getCPUUsage gets the current CPU usage percentage
func (m *SystemResourceMonitor) getCPUUsage() (float64, error) {
	// This is a simplified CPU usage calculation
	// For production, you might want to use a more sophisticated approach
	
	now := time.Now()
	if m.lastCPUTime.IsZero() {
		m.lastCPUTime = now
		m.lastCPUUsage = 0.0
		return 0.0, nil
	}
	
	// Simple approach: use number of goroutines as a proxy for CPU activity
	// This is not accurate but provides a basic indication
	numGoroutines := float64(runtime.NumGoroutine())
	numCPU := float64(runtime.NumCPU())
	
	// Calculate a rough CPU usage based on goroutine activity
	usage := (numGoroutines / (numCPU * 10)) * 100
	if usage > 100 {
		usage = 100
	}
	
	m.lastCPUTime = now
	m.lastCPUUsage = usage
	
	return usage, nil
}

// getDiskUsage gets the disk usage percentage for the given path
func (m *SystemResourceMonitor) getDiskUsage(path string) (float64, error) {
	return m.getDiskUsagePlatform(path)
}