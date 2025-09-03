package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gym-door-bridge/internal/cloud/config"
)

// RedisQueue implements message queue using Redis
type RedisQueue struct {
	client *redis.Client
	ctx    context.Context
}

// Message represents a queue message
type Message struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Retries   int                    `json:"retries"`
}

// NewRedisQueue creates a new Redis-based message queue
func NewRedisQueue(cfg config.RedisConfig) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Password,
		DB:       cfg.Database,
		PoolSize: cfg.PoolSize,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisQueue{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// Publish publishes a message to a queue
func (q *RedisQueue) Publish(queueName string, message *Message) error {
	message.Timestamp = time.Now()
	
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return q.client.LPush(q.ctx, queueName, data).Err()
}

// Subscribe subscribes to a queue and processes messages
func (q *RedisQueue) Subscribe(queueName string, handler func(*Message) error) error {
	for {
		// Block and wait for messages
		result, err := q.client.BRPop(q.ctx, 0, queueName).Result()
		if err != nil {
			return fmt.Errorf("failed to receive message: %w", err)
		}

		if len(result) < 2 {
			continue
		}

		var message Message
		if err := json.Unmarshal([]byte(result[1]), &message); err != nil {
			fmt.Printf("Failed to unmarshal message: %v\n", err)
			continue
		}

		// Process message
		if err := handler(&message); err != nil {
			fmt.Printf("Failed to process message %s: %v\n", message.ID, err)
			
			// Retry logic
			message.Retries++
			if message.Retries < 3 {
				// Re-queue for retry
				if retryErr := q.Publish(queueName+":retry", &message); retryErr != nil {
					fmt.Printf("Failed to re-queue message for retry: %v\n", retryErr)
				}
			} else {
				// Move to dead letter queue
				if dlqErr := q.Publish(queueName+":dlq", &message); dlqErr != nil {
					fmt.Printf("Failed to move message to dead letter queue: %v\n", dlqErr)
				}
			}
		}
	}
}

// GetQueueLength returns the length of a queue
func (q *RedisQueue) GetQueueLength(queueName string) (int64, error) {
	return q.client.LLen(q.ctx, queueName).Result()
}

// Health checks the Redis connection health
func (q *RedisQueue) Health() error {
	return q.client.Ping(q.ctx).Err()
}

// PublishEvent publishes an event to the events queue
func (q *RedisQueue) PublishEvent(eventType string, deviceID string, userID *string, data map[string]interface{}) error {
	message := &Message{
		ID:   fmt.Sprintf("%d", time.Now().UnixNano()),
		Type: eventType,
		Data: map[string]interface{}{
			"device_id": deviceID,
			"user_id":   userID,
			"event_data": data,
		},
	}

	return q.Publish("events", message)
}

// PublishDeviceHeartbeat publishes a device heartbeat event
func (q *RedisQueue) PublishDeviceHeartbeat(deviceID string, status map[string]interface{}) error {
	return q.PublishEvent("device_heartbeat", deviceID, nil, status)
}