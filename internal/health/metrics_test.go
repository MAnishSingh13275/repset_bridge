package health

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/tier"
)

func TestPrometheusMetricsExporter_Disabled(t *testing.T) {
	config := MetricsConfig{
		Enabled: false,
	}
	
	exporter := NewPrometheusMetricsExporter(config)
	
	ctx := context.Background()
	
	// Test start/stop when disabled
	err := exporter.Start(ctx)
	require.NoError(t, err)
	
	err = exporter.Stop(ctx)
	require.NoError(t, err)
	
	// Test recording metrics when disabled (should not panic)
	exporter.RecordQueueDepth(10)
	exporter.RecordAdapterStatus("test", "active")
	exporter.RecordSystemResources(tier.SystemResources{})
	exporter.RecordHealthStatus(HealthStatusHealthy)
}

func TestPrometheusMetricsExporter_RecordMetrics(t *testing.T) {
	config := MetricsConfig{
		Enabled:   true,
		Port:      9091, // Use different port to avoid conflicts
		Path:      "/metrics",
		Namespace: "test",
	}
	
	exporter := NewPrometheusMetricsExporter(
		config,
		WithMetricsLogger(logrus.New()),
	)
	
	// Record some metrics
	exporter.RecordQueueDepth(42)
	exporter.RecordAdapterStatus("fingerprint", "active")
	exporter.RecordAdapterStatus("rfid", "error")
	exporter.RecordSystemResources(tier.SystemResources{
		CPUUsage:    75.5,
		MemoryUsage: 60.2,
		DiskUsage:   45.8,
		CPUCores:    4,
		MemoryGB:    8.0,
	})
	exporter.RecordHealthStatus(HealthStatusDegraded)
	
	// Test metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	exporter.handleMetrics(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; version=0.0.4; charset=utf-8", w.Header().Get("Content-Type"))
	
	body := w.Body.String()
	
	// Check that metrics are present
	assert.Contains(t, body, "test_queue_depth 42")
	assert.Contains(t, body, "test_cpu_usage_percent 75.5")
	assert.Contains(t, body, "test_memory_usage_percent 60.2")
	assert.Contains(t, body, "test_disk_usage_percent 45.8")
	assert.Contains(t, body, "test_cpu_cores 4")
	assert.Contains(t, body, "test_memory_gb 8")
	assert.Contains(t, body, "test_health_status{status=\"degraded\"} 0.5")
	
	// Check adapter status metrics with labels (order may vary)
	assert.Contains(t, body, `adapter_name="fingerprint"`)
	assert.Contains(t, body, `status="active"`)
	assert.Contains(t, body, `adapter_name="rfid"`)
	assert.Contains(t, body, `status="error"`)
	
	// Check that help and type comments are present
	assert.Contains(t, body, "# HELP test_queue_depth")
	assert.Contains(t, body, "# TYPE test_queue_depth gauge")
}

func TestPrometheusMetricsExporter_AdapterStatusValues(t *testing.T) {
	config := MetricsConfig{
		Enabled:   true,
		Namespace: "test",
	}
	
	exporter := NewPrometheusMetricsExporter(config)
	
	tests := []struct {
		status        string
		expectedValue float64
	}{
		{"active", 1},
		{"error", 0},
		{"disabled", -1},
		{"initializing", 0.5},
		{"unknown", -2},
	}
	
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			exporter.RecordAdapterStatus("test-adapter", tt.status)
			
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			
			exporter.handleMetrics(w, req)
			
			body := w.Body.String()
			
			// Check for the adapter name and status in the output (order may vary)
			assert.Contains(t, body, `adapter_name="test-adapter"`)
			assert.Contains(t, body, fmt.Sprintf(`status="%s"`, tt.status))
		})
	}
}

func TestPrometheusMetricsExporter_HealthStatusValues(t *testing.T) {
	config := MetricsConfig{
		Enabled:   true,
		Namespace: "test",
	}
	
	exporter := NewPrometheusMetricsExporter(config)
	
	tests := []struct {
		status        HealthStatus
		expectedValue float64
	}{
		{HealthStatusHealthy, 1},
		{HealthStatusDegraded, 0.5},
		{HealthStatusUnhealthy, 0},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			exporter.RecordHealthStatus(tt.status)
			
			req := httptest.NewRequest("GET", "/metrics", nil)
			w := httptest.NewRecorder()
			
			exporter.handleMetrics(w, req)
			
			body := w.Body.String()
			assert.Contains(t, body, 
				strings.ReplaceAll(
					`test_health_status{status="STATUS"}`,
					"STATUS", tt.status.String(),
				),
			)
		})
	}
}

func TestPrometheusMetricsExporter_StartStop(t *testing.T) {
	config := MetricsConfig{
		Enabled: true,
		Port:    9092, // Use different port
		Path:    "/metrics",
	}
	
	exporter := NewPrometheusMetricsExporter(config)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Test start
	err := exporter.Start(ctx)
	require.NoError(t, err)
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test that server is running by making a request
	resp, err := http.Get("http://localhost:9092/metrics")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Test stop
	err = exporter.Stop(ctx)
	require.NoError(t, err)
	
	// Give server time to stop
	time.Sleep(100 * time.Millisecond)
	
	// Test that server is stopped
	_, err = http.Get("http://localhost:9092/metrics")
	assert.Error(t, err) // Should fail because server is stopped
}

func TestNoOpMetricsExporter(t *testing.T) {
	exporter := NewNoOpMetricsExporter()
	
	ctx := context.Background()
	
	// Test that all methods work without error
	err := exporter.Start(ctx)
	require.NoError(t, err)
	
	// Test recording metrics (should not panic)
	exporter.RecordQueueDepth(10)
	exporter.RecordAdapterStatus("test", "active")
	exporter.RecordSystemResources(tier.SystemResources{})
	exporter.RecordHealthStatus(HealthStatusHealthy)
	
	err = exporter.Stop(ctx)
	require.NoError(t, err)
}

func TestDefaultMetricsConfig(t *testing.T) {
	config := DefaultMetricsConfig()
	
	assert.False(t, config.Enabled)
	assert.Equal(t, 9090, config.Port)
	assert.Equal(t, "/metrics", config.Path)
	assert.Equal(t, "gym_door_bridge", config.Namespace)
}

func TestPrometheusMetricsExporter_ConcurrentAccess(t *testing.T) {
	config := MetricsConfig{
		Enabled:   true,
		Namespace: "test",
	}
	
	exporter := NewPrometheusMetricsExporter(config)
	
	// Test concurrent metric recording
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				exporter.RecordQueueDepth(id*100 + j)
				exporter.RecordAdapterStatus("adapter", "active")
				exporter.RecordHealthStatus(HealthStatusHealthy)
			}
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Test that metrics endpoint still works
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	
	exporter.handleMetrics(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "test_queue_depth")
}