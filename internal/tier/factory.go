package tier

import (
	"time"

	"github.com/sirupsen/logrus"
)

// DetectorFactory provides a convenient way to create tier detectors
type DetectorFactory struct {
	logger *logrus.Logger
}

// NewDetectorFactory creates a new detector factory
func NewDetectorFactory(logger *logrus.Logger) *DetectorFactory {
	return &DetectorFactory{
		logger: logger,
	}
}

// CreateDetector creates a new tier detector with default settings
func (f *DetectorFactory) CreateDetector() *Detector {
	return NewDetector(
		WithLogger(f.logger),
		WithEvaluationInterval(30*time.Second),
	)
}

// CreateDetectorWithCallback creates a new tier detector with a tier change callback
func (f *DetectorFactory) CreateDetectorWithCallback(callback func(oldTier, newTier Tier)) *Detector {
	return NewDetector(
		WithLogger(f.logger),
		WithEvaluationInterval(30*time.Second),
		WithTierChangeCallback(callback),
	)
}

// CreateDetectorForTesting creates a detector with mock resource monitor for testing
func (f *DetectorFactory) CreateDetectorForTesting(resources SystemResources) *Detector {
	monitor := NewMockResourceMonitor(resources)
	return NewDetector(
		WithLogger(f.logger),
		WithResourceMonitor(monitor),
		WithEvaluationInterval(100*time.Millisecond), // Fast evaluation for testing
	)
}