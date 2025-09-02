//go:build !windows && !darwin && !linux

package tier

import (
	"fmt"
	"runtime"
)

// Stub implementation for unsupported platforms
func (m *SystemResourceMonitor) getTotalMemoryWindows() (uint64, error) {
	return 0, fmt.Errorf("Windows memory detection not supported on %s", runtime.GOOS)
}

func (m *SystemResourceMonitor) getTotalMemoryDarwin() (uint64, error) {
	return 0, fmt.Errorf("macOS memory detection not supported on %s", runtime.GOOS)
}

func (m *SystemResourceMonitor) getTotalMemoryLinux() (uint64, error) {
	return 0, fmt.Errorf("Linux memory detection not supported on %s", runtime.GOOS)
}

func (m *SystemResourceMonitor) getDiskUsagePlatform(path string) (float64, error) {
	// Return a default value for unsupported platforms
	return 50.0, nil
}