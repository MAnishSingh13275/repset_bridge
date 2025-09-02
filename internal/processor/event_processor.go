package processor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gym-door-bridge/internal/logging"
	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

// DatabaseInterface defines the database methods needed by the event processor
type DatabaseInterface interface {
	HasSimilarEvent(externalUserID, eventType string, windowStart, windowEnd time.Time) (bool, error)
	ResolveExternalUserID(externalUserID string) (string, error)
}

// EventProcessorImpl implements the EventProcessor interface
type EventProcessorImpl struct {
	config ProcessorConfig
	db     DatabaseInterface
	logger *logrus.Entry
	stats  ProcessorStats
	mutex  sync.RWMutex
}

// NewEventProcessor creates a new event processor instance
func NewEventProcessor(db DatabaseInterface, logger *logrus.Logger) *EventProcessorImpl {
	return &EventProcessorImpl{
		db:     db,
		logger: logging.NewServiceLogger(logger, "event-processor"),
		stats:  ProcessorStats{},
	}
}

// Initialize sets up the processor with the provided configuration
func (p *EventProcessorImpl) Initialize(ctx context.Context, config ProcessorConfig) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.config = config
	
	// Validate configuration
	if p.config.DeviceID == "" {
		return fmt.Errorf("deviceId is required in processor configuration")
	}

	if p.config.DeduplicationWindow <= 0 {
		p.config.DeduplicationWindow = 300 // Default 5 minutes
	}

	return nil
}

// ProcessEvent converts a raw hardware event to a standard event
func (p *EventProcessorImpl) ProcessEvent(ctx context.Context, rawEvent types.RawHardwareEvent) (ProcessingResult, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Validate the raw event
	if err := p.ValidateEvent(rawEvent); err != nil {
		p.stats.TotalInvalid++
		return ProcessingResult{
			Processed: false,
			Reason:    fmt.Sprintf("validation failed: %s", err.Error()),
		}, nil
	}

	// Check for duplicates if deduplication is enabled
	if p.config.EnableDeduplication {
		isDuplicate, err := p.IsEventDuplicate(ctx, rawEvent)
		if err != nil {
			return ProcessingResult{}, fmt.Errorf("failed to check for duplicates: %w", err)
		}
		
		if isDuplicate {
			p.stats.TotalDuplicates++
			return ProcessingResult{
				Processed: false,
				Reason:    "duplicate event within deduplication window",
			}, nil
		}
	}

	// Generate event ID
	eventID := p.GenerateEventID(rawEvent)

	// Resolve external user ID to internal user ID
	internalUserID, err := p.resolveUserMapping(ctx, rawEvent.ExternalUserID)
	if err != nil {
		p.logger.WithFields(logrus.Fields{
			"external_user_id": rawEvent.ExternalUserID,
			"event_id":         eventID,
			"error":            err.Error(),
		}).Error("Failed to resolve external user ID mapping")
		// Continue processing even if mapping resolution fails
	}

	// Create standard event with metadata enrichment
	standardEvent := types.StandardEvent{
		EventID:        eventID,
		ExternalUserID: rawEvent.ExternalUserID,
		InternalUserID: internalUserID,
		Timestamp:      rawEvent.Timestamp,
		EventType:      rawEvent.EventType,
		IsSimulated:    p.isSimulatedEvent(rawEvent),
		DeviceID:       p.config.DeviceID,
		RawData:        rawEvent.RawData,
	}

	// Update statistics
	p.stats.TotalProcessed++
	p.stats.LastProcessedAt = time.Now().Unix()

	return ProcessingResult{
		Event:     standardEvent,
		Processed: true,
	}, nil
}

// ValidateEvent checks if a raw event is valid for processing
func (p *EventProcessorImpl) ValidateEvent(rawEvent types.RawHardwareEvent) error {
	// Check external user ID
	if strings.TrimSpace(rawEvent.ExternalUserID) == "" {
		return ValidationError{
			Field:   "externalUserId",
			Message: "external user ID cannot be empty",
		}
	}

	// Check timestamp
	if rawEvent.Timestamp.IsZero() {
		return ValidationError{
			Field:   "timestamp",
			Message: "timestamp cannot be zero",
		}
	}

	// Check if timestamp is too far in the future (more than 1 hour)
	if rawEvent.Timestamp.After(time.Now().Add(time.Hour)) {
		return ValidationError{
			Field:   "timestamp",
			Message: "timestamp cannot be more than 1 hour in the future",
		}
	}

	// Check if timestamp is too far in the past (more than 24 hours)
	if rawEvent.Timestamp.Before(time.Now().Add(-24 * time.Hour)) {
		return ValidationError{
			Field:   "timestamp",
			Message: "timestamp cannot be more than 24 hours in the past",
		}
	}

	// Check event type
	if !types.IsValidEventType(rawEvent.EventType) {
		return ValidationError{
			Field:   "eventType",
			Message: fmt.Sprintf("invalid event type: %s", rawEvent.EventType),
		}
	}

	return nil
}

// GenerateEventID creates a unique event ID for the processed event
func (p *EventProcessorImpl) GenerateEventID(rawEvent types.RawHardwareEvent) string {
	// Create a deterministic hash based on event content and device
	hasher := sha256.New()
	
	// Include device ID to ensure uniqueness across devices
	hasher.Write([]byte(p.config.DeviceID))
	hasher.Write([]byte(rawEvent.ExternalUserID))
	hasher.Write([]byte(rawEvent.EventType))
	hasher.Write([]byte(rawEvent.Timestamp.Format(time.RFC3339Nano)))
	
	// Include raw data if present
	if rawEvent.RawData != nil {
		rawDataBytes, _ := json.Marshal(rawEvent.RawData)
		hasher.Write(rawDataBytes)
	}
	
	hash := hex.EncodeToString(hasher.Sum(nil))
	
	// Return a shorter, more readable event ID
	return fmt.Sprintf("evt_%s_%s", p.config.DeviceID[:8], hash[:16])
}

// IsEventDuplicate checks if an event is a duplicate within the deduplication window
func (p *EventProcessorImpl) IsEventDuplicate(ctx context.Context, rawEvent types.RawHardwareEvent) (bool, error) {
	if !p.config.EnableDeduplication {
		return false, nil
	}

	// Calculate the time window for deduplication
	windowStart := rawEvent.Timestamp.Add(-time.Duration(p.config.DeduplicationWindow) * time.Second)
	windowEnd := rawEvent.Timestamp.Add(time.Duration(p.config.DeduplicationWindow) * time.Second)

	// Check if a similar event exists within the window
	return p.db.HasSimilarEvent(
		rawEvent.ExternalUserID,
		rawEvent.EventType,
		windowStart,
		windowEnd,
	)
}

// GetStats returns processing statistics
func (p *EventProcessorImpl) GetStats() ProcessorStats {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	
	return p.stats
}

// resolveUserMapping resolves an external user ID to an internal user ID
func (p *EventProcessorImpl) resolveUserMapping(ctx context.Context, externalUserID string) (string, error) {
	internalUserID, err := p.db.ResolveExternalUserID(externalUserID)
	if err != nil {
		return "", fmt.Errorf("database error resolving external user ID: %w", err)
	}

	if internalUserID == "" {
		// Log unmapped external user ID for manual review
		p.logger.WithFields(logrus.Fields{
			"external_user_id": externalUserID,
			"device_id":        p.config.DeviceID,
		}).Warn("Unmapped external user ID encountered - manual mapping required")
		
		return "", nil // No mapping found, but not an error
	}

	p.logger.WithFields(logrus.Fields{
		"external_user_id": externalUserID,
		"internal_user_id": internalUserID,
	}).Debug("Successfully resolved external user ID to internal user ID")

	return internalUserID, nil
}

// isSimulatedEvent determines if an event is from a simulator
func (p *EventProcessorImpl) isSimulatedEvent(rawEvent types.RawHardwareEvent) bool {
	// Check if raw data indicates simulation
	if rawEvent.RawData != nil {
		if simulated, exists := rawEvent.RawData["simulated"]; exists {
			if sim, ok := simulated.(bool); ok {
				return sim
			}
		}
		
		// Check for simulator adapter indication
		if adapter, exists := rawEvent.RawData["adapter"]; exists {
			if adapterName, ok := adapter.(string); ok {
				return strings.Contains(strings.ToLower(adapterName), "simulator")
			}
		}
	}
	
	return false
}