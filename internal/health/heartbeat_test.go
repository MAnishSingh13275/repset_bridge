package health

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/tier"
)



func TestHeartbeatManager_Start_Stop(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	config := HeartbeatConfig{
		Interval:         100 * time.Millisecond, // Fast interval for testing
		Timeout:          5 * time.Second,
		MaxRetries:       1,
		RetryBackoff:     1 * time.Second,
		EnableSystemInfo: true,
	}
	
	manager := NewHeartbeatManager(
		config,
		mockClient,
		mockHealthMonitor,
		WithHeartbeatLogger(logrus.New()),
	)
	
	// Mock expectations - expect at least one heartbeat call
	mockClient.On("SendHeartbeat", mock.Anything, mock.AnythingOfType("*client.HeartbeatRequest")).Return(nil).Maybe()
	
	// Test start
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err := manager.Start(ctx)
	require.NoError(t, err)
	
	// Wait a bit to allow heartbeat to be sent
	time.Sleep(200 * time.Millisecond)
	
	// Test stop
	err = manager.Stop(ctx)
	require.NoError(t, err)
	
	// Verify stats
	stats := manager.GetStats()
	assert.False(t, stats.IsRunning)
	assert.Equal(t, config.Interval, stats.Interval)
	
	mockClient.AssertExpectations(t)
}

func TestHeartbeatManager_SendHeartbeat(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	config := HeartbeatConfig{
		Interval:         1 * time.Hour, // Long interval to prevent automatic sending
		Timeout:          5 * time.Second,
		MaxRetries:       1,
		RetryBackoff:     1 * time.Second,
		EnableSystemInfo: true,
	}
	
	manager := NewHeartbeatManager(config, mockClient, mockHealthMonitor)
	
	// Mock expectations
	expectedHeartbeat := &client.HeartbeatRequest{
		Status:     "healthy",
		Tier:       "normal",
		QueueDepth: 5,
		SystemInfo: &client.SystemInfo{
			CPUUsage:    25.0,
			MemoryUsage: 50.0,
			DiskSpace:   30.0,
		},
	}
	
	mockClient.On("SendHeartbeat", mock.Anything, mock.MatchedBy(func(hb *client.HeartbeatRequest) bool {
		return hb.Status == expectedHeartbeat.Status &&
			hb.Tier == expectedHeartbeat.Tier &&
			hb.QueueDepth == expectedHeartbeat.QueueDepth &&
			hb.SystemInfo != nil
	})).Return(nil)
	
	// Test sending heartbeat
	ctx := context.Background()
	err := manager.sendHeartbeat(ctx)
	require.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}

func TestHeartbeatManager_SendHeartbeat_WithRetries(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	config := HeartbeatConfig{
		Interval:         1 * time.Hour,
		Timeout:          1 * time.Second,
		MaxRetries:       2,
		RetryBackoff:     10 * time.Millisecond, // Fast backoff for testing
		EnableSystemInfo: false,
	}
	
	manager := NewHeartbeatManager(config, mockClient, mockHealthMonitor)
	
	// Mock expectations - fail first two attempts, succeed on third
	mockClient.On("SendHeartbeat", mock.Anything, mock.Anything).Return(assert.AnError).Twice()
	mockClient.On("SendHeartbeat", mock.Anything, mock.Anything).Return(nil).Once()
	
	// Test sending heartbeat with retries
	ctx := context.Background()
	err := manager.sendHeartbeat(ctx)
	require.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}

func TestHeartbeatManager_SendHeartbeat_AllRetriesFail(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	config := HeartbeatConfig{
		Interval:         1 * time.Hour,
		Timeout:          1 * time.Second,
		MaxRetries:       2,
		RetryBackoff:     10 * time.Millisecond,
		EnableSystemInfo: false,
	}
	
	manager := NewHeartbeatManager(config, mockClient, mockHealthMonitor)
	
	// Mock expectations - all attempts fail
	mockClient.On("SendHeartbeat", mock.Anything, mock.Anything).Return(assert.AnError).Times(3) // Initial + 2 retries
	
	// Test sending heartbeat with all retries failing
	ctx := context.Background()
	err := manager.sendHeartbeat(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "heartbeat failed after 3 attempts")
	
	mockClient.AssertExpectations(t)
}

func TestHeartbeatManager_UpdateConfig(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	initialConfig := HeartbeatConfig{
		Interval: 1 * time.Minute,
	}
	
	manager := NewHeartbeatManager(initialConfig, mockClient, mockHealthMonitor)
	
	// Test initial config
	stats := manager.GetStats()
	assert.Equal(t, 1*time.Minute, stats.Interval)
	
	// Update config
	newConfig := HeartbeatConfig{
		Interval: 30 * time.Second,
	}
	
	manager.UpdateConfig(newConfig)
	
	// Test updated config
	stats = manager.GetStats()
	assert.Equal(t, 30*time.Second, stats.Interval)
}

func TestGetTierHeartbeatConfig(t *testing.T) {
	tests := []struct {
		tier                 tier.Tier
		expectedInterval     time.Duration
		expectedSystemInfo   bool
	}{
		{tier.TierLite, 5 * time.Minute, false},
		{tier.TierNormal, 1 * time.Minute, false},
		{tier.TierFull, 30 * time.Second, true},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			config := GetTierHeartbeatConfig(tt.tier)
			
			assert.Equal(t, tt.expectedInterval, config.Interval)
			assert.Equal(t, tt.expectedSystemInfo, config.EnableSystemInfo)
			assert.Equal(t, 30*time.Second, config.Timeout)
			assert.Equal(t, 3, config.MaxRetries)
			assert.Equal(t, 10*time.Second, config.RetryBackoff)
		})
	}
}

func TestHeartbeatManager_ContextCancellation(t *testing.T) {
	// Setup
	mockClient := &MockHTTPClient{}
	mockHealthMonitor := createMockHealthMonitor()
	
	config := HeartbeatConfig{
		Interval:     50 * time.Millisecond,
		Timeout:      5 * time.Second,
		MaxRetries:   1,
		RetryBackoff: 1 * time.Second,
	}
	
	manager := NewHeartbeatManager(config, mockClient, mockHealthMonitor)
	
	// Mock expectations - allow heartbeat calls
	mockClient.On("SendHeartbeat", mock.Anything, mock.Anything).Return(nil).Maybe()
	
	// Test with context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err := manager.Start(ctx)
	require.NoError(t, err)
	
	// Wait for context to be cancelled
	<-ctx.Done()
	
	// Stop should complete quickly since context is already cancelled
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()
	
	err = manager.Stop(stopCtx)
	require.NoError(t, err)
	
	mockClient.AssertExpectations(t)
}

