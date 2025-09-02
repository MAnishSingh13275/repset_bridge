package health

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/queue"
	"gym-door-bridge/internal/tier"
	"gym-door-bridge/internal/types"
)

// Mock implementations for testing

type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) Initialize(ctx context.Context, config queue.QueueConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQueueManager) Enqueue(ctx context.Context, event types.StandardEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockQueueManager) GetPendingEvents(ctx context.Context, limit int) ([]queue.QueuedEvent, error) {
	args := m.Called(ctx, limit)
	return args.Get(0).([]queue.QueuedEvent), args.Error(1)
}

func (m *MockQueueManager) MarkEventsSent(ctx context.Context, eventIDs []int64) error {
	args := m.Called(ctx, eventIDs)
	return args.Error(0)
}

func (m *MockQueueManager) MarkEventsFailed(ctx context.Context, eventIDs []int64, errorMessage string) error {
	args := m.Called(ctx, eventIDs, errorMessage)
	return args.Error(0)
}

func (m *MockQueueManager) GetStats(ctx context.Context) (queue.QueueStats, error) {
	args := m.Called(ctx)
	return args.Get(0).(queue.QueueStats), args.Error(1)
}

func (m *MockQueueManager) Cleanup(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockQueueManager) GetQueueDepth(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockQueueManager) IsQueueFull(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockQueueManager) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockTierDetector struct {
	currentTier      tier.Tier
	currentResources tier.SystemResources
}

func (m *MockTierDetector) GetCurrentTier() tier.Tier {
	return m.currentTier
}

func (m *MockTierDetector) GetCurrentResources() tier.SystemResources {
	return m.currentResources
}

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) SendHeartbeat(ctx context.Context, heartbeat *client.HeartbeatRequest) error {
	args := m.Called(ctx, heartbeat)
	return args.Error(0)
}

// Helper function to create a mock health monitor
func createMockHealthMonitor() *HealthMonitor {
	// Create a simple health monitor with mock dependencies
	mockQueue := &MockQueueManager{}
	mockTierDetector := &MockTierDetector{
		currentTier: tier.TierNormal,
		currentResources: tier.SystemResources{
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskUsage:   30.0,
		},
	}
	
	// Set up mock expectations
	mockQueue.On("GetQueueDepth", mock.Anything).Return(5, nil).Maybe()
	mockQueue.On("GetStats", mock.Anything).Return(queue.QueueStats{
		QueueDepth: 5,
		LastSentAt: time.Now().Add(-1 * time.Minute),
	}, nil).Maybe()
	
	monitor := NewHealthMonitor(
		DefaultHealthCheckConfig(),
		mockQueue,
		mockTierDetector,
		NewSimpleAdapterRegistry(),
	)
	
	// Initialize health status
	ctx := context.Background()
	_ = monitor.UpdateHealth(ctx)
	
	return monitor
}