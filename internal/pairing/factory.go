package pairing

import (
	"gym-door-bridge/internal/auth"
	"gym-door-bridge/internal/client"
	"gym-door-bridge/internal/config"
	"github.com/sirupsen/logrus"
)

// NewPairingManagerWithRealDependencies creates a pairing manager with actual client and auth manager
func NewPairingManagerWithRealDependencies(cfg *config.Config, logger *logrus.Logger) (*PairingManager, error) {
	// Create auth manager
	authManager, err := auth.NewAuthManager()
	if err != nil {
		return nil, err
	}

	// Initialize auth manager to load existing credentials
	if err := authManager.Initialize(); err != nil {
		return nil, err
	}

	// Create HTTP client
	httpClient, err := client.NewHTTPClientWithAuthManager(cfg, authManager, logger)
	if err != nil {
		return nil, err
	}

	// Create pairing manager
	return NewPairingManager(httpClient, authManager, cfg, logger)
}