//go:build linux

package tier

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// Linux-specific memory detection
func (m *SystemResourceMonitor) getTotalMemoryLinux() (uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				memKB, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				// Convert from KB to bytes
				return memKB * 1024, nil
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	
	return 0, fmt.Errorf("could not find MemTotal in /proc/meminfo")
}// Linux
-specific disk usage detection
func (m *SystemResourceMonitor) getDiskUsagePlatform(path string) (float64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to get disk stats: %w", err)
	}
	
	// Calculate disk usage percentage
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	
	if total == 0 {
		return 0, nil
	}
	
	usage := (float64(used) / float64(total)) * 100
	return usage, nil
}// Stub
 methods for other platforms
func (m *SystemResourceMonitor) getTotalMemoryWindows() (uint64, error) {
	return 0, fmt.Errorf("Windows memory detection not supported on Linux")
}

func (m *SystemResourceMonitor) getTotalMemoryDarwin() (uint64, error) {
	return 0, fmt.Errorf("macOS memory detection not supported on Linux")
}