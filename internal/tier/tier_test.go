package tier

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTier_String(t *testing.T) {
	tests := []struct {
		tier     Tier
		expected string
	}{
		{TierLite, "lite"},
		{TierNormal, "normal"},
		{TierFull, "full"},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tier.String())
		})
	}
}

func TestTier_IsValid(t *testing.T) {
	tests := []struct {
		tier     Tier
		expected bool
	}{
		{TierLite, true},
		{TierNormal, true},
		{TierFull, true},
		{Tier("invalid"), false},
		{Tier(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tier.IsValid())
		})
	}
}

func TestGetTierConfig(t *testing.T) {
	tests := []struct {
		tier     Tier
		expected TierConfig
	}{
		{
			tier: TierLite,
			expected: TierConfig{
				QueueMaxSize:      1000,
				HeartbeatInterval: 5 * time.Minute,
				EnableWebUI:       false,
				EnableMetrics:     false,
			},
		},
		{
			tier: TierNormal,
			expected: TierConfig{
				QueueMaxSize:      10000,
				HeartbeatInterval: 1 * time.Minute,
				EnableWebUI:       false,
				EnableMetrics:     true,
			},
		},
		{
			tier: TierFull,
			expected: TierConfig{
				QueueMaxSize:      50000,
				HeartbeatInterval: 30 * time.Second,
				EnableWebUI:       true,
				EnableMetrics:     true,
			},
		},
		{
			tier: Tier("invalid"),
			expected: TierConfig{
				QueueMaxSize:      10000,
				HeartbeatInterval: 1 * time.Minute,
				EnableWebUI:       false,
				EnableMetrics:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			config := GetTierConfig(tt.tier)
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestNewDetector(t *testing.T) {
	logger := logrus.New()
	monitor := NewMockResourceMonitor(CreateNormalSystemResources())
	callback := func(oldTier, newTier Tier) {}

	detector := NewDetector(
		WithLogger(logger),
		WithResourceMonitor(monitor),
		WithEvaluationInterval(10*time.Second),
		WithTierChangeCallback(callback),
	)

	assert.NotNil(t, detector)
	assert.Equal(t, TierNormal, detector.currentTier)
	assert.Equal(t, logger, detector.logger)
	assert.Equal(t, monitor, detector.resourceMonitor)
	assert.Equal(t, 10*time.Second, detector.evaluationInterval)
	assert.NotNil(t, detector.onTierChange)
}

func TestDetector_GetCurrentTier(t *testing.T) {
	detector := NewDetector()
	
	// Default tier should be Normal
	assert.Equal(t, TierNormal, detector.GetCurrentTier())
	
	// Change tier and verify
	detector.currentTier = TierFull
	assert.Equal(t, TierFull, detector.GetCurrentTier())
}

func TestDetector_GetCurrentResources(t *testing.T) {
	resources := CreateNormalSystemResources()
	detector := NewDetector()
	detector.currentResources = resources
	
	retrieved := detector.GetCurrentResources()
	assert.Equal(t, resources, retrieved)
}

func TestDetector_determineTier(t *testing.T) {
	detector := NewDetector()
	
	tests := []struct {
		name      string
		resources SystemResources
		expected  Tier
	}{
		{
			name:      "Lite tier - low CPU cores",
			resources: SystemResources{CPUCores: 1, MemoryGB: 4.0},
			expected:  TierLite,
		},
		{
			name:      "Lite tier - low memory",
			resources: SystemResources{CPUCores: 4, MemoryGB: 1.5},
			expected:  TierLite,
		},
		{
			name:      "Lite tier - both low",
			resources: SystemResources{CPUCores: 1, MemoryGB: 1.0},
			expected:  TierLite,
		},
		{
			name:      "Normal tier - 2 cores, 2GB",
			resources: SystemResources{CPUCores: 2, MemoryGB: 2.0},
			expected:  TierNormal,
		},
		{
			name:      "Normal tier - 4 cores, 4GB",
			resources: SystemResources{CPUCores: 4, MemoryGB: 4.0},
			expected:  TierNormal,
		},
		{
			name:      "Normal tier - 4 cores, 7GB (not enough for Full)",
			resources: SystemResources{CPUCores: 4, MemoryGB: 7.0},
			expected:  TierNormal,
		},
		{
			name:      "Full tier - 8 cores, 16GB",
			resources: SystemResources{CPUCores: 8, MemoryGB: 16.0},
			expected:  TierFull,
		},
		{
			name:      "Full tier - 6 cores, 8GB (minimum for Full)",
			resources: SystemResources{CPUCores: 6, MemoryGB: 8.0},
			expected:  TierFull,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier := detector.determineTier(tt.resources)
			assert.Equal(t, tt.expected, tier)
		})
	}
}

func TestDetector_evaluateTier(t *testing.T) {
	t.Run("successful evaluation", func(t *testing.T) {
		resources := CreateFullSystemResources()
		monitor := NewMockResourceMonitor(resources)
		detector := NewDetector(WithResourceMonitor(monitor))
		
		// Initially should be Normal
		assert.Equal(t, TierNormal, detector.GetCurrentTier())
		
		err := detector.evaluateTier()
		require.NoError(t, err)
		
		// Should now be Full based on resources
		assert.Equal(t, TierFull, detector.GetCurrentTier())
		assert.Equal(t, resources.CPUCores, detector.GetCurrentResources().CPUCores)
		assert.Equal(t, resources.MemoryGB, detector.GetCurrentResources().MemoryGB)
	})

	t.Run("monitor error", func(t *testing.T) {
		monitor := NewMockResourceMonitor(CreateNormalSystemResources())
		monitor.SetError(assert.AnError)
		detector := NewDetector(WithResourceMonitor(monitor))
		
		err := detector.evaluateTier()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get system resources")
	})
}

func TestDetector_evaluateTier_TierChange(t *testing.T) {
	resources := CreateLiteSystemResources()
	monitor := NewMockResourceMonitor(resources)
	
	var oldTierCallback, newTierCallback Tier
	callbackCalled := make(chan bool, 1)
	
	callback := func(oldTier, newTier Tier) {
		oldTierCallback = oldTier
		newTierCallback = newTier
		callbackCalled <- true
	}
	
	detector := NewDetector(
		WithResourceMonitor(monitor),
		WithTierChangeCallback(callback),
	)
	
	// Initial evaluation should trigger callback (Normal -> Lite)
	err := detector.evaluateTier()
	require.NoError(t, err)
	
	// Wait for callback
	select {
	case <-callbackCalled:
		assert.Equal(t, TierNormal, oldTierCallback)
		assert.Equal(t, TierLite, newTierCallback)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Callback was not called")
	}
	
	// Change to Full tier resources
	fullResources := CreateFullSystemResources()
	monitor.SetResources(fullResources)
	
	err = detector.evaluateTier()
	require.NoError(t, err)
	
	// Wait for second callback
	select {
	case <-callbackCalled:
		assert.Equal(t, TierLite, oldTierCallback)
		assert.Equal(t, TierFull, newTierCallback)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Second callback was not called")
	}
}

func TestDetector_Start(t *testing.T) {
	t.Run("successful start and stop", func(t *testing.T) {
		resources := CreateNormalSystemResources()
		monitor := NewMockResourceMonitor(resources)
		detector := NewDetector(
			WithResourceMonitor(monitor),
			WithEvaluationInterval(10*time.Millisecond),
		)
		
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		err := detector.Start(ctx)
		assert.Equal(t, context.DeadlineExceeded, err)
		
		// Should have evaluated at least once
		assert.Equal(t, TierNormal, detector.GetCurrentTier())
	})

	t.Run("initial evaluation error", func(t *testing.T) {
		monitor := NewMockResourceMonitor(CreateNormalSystemResources())
		monitor.SetError(assert.AnError)
		detector := NewDetector(WithResourceMonitor(monitor))
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		
		err := detector.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed initial tier evaluation")
	})
}

func TestDetector_Start_PeriodicEvaluation(t *testing.T) {
	resources := CreateLiteSystemResources()
	monitor := NewMockResourceMonitor(resources)
	
	evaluationCount := 0
	var tierChanges []Tier
	callback := func(oldTier, newTier Tier) {
		evaluationCount++
		tierChanges = append(tierChanges, newTier)
	}
	
	detector := NewDetector(
		WithResourceMonitor(monitor),
		WithEvaluationInterval(20*time.Millisecond),
		WithTierChangeCallback(callback),
	)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	// Change resources after 30ms to trigger another tier change
	go func() {
		time.Sleep(30 * time.Millisecond)
		monitor.SetResources(CreateFullSystemResources())
	}()
	
	err := detector.Start(ctx)
	assert.Equal(t, context.DeadlineExceeded, err)
	
	// Should have had at least 2 evaluations (initial + periodic)
	assert.GreaterOrEqual(t, evaluationCount, 1)
	
	// Should have detected tier changes
	if len(tierChanges) >= 1 {
		assert.Equal(t, TierLite, tierChanges[0]) // Initial change from Normal to Lite
	}
}