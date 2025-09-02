package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gym-door-bridge/internal/tier"
)

// MetricsConfig holds configuration for the metrics exporter
type MetricsConfig struct {
	Enabled    bool   `json:"enabled"`    // Enable metrics collection
	Port       int    `json:"port"`       // Port for metrics endpoint
	Path       string `json:"path"`       // Path for metrics endpoint
	Namespace  string `json:"namespace"`  // Metrics namespace
}

// DefaultMetricsConfig returns the default metrics configuration
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Enabled:   false,
		Port:      9090,
		Path:      "/metrics",
		Namespace: "gym_door_bridge",
	}
}

// PrometheusMetricsExporter implements MetricsExporter using Prometheus-style metrics
// This is a simple implementation without external dependencies
type PrometheusMetricsExporter struct {
	mu       sync.RWMutex
	config   MetricsConfig
	logger   *logrus.Logger
	server   *http.Server
	
	// Metrics storage (simple in-memory storage)
	metrics  map[string]MetricValue
	
	// Control
	isRunning bool
}

// MetricValue represents a metric value with timestamp
type MetricValue struct {
	Value      float64           `json:"value"`
	Timestamp  time.Time         `json:"timestamp"`
	Labels     map[string]string `json:"labels,omitempty"`
	MetricName string            `json:"metricName,omitempty"`
}

// PrometheusMetricsExporterOption is a functional option for configuring the PrometheusMetricsExporter
type PrometheusMetricsExporterOption func(*PrometheusMetricsExporter)

// WithMetricsLogger sets the logger for the metrics exporter
func WithMetricsLogger(logger *logrus.Logger) PrometheusMetricsExporterOption {
	return func(p *PrometheusMetricsExporter) {
		p.logger = logger
	}
}

// NewPrometheusMetricsExporter creates a new Prometheus-style metrics exporter
func NewPrometheusMetricsExporter(config MetricsConfig, opts ...PrometheusMetricsExporterOption) *PrometheusMetricsExporter {
	p := &PrometheusMetricsExporter{
		config:  config,
		logger:  logrus.New(),
		metrics: make(map[string]MetricValue),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(p)
	}
	
	return p
}

// Start starts the metrics exporter
func (p *PrometheusMetricsExporter) Start(ctx context.Context) error {
	if !p.config.Enabled {
		p.logger.Info("Metrics exporter disabled")
		return nil
	}
	
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		return fmt.Errorf("metrics exporter is already running")
	}
	p.isRunning = true
	p.mu.Unlock()
	
	p.logger.Info("Starting metrics exporter", "port", p.config.Port, "path", p.config.Path)
	
	// Set up HTTP server for metrics endpoint
	mux := http.NewServeMux()
	mux.HandleFunc(p.config.Path, p.handleMetrics)
	
	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.Port),
		Handler: mux,
	}
	
	// Start HTTP server in a goroutine
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			p.logger.WithError(err).Error("Metrics HTTP server failed")
		}
	}()
	
	p.logger.Info("Metrics endpoint started", "url", fmt.Sprintf("http://localhost:%d%s", p.config.Port, p.config.Path))
	
	return nil
}

// Stop stops the metrics exporter
func (p *PrometheusMetricsExporter) Stop(ctx context.Context) error {
	if !p.config.Enabled {
		return nil
	}
	
	p.mu.Lock()
	if !p.isRunning {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()
	
	p.logger.Info("Stopping metrics exporter")
	
	// Stop HTTP server
	if p.server != nil {
		if err := p.server.Shutdown(ctx); err != nil {
			p.logger.WithError(err).Error("Failed to shutdown metrics HTTP server")
			return err
		}
	}
	
	p.mu.Lock()
	p.isRunning = false
	p.mu.Unlock()
	
	return nil
}

// RecordQueueDepth records the current queue depth
func (p *PrometheusMetricsExporter) RecordQueueDepth(depth int) {
	if !p.config.Enabled {
		return
	}
	
	p.recordMetric("queue_depth", float64(depth), nil)
}

// RecordAdapterStatus records the status of an adapter
func (p *PrometheusMetricsExporter) RecordAdapterStatus(name string, status string) {
	if !p.config.Enabled {
		return
	}
	
	// Convert status to numeric value for easier monitoring
	var statusValue float64
	switch status {
	case "active":
		statusValue = 1
	case "error":
		statusValue = 0
	case "disabled":
		statusValue = -1
	case "initializing":
		statusValue = 0.5
	default:
		statusValue = -2 // Unknown status
	}
	
	labels := map[string]string{
		"adapter_name": name,
		"status":       status,
	}
	
	p.recordMetric("adapter_status", statusValue, labels)
}

// RecordSystemResources records system resource usage
func (p *PrometheusMetricsExporter) RecordSystemResources(resources tier.SystemResources) {
	if !p.config.Enabled {
		return
	}
	
	p.recordMetric("cpu_usage_percent", resources.CPUUsage, nil)
	p.recordMetric("memory_usage_percent", resources.MemoryUsage, nil)
	p.recordMetric("disk_usage_percent", resources.DiskUsage, nil)
	p.recordMetric("cpu_cores", float64(resources.CPUCores), nil)
	p.recordMetric("memory_gb", resources.MemoryGB, nil)
}

// RecordHealthStatus records the overall health status
func (p *PrometheusMetricsExporter) RecordHealthStatus(status HealthStatus) {
	if !p.config.Enabled {
		return
	}
	
	// Convert health status to numeric value
	var statusValue float64
	switch status {
	case HealthStatusHealthy:
		statusValue = 1
	case HealthStatusDegraded:
		statusValue = 0.5
	case HealthStatusUnhealthy:
		statusValue = 0
	default:
		statusValue = -1 // Unknown status
	}
	
	labels := map[string]string{
		"status": status.String(),
	}
	
	p.recordMetric("health_status", statusValue, labels)
}

// recordMetric records a metric value
func (p *PrometheusMetricsExporter) recordMetric(name string, value float64, labels map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	metricName := p.config.Namespace + "_" + name
	
	// Create a unique key for metrics with labels
	key := metricName
	if len(labels) > 0 {
		// Add labels to the key to make it unique
		for k, v := range labels {
			key += "_" + k + "_" + v
		}
	}
	
	p.metrics[key] = MetricValue{
		Value:      value,
		Timestamp:  time.Now(),
		Labels:     labels,
		MetricName: metricName, // Store the original metric name for output
	}
}

// handleMetrics handles HTTP requests for metrics
func (p *PrometheusMetricsExporter) handleMetrics(w http.ResponseWriter, r *http.Request) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	
	// Write metrics in Prometheus format
	for _, metric := range p.metrics {
		metricName := metric.MetricName
		if metricName == "" {
			// Fallback for metrics without MetricName (shouldn't happen with new code)
			continue
		}
		
		// Write help and type comments
		fmt.Fprintf(w, "# HELP %s %s\n", metricName, metricName)
		fmt.Fprintf(w, "# TYPE %s gauge\n", metricName)
		
		// Write metric value with labels
		if len(metric.Labels) > 0 {
			labelStr := ""
			first := true
			for key, value := range metric.Labels {
				if !first {
					labelStr += ","
				}
				labelStr += fmt.Sprintf(`%s="%s"`, key, value)
				first = false
			}
			fmt.Fprintf(w, "%s{%s} %f %d\n", metricName, labelStr, metric.Value, metric.Timestamp.Unix()*1000)
		} else {
			fmt.Fprintf(w, "%s %f %d\n", metricName, metric.Value, metric.Timestamp.Unix()*1000)
		}
	}
}

// NoOpMetricsExporter is a no-op implementation of MetricsExporter
type NoOpMetricsExporter struct{}

// NewNoOpMetricsExporter creates a new no-op metrics exporter
func NewNoOpMetricsExporter() *NoOpMetricsExporter {
	return &NoOpMetricsExporter{}
}

// Start does nothing
func (n *NoOpMetricsExporter) Start(ctx context.Context) error {
	return nil
}

// Stop does nothing
func (n *NoOpMetricsExporter) Stop(ctx context.Context) error {
	return nil
}

// RecordQueueDepth does nothing
func (n *NoOpMetricsExporter) RecordQueueDepth(depth int) {}

// RecordAdapterStatus does nothing
func (n *NoOpMetricsExporter) RecordAdapterStatus(name string, status string) {}

// RecordSystemResources does nothing
func (n *NoOpMetricsExporter) RecordSystemResources(resources tier.SystemResources) {}

// RecordHealthStatus does nothing
func (n *NoOpMetricsExporter) RecordHealthStatus(status HealthStatus) {}