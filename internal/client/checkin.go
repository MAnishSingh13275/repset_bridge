package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

// CheckinResponse represents the response from the checkin endpoint
type CheckinResponse struct {
	Success      bool     `json:"success"`
	ProcessedIDs []string `json:"processedIds,omitempty"`
	FailedIDs    []string `json:"failedIds,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
}

// HTTPClientInterface defines the interface for HTTP client operations
type HTTPClientInterface interface {
	Do(ctx context.Context, req *Request) (*Response, error)
}

// CheckinClient handles submission of check-in events to the cloud API
type CheckinClient struct {
	httpClient HTTPClientInterface
	logger     *logrus.Logger
}

// NewCheckinClient creates a new checkin client
func NewCheckinClient(httpClient HTTPClientInterface, logger *logrus.Logger) *CheckinClient {
	return &CheckinClient{
		httpClient: httpClient,
		logger:     logger,
	}
}

// SubmitEvents submits a batch of events to the /api/v1/checkin endpoint
func (c *CheckinClient) SubmitEvents(ctx context.Context, events []types.StandardEvent) (*CheckinResponse, error) {
	if len(events) == 0 {
		return &CheckinResponse{Success: true}, nil
	}

	// Generate idempotency keys for events that don't have them
	eventsWithKeys := make([]types.StandardEvent, len(events))
	for i, event := range events {
		eventsWithKeys[i] = event
		if eventsWithKeys[i].EventID == "" {
			idempotencyKey, err := c.generateIdempotencyKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate idempotency key: %w", err)
			}
			eventsWithKeys[i].EventID = idempotencyKey
		}
	}

	// Convert StandardEvents to CheckinEvents
	checkinEvents := make([]CheckinEvent, len(eventsWithKeys))
	for i, event := range eventsWithKeys {
		checkinEvents[i] = CheckinEvent{
			EventID:        event.EventID,
			ExternalUserID: event.ExternalUserID,
			Timestamp:      event.Timestamp.Format(time.RFC3339),
			EventType:      event.EventType,
			IsSimulated:    event.IsSimulated,
			DeviceID:       event.DeviceID,
		}
	}

	// Create request payload
	request := CheckinRequest{
		Events: checkinEvents,
	}

	c.logger.Info("Submitting events to checkin endpoint", 
		"event_count", len(events),
		"first_event_id", eventsWithKeys[0].EventID)

	// Make HTTP request
	httpReq := &Request{
		Method:      http.MethodPost,
		Path:        "/api/v1/checkin",
		Body:        request,
		RequireAuth: true,
	}

	resp, err := c.httpClient.Do(ctx, httpReq)
	if err != nil {
		c.logger.Error("Failed to submit events", "error", err)
		return nil, fmt.Errorf("failed to submit events: %w", err)
	}

	// Parse response
	var checkinResp CheckinResponse
	if err := parseJSONResponse(resp, &checkinResp); err != nil {
		return nil, fmt.Errorf("failed to parse checkin response: %w", err)
	}

	// Log results
	if checkinResp.Success {
		c.logger.Info("Successfully submitted events", 
			"processed_count", len(checkinResp.ProcessedIDs))
	} else {
		c.logger.Warn("Event submission partially failed", 
			"processed_count", len(checkinResp.ProcessedIDs),
			"failed_count", len(checkinResp.FailedIDs),
			"error", checkinResp.ErrorMessage)
	}

	return &checkinResp, nil
}

// SubmitSingleEvent submits a single event to the checkin endpoint
func (c *CheckinClient) SubmitSingleEvent(ctx context.Context, event types.StandardEvent) (*CheckinResponse, error) {
	return c.SubmitEvents(ctx, []types.StandardEvent{event})
}

// generateIdempotencyKey generates a unique idempotency key for an event
func (c *CheckinClient) generateIdempotencyKey() (string, error) {
	// Generate a random 16-byte key
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create a unique key with timestamp prefix for better debugging
	timestamp := time.Now().Unix()
	key := fmt.Sprintf("evt_%d_%s", timestamp, hex.EncodeToString(bytes))
	
	return key, nil
}

// ValidateEvents validates a batch of events before submission
func (c *CheckinClient) ValidateEvents(events []types.StandardEvent) error {
	if len(events) == 0 {
		return fmt.Errorf("no events to validate")
	}

	for i, event := range events {
		if err := c.validateSingleEvent(event); err != nil {
			return fmt.Errorf("event %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateSingleEvent validates a single event
func (c *CheckinClient) validateSingleEvent(event types.StandardEvent) error {
	if event.ExternalUserID == "" {
		return fmt.Errorf("externalUserId is required")
	}

	if event.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	if !types.IsValidEventType(event.EventType) {
		return fmt.Errorf("invalid event type: %s", event.EventType)
	}

	if event.DeviceID == "" {
		return fmt.Errorf("deviceId is required")
	}

	// Check timestamp is not too far in the future (allow 5 minutes skew)
	maxFutureTime := time.Now().Add(5 * time.Minute)
	if event.Timestamp.After(maxFutureTime) {
		return fmt.Errorf("timestamp is too far in the future: %v", event.Timestamp)
	}

	// Check timestamp is not too old (allow 7 days)
	minPastTime := time.Now().Add(-7 * 24 * time.Hour)
	if event.Timestamp.Before(minPastTime) {
		return fmt.Errorf("timestamp is too old: %v", event.Timestamp)
	}

	return nil
}

// GetMaxBatchSize returns the maximum number of events that can be submitted in a single batch
func (c *CheckinClient) GetMaxBatchSize() int {
	// Based on typical API limits and payload size considerations
	return 100
}

// SplitIntoBatches splits a large slice of events into smaller batches for submission
func (c *CheckinClient) SplitIntoBatches(events []types.StandardEvent, batchSize int) [][]types.StandardEvent {
	if batchSize <= 0 {
		batchSize = c.GetMaxBatchSize()
	}

	var batches [][]types.StandardEvent
	for i := 0; i < len(events); i += batchSize {
		end := i + batchSize
		if end > len(events) {
			end = len(events)
		}
		batches = append(batches, events[i:end])
	}

	return batches
}

// SubmitEventsInBatches submits events in multiple batches if necessary
func (c *CheckinClient) SubmitEventsInBatches(ctx context.Context, events []types.StandardEvent) ([]*CheckinResponse, error) {
	if len(events) == 0 {
		return []*CheckinResponse{{Success: true}}, nil
	}

	// Validate all events first
	if err := c.ValidateEvents(events); err != nil {
		return nil, fmt.Errorf("event validation failed: %w", err)
	}

	// Split into batches
	batches := c.SplitIntoBatches(events, c.GetMaxBatchSize())
	responses := make([]*CheckinResponse, len(batches))

	c.logger.Info("Submitting events in batches", 
		"total_events", len(events),
		"batch_count", len(batches))

	// Submit each batch
	for i, batch := range batches {
		c.logger.Debug("Submitting batch", "batch", i+1, "events", len(batch))
		
		resp, err := c.SubmitEvents(ctx, batch)
		if err != nil {
			return responses[:i], fmt.Errorf("failed to submit batch %d: %w", i+1, err)
		}
		
		responses[i] = resp
		
		// Add small delay between batches to avoid overwhelming the server
		if i < len(batches)-1 {
			select {
			case <-ctx.Done():
				return responses[:i+1], ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}

	return responses, nil
}