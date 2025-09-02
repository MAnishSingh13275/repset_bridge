//go:build darwin

package tier

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// macOS-specific memory detection
func (m *SystemResourceMonitor) getTotalMemoryDarwin() (uint64, error) {
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	memStr := strings.TrimSpace(string(output))
	memBytes, err := strconv.ParseUint(memStr, 10, 64)
	if err != nil {
		return 0, err
	}
	
	return memBytes, nil
}// m
acOS-specific disk usage detection
func (m *SystemResourceMonitor) getDiskUsagePlatform(path string) (float64, error) {
	cmd := exec.Command("df", "-k", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, fmt.Errorf("unexpected df output format")
	}
	
	// Parse the second line (first line is headers)
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return 0, fmt.Errorf("unexpected df output format")
	}
	
	// Fields: Filesystem, 1K-blocks, Used, Available, Use%, Mounted on
	totalStr := fields[1]
	usedStr := fields[2]
	
	total, err := strconv.ParseUint(totalStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse total disk space: %w", err)
	}
	
	used, err := strconv.ParseUint(usedStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse used disk space: %w", err)
	}
	
	if total == 0 {
		return 0, nil
	}
	
	usage := (float64(used) / float64(total)) * 100
	return usage, nil
}/
/ Stub methods for other platforms
func (m *SystemResourceMonitor) getTotalMemoryWindows() (uint64, error) {
	return 0, fmt.Errorf("Windows memory detection not supported on macOS")
}

func (m *SystemResourceMonitor) getTotalMemoryLinux() (uint64, error) {
	return 0, fmt.Errorf("Linux memory detection not supported on macOS")
}