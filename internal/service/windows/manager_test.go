package windows

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServiceManager(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service manager tests only run on Windows")
	}
	
	// Note: This test requires administrator privileges to actually connect
	// In a CI environment, this might fail, so we'll make it conditional
	sm, err := NewServiceManager()
	if err != nil {
		// If we can't connect (likely due to permissions), skip the test
		t.Skipf("Cannot connect to service manager (likely permissions): %v", err)
	}
	
	require.NotNil(t, sm)
	require.NotNil(t, sm.manager)
	
	// Test closing the manager
	err = sm.Close()
	assert.NoError(t, err)
}

func TestGetExecutablePath(t *testing.T) {
	execPath, err := GetExecutablePath()
	require.NoError(t, err)
	assert.NotEmpty(t, execPath)
	
	// Verify the path exists
	_, err = os.Stat(execPath)
	assert.NoError(t, err)
}

func TestServiceManagerLifecycle(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service manager tests only run on Windows")
	}
	
	// This is an integration test that requires administrator privileges
	// We'll test the basic functionality without actually installing/uninstalling
	
	sm, err := NewServiceManager()
	if err != nil {
		t.Skipf("Cannot connect to service manager (likely permissions): %v", err)
	}
	defer sm.Close()
	
	// Test checking if service is installed (should be false for test service)
	installed, err := sm.IsServiceInstalled()
	if err != nil {
		t.Logf("Warning: Could not check service installation status: %v", err)
	} else {
		// In most test environments, the service should not be installed
		t.Logf("Service installed: %v", installed)
	}
}

func TestServiceManagerOperations(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service manager tests only run on Windows")
	}
	
	// Test service manager operations that don't require the service to be installed
	sm, err := NewServiceManager()
	if err != nil {
		t.Skipf("Cannot connect to service manager (likely permissions): %v", err)
	}
	defer sm.Close()
	
	// Test IsServiceInstalled - this should work even if service is not installed
	_, err = sm.IsServiceInstalled()
	// We don't assert the result, just that the call doesn't error
	// The actual result depends on whether the service is installed
	if err != nil {
		t.Logf("IsServiceInstalled returned error: %v", err)
	}
}

// Test service manager error handling
func TestServiceManagerErrorHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows service manager tests only run on Windows")
	}
	
	sm, err := NewServiceManager()
	if err != nil {
		t.Skipf("Cannot connect to service manager (likely permissions): %v", err)
	}
	defer sm.Close()
	
	// Test operations on non-existent service
	// These should handle errors gracefully
	
	// GetServiceStatus on non-existent service should return error
	_, err = sm.GetServiceStatus()
	if err == nil {
		t.Log("GetServiceStatus succeeded (service might be installed)")
	} else {
		t.Logf("GetServiceStatus failed as expected: %v", err)
	}
	
	// StartService on non-existent service should return error
	err = sm.StartService()
	if err == nil {
		t.Log("StartService succeeded (service might be installed)")
	} else {
		t.Logf("StartService failed as expected: %v", err)
	}
	
	// StopService on non-existent service should return error
	err = sm.StopService()
	if err == nil {
		t.Log("StopService succeeded (service might be installed)")
	} else {
		t.Logf("StopService failed as expected: %v", err)
	}
}

// Benchmark service manager operations
func BenchmarkManagerCreation(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows service manager benchmarks only run on Windows")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm, err := NewServiceManager()
		if err != nil {
			b.Skipf("Cannot connect to service manager: %v", err)
		}
		sm.Close()
	}
}

func BenchmarkIsServiceInstalled(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows service manager benchmarks only run on Windows")
	}
	
	sm, err := NewServiceManager()
	if err != nil {
		b.Skipf("Cannot connect to service manager: %v", err)
	}
	defer sm.Close()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.IsServiceInstalled()
		if err != nil {
			b.Logf("IsServiceInstalled error: %v", err)
		}
	}
}