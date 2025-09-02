package database

import (
	"database/sql"
	"fmt"
)

// CreateExternalUserMapping creates a new external user mapping
func (db *DB) CreateExternalUserMapping(externalUserID, internalUserID, userName, notes string) (*ExternalUserMapping, error) {
	if externalUserID == "" {
		return nil, fmt.Errorf("external user ID cannot be empty")
	}
	if internalUserID == "" {
		return nil, fmt.Errorf("internal user ID cannot be empty")
	}

	query := `
		INSERT INTO external_user_mappings (external_user_id, internal_user_id, user_name, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, external_user_id, internal_user_id, user_name, notes, created_at, updated_at
	`

	var mapping ExternalUserMapping
	err := db.conn.QueryRow(query, externalUserID, internalUserID, userName, notes).Scan(
		&mapping.ID,
		&mapping.ExternalUserID,
		&mapping.InternalUserID,
		&mapping.UserName,
		&mapping.Notes,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create external user mapping: %w", err)
	}

	return &mapping, nil
}

// GetExternalUserMapping retrieves a mapping by external user ID
func (db *DB) GetExternalUserMapping(externalUserID string) (*ExternalUserMapping, error) {
	if externalUserID == "" {
		return nil, fmt.Errorf("external user ID cannot be empty")
	}

	query := `
		SELECT id, external_user_id, internal_user_id, user_name, notes, created_at, updated_at
		FROM external_user_mappings
		WHERE external_user_id = ?
	`

	var mapping ExternalUserMapping
	err := db.conn.QueryRow(query, externalUserID).Scan(
		&mapping.ID,
		&mapping.ExternalUserID,
		&mapping.InternalUserID,
		&mapping.UserName,
		&mapping.Notes,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No mapping found
		}
		return nil, fmt.Errorf("failed to get external user mapping: %w", err)
	}

	return &mapping, nil
}

// GetExternalUserMappingByInternalID retrieves a mapping by internal user ID
func (db *DB) GetExternalUserMappingByInternalID(internalUserID string) (*ExternalUserMapping, error) {
	if internalUserID == "" {
		return nil, fmt.Errorf("internal user ID cannot be empty")
	}

	query := `
		SELECT id, external_user_id, internal_user_id, user_name, notes, created_at, updated_at
		FROM external_user_mappings
		WHERE internal_user_id = ?
	`

	var mapping ExternalUserMapping
	err := db.conn.QueryRow(query, internalUserID).Scan(
		&mapping.ID,
		&mapping.ExternalUserID,
		&mapping.InternalUserID,
		&mapping.UserName,
		&mapping.Notes,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No mapping found
		}
		return nil, fmt.Errorf("failed to get external user mapping by internal ID: %w", err)
	}

	return &mapping, nil
}

// UpdateExternalUserMapping updates an existing external user mapping
func (db *DB) UpdateExternalUserMapping(externalUserID, internalUserID, userName, notes string) (*ExternalUserMapping, error) {
	if externalUserID == "" {
		return nil, fmt.Errorf("external user ID cannot be empty")
	}
	if internalUserID == "" {
		return nil, fmt.Errorf("internal user ID cannot be empty")
	}

	query := `
		UPDATE external_user_mappings
		SET internal_user_id = ?, user_name = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
		WHERE external_user_id = ?
		RETURNING id, external_user_id, internal_user_id, user_name, notes, created_at, updated_at
	`

	var mapping ExternalUserMapping
	err := db.conn.QueryRow(query, internalUserID, userName, notes, externalUserID).Scan(
		&mapping.ID,
		&mapping.ExternalUserID,
		&mapping.InternalUserID,
		&mapping.UserName,
		&mapping.Notes,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("external user mapping not found for external user ID: %s", externalUserID)
		}
		return nil, fmt.Errorf("failed to update external user mapping: %w", err)
	}

	return &mapping, nil
}

// DeleteExternalUserMapping deletes an external user mapping
func (db *DB) DeleteExternalUserMapping(externalUserID string) error {
	if externalUserID == "" {
		return fmt.Errorf("external user ID cannot be empty")
	}

	query := `DELETE FROM external_user_mappings WHERE external_user_id = ?`
	result, err := db.conn.Exec(query, externalUserID)
	if err != nil {
		return fmt.Errorf("failed to delete external user mapping: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("external user mapping not found for external user ID: %s", externalUserID)
	}

	return nil
}

// ListExternalUserMappings retrieves all external user mappings with optional pagination
func (db *DB) ListExternalUserMappings(limit, offset int) ([]ExternalUserMapping, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, external_user_id, internal_user_id, user_name, notes, created_at, updated_at
		FROM external_user_mappings
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list external user mappings: %w", err)
	}
	defer rows.Close()

	var mappings []ExternalUserMapping
	for rows.Next() {
		var mapping ExternalUserMapping
		err := rows.Scan(
			&mapping.ID,
			&mapping.ExternalUserID,
			&mapping.InternalUserID,
			&mapping.UserName,
			&mapping.Notes,
			&mapping.CreatedAt,
			&mapping.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan external user mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating external user mappings: %w", err)
	}

	return mappings, nil
}

// CountExternalUserMappings returns the total count of external user mappings
func (db *DB) CountExternalUserMappings() (int64, error) {
	query := `SELECT COUNT(*) FROM external_user_mappings`
	
	var count int64
	err := db.conn.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count external user mappings: %w", err)
	}

	return count, nil
}

// ResolveExternalUserID resolves an external user ID to an internal user ID
// Returns the internal user ID if mapping exists, empty string if not found
func (db *DB) ResolveExternalUserID(externalUserID string) (string, error) {
	mapping, err := db.GetExternalUserMapping(externalUserID)
	if err != nil {
		return "", err
	}
	
	if mapping == nil {
		return "", nil // No mapping found
	}
	
	return mapping.InternalUserID, nil
}