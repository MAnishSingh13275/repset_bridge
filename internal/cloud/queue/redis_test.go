package queue

import (
	"testing"
	"time"

	"gym-door-bridge/internal/cloud/config"
)

func TestNewRedisQueue(t *testing.T) {
	cfg := config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
		PoolSize: 10,
	}

	// This test will fail if Redis is not available, which is expected
	queue, err := NewRedisQueue(cfg)
	if err != nil {
		t.Logf("Redis connection failed as expected without Redis server: %v", err)
		return
	}

	defer queue.Close()

	// Test health check
	err = queue.Health()
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestMessage(t *testing.T) {
	message := &Message{
		ID:        "test-123",
		Type:      "test_event",
		Data:      map[string]interface{}{"key": "value"},
		Timestamp: time.Now(),
		Retries:   0,
	}

	if message.ID != "test-123" {
		t.Errorf("Expected ID test-123, got %s", message.ID)
	}

	if message.Type != "test_event" {
		t.Errorf("Expected Type test_event, got %s", message.Type)
	}

	if message.Data["key"] != "value" {
		t.Errorf("Expected Data key=value, got %v", message.Data["key"])
	}

	if message.Retries != 0 {
		t.Errorf("Expected Retries 0, got %d", message.Retries)
	}
}

func TestPublishEvent(t *testing.T) {
	cfg := config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
		PoolSize: 10,
	}

	queue, err := NewRedisQueue(cfg)
	if err != nil {
		t.Skip("Skipping test - Redis not available")
	}
	defer queue.Close()

	// Test publishing an event
	err = queue.PublishEvent("door_unlock", "device-123", nil, map[string]interface{}{
		"duration": 5000,
		"reason":   "test",
	})

	if err != nil {
		t.Errorf("Failed to publish event: %v", err)
	}
}

func TestPublishDeviceHeartbeat(t *testing.T) {
	cfg := config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		Database: 0,
		PoolSize: 10,
	}

	queue, err := NewRedisQueue(cfg)
	if err != nil {
		t.Skip("Skipping test - Redis not available")
	}
	defer queue.Close()

	// Test publishing a device heartbeat
	status := map[string]interface{}{
		"status":     "healthy",
		"uptime":     3600,
		"queue_depth": 0,
	}

	err = queue.PublishDeviceHeartbeat("device-123", status)
	if err != nil {
		t.Errorf("Failed to publish device heartbeat: %v", err)
	}
}

func TestRedisQueueClose(t *testing.T) {
	// Test closing a queue with nil client
	queue := &RedisQueue{client: nil}
	err := queue.Close()
	if err != nil {
		t.Errorf("Expected no error closing nil client, got %v", err)
	}
}