package processor

import (
	"context"
	"testing"
	"time"

	"gym-door-bridge/internal/types"
	"github.com/sirupsen/logrus"
)

func TestEventProcessorImpl_Initialize(t *testing.T) {
	db := &MockDB{} // Mock DB for testing
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	processor := NewEventProcessor(db, logger)

	tests := []struct {
		name    string
		config  ProcessorConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ProcessorConfig{
				DeviceID:            "test-device-123",
				EnableDeduplication: true,
				DeduplicationWindow: 300,
			},
			wantErr: false,
		},
		{
			name: "missing device ID",
			config: ProcessorConfig{
				EnableDeduplication: true,
				DeduplicationWindow: 300,
			},
			wantErr: true,
		},
		{
			name: "zero deduplication window gets default",
			config: ProcessorConfig{
				DeviceID:            "test-device-123",
				EnableDeduplication: true,
				DeduplicationWindow: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.Initialize(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.config.DeduplicationWindow == 0 {
				if processor.config.DeduplicationWindow != 300 {
					t.Errorf("Expected default deduplication window of 300, got %d", processor.config.DeduplicationWindow)
				}
			}
		})
	}
}

func TestEventProcessorImpl_ValidateEvent(t *testing.T) {
	processor := &EventProcessorImpl{}

	now := time.Now()
	
	tests := []struct {
		name     string
		rawEvent types.RawHardwareEvent
		wantErr  bool
		errField string
	}{
		{
			name: "valid event",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "user123",
				Timestamp:      now,
				EventType:      types.EventTypeEntry,
			},
			wantErr: false,
		},
		{
			name: "empty external user ID",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "",
				Timestamp:      now,
				EventType:      types.EventTypeEntry,
			},
			wantErr:  true,
			errField: "externalUserId",
		},
		{
			name: "whitespace only external user ID",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "   ",
				Timestamp:      now,
				EventType:      types.EventTypeEntry,
			},
			wantErr:  true,
			errField: "externalUserId",
		},
		{
			name: "zero timestamp",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "user123",
				Timestamp:      time.Time{},
				EventType:      types.EventTypeEntry,
			},
			wantErr:  true,
			errField: "timestamp",
		},
		{
			name: "future timestamp",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "user123",
				Timestamp:      now.Add(2 * time.Hour),
				EventType:      types.EventTypeEntry,
			},
			wantErr:  true,
			errField: "timestamp",
		},
		{
			name: "old timestamp",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "user123",
				Timestamp:      now.Add(-25 * time.Hour),
				EventType:      types.EventTypeEntry,
			},
			wantErr:  true,
			errField: "timestamp",
		},
		{
			name: "invalid event type",
			rawEvent: types.RawHardwareEvent{
				ExternalUserID: "user123",
				Timestamp:      now,
				EventType:      "invalid",
			},
			wantErr:  true,
			errField: "eventType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ValidateEvent(tt.rawEvent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if validationErr, ok := err.(ValidationError); ok {
					if validationErr.Field != tt.errField {
						t.Errorf("Expected error field %s, got %s", tt.errField, validationErr.Field)
					}
				} else {
					t.Errorf("Expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestEventProcessorImpl_GenerateEventID(t *testing.T) {
	processor := &EventProcessorImpl{
		config: ProcessorConfig{
			DeviceID: "test-device-12345678",
		},
	}

	now := time.Now()
	rawEvent := types.RawHardwareEvent{
		ExternalUserID: "user123",
		Timestamp:      now,
		EventType:      types.EventTypeEntry,
		RawData: map[string]interface{}{
			"test": "data",
		},
	}

	// Test that the same event generates the same ID
	id1 := processor.GenerateEventID(rawEvent)
	id2 := processor.GenerateEventID(rawEvent)

	if id1 != id2 {
		t.Errorf("Expected same event to generate same ID, got %s and %s", id1, id2)
	}

	// Test that different events generate different IDs
	rawEvent2 := rawEvent
	rawEvent2.ExternalUserID = "user456"
	id3 := processor.GenerateEventID(rawEvent2)

	if id1 == id3 {
		t.Errorf("Expected different events to generate different IDs, both got %s", id1)
	}

	// Test ID format
	expectedPrefix := "evt_test-dev"
	if len(id1) < len(expectedPrefix) || id1[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected ID to start with %s, got %s", expectedPrefix, id1)
	}
}

func TestEventProcessorImpl_isSimulatedEvent(t *testing.T) {
	processor := &EventProcessorImpl{}

	tests := []struct {
		name     string
		rawEvent types.RawHardwareEvent
		want     bool
	}{
		{
			name: "no raw data",
			rawEvent: types.RawHardwareEvent{
				RawData: nil,
			},
			want: false,
		},
		{
			name: "simulated flag true",
			rawEvent: types.RawHardwareEvent{
				RawData: map[string]interface{}{
					"simulated": true,
				},
			},
			want: true,
		},
		{
			name: "simulated flag false",
			rawEvent: types.RawHardwareEvent{
				RawData: map[string]interface{}{
					"simulated": false,
				},
			},
			want: false,
		},
		{
			name: "simulator adapter",
			rawEvent: types.RawHardwareEvent{
				RawData: map[string]interface{}{
					"adapter": "SimulatorAdapter",
				},
			},
			want: true,
		},
		{
			name: "real adapter",
			rawEvent: types.RawHardwareEvent{
				RawData: map[string]interface{}{
					"adapter": "FingerprintAdapter",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processor.isSimulatedEvent(tt.rawEvent)
			if got != tt.want {
				t.Errorf("isSimulatedEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEventProcessorImpl_ProcessEvent(t *testing.T) {
	// Create a mock database that implements the required methods
	mockDB := &MockDB{
		similarEvents: make(map[string]bool),
		userMappings:  make(map[string]string),
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	processor := NewEventProcessor(mockDB, logger)
	
	config := ProcessorConfig{
		DeviceID:            "test-device-123",
		EnableDeduplication: true,
		DeduplicationWindow: 300,
	}
	
	err := processor.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to initialize processor: %v", err)
	}

	now := time.Now()
	validEvent := types.RawHardwareEvent{
		ExternalUserID: "user123",
		Timestamp:      now,
		EventType:      types.EventTypeEntry,
		RawData: map[string]interface{}{
			"adapter": "TestAdapter",
		},
	}

	t.Run("valid event processing", func(t *testing.T) {
		result, err := processor.ProcessEvent(context.Background(), validEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		if !result.Processed {
			t.Errorf("Expected event to be processed, got Processed = false, Reason = %s", result.Reason)
			return
		}

		// Verify standard event fields
		if result.Event.ExternalUserID != validEvent.ExternalUserID {
			t.Errorf("Expected ExternalUserID %s, got %s", validEvent.ExternalUserID, result.Event.ExternalUserID)
		}

		if result.Event.EventType != validEvent.EventType {
			t.Errorf("Expected EventType %s, got %s", validEvent.EventType, result.Event.EventType)
		}

		if result.Event.DeviceID != config.DeviceID {
			t.Errorf("Expected DeviceID %s, got %s", config.DeviceID, result.Event.DeviceID)
		}

		if result.Event.EventID == "" {
			t.Error("Expected EventID to be generated")
		}
	})

	t.Run("invalid event", func(t *testing.T) {
		invalidEvent := validEvent
		invalidEvent.ExternalUserID = ""

		result, err := processor.ProcessEvent(context.Background(), invalidEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		if result.Processed {
			t.Error("Expected invalid event to not be processed")
		}

		if result.Reason == "" {
			t.Error("Expected reason for not processing invalid event")
		}
	})

	t.Run("duplicate event", func(t *testing.T) {
		// Set up mock to return true for duplicate check
		mockDB.similarEvents["user123:entry"] = true

		result, err := processor.ProcessEvent(context.Background(), validEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		if result.Processed {
			t.Error("Expected duplicate event to not be processed")
		}

		if result.Reason != "duplicate event within deduplication window" {
			t.Errorf("Expected duplicate reason, got: %s", result.Reason)
		}
	})

	t.Run("user mapping resolution - mapped user", func(t *testing.T) {
		// Reset mock and add user mapping
		mockDB.similarEvents = make(map[string]bool)
		mockDB.userMappings["user123"] = "internal_user_456"
		
		result, err := processor.ProcessEvent(context.Background(), validEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		if !result.Processed {
			t.Errorf("Expected event to be processed, got Processed = false, Reason = %s", result.Reason)
			return
		}

		if result.Event.InternalUserID != "internal_user_456" {
			t.Errorf("Expected InternalUserID to be resolved to 'internal_user_456', got '%s'", result.Event.InternalUserID)
		}
	})

	t.Run("user mapping resolution - unmapped user", func(t *testing.T) {
		// Reset mock with no user mappings
		mockDB.similarEvents = make(map[string]bool)
		mockDB.userMappings = make(map[string]string)
		
		result, err := processor.ProcessEvent(context.Background(), validEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		if !result.Processed {
			t.Errorf("Expected event to be processed, got Processed = false, Reason = %s", result.Reason)
			return
		}

		if result.Event.InternalUserID != "" {
			t.Errorf("Expected InternalUserID to be empty for unmapped user, got '%s'", result.Event.InternalUserID)
		}
	})

	t.Run("statistics update", func(t *testing.T) {
		// Reset mock
		mockDB.similarEvents = make(map[string]bool)
		mockDB.userMappings = make(map[string]string)
		
		initialStats := processor.GetStats()
		initialTime := time.Now().Unix()
		
		_, err := processor.ProcessEvent(context.Background(), validEvent)
		if err != nil {
			t.Errorf("ProcessEvent() error = %v", err)
			return
		}

		newStats := processor.GetStats()
		if newStats.TotalProcessed != initialStats.TotalProcessed+1 {
			t.Errorf("Expected TotalProcessed to increment by 1, got %d -> %d", 
				initialStats.TotalProcessed, newStats.TotalProcessed)
		}

		if newStats.LastProcessedAt < initialTime {
			t.Errorf("Expected LastProcessedAt to be updated to at least %d, got %d", 
				initialTime, newStats.LastProcessedAt)
		}
	})
}

// MockDB implements the database methods needed for testing
type MockDB struct {
	similarEvents map[string]bool
	userMappings  map[string]string
}

func (m *MockDB) HasSimilarEvent(externalUserID, eventType string, windowStart, windowEnd time.Time) (bool, error) {
	key := externalUserID + ":" + eventType
	return m.similarEvents[key], nil
}

func (m *MockDB) ResolveExternalUserID(externalUserID string) (string, error) {
	internalUserID, exists := m.userMappings[externalUserID]
	if !exists {
		return "", nil // No mapping found
	}
	return internalUserID, nil
}

func TestEventProcessorImpl_resolveUserMapping(t *testing.T) {
	mockDB := &MockDB{
		userMappings: map[string]string{
			"fp_12345": "user_abc123",
			"rfid_678": "user_def456",
		},
	}
	
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	processor := NewEventProcessor(mockDB, logger)
	
	config := ProcessorConfig{
		DeviceID: "test-device-123",
	}
	
	err := processor.Initialize(context.Background(), config)
	if err != nil {
		t.Fatalf("Failed to initialize processor: %v", err)
	}

	tests := []struct {
		name           string
		externalUserID string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "mapped user",
			externalUserID: "fp_12345",
			expectedResult: "user_abc123",
			expectError:    false,
		},
		{
			name:           "another mapped user",
			externalUserID: "rfid_678",
			expectedResult: "user_def456",
			expectError:    false,
		},
		{
			name:           "unmapped user",
			externalUserID: "fp_99999",
			expectedResult: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.resolveUserMapping(context.Background(), tt.externalUserID)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("expected result '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}