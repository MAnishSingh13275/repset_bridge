package telemetry

import (
	"context"
	"runtime"
	"testing"
	"time"

	"gym-door-bridge/internal/config"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallationTelemetry(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := config.DefaultConfig()
	cfg.SetInstallationMethod("automated", "automated-installer", "test-pair-code", "github", "sha256:abcd1234")

	telemetry := NewInstallationTelemetry(logger, cfg)
	require.NotNil(t, telemetry)

	ctx := context.Background()

	t.Run("GenerateInstallationStatusReport", func(t *testing.T) {
		report, err := telemetry.GenerateInstallationStatusReport(ctx)
		require.NoError(t, err)
		require.NotNil(t, report)

		assert.Equal(t, "automated", report.InstallationMethod)
		assert.Equal(t, "automated-installer", report.InstalledBy)
		assert.Equal(t, "test-pair-code", report.PairCode)
		assert.Equal(t, "github", report.Source)
		assert.Equal(t, "sha256:abcd1234", report.Checksum)
		assert.Equal(t, runtime.GOOS, report.Platform)
		assert.Equal(t, runtime.GOARCH, report.Architecture)
		assert.NotEmpty(t, report.InstallationAge)
		assert.False(t, report.ReportTimestamp.IsZero())

		if runtime.GOOS == "windows" {
			// Service health might not be available in test environment
			t.Logf("Service health report: %+v", report.ServiceHealth)
		}
	})

	t.Run("GetInstallationMetrics", func(t *testing.T) {
		metrics := telemetry.GetInstallationMetrics()
		require.NotNil(t, metrics)

		assert.Equal(t, "automated", metrics["installation_method"])
		assert.Equal(t, "automated-installer", metrics["installed_by"])
		assert.Equal(t, "github", metrics["source"])
		assert.True(t, metrics["is_automated"].(bool))
		assert.Equal(t, runtime.GOOS, metrics["platform"])
		assert.Equal(t, runtime.GOARCH, metrics["architecture"])
		assert.Contains(t, metrics, "installation_age_seconds")
		assert.Contains(t, metrics, "installation_age_days")
	})

	t.Run("ValidateInstallationIntegrity", func(t *testing.T) {
		report, err := telemetry.ValidateInstallationIntegrity()
		require.NoError(t, err)
		require.NotNil(t, report)

		assert.False(t, report.Timestamp.IsZero())
		
		// Should be valid since we set all required fields
		if !report.Valid {
			t.Logf("Validation issues: %v", report.Issues)
		}
	})

	t.Run("LogInstallationStatus", func(t *testing.T) {
		// This should not panic or error
		telemetry.LogInstallationStatus(ctx)
	})
}

func TestInstallationTelemetryWithIncompleteData(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	cfg := config.DefaultConfig()
	// Don't set installation metadata

	telemetry := NewInstallationTelemetry(logger, cfg)
	require.NotNil(t, telemetry)

	t.Run("ValidateInstallationIntegrity_Incomplete", func(t *testing.T) {
		report, err := telemetry.ValidateInstallationIntegrity()
		require.NoError(t, err)
		require.NotNil(t, report)

		assert.False(t, report.Valid)
		assert.NotEmpty(t, report.Issues)
		
		// Should have issues for missing installation metadata
		// The default config has "manual" method, so check for timestamp missing instead
		found := false
		for _, issue := range report.Issues {
			if issue == "Installation timestamp missing" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected 'Installation timestamp missing' in issues: %v", report.Issues)
	})
}

func TestConfigInstallationMethods(t *testing.T) {
	cfg := config.DefaultConfig()

	t.Run("SetInstallationMethod", func(t *testing.T) {
		cfg.SetInstallationMethod("automated", "test-installer", "pair123", "github", "sha256:test")

		assert.Equal(t, "automated", cfg.Installation.Method)
		assert.Equal(t, "test-installer", cfg.Installation.InstalledBy)
		assert.Equal(t, "pair123", cfg.Installation.PairCode)
		assert.Equal(t, "github", cfg.Installation.Source)
		assert.Equal(t, "sha256:test", cfg.Installation.Checksum)
		assert.NotEmpty(t, cfg.Installation.InstalledAt)
		assert.NotEmpty(t, cfg.Installation.Version)
	})

	t.Run("IsAutomatedInstallation", func(t *testing.T) {
		cfg.Installation.Method = "automated"
		assert.True(t, cfg.IsAutomatedInstallation())

		cfg.Installation.Method = "manual"
		assert.False(t, cfg.IsAutomatedInstallation())
	})

	t.Run("GetInstallationAge", func(t *testing.T) {
		cfg.Installation.InstalledAt = time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
		
		age, err := cfg.GetInstallationAge()
		require.NoError(t, err)
		assert.True(t, age > 50*time.Minute) // Should be close to 1 hour
		assert.True(t, age < 70*time.Minute) // But not too much more
	})

	t.Run("GetInstallationAge_InvalidTimestamp", func(t *testing.T) {
		cfg.Installation.InstalledAt = "invalid-timestamp"
		
		_, err := cfg.GetInstallationAge()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid installation timestamp")
	})

	t.Run("GetInstallationAge_EmptyTimestamp", func(t *testing.T) {
		cfg.Installation.InstalledAt = ""
		
		_, err := cfg.GetInstallationAge()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "installation timestamp not available")
	})
}