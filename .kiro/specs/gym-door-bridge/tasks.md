# Implementation Plan

- [x] 1. Initialize Go project and core structure

  - Create go.mod file and basic project structure (cmd/, internal/, pkg/)
  - Set up main.go entry point with basic CLI argument parsing
  - Create internal/config package for configuration management
  - Add basic logging setup using structured logging library
  - _Requirements: 1.4, 2.1_

- [x] 2. Define core interfaces and types

  - Create internal/types package with core data structures (RawHardwareEvent, StandardEvent)
  - Define HardwareAdapter interface in internal/adapters package
  - Create EventProcessor interface in internal/processor package
  - Add QueueManager interface in internal/queue package
  - _Requirements: 1.1, 1.2, 1.4_

- [x] 3. Implement SQLite database layer

  - Create internal/database package with SQLite connection management
  - Implement schema with migrations (event_queue, device_config, adapter_status tables)
  - Enable WAL mode for durability and performance
  - Add AES-GCM encryption for sensitive payloads before storage
  - Configure tier-specific pragmas (Lite=NORMAL, Full=FULL sync)
  - Add CRUD operations for event storage and configuration
  - Write unit tests for database operations
  - _Requirements: 4.1, 4.2, 4.5_

- [x] 4. Build authentication and security foundation

  - Create internal/auth package with HMAC-SHA256 implementation
  - Implement device credential management (deviceId, deviceKey storage via OS keychain/DPAPI)
  - Add request signing and validation functions
  - Write unit tests for authentication flows
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 5. Create simulator hardware adapter

  - Implement SimulatorAdapter in internal/adapters/simulator
  - Add mock event generation for testing purposes
  - Implement door unlock simulation with logging
  - Write unit tests for simulator functionality
  - _Requirements: 1.3, 9.5_

- [x] 6. Implement event processing pipeline

  - Create EventProcessor in internal/processor package
  - Add event normalization from RawHardwareEvent to StandardEvent
  - Implement metadata enrichment (deviceId, timestamps, event IDs)
  - Add event validation and deduplication logic
  - Write unit tests for event transformation
  - _Requirements: 1.1, 1.2, 4.4_

- [x] 7. Build offline queue management

  - Implement QueueManager in internal/queue package
  - Add local event storage in SQLite with encrypted payloads
  - Create batch processing for event replay
  - Implement queue size limits per tier (Lite=1k, Normal=10k, Full=50k) with FIFO eviction
  - Write unit tests for offline storage and replay
  - _Requirements: 4.1, 4.2, 4.3, 4.5_

- [x] 8. Create HTTP client with authentication

  - Implement HTTP client in internal/client package
  - Add HMAC authentication headers to all requests (with clock skew tolerance)
  - Create retry logic with exponential backoff
  - Add network connectivity detection
  - Write unit tests for HTTP communication
  - _Requirements: 3.2, 4.1, 4.2_

- [x] 9. Implement device pairing functionality

  - Create pairing logic in internal/pairing package
  - Add /api/v1/devices/pair endpoint client implementation
  - Implement device registration with Pair Code
  - Store received deviceId and deviceKey securely
  - Write unit tests for pairing flow

  - _Requirements: 3.1, 6.3_

- [x] 10. Add check-in event submission

  - Implement /api/v1/checkin endpoint client
  - Add batch event submission with proper authentication
  - Create idempotency key generation for events
  - Add error handling for submission failures
  - Write unit tests for event submission
  - _Requirements: 4.4, 7.1_

- [x] 11. Build performance tier detection

  - Create internal/tier package for system resource monitoring
  - Implement CPU, memory, and disk usage detection
  - Add automatic tier assignment (Lite/Normal/Full)
  - Periodically re-evaluate tier assignment
  - Write unit tests for tier detection
  - _Requirements: 2.3, 2.4, 2.5, 2.6_

- [x] 12. Implement health monitoring system

  - Create internal/health package with health check endpoint
  - Add system status reporting (queue depth, adapter status, resources)
  - Implement HeartbeatManager for periodic cloud updates
  - Create /api/v1/devices/heartbeat endpoint client
  - Add optional OpenTelemetry metrics exporter (disabled by default)
  - Write unit tests for health reporting
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 13. Add door control functionality

  - Extend HardwareAdapter interface with UnlockDoor method
  - Implement door unlock in SimulatorAdapter
  - Create HTTP endpoint /open-door for remote door control
  - Add configurable unlock duration with automatic re-lock (safe fallback = locked)
  - Write unit tests for door control operations
  - _Requirements: 9.1, 9.2, 9.3, 9.5_

- [x] 14. Create external user mapping system

  - Add external_user_mappings table to database schema
  - Implement user mapping resolution in event processing
  - Add logging for unmapped external user IDs
  - Create mapping management functions
  - Write unit tests for user mapping resolution
  - _Requirements: 7.1, 7.3, 7.5_

- [x] 15. Build service management for Windows

  - Create internal/service/windows package
  - Implement Windows service wrapper using golang.org/x/sys/windows/svc
  - Add service installation, start, stop, and uninstall functions
  - Create service configuration and lifecycle management
  - Sign binaries with Authenticode for distribution
  - Write integration tests for Windows service operations
  - _Requirements: 2.1, 2.2, 6.4_

- [x] 16. Build service management for macOS

  - Create internal/service/macos package
  - Implement macOS daemon support with launchd integration
  - Add plist file generation and daemon lifecycle management
  - Add notarization for macOS binaries
  - Create installation and uninstallation scripts
  - Write integration tests for macOS daemon operations
  - _Requirements: 2.1, 2.2, 6.4_

- [x] 17. Create installation scripts

  - Build PowerShell installation script for Windows
  - Create bash installation script for macOS
  - Implement automatic pairing during installation process
  - Add configuration file generation and service registration
  - Write integration tests for installation flows
  - _Requirements: 6.1, 6.2, 6.3, 6.5_

- [x] 18. Implement update mechanism

  - Create internal/updater package for update management
  - Add manifest.json checking from CDN
  - Implement binary download with Ed25519 signature verification
  - Add staged rollout support (percentage rollout field in manifest)
  - Create graceful restart with queue preservation
  - Rollback if new version fails health checks
  - Write integration tests for update process
  - _Requirements: 10.1, 10.2, 10.5_

- [x] 19. Add comprehensive error handling and logging

  - Enhance logging throughout all packages with structured logging
  - Implement error recovery mechanisms for hardware and network failures
  - Add graceful degradation for resource constraints
  - Create error classification and handling strategies
  - Write unit tests for error scenarios
  - _Requirements: 1.5, 2.6, 8.4_

- [x] 20. Build additional hardware adapters

  - Create WebhookAdapter for HTTP-based integrations in internal/adapters/webhook
  - Add framework for fingerprint and RFID adapter implementations
  - Support config-driven adapter discovery (enable/disable in config)
  - Implement adapter lifecycle management and status tracking
  - Write unit tests for adapter framework
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 21. Implement monitoring and alerting

  - Add alert generation for offline devices and queue thresholds
  - Create security event logging for HMAC failures
  - Implement performance monitoring with diagnostic collection
  - Add metrics collection and reporting to admin portal
  - Write unit tests for monitoring functionality
  - _Requirements: 8.1, 8.2, 8.3, 8.5_

- [x] 22. Create comprehensive test suite

  - Build integration tests for complete hardware-to-cloud flow
  - Add load testing for different performance tiers
  - Implement security testing for authentication flows
  - Create end-to-end tests simulating real deployment scenarios
  - _Requirements: All requirements validation_

- [x] 23. Build deployment artifacts

  - Set up build scripts for cross-platform binary compilation
  - Create Docker image with multi-stage builds
  - Sign binaries (Windows Authenticode, macOS notarization)
  - Generate deployment documentation and troubleshooting guides
  - Create manifest.json template for update distribution
  - _Requirements: 10.3, 10.4, 10.5_
