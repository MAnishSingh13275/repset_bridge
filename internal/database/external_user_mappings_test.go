package database

import (
	"testing"
	"time"
)

func TestCreateExternalUserMapping(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	tests := []struct {
		name           string
		externalUserID string
		internalUserID string
		userName       string
		notes          string
		expectError    bool
	}{
		{
			name:           "valid mapping",
			externalUserID: "fp_12345",
			internalUserID: "user_abc123",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    false,
		},
		{
			name:           "empty external user ID",
			externalUserID: "",
			internalUserID: "user_abc123",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    true,
		},
		{
			name:           "empty internal user ID",
			externalUserID: "fp_12345",
			internalUserID: "",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    true,
		},
		{
			name:           "minimal mapping",
			externalUserID: "fp_67890",
			internalUserID: "user_def456",
			userName:       "",
			notes:          "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := db.CreateExternalUserMapping(tt.externalUserID, tt.internalUserID, tt.userName, tt.notes)

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

			if mapping == nil {
				t.Errorf("expected mapping but got nil")
				return
			}

			if mapping.ExternalUserID != tt.externalUserID {
				t.Errorf("expected external user ID %s, got %s", tt.externalUserID, mapping.ExternalUserID)
			}

			if mapping.InternalUserID != tt.internalUserID {
				t.Errorf("expected internal user ID %s, got %s", tt.internalUserID, mapping.InternalUserID)
			}

			if mapping.UserName != tt.userName {
				t.Errorf("expected user name %s, got %s", tt.userName, mapping.UserName)
			}

			if mapping.Notes != tt.notes {
				t.Errorf("expected notes %s, got %s", tt.notes, mapping.Notes)
			}

			if mapping.ID == 0 {
				t.Errorf("expected non-zero ID")
			}

			if mapping.CreatedAt.IsZero() {
				t.Errorf("expected non-zero created at timestamp")
			}

			if mapping.UpdatedAt.IsZero() {
				t.Errorf("expected non-zero updated at timestamp")
			}
		})
	}
}

func TestGetExternalUserMapping(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Create a test mapping
	created, err := db.CreateExternalUserMapping("fp_12345", "user_abc123", "John Doe", "Test user")
	if err != nil {
		t.Fatalf("failed to create test mapping: %v", err)
	}

	tests := []struct {
		name           string
		externalUserID string
		expectFound    bool
		expectError    bool
	}{
		{
			name:           "existing mapping",
			externalUserID: "fp_12345",
			expectFound:    true,
			expectError:    false,
		},
		{
			name:           "non-existing mapping",
			externalUserID: "fp_99999",
			expectFound:    false,
			expectError:    false,
		},
		{
			name:           "empty external user ID",
			externalUserID: "",
			expectFound:    false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := db.GetExternalUserMapping(tt.externalUserID)

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

			if tt.expectFound {
				if mapping == nil {
					t.Errorf("expected mapping but got nil")
					return
				}

				if mapping.ExternalUserID != created.ExternalUserID {
					t.Errorf("expected external user ID %s, got %s", created.ExternalUserID, mapping.ExternalUserID)
				}

				if mapping.InternalUserID != created.InternalUserID {
					t.Errorf("expected internal user ID %s, got %s", created.InternalUserID, mapping.InternalUserID)
				}
			} else {
				if mapping != nil {
					t.Errorf("expected nil mapping but got %+v", mapping)
				}
			}
		})
	}
}

func TestUpdateExternalUserMapping(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Create a test mapping
	_, err := db.CreateExternalUserMapping("fp_12345", "user_abc123", "John Doe", "Test user")
	if err != nil {
		t.Fatalf("failed to create test mapping: %v", err)
	}

	// Wait a moment to ensure updated_at will be different
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name           string
		externalUserID string
		internalUserID string
		userName       string
		notes          string
		expectError    bool
	}{
		{
			name:           "valid update",
			externalUserID: "fp_12345",
			internalUserID: "user_xyz789",
			userName:       "Jane Doe",
			notes:          "Updated user",
			expectError:    false,
		},
		{
			name:           "non-existing mapping",
			externalUserID: "fp_99999",
			internalUserID: "user_abc123",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    true,
		},
		{
			name:           "empty external user ID",
			externalUserID: "",
			internalUserID: "user_abc123",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    true,
		},
		{
			name:           "empty internal user ID",
			externalUserID: "fp_12345",
			internalUserID: "",
			userName:       "John Doe",
			notes:          "Test user",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, err := db.UpdateExternalUserMapping(tt.externalUserID, tt.internalUserID, tt.userName, tt.notes)

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

			if mapping == nil {
				t.Errorf("expected mapping but got nil")
				return
			}

			if mapping.ExternalUserID != tt.externalUserID {
				t.Errorf("expected external user ID %s, got %s", tt.externalUserID, mapping.ExternalUserID)
			}

			if mapping.InternalUserID != tt.internalUserID {
				t.Errorf("expected internal user ID %s, got %s", tt.internalUserID, mapping.InternalUserID)
			}

			if mapping.UserName != tt.userName {
				t.Errorf("expected user name %s, got %s", tt.userName, mapping.UserName)
			}

			if mapping.Notes != tt.notes {
				t.Errorf("expected notes %s, got %s", tt.notes, mapping.Notes)
			}
		})
	}
}

func TestDeleteExternalUserMapping(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Create a test mapping
	_, err := db.CreateExternalUserMapping("fp_12345", "user_abc123", "John Doe", "Test user")
	if err != nil {
		t.Fatalf("failed to create test mapping: %v", err)
	}

	tests := []struct {
		name           string
		externalUserID string
		expectError    bool
	}{
		{
			name:           "existing mapping",
			externalUserID: "fp_12345",
			expectError:    false,
		},
		{
			name:           "non-existing mapping",
			externalUserID: "fp_99999",
			expectError:    true,
		},
		{
			name:           "empty external user ID",
			externalUserID: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.DeleteExternalUserMapping(tt.externalUserID)

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

			// Verify the mapping was deleted
			mapping, err := db.GetExternalUserMapping(tt.externalUserID)
			if err != nil {
				t.Errorf("unexpected error checking deleted mapping: %v", err)
				return
			}

			if mapping != nil {
				t.Errorf("expected mapping to be deleted but still exists: %+v", mapping)
			}
		})
	}
}

func TestListExternalUserMappings(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Create test mappings
	testMappings := []struct {
		externalUserID string
		internalUserID string
		userName       string
	}{
		{"fp_12345", "user_abc123", "John Doe"},
		{"fp_67890", "user_def456", "Jane Smith"},
		{"rfid_11111", "user_ghi789", "Bob Johnson"},
	}

	for _, tm := range testMappings {
		_, err := db.CreateExternalUserMapping(tm.externalUserID, tm.internalUserID, tm.userName, "")
		if err != nil {
			t.Fatalf("failed to create test mapping: %v", err)
		}
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
	}{
		{
			name:          "all mappings",
			limit:         10,
			offset:        0,
			expectedCount: 3,
		},
		{
			name:          "limited mappings",
			limit:         2,
			offset:        0,
			expectedCount: 2,
		},
		{
			name:          "offset mappings",
			limit:         10,
			offset:        1,
			expectedCount: 2,
		},
		{
			name:          "default limit",
			limit:         0,
			offset:        0,
			expectedCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mappings, err := db.ListExternalUserMappings(tt.limit, tt.offset)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(mappings) != tt.expectedCount {
				t.Errorf("expected %d mappings, got %d", tt.expectedCount, len(mappings))
			}

			// Verify mappings are ordered by created_at DESC
			for i := 1; i < len(mappings); i++ {
				if mappings[i-1].CreatedAt.Before(mappings[i].CreatedAt) {
					t.Errorf("mappings not ordered by created_at DESC")
				}
			}
		})
	}
}

func TestCountExternalUserMappings(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Initially should be 0
	count, err := db.CountExternalUserMappings()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}

	// Create test mappings
	testMappings := []struct {
		externalUserID string
		internalUserID string
	}{
		{"fp_12345", "user_abc123"},
		{"fp_67890", "user_def456"},
		{"rfid_11111", "user_ghi789"},
	}

	for _, tm := range testMappings {
		_, err := db.CreateExternalUserMapping(tm.externalUserID, tm.internalUserID, "", "")
		if err != nil {
			t.Fatalf("failed to create test mapping: %v", err)
		}
	}

	// Should now be 3
	count, err = db.CountExternalUserMappings()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestResolveExternalUserID(t *testing.T) {
	db := setupTestDB(t, TierNormal)

	// Create a test mapping
	_, err := db.CreateExternalUserMapping("fp_12345", "user_abc123", "John Doe", "Test user")
	if err != nil {
		t.Fatalf("failed to create test mapping: %v", err)
	}

	tests := []struct {
		name           string
		externalUserID string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "existing mapping",
			externalUserID: "fp_12345",
			expectedResult: "user_abc123",
			expectError:    false,
		},
		{
			name:           "non-existing mapping",
			externalUserID: "fp_99999",
			expectedResult: "",
			expectError:    false,
		},
		{
			name:           "empty external user ID",
			externalUserID: "",
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := db.ResolveExternalUserID(tt.externalUserID)

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
				t.Errorf("expected result %s, got %s", tt.expectedResult, result)
			}
		})
	}
}