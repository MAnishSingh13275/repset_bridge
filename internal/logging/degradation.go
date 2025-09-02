package logging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DegradationLevel represents different levels of service degradation
type DegradationLevel int

const (
	// Normal operation - no degradation
	DegradationNone DegradationLevel = iota
	// Minor degradation - reduce non-essential features
	DegradationMinor
	// Moderate degradation - reduce performance and features
	DegradationModerate
	// Severe degradation - minimal functionality only
	DegradationSevere
	// Critical degradation - emergency mode
	DegradationCritical
)

// String returns the string representation of degradation level
func (d DegradationLevel) String() string {
	switch d {
	case DegradationNone:
		return "none"
	case DegradationMinor:
		return "minor"
	case DegradationModerate:
		return "moderate"
	case DegradationSevere:
		return "severe"
	case DegradationCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// DegradationAction represents an action to take when degrading service
type DegradationAction struct {
	Name        string                                    `json:"name"`
	Description string                                    `json:"description"`
	Level       DegradationLevel                          `json:"level"`
	Action      func(ctx context.Context) error           `json:"-"`
	Rollback    func(ctx context.Context) error           `json:"-"`
	Priority    int                                       `json:"priority"` // Lower number = higher priority
}

// DegradationManager manages graceful service degradation
type DegradationManager struct {
	logger           *logrus.Logger
	currentLevel     DegradationLevel
	actions          []DegradationAction
	activeActions    map[string]bool
	degradationStart time.Time
	mutex            sync.RWMutex
}

// NewDegradationManager creates a new degradation manager
func NewDegradationManager(logger *logrus.Logger) *DegradationManager {
	dm := &DegradationManager{
		logger:        logger,
		currentLevel:  DegradationNone,
		actions:       make([]DegradationAction, 0),
		activeActions: make(map[string]bool),
	}

	// Set up default degradation actions
	dm.setupDefaultActions()
	
	return dm
}

// setupDefaultActions configures default degradation actions
func (dm *DegradationManager) setupDefaultActions() {
	// Minor degradation actions
	dm.RegisterAction(DegradationAction{
		Name:        "reduce_heartbeat_frequency",
		Description: "Reduce heartbeat frequency to conserve resources",
		Level:       DegradationMinor,
		Priority:    1,
		Action: func(ctx context.Context) error {
			dm.logger.Info("Reducing heartbeat frequency for resource conservation")
			// This would be implemented by the heartbeat manager
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Restoring normal heartbeat frequency")
			return nil
		},
	})

	dm.RegisterAction(DegradationAction{
		Name:        "disable_detailed_metrics",
		Description: "Disable detailed metrics collection to reduce CPU usage",
		Level:       DegradationMinor,
		Priority:    2,
		Action: func(ctx context.Context) error {
			dm.logger.Info("Disabling detailed metrics collection")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Re-enabling detailed metrics collection")
			return nil
		},
	})

	// Moderate degradation actions
	dm.RegisterAction(DegradationAction{
		Name:        "reduce_queue_size",
		Description: "Reduce event queue size to conserve memory",
		Level:       DegradationModerate,
		Priority:    1,
		Action: func(ctx context.Context) error {
			dm.logger.Info("Reducing event queue size for memory conservation")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Restoring normal event queue size")
			return nil
		},
	})

	dm.RegisterAction(DegradationAction{
		Name:        "disable_non_essential_adapters",
		Description: "Disable non-essential hardware adapters",
		Level:       DegradationModerate,
		Priority:    2,
		Action: func(ctx context.Context) error {
			dm.logger.Info("Disabling non-essential hardware adapters")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Re-enabling hardware adapters")
			return nil
		},
	})

	// Severe degradation actions
	dm.RegisterAction(DegradationAction{
		Name:        "disable_file_logging",
		Description: "Disable file logging to reduce disk I/O",
		Level:       DegradationSevere,
		Priority:    1,
		Action: func(ctx context.Context) error {
			dm.logger.Info("Disabling file logging to reduce disk I/O")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Re-enabling file logging")
			return nil
		},
	})

	dm.RegisterAction(DegradationAction{
		Name:        "emergency_queue_flush",
		Description: "Flush event queue to prevent memory overflow",
		Level:       DegradationSevere,
		Priority:    2,
		Action: func(ctx context.Context) error {
			dm.logger.Warn("Emergency queue flush - some events may be lost")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			// No rollback for emergency flush
			return nil
		},
	})

	// Critical degradation actions
	dm.RegisterAction(DegradationAction{
		Name:        "minimal_operation_mode",
		Description: "Switch to minimal operation mode - essential functions only",
		Level:       DegradationCritical,
		Priority:    1,
		Action: func(ctx context.Context) error {
			dm.logger.Error("Entering minimal operation mode - essential functions only")
			return nil
		},
		Rollback: func(ctx context.Context) error {
			dm.logger.Info("Exiting minimal operation mode")
			return nil
		},
	})
}

// RegisterAction registers a new degradation action
func (dm *DegradationManager) RegisterAction(action DegradationAction) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	
	dm.actions = append(dm.actions, action)
	
	dm.logger.WithFields(logrus.Fields{
		"action": action.Name,
		"level":  action.Level.String(),
		"priority": action.Priority,
	}).Info("Registered degradation action")
}

// DegradeToLevel degrades service to the specified level
func (dm *DegradationManager) DegradeToLevel(ctx context.Context, targetLevel DegradationLevel, reason string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if targetLevel <= dm.currentLevel {
		dm.logger.WithFields(logrus.Fields{
			"current_level": dm.currentLevel.String(),
			"target_level":  targetLevel.String(),
		}).Debug("Already at or below target degradation level")
		return nil
	}

	dm.logger.WithFields(logrus.Fields{
		"from_level": dm.currentLevel.String(),
		"to_level":   targetLevel.String(),
		"reason":     reason,
	}).Warn("Degrading service level")

	// Record when degradation started
	if dm.currentLevel == DegradationNone {
		dm.degradationStart = time.Now()
	}

	// Execute actions for the target level and below
	for _, action := range dm.actions {
		if action.Level <= targetLevel && !dm.activeActions[action.Name] {
			dm.logger.WithFields(logrus.Fields{
				"action": action.Name,
				"level":  action.Level.String(),
			}).Info("Executing degradation action")

			if action.Action != nil {
				if err := action.Action(ctx); err != nil {
					dm.logger.WithError(err).WithField("action", action.Name).Error("Failed to execute degradation action")
					continue
				}
			}

			dm.activeActions[action.Name] = true
		}
	}

	dm.currentLevel = targetLevel
	
	return nil
}

// RestoreToLevel restores service to the specified level
func (dm *DegradationManager) RestoreToLevel(ctx context.Context, targetLevel DegradationLevel, reason string) error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if targetLevel >= dm.currentLevel {
		dm.logger.WithFields(logrus.Fields{
			"current_level": dm.currentLevel.String(),
			"target_level":  targetLevel.String(),
		}).Debug("Already at or above target degradation level")
		return nil
	}

	dm.logger.WithFields(logrus.Fields{
		"from_level": dm.currentLevel.String(),
		"to_level":   targetLevel.String(),
		"reason":     reason,
	}).Info("Restoring service level")

	// Rollback actions that are above the target level
	for _, action := range dm.actions {
		if action.Level > targetLevel && dm.activeActions[action.Name] {
			dm.logger.WithFields(logrus.Fields{
				"action": action.Name,
				"level":  action.Level.String(),
			}).Info("Rolling back degradation action")

			if action.Rollback != nil {
				if err := action.Rollback(ctx); err != nil {
					dm.logger.WithError(err).WithField("action", action.Name).Error("Failed to rollback degradation action")
					continue
				}
			}

			delete(dm.activeActions, action.Name)
		}
	}

	dm.currentLevel = targetLevel

	// If fully restored, clear degradation start time
	if targetLevel == DegradationNone {
		duration := time.Since(dm.degradationStart)
		dm.logger.WithField("duration", duration).Info("Service fully restored from degradation")
		dm.degradationStart = time.Time{}
	}
	
	return nil
}

// GetCurrentLevel returns the current degradation level
func (dm *DegradationManager) GetCurrentLevel() DegradationLevel {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	return dm.currentLevel
}

// GetDegradationDuration returns how long the service has been degraded
func (dm *DegradationManager) GetDegradationDuration() time.Duration {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	if dm.degradationStart.IsZero() {
		return 0
	}
	
	return time.Since(dm.degradationStart)
}

// GetActiveActions returns a list of currently active degradation actions
func (dm *DegradationManager) GetActiveActions() []string {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	actions := make([]string, 0, len(dm.activeActions))
	for action := range dm.activeActions {
		actions = append(actions, action)
	}
	
	return actions
}

// IsActionActive checks if a specific degradation action is currently active
func (dm *DegradationManager) IsActionActive(actionName string) bool {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	return dm.activeActions[actionName]
}

// HandleResourceConstraint handles resource constraint errors by degrading service
func (dm *DegradationManager) HandleResourceConstraint(ctx context.Context, resourceType string, usage, threshold float64) error {
	usagePercent := (usage / threshold) * 100
	
	var targetLevel DegradationLevel
	var reason string

	switch {
	case usagePercent >= 95:
		targetLevel = DegradationCritical
		reason = fmt.Sprintf("%s usage critical: %.1f%% (threshold: %.1f%%)", resourceType, usagePercent, 95.0)
	case usagePercent >= 90:
		targetLevel = DegradationSevere
		reason = fmt.Sprintf("%s usage severe: %.1f%% (threshold: %.1f%%)", resourceType, usagePercent, 90.0)
	case usagePercent >= 80:
		targetLevel = DegradationModerate
		reason = fmt.Sprintf("%s usage moderate: %.1f%% (threshold: %.1f%%)", resourceType, usagePercent, 80.0)
	case usagePercent >= 70:
		targetLevel = DegradationMinor
		reason = fmt.Sprintf("%s usage elevated: %.1f%% (threshold: %.1f%%)", resourceType, usagePercent, 70.0)
	default:
		// Usage is acceptable, try to restore if currently degraded
		if dm.GetCurrentLevel() > DegradationNone {
			return dm.RestoreToLevel(ctx, DegradationNone, fmt.Sprintf("%s usage normalized: %.1f%%", resourceType, usagePercent))
		}
		return nil
	}

	return dm.DegradeToLevel(ctx, targetLevel, reason)
}

// GetDegradationStatus returns the current degradation status
func (dm *DegradationManager) GetDegradationStatus() map[string]interface{} {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	
	status := map[string]interface{}{
		"current_level":    dm.currentLevel.String(),
		"active_actions":   len(dm.activeActions),
		"degraded":         dm.currentLevel > DegradationNone,
	}

	if !dm.degradationStart.IsZero() {
		status["degradation_duration"] = time.Since(dm.degradationStart).String()
		status["degradation_start"] = dm.degradationStart
	}

	if len(dm.activeActions) > 0 {
		actions := make([]string, 0, len(dm.activeActions))
		for action := range dm.activeActions {
			actions = append(actions, action)
		}
		status["active_action_names"] = actions
	}

	return status
}