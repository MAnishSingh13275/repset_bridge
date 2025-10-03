package windows

import (
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/svc"
)

func TestServiceHealthMonitor(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Service health monitor tests only run on Windows")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	config := DefaultServiceHealthMonitorConfig()
	config.MonitorInterval = 1 * time.Second // Faster for testing

	monitor, err := NewServiceHealthMonitor(logger, config)
	require.NoError(t, err)
	require.NotNil(t, monitor)

	defer monitor.Close()

	t.Run("GetHealthSummary", func(t *testing.T) {
		summary := monitor.GetHealthSummary()
		assert.NotNil(t, summary)
		assert.Contains(t, summary, "is_monitoring")
		assert.Contains(t, summary, "monitor_interval")
		assert.Contains(t, summary, "recovery_enabled")
	})

	t.Run("GetCurrentHealth", func(t *testing.T) {
		health, err := monitor.GetCurrentHealth()
		if err != nil {
			// Service might not be installed in test environment
			t.Logf("Service not available for testing: %v", err)
			return
		}
		
		assert.NotNil(t, health)
		assert.NotEmpty(t, health.ServiceName)
		assert.NotEmpty(t, health.Status)
	})
}

func TestServiceHealthMonitorConfig(t *testing.T) {
	config := DefaultServiceHealthMonitorConfig()
	
	assert.Equal(t, 30*time.Second, config.MonitorInterval)
	assert.Equal(t, 100, config.MaxHistorySize)
	assert.True(t, config.RecoveryEnabled)
	assert.Equal(t, 3, config.MaxRecoveryAttempts)
	assert.Equal(t, 1*time.Hour, config.RecoveryResetTime)
}

func TestServiceHealthInfo(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Service health tests only run on Windows")
	}

	serviceManager, err := NewServiceManager()
	if err != nil {
		t.Skipf("Cannot create service manager: %v", err)
	}
	defer serviceManager.Close()

	t.Run("TranslateServiceState", func(t *testing.T) {
		testCases := []struct {
			state    svc.State
			expected string
		}{
			{svc.Stopped, "Stopped"},
			{svc.StartPending, "Starting"},
			{svc.StopPending, "Stopping"},
			{svc.Running, "Running"},
		}

		for _, tc := range testCases {
			result := serviceManager.translateServiceState(tc.state)
			assert.Contains(t, result, tc.expected)
		}
	})

	t.Run("TranslateStartType", func(t *testing.T) {
		testCases := []struct {
			startType uint32
			expected  string
		}{
			{2, "Automatic"},
			{3, "Manual"},
			{4, "Disabled"},
		}

		for _, tc := range testCases {
			result := serviceManager.translateStartType(tc.startType)
			assert.Equal(t, tc.expected, result)
		}
	})
}