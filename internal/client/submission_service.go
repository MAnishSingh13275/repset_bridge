package client

import (
	"context"
	"fmt"
	"time"

	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

// CheckinClientInterface defines the interface for checkin client operations
type CheckinClientInterface interface {
	SubmitEvents(ctx context.Context, events []types.StandardEvent) (*CheckinResponse, error)
}

// SubmissionService handles the submission of queued events to the cloud
type SubmissionService struct {
	queueManager  queue.QueueManager
	checkinClient CheckinClientInterface
	logger        *logrus.Logger
	config        SubmissionConfig
}

// SubmissionConfig holds configuration for the submission service
type SubmissionConfig struct {
	BatchSize       int           `json:"batchSize"`       // Number of events to submit per batch
	RetryInterval   time.Duration `json:"retryInterval"`   // Time between retry attempts
	MaxRetries      int           `json:"maxRetries"`      // Maximum number of retry attempts per event
	SubmitInterval  time.Duration `json:"submitInterval"`  // Time between submission attempts
	MaxConcurrency  int           `json:"maxConcurrency"`  // Maximum number of concurrent submissions
}

// DefaultSubmissionConfig returns a submission configuration with sensible defaults
func DefaultSubmissionConfig() SubmissionConfig {
	return SubmissionConfig{
		BatchSize:       50,
		RetryInterval:   30 * time.Second,
		MaxRetries:      5,
		SubmitInterval:  10 * time.Second,
		MaxConcurrency:  3,
	}
}

// SubmissionResult represents the result of a submission attempt
type SubmissionResult struct {
	TotalEvents     int           `json:"totalEvents"`
	SuccessfulEvents int          `json:"successfulEvents"`
	FailedEvents    int           `json:"failedEvents"`
	Duration        time.Duration `json:"duration"`
	Errors          []string      `json:"errors,omitempty"`
}

// NewSubmissionService creates a new submission service
func NewSubmissionService(queueManager queue.QueueManager, checkinClient CheckinClientInterface, logger *logrus.Logger) *SubmissionService {
	return &SubmissionService{
		queueManager:  queueManager,
		checkinClient: checkinClient,
		logger:        logger,
		config:        DefaultSubmissionConfig(),
	}
}

// SetConfig updates the submission service configuration
func (s *SubmissionService) SetConfig(config SubmissionConfig) {
	s.config = config
}

// SubmitPendingEvents submits all pending events from the queue to the cloud
func (s *SubmissionService) SubmitPendingEvents(ctx context.Context) (*SubmissionResult, error) {
	startTime := time.Now()
	result := &SubmissionResult{}

	s.logger.Info("Starting submission of pending events")

	// Get pending events from queue
	pendingEvents, err := s.queueManager.GetPendingEvents(ctx, s.config.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending events: %w", err)
	}

	if len(pendingEvents) == 0 {
		s.logger.Debug("No pending events to submit")
		result.Duration = time.Since(startTime)
		return result, nil
	}

	result.TotalEvents = len(pendingEvents)
	s.logger.Info("Found pending events to submit", "count", len(pendingEvents))

	// Convert queued events to standard events
	standardEvents := make([]queue.QueuedEvent, 0, len(pendingEvents))
	for _, queuedEvent := range pendingEvents {
		// Skip events that have exceeded max retries
		if queuedEvent.RetryCount >= s.config.MaxRetries {
			s.logger.Warn("Skipping event that exceeded max retries", 
				"event_id", queuedEvent.Event.EventID,
				"retry_count", queuedEvent.RetryCount,
				"max_retries", s.config.MaxRetries)
			continue
		}
		standardEvents = append(standardEvents, queuedEvent)
	}

	if len(standardEvents) == 0 {
		s.logger.Info("No events eligible for submission (all exceeded max retries)")
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Extract standard events for submission
	eventsToSubmit := make([]queue.QueuedEvent, len(standardEvents))
	copy(eventsToSubmit, standardEvents)

	// Submit events in batches
	submissionResult, err := s.submitEventBatches(ctx, eventsToSubmit)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	// Update result
	result.SuccessfulEvents = submissionResult.SuccessfulEvents
	result.FailedEvents = submissionResult.FailedEvents
	result.Duration = time.Since(startTime)
	result.Errors = append(result.Errors, submissionResult.Errors...)

	s.logger.Info("Completed submission of pending events",
		"total", result.TotalEvents,
		"successful", result.SuccessfulEvents,
		"failed", result.FailedEvents,
		"duration", result.Duration)

	return result, nil
}

// submitEventBatches submits events in batches and handles the results
func (s *SubmissionService) submitEventBatches(ctx context.Context, queuedEvents []queue.QueuedEvent) (*SubmissionResult, error) {
	result := &SubmissionResult{}

	// Convert to standard events for submission
	standardEvents := make([]queue.QueuedEvent, len(queuedEvents))
	copy(standardEvents, queuedEvents)

	// Split into batches
	batches := s.splitQueuedEventsIntoBatches(standardEvents, s.config.BatchSize)
	
	s.logger.Debug("Submitting events in batches", 
		"total_events", len(queuedEvents),
		"batch_count", len(batches),
		"batch_size", s.config.BatchSize)

	// Submit each batch
	for i, batch := range batches {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		s.logger.Debug("Submitting batch", "batch", i+1, "events", len(batch))

		batchResult, err := s.submitSingleBatch(ctx, batch)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("batch %d failed: %v", i+1, err))
			// Mark all events in this batch as failed
			eventIDs := make([]int64, len(batch))
			for j, event := range batch {
				eventIDs[j] = event.ID
			}
			if markErr := s.queueManager.MarkEventsFailed(ctx, eventIDs, err.Error()); markErr != nil {
				s.logger.Error("Failed to mark events as failed", "error", markErr)
			}
			result.FailedEvents += len(batch)
			continue
		}

		result.SuccessfulEvents += batchResult.SuccessfulEvents
		result.FailedEvents += batchResult.FailedEvents
		result.Errors = append(result.Errors, batchResult.Errors...)

		// Add small delay between batches
		if i < len(batches)-1 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	return result, nil
}

// submitSingleBatch submits a single batch of events and handles the response
func (s *SubmissionService) submitSingleBatch(ctx context.Context, batch []queue.QueuedEvent) (*SubmissionResult, error) {
	result := &SubmissionResult{}

	// Extract standard events from queued events
	standardEvents := make([]types.StandardEvent, len(batch))
	eventIDMap := make(map[string]int64) // Map event ID to database row ID
	
	for i, queuedEvent := range batch {
		standardEvents[i] = queuedEvent.Event
		eventIDMap[queuedEvent.Event.EventID] = queuedEvent.ID
	}

	s.logger.Debug("Submitting batch to checkin endpoint", "event_count", len(standardEvents))

	// Submit events using checkin client
	checkinResp, err := s.checkinClient.SubmitEvents(ctx, standardEvents)
	if err != nil {
		// All events failed
		eventIDs := make([]int64, len(batch))
		for i, event := range batch {
			eventIDs[i] = event.ID
		}
		
		if markErr := s.queueManager.MarkEventsFailed(ctx, eventIDs, err.Error()); markErr != nil {
			s.logger.Error("Failed to mark events as failed", "error", markErr)
		}
		
		result.FailedEvents = len(batch)
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("checkin submission failed: %w", err)
	}

	// Handle successful events
	if len(checkinResp.ProcessedIDs) > 0 {
		successfulEventIDs := make([]int64, 0, len(checkinResp.ProcessedIDs))
		for _, eventID := range checkinResp.ProcessedIDs {
			if dbID, exists := eventIDMap[eventID]; exists {
				successfulEventIDs = append(successfulEventIDs, dbID)
			}
		}
		
		if len(successfulEventIDs) > 0 {
			if err := s.queueManager.MarkEventsSent(ctx, successfulEventIDs); err != nil {
				s.logger.Error("Failed to mark successful events as sent", "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("failed to mark events as sent: %v", err))
			} else {
				result.SuccessfulEvents = len(successfulEventIDs)
				s.logger.Debug("Marked events as successfully sent", "count", len(successfulEventIDs))
			}
		}
	}

	// Handle failed events
	if len(checkinResp.FailedIDs) > 0 {
		failedEventIDs := make([]int64, 0, len(checkinResp.FailedIDs))
		for _, eventID := range checkinResp.FailedIDs {
			if dbID, exists := eventIDMap[eventID]; exists {
				failedEventIDs = append(failedEventIDs, dbID)
			}
		}
		
		if len(failedEventIDs) > 0 {
			errorMsg := checkinResp.ErrorMessage
			if errorMsg == "" {
				errorMsg = "submission failed without specific error"
			}
			
			if err := s.queueManager.MarkEventsFailed(ctx, failedEventIDs, errorMsg); err != nil {
				s.logger.Error("Failed to mark failed events", "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("failed to mark events as failed: %v", err))
			} else {
				result.FailedEvents = len(failedEventIDs)
				s.logger.Debug("Marked events as failed", "count", len(failedEventIDs))
			}
		}
	}

	// Log summary
	s.logger.Info("Batch submission completed",
		"total", len(batch),
		"successful", result.SuccessfulEvents,
		"failed", result.FailedEvents)

	return result, nil
}

// splitQueuedEventsIntoBatches splits queued events into batches
func (s *SubmissionService) splitQueuedEventsIntoBatches(events []queue.QueuedEvent, batchSize int) [][]queue.QueuedEvent {
	if batchSize <= 0 {
		batchSize = s.config.BatchSize
	}

	var batches [][]queue.QueuedEvent
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		batches = append(batches, events[i:end])
	}

	return batches
}

// StartPeriodicSubmission starts a goroutine that periodically submits pending events
func (s *SubmissionService) StartPeriodicSubmission(ctx context.Context) {
	ticker := time.NewTicker(s.config.SubmitInterval)
	defer ticker.Stop()

	s.logger.Info("Starting periodic event submission", "interval", s.config.SubmitInterval)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Stopping periodic event submission")
			return
		case <-ticker.C:
			if _, err := s.SubmitPendingEvents(ctx); err != nil {
				s.logger.Error("Periodic event submission failed", "error", err)
			}
		}
	}
}

// GetQueueStats returns current queue statistics
func (s *SubmissionService) GetQueueStats(ctx context.Context) (queue.QueueStats, error) {
	return s.queueManager.GetStats(ctx)
}