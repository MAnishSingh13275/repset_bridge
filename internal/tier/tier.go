package tier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Tier represents the performance tier of the system
type Tier string

const (
	TierLite   Tier = "lite"
	TierNormal Tier = "normal"
	TierFull   Tier = "full"
)

// String returns the string representation of the tier
func (t Tier) String() string {
	return string(t)
}

// IsValid checks if the tier is valid
func (t Tier) IsValid() bool {
	switch t {
	case TierLite, TierNormal, TierFull:
		return true
	default:
		return false
	}
}

// SystemResources represents the current system resource usage
type SystemResources struct {
	CPUCores     int     `json:"cpuCores"`
	MemoryGB     float64 `json:"memoryGB"`
	CPUUsage     float64 `json:"cpuUsage"`     // Percentage
	MemoryUsage  float64 `json:"memoryUsage"`  // Percentage
	DiskUsage    float64 `json:"diskUsage"`    // Percentage
	LastUpdated  time.Time `json:"lastUpdated"`
}

// TierConfig represents the configuration for each tier
type TierConfig struct {
	QueueMaxSize      int           `json:"queueMaxSize"`
	HeartbeatInterval time.Duration `json:"heartbeatInterval"`
	EnableWebUI       bool          `json:"enableWebUI"`
	EnableMetrics     bool          `json:"enableMetrics"`
}

// GetTierConfig returns the configuration for a given tier
func GetTierConfig(t Tier) TierConfig {
	switch t {
	case TierLite:
		return TierConfig{
			QueueMaxSize:      1000,
			HeartbeatInterval: 5 * time.Minute,
			EnableWebUI:       false,
			EnableMetrics:     false,
		}
	case TierNormal:
		return TierConfig{
			QueueMaxSize:      10000,
			HeartbeatInterval: 1 * time.Minute,
			EnableWebUI:       false,
			EnableMetrics:     true,
		}
	case TierFull:
		return TierConfig{
			QueueMaxSize:      50000,
			HeartbeatInterval: 30 * time.Second,
			EnableWebUI:       true,
			EnableMetrics:     true,
		}
	default:
		// Default to normal tier
		return GetTierConfig(TierNormal)
	}
}

// Detector handles system resource monitoring and tier detection
type Detector struct {
	mu                sync.RWMutex
	currentTier       Tier
	currentResources  SystemResources
	logger            *logrus.Logger
	resourceMonitor   ResourceMonitor
	evaluationInterval time.Duration
	
	// Callbacks
	onTierChange func(oldTier, newTier Tier)
}

// DetectorOption is a functional option for configuring the Detector
type DetectorOption func(*Detector)

// WithLogger sets the logger for the detector
func WithLogger(logger *logrus.Logger) DetectorOption {
	return func(d *Detector) {
		d.logger = logger
	}
}

// WithResourceMonitor sets a custom resource monitor
func WithResourceMonitor(monitor ResourceMonitor) DetectorOption {
	return func(d *Detector) {
		d.resourceMonitor = monitor
	}
}

// WithEvaluationInterval sets the interval for tier re-evaluation
func WithEvaluationInterval(interval time.Duration) DetectorOption {
	return func(d *Detector) {
		d.evaluationInterval = interval
	}
}

// WithTierChangeCallback sets a callback for tier changes
func WithTierChangeCallback(callback func(oldTier, newTier Tier)) DetectorOption {
	return func(d *Detector) {
		d.onTierChange = callback
	}
}

// NewDetector creates a new tier detector
func NewDetector(opts ...DetectorOption) *Detector {
	d := &Detector{
		currentTier:        TierNormal, // Default to normal
		logger:             logrus.New(),
		evaluationInterval: 30 * time.Second, // Default evaluation interval
	}
	
	// Apply options
	for _, opt := range opts {
		opt(d)
	}
	
	// Set default resource monitor if none provided
	if d.resourceMonitor == nil {
		d.resourceMonitor = NewSystemResourceMonitor()
	}
	
	return d
}

// Start begins the tier detection process
func (d *Detector) Start(ctx context.Context) error {
	d.logger.Info("Starting tier detector")
	
	// Initial tier detection
	if err := d.evaluateTier(); err != nil {
		d.logger.WithError(err).Error("Failed initial tier evaluation")
		return fmt.Errorf("failed initial tier evaluation: %w", err)
	}
	
	// Start periodic evaluation
	ticker := time.NewTicker(d.evaluationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Stopping tier detector")
			return ctx.Err()
		case <-ticker.C:
			if err := d.evaluateTier(); err != nil {
				d.logger.WithError(err).Error("Failed to evaluate tier")
			}
		}
	}
}

// GetCurrentTier returns the current performance tier
func (d *Detector) GetCurrentTier() Tier {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentTier
}

// GetCurrentResources returns the current system resources
func (d *Detector) GetCurrentResources() SystemResources {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.currentResources
}

// evaluateTier evaluates the current system resources and determines the appropriate tier
func (d *Detector) evaluateTier() error {
	// Get current system resources
	resources, err := d.resourceMonitor.GetSystemResources()
	if err != nil {
		return fmt.Errorf("failed to get system resources: %w", err)
	}
	
	// Determine appropriate tier based on resources
	newTier := d.determineTier(resources)
	
	d.mu.Lock()
	oldTier := d.currentTier
	d.currentResources = resources
	d.currentTier = newTier
	d.mu.Unlock()
	
	// Log tier change if it occurred
	if oldTier != newTier {
		d.logger.WithFields(logrus.Fields{
			"old_tier": oldTier,
			"new_tier": newTier,
			"cpu_cores": resources.CPUCores,
			"memory_gb": resources.MemoryGB,
			"cpu_usage": resources.CPUUsage,
			"memory_usage": resources.MemoryUsage,
		}).Info("Performance tier changed")
		
		// Call tier change callback if set
		if d.onTierChange != nil {
			go d.onTierChange(oldTier, newTier)
		}
	}
	
	return nil
}

// determineTier determines the appropriate tier based on system resources
func (d *Detector) determineTier(resources SystemResources) Tier {
	// Tier determination logic based on design document:
	// - Lite Mode: <2 CPU cores OR <2GB RAM
	// - Normal Mode: 2-4 CPU cores + ≥2GB RAM  
	// - Full Mode: >4 CPU cores + ≥8GB RAM
	
	if resources.CPUCores < 2 || resources.MemoryGB < 2.0 {
		return TierLite
	}
	
	if resources.CPUCores > 4 && resources.MemoryGB >= 8.0 {
		return TierFull
	}
	
	return TierNormal
}