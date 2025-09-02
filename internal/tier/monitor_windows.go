//go:build windows

package tier

import (
	"syscall"
	"unsafe"
)

// Windows-specific memory detection
func (m *SystemResourceMonitor) getTotalMemoryWindows() (uint64, error) {
	kernel32, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return 0, err
	}
	defer syscall.FreeLibrary(kernel32)
	
	globalMemoryStatusEx, err := syscall.GetProcAddress(kernel32, "GlobalMemoryStatusEx")
	if err != nil {
		return 0, err
	}
	
	type memoryStatusEx struct {
		dwLength                uint32
		dwMemoryLoad            uint32
		ullTotalPhys            uint64
		ullAvailPhys            uint64
		ullTotalPageFile        uint64
		ullAvailPageFile        uint64
		ullTotalVirtual         uint64
		ullAvailVirtual         uint64
		ullAvailExtendedVirtual uint64
	}
	
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))
	
	ret, _, _ := syscall.Syscall(globalMemoryStatusEx, 1, uintptr(unsafe.Pointer(&memStatus)), 0, 0)
	if ret == 0 {
		return 0, syscall.GetLastError()
	}
	
	return memStatus.ullTotalPhys, nil
}

// Windows-specific disk usage detection
func (m *SystemResourceMonitor) getDiskUsagePlatform(path string) (float64, error) {
	kernel32, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		return 0, err
	}
	defer syscall.FreeLibrary(kernel32)
	
	getDiskFreeSpaceEx, err := syscall.GetProcAddress(kernel32, "GetDiskFreeSpaceExW")
	if err != nil {
		return 0, err
	}
	
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	
	ret, _, _ := syscall.Syscall6(getDiskFreeSpaceEx, 4,
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
		0, 0)
	
	if ret == 0 {
		return 0, syscall.GetLastError()
	}
	
	if totalNumberOfBytes == 0 {
		return 0, nil
	}
	
	used := totalNumberOfBytes - totalNumberOfFreeBytes
	usage := (float64(used) / float64(totalNumberOfBytes)) * 100
	return usage, nil
}

// Stub methods for other platforms
func (m *SystemResourceMonitor) getTotalMemoryDarwin() (uint64, error) {
	return 0, syscall.ENOSYS
}

func (m *SystemResourceMonitor) getTotalMemoryLinux() (uint64, error) {
	return 0, syscall.ENOSYS
}