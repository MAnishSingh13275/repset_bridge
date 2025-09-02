package logging

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDegradationLevelString(t *testing.T) {
	tests := []struct {
		level    DegradationLevel
		expected string
	}{
		{DegradationNone, "none"},
		{DegradationMinor, "minor"},
		{DegradationModerate, "moderate"},
		{DegradationSevere, "severe"},
		{DegradationCritical, "critical"},
		{DegradationLevel(999), "unknown"}, // Invalid level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestNewDegradationManager(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	assert.NotNil(t, dm)
	assert.Equal(t, logger, dm.logger)
	assert.Equal(t, DegradationNone, dm.currentLevel)
	assert.NotNil(t, dm.actions)
	assert.NotNil(t, dm.activeActions)
	assert.True(t, len(dm.actions) > 0) // Should have default actions
}

func TestRegisterAction(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	initialCount := len(dm.actions)

	action := DegradationAction{
		Name:        "test_action",
		Description: "Test degradation action",
		Level:       DegradationMinor,
		Priority:    1,
	}

	dm.RegisterAction(action)

	assert.Equal(t, initialCount+1, len(dm.actions))
	
	// Find the registered action
	found := false
	for _, a := range dm.actions {
		if a.Name == "test_action" {
			found = true
			assert.Equal(t, action.Description, a.Description)
			assert.Equal(t, action.Level, a.Level)
			assert.Equal(t, action.Priority, a.Priority)
			break
		}
	}
	assert.True(t, found, "Registered action not found")
}

func TestDegradeToLevel(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Initially at none level
	assert.Equal(t, DegradationNone, dm.GetCurrentLevel())

	// Degrade to minor level
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test degradation")
	assert.NoError(t, err)
	assert.Equal(t, DegradationMinor, dm.GetCurrentLevel())

	// Check that minor level actions are active
	activeActions := dm.GetActiveActions()
	assert.True(t, len(activeActions) > 0)

	// Degrade to moderate level
	err = dm.DegradeToLevel(ctx, DegradationModerate, "further degradation")
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, dm.GetCurrentLevel())

	// Should have more active actions now
	newActiveActions := dm.GetActiveActions()
	assert.True(t, len(newActiveActions) >= len(activeActions))
}

func TestDegradeToLevelSameOrLower(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Degrade to moderate level first
	err := dm.DegradeToLevel(ctx, DegradationModerate, "initial degradation")
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, dm.GetCurrentLevel())

	// Try to degrade to same level (should be no-op)
	err = dm.DegradeToLevel(ctx, DegradationModerate, "same level")
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, dm.GetCurrentLevel())

	// Try to degrade to lower level (should be no-op)
	err = dm.DegradeToLevel(ctx, DegradationMinor, "lower level")
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, dm.GetCurrentLevel()) // Should remain at moderate
}

func TestRestoreToLevel(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// First degrade to severe level
	err := dm.DegradeToLevel(ctx, DegradationSevere, "test degradation")
	assert.NoError(t, err)
	assert.Equal(t, DegradationSevere, dm.GetCurrentLevel())

	activeActions := dm.GetActiveActions()
	assert.True(t, len(activeActions) > 0)

	// Restore to moderate level
	err = dm.RestoreToLevel(ctx, DegradationModerate, "partial restoration")
	assert.NoError(t, err)
	assert.Equal(t, DegradationModerate, dm.GetCurrentLevel())

	// Should have fewer active actions now
	newActiveActions := dm.GetActiveActions()
	assert.True(t, len(newActiveActions) <= len(activeActions))

	// Restore to none level
	err = dm.RestoreToLevel(ctx, DegradationNone, "full restoration")
	assert.NoError(t, err)
	assert.Equal(t, DegradationNone, dm.GetCurrentLevel())

	// Should have no active actions
	finalActiveActions := dm.GetActiveActions()
	assert.Equal(t, 0, len(finalActiveActions))
}

func TestRestoreToLevelSameOrHigher(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Degrade to minor level first
	err := dm.DegradeToLevel(ctx, DegradationMinor, "initial degradation")
	assert.NoError(t, err)
	assert.Equal(t, DegradationMinor, dm.GetCurrentLevel())

	// Try to restore to same level (should be no-op)
	err = dm.RestoreToLevel(ctx, DegradationMinor, "same level")
	assert.NoError(t, err)
	assert.Equal(t, DegradationMinor, dm.GetCurrentLevel())

	// Try to restore to higher level (should be no-op)
	err = dm.RestoreToLevel(ctx, DegradationModerate, "higher level")
	assert.NoError(t, err)
	assert.Equal(t, DegradationMinor, dm.GetCurrentLevel()) // Should remain at minor
}

func TestGetDegradationDuration(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Initially no degradation
	duration := dm.GetDegradationDuration()
	assert.Equal(t, time.Duration(0), duration)

	// Degrade and check duration
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test degradation")
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Small delay

	duration = dm.GetDegradationDuration()
	assert.True(t, duration > 0)
	assert.True(t, duration >= 10*time.Millisecond)

	// Restore and check duration is reset
	err = dm.RestoreToLevel(ctx, DegradationNone, "restoration")
	assert.NoError(t, err)

	duration = dm.GetDegradationDuration()
	assert.Equal(t, time.Duration(0), duration)
}

func TestIsActionActive(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Initially no actions are active
	assert.False(t, dm.IsActionActive("reduce_heartbeat_frequency"))

	// Degrade to minor level
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test degradation")
	assert.NoError(t, err)

	// Minor level actions should be active
	assert.True(t, dm.IsActionActive("reduce_heartbeat_frequency"))
	assert.True(t, dm.IsActionActive("disable_detailed_metrics"))

	// Moderate level actions should not be active yet
	assert.False(t, dm.IsActionActive("reduce_queue_size"))
}

func TestHandleResourceConstraint(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	tests := []struct {
		name           string
		usage          float64
		threshold      float64
		expectedLevel  DegradationLevel
	}{
		{
			name:          "normal usage",
			usage:         50.0,
			threshold:     100.0,
			expectedLevel: DegradationNone,
		},
		{
			name:          "elevated usage",
			usage:         75.0,
			threshold:     100.0,
			expectedLevel: DegradationMinor,
		},
		{
			name:          "moderate usage",
			usage:         85.0,
			threshold:     100.0,
			expectedLevel: DegradationModerate,
		},
		{
			name:          "severe usage",
			usage:         92.0,
			threshold:     100.0,
			expectedLevel: DegradationSevere,
		},
		{
			name:          "critical usage",
			usage:         98.0,
			threshold:     100.0,
			expectedLevel: DegradationCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to none level
			dm.RestoreToLevel(ctx, DegradationNone, "reset")

			err := dm.HandleResourceConstraint(ctx, "memory", tt.usage, tt.threshold)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, dm.GetCurrentLevel())
		})
	}
}

func TestHandleResourceConstraintRestore(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// First degrade due to high usage
	err := dm.HandleResourceConstraint(ctx, "memory", 95.0, 100.0)
	assert.NoError(t, err)
	assert.Equal(t, DegradationCritical, dm.GetCurrentLevel())

	// Then restore due to normal usage
	err = dm.HandleResourceConstraint(ctx, "memory", 50.0, 100.0)
	assert.NoError(t, err)
	assert.Equal(t, DegradationNone, dm.GetCurrentLevel())
}

func TestGetDegradationStatus(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Initially not degraded
	status := dm.GetDegradationStatus()
	assert.Equal(t, "none", status["current_level"])
	assert.Equal(t, 0, status["active_actions"])
	assert.Equal(t, false, status["degraded"])
	assert.NotContains(t, status, "degradation_duration")
	assert.NotContains(t, status, "degradation_start")

	// Degrade and check status
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test degradation")
	assert.NoError(t, err)

	status = dm.GetDegradationStatus()
	assert.Equal(t, "minor", status["current_level"])
	assert.True(t, status["active_actions"].(int) > 0)
	assert.Equal(t, true, status["degraded"])
	assert.Contains(t, status, "degradation_duration")
	assert.Contains(t, status, "degradation_start")
	assert.Contains(t, status, "active_action_names")
}

func TestDegradationWithCustomActions(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Register custom actions with callbacks
	actionCalled := false
	rollbackCalled := false

	customAction := DegradationAction{
		Name:        "custom_test_action",
		Description: "Custom test action",
		Level:       DegradationMinor,
		Priority:    1,
		Action: func(ctx context.Context) error {
			actionCalled = true
			return nil
		},
		Rollback: func(ctx context.Context) error {
			rollbackCalled = true
			return nil
		},
	}

	dm.RegisterAction(customAction)

	// Degrade to trigger the action
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test custom action")
	assert.NoError(t, err)
	assert.True(t, actionCalled)
	assert.True(t, dm.IsActionActive("custom_test_action"))

	// Restore to trigger rollback
	err = dm.RestoreToLevel(ctx, DegradationNone, "test rollback")
	assert.NoError(t, err)
	assert.True(t, rollbackCalled)
	assert.False(t, dm.IsActionActive("custom_test_action"))
}

func TestDegradationActionFailure(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Register action that fails
	failingAction := DegradationAction{
		Name:        "failing_action",
		Description: "Action that fails",
		Level:       DegradationMinor,
		Priority:    1,
		Action: func(ctx context.Context) error {
			return assert.AnError
		},
	}

	dm.RegisterAction(failingAction)

	// Degrade should not fail even if action fails
	err := dm.DegradeToLevel(ctx, DegradationMinor, "test failing action")
	assert.NoError(t, err)
	assert.Equal(t, DegradationMinor, dm.GetCurrentLevel())

	// The failing action should not be marked as active
	assert.False(t, dm.IsActionActive("failing_action"))
}

func TestConcurrentDegradation(t *testing.T) {
	logger := logrus.New()
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	// Test concurrent degradation and restoration
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			dm.DegradeToLevel(ctx, DegradationMinor, "concurrent test")
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			dm.RestoreToLevel(ctx, DegradationNone, "concurrent test")
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should not panic and should be in a valid state
	level := dm.GetCurrentLevel()
	assert.True(t, level >= DegradationNone && level <= DegradationCritical)
}

func BenchmarkDegradeToLevel(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dm.DegradeToLevel(ctx, DegradationMinor, "benchmark test")
		dm.RestoreToLevel(ctx, DegradationNone, "benchmark reset")
	}
}

func BenchmarkHandleResourceConstraint(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce logging overhead
	dm := NewDegradationManager(logger)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usage := float64(50 + (i%50)) // Vary usage between 50-99%
		dm.HandleResourceConstraint(ctx, "memory", usage, 100.0)
	}
}