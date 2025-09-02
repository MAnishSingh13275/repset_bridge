package tier

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTierDetector_Integration(t *testing.T) {
	t.Run("factory creates working detector", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
		
		factory := NewDetectorFactory(logger)
		detector := factory.CreateDetector()
		
		assert.NotNil(t, detector)
		assert.Equal(t, TierNormal, detector.GetCurrentTier())
		
		// Trigger initial evaluation to populate resources
		err := detector.evaluateTier()
		require.NoError(t, err)
		
		// Test that we can get current resources
		resources := detector.GetCurrentResources()
		assert.Greater(t, resources.CPUCores, 0)
	})

	t.Run("tier change callback integration", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		
		factory := NewDetectorFactory(logger)
		
		var tierChanges []struct {
			old, new Tier
		}
		
		callback := func(oldTier, newTier Tier) {
			tierChanges = append(tierChanges, struct{ old, new Tier }{oldTier, newTier})
		}
		
		detector := factory.CreateDetectorWithCallback(callback)
		
		// Use mock monitor to control tier changes
		mockMonitor := NewMockResourceMonitor(CreateLiteSystemResources())
		detector.resourceMonitor = mockMonitor
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		// Start detector (this will trigger initial evaluation)
		go func() {
			detector.Start(ctx)
		}()
		
		// Wait for initial evaluation
		time.Sleep(10 * time.Millisecond)
		
		// Change to full tier
		mockMonitor.SetResources(CreateFullSystemResources())
		
		// Trigger another evaluation
		err := detector.evaluateTier()
		require.NoError(t, err)
		
		// Wait for callback processing
		time.Sleep(10 * time.Millisecond)
		
		// Should have at least one tier change
		assert.GreaterOrEqual(t, len(tierChanges), 1)
		
		// First change should be from Normal to Lite
		if len(tierChanges) >= 1 {
			assert.Equal(t, TierNormal, tierChanges[0].old)
			assert.Equal(t, TierLite, tierChanges[0].new)
		}
		
		// Second change should be from Lite to Full
		if len(tierChanges) >= 2 {
			assert.Equal(t, TierLite, tierChanges[1].old)
			assert.Equal(t, TierFull, tierChanges[1].new)
		}
	})

	t.Run("tier configuration matches detected tier", func(t *testing.T) {
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)
		
		factory := NewDetectorFactory(logger)
		
		// Test with different resource scenarios
		testCases := []struct {
			name      string
			resources SystemResources
			expected  Tier
		}{
			{
				name:      "lite system",
				resources: CreateLiteSystemResources(),
				expected:  TierLite,
			},
			{
				name:      "normal system",
				resources: CreateNormalSystemResources(),
				expected:  TierNormal,
			},
			{
				name:      "full system",
				resources: CreateFullSystemResources(),
				expected:  TierFull,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				detector := factory.CreateDetectorForTesting(tc.resources)
				
				// Trigger evaluation
				err := detector.evaluateTier()
				require.NoError(t, err)
				
				// Check detected tier
				detectedTier := detector.GetCurrentTier()
				assert.Equal(t, tc.expected, detectedTier)
				
				// Check that tier configuration is appropriate
				config := GetTierConfig(detectedTier)
				
				switch detectedTier {
				case TierLite:
					assert.Equal(t, 1000, config.QueueMaxSize)
					assert.Equal(t, 5*time.Minute, config.HeartbeatInterval)
					assert.False(t, config.EnableWebUI)
					assert.False(t, config.EnableMetrics)
				case TierNormal:
					assert.Equal(t, 10000, config.QueueMaxSize)
					assert.Equal(t, 1*time.Minute, config.HeartbeatInterval)
					assert.False(t, config.EnableWebUI)
					assert.True(t, config.EnableMetrics)
				case TierFull:
					assert.Equal(t, 50000, config.QueueMaxSize)
					assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
					assert.True(t, config.EnableWebUI)
					assert.True(t, config.EnableMetrics)
				}
			})
		}
	})
}

func TestDetectorFactory(t *testing.T) {
	logger := logrus.New()
	factory := NewDetectorFactory(logger)
	
	t.Run("creates detector with default settings", func(t *testing.T) {
		detector := factory.CreateDetector()
		
		assert.NotNil(t, detector)
		assert.Equal(t, logger, detector.logger)
		assert.Equal(t, 30*time.Second, detector.evaluationInterval)
		assert.Nil(t, detector.onTierChange)
	})
	
	t.Run("creates detector with callback", func(t *testing.T) {
		callback := func(oldTier, newTier Tier) {}
		detector := factory.CreateDetectorWithCallback(callback)
		
		assert.NotNil(t, detector)
		assert.Equal(t, logger, detector.logger)
		assert.Equal(t, 30*time.Second, detector.evaluationInterval)
		assert.NotNil(t, detector.onTierChange)
	})
	
	t.Run("creates detector for testing", func(t *testing.T) {
		resources := CreateNormalSystemResources()
		detector := factory.CreateDetectorForTesting(resources)
		
		assert.NotNil(t, detector)
		assert.Equal(t, logger, detector.logger)
		assert.Equal(t, 100*time.Millisecond, detector.evaluationInterval)
		
		// Should use mock monitor
		mockMonitor, ok := detector.resourceMonitor.(*MockResourceMonitor)
		assert.True(t, ok)
		assert.NotNil(t, mockMonitor)
	})
}