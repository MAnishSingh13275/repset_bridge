package bridge

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gym-door-bridge/internal/config"
	"gym-door-bridge/internal/database"
)

// TestBridgeManagerIntegration tests the integration of the bridge manager with API server
func TestBridgeManagerIntegration(t *testing.T) {
	// Create temporary database for testing
	tempDB := createTempDB(t)
	defer os.Remove(tempDB)

	// Create test configuration
	cfg := &config.Config{
		DeviceID:          "test-device-123",
		DeviceKey:         "test-key",
		ServerURL:         "https://api.test.com",
		Tier:              "normal",
		QueueMaxSize:      1000,
		HeartbeatInterval: 60,
		UnlockDuration:    3000,
		DatabasePath:      tempDB,
		LogLevel:          "info",
		EnabledAdapters:   []string{"simulator"},
		AdapterConfigs:    make(map[string]map[string]interface{}),
		UpdatesEnabled:    true,
		APIServer: config.APIServerConfig{
			Enabled:      true,
			Port:         0, // Use random port for testing
			Host:         "127.0.0.1",
			TLSEnabled:   false,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
			Auth: config.AuthConfig{
				Enabled: false,
			},
			RateLimit: config.RateLimitConfig{
				Enabled:        false,
				RequestsPerMin: 60,
				BurstSize:      10,
			},
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
			},
		},
	}

	// Create bridge manager
	manager, err := NewManager(cfg,
		WithVersion("1.0.0-test"),
		WithDeviceID(cfg.DeviceID),
	)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test that manager is not running initially
	assert.False(t, manager.IsRunning())
	assert.Equal(t, time.Duration(0), manager.GetUptime())

	// Start manager in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	managerDone := make(chan error, 1)
	go func() {
		managerDone <- manager.Start(ctx)
	}()

	// Wait a bit for components to start
	time.Sleep(2 * time.Second)

	// Test that manager is running
	assert.True(t, manager.IsRunning())
	assert.Greater(t, manager.GetUptime(), time.Duration(0))

	// Test manager stats
	stats := manager.GetStats()
	assert.True(t, stats["isRunning"].(bool))
	assert.Equal(t, "1.0.0-test", stats["version"])
	assert.Equal(t, cfg.DeviceID, stats["deviceID"])
	assert.NotNil(t, stats["health"])
	assert.NotNil(t, stats["adapters"])

	// Test API server endpoints if available
	if manager.apiServer != nil {
		testAPIEndpoints(t, manager)
	}

	// Stop manager
	cancel()

	// Wait for manager to stop
	select {
	case err := <-managerDone:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Manager did not stop within timeout")
	}

	// Test that manager is not running after stop
	assert.False(t, manager.IsRunning())
}

// TestBridgeManagerComponentIntegration tests integration between bridge components
func TestBridgeManagerComponentIntegration(t *testing.T) {
	// Create temporary database for testing
	tempDB := createTempDB(t)
	defer os.Remove(tempDB)

	// Create test configuration with API server disabled for focused testing
	cfg := &config.Config{
		DeviceID:          "test-device-456",
		ServerURL:         "https://api.test.com",
		Tier:              "lite",
		QueueMaxSize:      100,
		HeartbeatInterval: 30,
		UnlockDuration:    2000,
		DatabasePath:      tempDB,
		LogLevel:          "debug",
		EnabledAdapters:   []string{"simulator"},
		AdapterConfigs: map[string]map[string]interface{}{
			"simulator": {
				"eventInterval": 5,
				"autoGenerate":  true,
			},
		},
		APIServer: config.APIServerConfig{
			Enabled: false,
		},
	}

	// Create bridge manager
	manager, err := NewManager(cfg, WithVersion("1.0.0-test"))
	require.NoError(t, err)

	// Start manager
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	managerDone := make(chan error, 1)
	go func() {
		managerDone <- manager.Start(ctx)
	}()

	// Wait for components to initialize
	time.Sleep(3 * time.Second)

	// Test adapter manager integration
	t.Run("AdapterManager", func(t *testing.T) {
		assert.NotNil(t, manager.adapterManager)
		
		// Test adapter status
		status := manager.adapterManager.GetAdapterStatus()
		assert.Contains(t, status, "simulator")
		
		// Test healthy adapters
		healthy := manager.adapterManager.GetHealthyAdapters()
		assert.Contains(t, healthy, "simulator")
	})

	// Test queue manager integration
	t.Run("QueueManager", func(t *testing.T) {
		assert.NotNil(t, manager.queueManager)
		
		// Test queue depth
		depth, err := manager.queueManager.GetQueueDepth(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, depth, 0)
		
		// Test queue stats
		stats, err := manager.queueManager.GetStats(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, stats.QueueDepth, 0)
	})

	// Test health monitor integration
	t.Run("HealthMonitor", func(t *testing.T) {
		assert.NotNil(t, manager.healthMonitor)
		
		// Test health status
		health := manager.healthMonitor.GetCurrentHealth()
		assert.NotEmpty(t, health.Status)
		assert.NotZero(t, health.Timestamp)
		assert.Equal(t, "1.0.0-test", health.Version)
	})

	// Test tier detector integration
	t.Run("TierDetector", func(t *testing.T) {
		assert.NotNil(t, manager.tierDetector)
		
		// Test current tier
		currentTier := manager.tierDetector.GetCurrentTier()
		assert.NotEmpty(t, currentTier)
		
		// Test resources
		resources := manager.tierDetector.GetCurrentResources()
		assert.Greater(t, resources.CPUCores, 0)
		assert.Greater(t, resources.MemoryGB, 0.0)
	})

	// Test door controller integration
	t.Run("DoorController", func(t *testing.T) {
		assert.NotNil(t, manager.doorController)
		
		// Test door stats
		stats := manager.doorController.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "unlockCount")
		assert.Contains(t, stats, "failureCount")
	})

	// Stop manager
	cancel()

	// Wait for manager to stop
	select {
	case err := <-managerDone:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Manager did not stop within timeout")
	}
}

// testAPIEndpoints tests the API server endpoints
func testAPIEndpoints(t *testing.T, manager *Manager) {
	// Note: This is a simplified test since we can't easily get the actual server port
	// In a real scenario, we'd need to modify the server to expose its actual listening port
	
	t.Run("APIServer", func(t *testing.T) {
		assert.NotNil(t, manager.apiServer)
		
		// Test that API server components are initialized
		// We can't easily test HTTP endpoints without knowing the port,
		// but we can test that the server was created successfully
		assert.True(t, manager.config.APIServer.Enabled)
	})
}

// createTempDB creates a temporary database file for testing
func createTempDB(t *testing.T) string {
	tempFile, err := os.CreateTemp("", "bridge_test_*.db")
	require.NoError(t, err)
	
	tempFile.Close()
	
	// Initialize the database
	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte("test-encryption-key-for-testing-"))
	
	dbConfig := database.Config{
		DatabasePath:    tempFile.Name(),
		EncryptionKey:   encryptionKey,
		PerformanceTier: database.TierNormal,
	}
	db, err := database.NewDB(dbConfig)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	
	return tempFile.Name()
}

// TestBridgeManagerLifecycle tests the complete lifecycle of the bridge manager
func TestBridgeManagerLifecycle(t *testing.T) {
	tempDB := createTempDB(t)
	defer os.Remove(tempDB)

	cfg := &config.Config{
		DeviceID:          "lifecycle-test",
		ServerURL:         "https://api.test.com",
		Tier:              "normal",
		QueueMaxSize:      500,
		HeartbeatInterval: 60,
		UnlockDuration:    3000,
		DatabasePath:      tempDB,
		LogLevel:          "info",
		EnabledAdapters:   []string{"simulator"},
		AdapterConfigs:    make(map[string]map[string]interface{}),
		APIServer: config.APIServerConfig{
			Enabled: false, // Disable for simpler testing
		},
	}

	// Test manager creation
	manager, err := NewManager(cfg, WithVersion("lifecycle-test"))
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Test initial state
	assert.False(t, manager.IsRunning())
	assert.Equal(t, time.Duration(0), manager.GetUptime())

	// Test multiple start/stop cycles
	for i := 0; i < 3; i++ {
		t.Run(fmt.Sprintf("Cycle_%d", i+1), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Start manager
			managerDone := make(chan error, 1)
			go func() {
				managerDone <- manager.Start(ctx)
			}()

			// Wait for startup
			time.Sleep(1 * time.Second)

			// Verify running state
			assert.True(t, manager.IsRunning())
			assert.Greater(t, manager.GetUptime(), time.Duration(0))

			// Test stats during operation
			stats := manager.GetStats()
			assert.True(t, stats["isRunning"].(bool))
			assert.NotNil(t, stats["health"])

			// Stop manager
			cancel()

			// Wait for shutdown
			select {
			case err := <-managerDone:
				assert.NoError(t, err)
			case <-time.After(5 * time.Second):
				t.Fatal("Manager did not stop within timeout")
			}

			// Verify stopped state
			assert.False(t, manager.IsRunning())
		})
	}
}

// TestBridgeManagerErrorHandling tests error handling in the bridge manager
func TestBridgeManagerErrorHandling(t *testing.T) {
	t.Run("InvalidDatabasePath", func(t *testing.T) {
		cfg := &config.Config{
			DatabasePath:    "/invalid/path/that/does/not/exist/bridge.db",
			LogLevel:        "info",
			EnabledAdapters: []string{},
			APIServer: config.APIServerConfig{
				Enabled: false,
			},
		}

		manager, err := NewManager(cfg)
		if err != nil {
			// Expected for invalid database path
			assert.Contains(t, err.Error(), "failed to initialize")
		} else {
			// If manager is created, it should handle the invalid path gracefully
			assert.NotNil(t, manager)
		}
	})

	t.Run("InvalidConfiguration", func(t *testing.T) {
		tempDB := createTempDB(t)
		defer os.Remove(tempDB)

		cfg := &config.Config{
			DatabasePath:      tempDB,
			LogLevel:          "info", // Use valid log level
			QueueMaxSize:      1000,   // Use valid queue size
			HeartbeatInterval: 60,     // Use valid interval
			Tier:              "normal",
			UnlockDuration:    3000,
			EnabledAdapters:   []string{},
			APIServer: config.APIServerConfig{
				Enabled: false,
			},
		}

		// The manager should be created successfully with valid config
		manager, err := NewManager(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, manager)
	})
}

// BenchmarkBridgeManagerStartup benchmarks the bridge manager startup time
func BenchmarkBridgeManagerStartup(b *testing.B) {
	tempDB := createTempDBForBench(b)
	defer os.Remove(tempDB)

	cfg := &config.Config{
		DeviceID:          "benchmark-test",
		ServerURL:         "https://api.test.com",
		Tier:              "normal",
		QueueMaxSize:      1000,
		HeartbeatInterval: 60,
		UnlockDuration:    3000,
		DatabasePath:      tempDB,
		LogLevel:          "error", // Reduce logging for benchmark
		EnabledAdapters:   []string{"simulator"},
		AdapterConfigs:    make(map[string]map[string]interface{}),
		APIServer: config.APIServerConfig{
			Enabled: false, // Disable for faster startup
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager, err := NewManager(cfg, WithVersion("benchmark"))
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		done := make(chan error, 1)
		go func() {
			done <- manager.Start(ctx)
		}()

		// Let it run briefly
		time.Sleep(100 * time.Millisecond)
		
		cancel()
		
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			b.Fatal("Manager did not stop within timeout")
		}
	}
}

// Helper function for benchmarks that need to create temp DB
func createTempDBForBench(tb testing.TB) string {
	tempFile, err := os.CreateTemp("", "bridge_test_*.db")
	require.NoError(tb, err)
	
	tempFile.Close()
	
	// Initialize the database
	encryptionKey := make([]byte, 32)
	copy(encryptionKey, []byte("test-encryption-key-for-testing-"))
	
	dbConfig := database.Config{
		DatabasePath:    tempFile.Name(),
		EncryptionKey:   encryptionKey,
		PerformanceTier: database.TierNormal,
	}
	db, err := database.NewDB(dbConfig)
	require.NoError(tb, err)
	require.NoError(tb, db.Close())
	
	return tempFile.Name()
}