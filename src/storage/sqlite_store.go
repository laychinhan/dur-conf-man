package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"config-manager/src/models"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore handles all database operations for configurations and versions
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite storage instance
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// CreateConfiguration creates a new configuration with version 1
// Implements the data access pattern from data-model.md
func (s *SQLiteStore) CreateConfiguration(name, jsonData string) (*models.Configuration, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Failed to rollback transaction: %v", err)
		}
	}()

	now := time.Now()

	// 1. Insert new configuration record
	configQuery := `
		INSERT INTO configurations (name, current_version, created_at, updated_at)
		VALUES (?, ?, ?, ?)`

	_, err = tx.Exec(configQuery, name, 1, now, now)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, &ConfigAlreadyExistsError{ConfigName: name}
		}
		return nil, fmt.Errorf("failed to insert configuration: %w", err)
	}

	// 2. Insert version 1 record
	versionQuery := `
		INSERT INTO versions (configuration_name, version_number, json_data, created_at)
		VALUES (?, ?, ?, ?)`

	_, err = tx.Exec(versionQuery, name, 1, jsonData, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert version: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Configuration{
		Name:           name,
		CurrentVersion: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// UpdateConfiguration updates an existing configuration, increments version, and returns updated config
func (s *SQLiteStore) UpdateConfiguration(name, jsonData string) (*models.Configuration, error) {
	// Check if configuration exists
	var currentVersion int
	row := s.db.QueryRow("SELECT current_version FROM configurations WHERE name = ?", name)
	if err := row.Scan(&currentVersion); err != nil {
		if err == sql.ErrNoRows {
			return nil, &ConfigNotFoundError{ConfigName: name}
		}
		return nil, fmt.Errorf("failed to query configuration: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Failed to rollback transaction: %v", err)
		}
	}()

	newVersion := currentVersion + 1
	now := time.Now()

	// Insert new version row
	versionQuery := `
		INSERT INTO versions (configuration_name, version_number, json_data, created_at)
		VALUES (?, ?, ?, ?)`
	_, err = tx.Exec(versionQuery, name, newVersion, jsonData, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new version: %w", err)
	}

	// Update current_version in configurations table
	updateConfigQuery := `
		UPDATE configurations SET current_version = ?, updated_at = ? WHERE name = ?`
	_, err = tx.Exec(updateConfigQuery, newVersion, now, name)
	if err != nil {
		return nil, fmt.Errorf("failed to update configuration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Configuration{
		Name:           name,
		CurrentVersion: newVersion,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// RollbackConfiguration creates a new version with data from target version
func (s *SQLiteStore) RollbackConfiguration(name string, targetVersion int) (*models.Configuration, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Failed to rollback transaction: %v", err)
		}
	}()

	// 1. Validate target version exists and get its data
	var targetJsonData string
	versionQuery := `SELECT json_data FROM versions WHERE configuration_name = ? AND version_number = ?`
	err = tx.QueryRow(versionQuery, name, targetVersion).Scan(&targetJsonData)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &VersionNotFoundError{ConfigName: name, Version: targetVersion}
		}
		return nil, fmt.Errorf("failed to get target version data: %w", err)
	}

	// 2. Get current version number and created_at
	var currentVersion int
	var createdAtStr string
	configQuery := `SELECT current_version, created_at FROM configurations WHERE name = ?`
	err = tx.QueryRow(configQuery, name).Scan(&currentVersion, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &ConfigNotFoundError{ConfigName: name}
		}
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	// Parse SQLite timestamp format using helper
	createdAt, err := parseTimestamp(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	// 3. Insert new version with target's JSON data
	newVersion := currentVersion + 1
	now := time.Now()
	insertVersionQuery := `
		INSERT INTO versions (configuration_name, version_number, json_data, created_at)
		VALUES (?, ?, ?, ?)`

	_, err = tx.Exec(insertVersionQuery, name, newVersion, targetJsonData, now)
	if err != nil {
		return nil, fmt.Errorf("failed to insert rollback version: %w", err)
	}

	// 4. Update configuration's current_version
	updateQuery := `UPDATE configurations SET current_version = ?, updated_at = ? WHERE name = ?`
	_, err = tx.Exec(updateQuery, newVersion, now, name)
	if err != nil {
		return nil, fmt.Errorf("failed to update current version: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Configuration{
		Name:           name,
		CurrentVersion: newVersion,
		CreatedAt:      createdAt,
		UpdatedAt:      now,
	}, nil
}

// GetLatestConfiguration retrieves the latest version of a configuration
func (s *SQLiteStore) GetLatestConfiguration(name string) (*models.Configuration, *models.Version, error) {
	query := `
		SELECT c.name, c.current_version, c.created_at, c.updated_at,
		       v.id, v.version_number, v.json_data, v.created_at
		FROM configurations c
		JOIN versions v ON c.name = v.configuration_name AND c.current_version = v.version_number
		WHERE c.name = ?`

	var config models.Configuration
	var version models.Version
	var configCreatedAtStr, configUpdatedAtStr, versionCreatedAtStr string

	err := s.db.QueryRow(query, name).Scan(
		&config.Name, &config.CurrentVersion, &configCreatedAtStr, &configUpdatedAtStr,
		&version.ID, &version.VersionNumber, &version.JsonData, &versionCreatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, &ConfigNotFoundError{ConfigName: name}
		}
		return nil, nil, fmt.Errorf("failed to get latest configuration: %w", err)
	}

	// Parse timestamps
	config.CreatedAt, err = parseTimestamp(configCreatedAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config created_at: %w", err)
	}

	config.UpdatedAt, err = parseTimestamp(configUpdatedAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config updated_at: %w", err)
	}

	version.CreatedAt, err = parseTimestamp(versionCreatedAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse version created_at: %w", err)
	}

	version.ConfigurationName = name
	return &config, &version, nil
}

// GetConfigurationVersion retrieves a specific version of a configuration
func (s *SQLiteStore) GetConfigurationVersion(name string, versionNumber int) (*models.Version, error) {
	query := `
		SELECT id, configuration_name, version_number, json_data, created_at
		FROM versions 
		WHERE configuration_name = ? AND version_number = ?`

	var version models.Version
	var createdAtStr string
	err := s.db.QueryRow(query, name, versionNumber).Scan(
		&version.ID, &version.ConfigurationName, &version.VersionNumber,
		&version.JsonData, &createdAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, &VersionNotFoundError{ConfigName: name, Version: versionNumber}
		}
		return nil, fmt.Errorf("failed to get configuration version: %w", err)
	}

	// Parse timestamp
	version.CreatedAt, err = parseTimestamp(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse created_at: %w", err)
	}

	return &version, nil
}

// ListVersions retrieves all versions of a configuration
func (s *SQLiteStore) ListVersions(name string) (*models.Configuration, []models.Version, error) {
	// First check if configuration exists
	var config models.Configuration
	var createdAtStr, updatedAtStr string
	configQuery := `SELECT name, current_version, created_at, updated_at FROM configurations WHERE name = ?`
	err := s.db.QueryRow(configQuery, name).Scan(
		&config.Name, &config.CurrentVersion, &createdAtStr, &updatedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, &ConfigNotFoundError{ConfigName: name}
		}
		return nil, nil, fmt.Errorf("failed to get configuration: %w", err)
	}

	// Parse configuration timestamps
	config.CreatedAt, err = parseTimestamp(createdAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config created_at: %w", err)
	}

	config.UpdatedAt, err = parseTimestamp(updatedAtStr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config updated_at: %w", err)
	}

	// Get all versions ordered by version number descending
	versionsQuery := `
		SELECT id, configuration_name, version_number, json_data, created_at
		FROM versions 
		WHERE configuration_name = ?
		ORDER BY version_number DESC`

	rows, err := s.db.Query(versionsQuery, name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var versions []models.Version
	for rows.Next() {
		var version models.Version
		var versionCreatedAtStr string
		err := rows.Scan(
			&version.ID, &version.ConfigurationName, &version.VersionNumber,
			&version.JsonData, &versionCreatedAtStr,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan version: %w", err)
		}

		// Parse version timestamp
		version.CreatedAt, err = parseTimestamp(versionCreatedAtStr)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse version created_at: %w", err)
		}

		versions = append(versions, version)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating versions: %w", err)
	}

	return &config, versions, nil
}

// parseTimestamp parses SQLite timestamp strings with fallback formats
func parseTimestamp(timestampStr string) (time.Time, error) {
	// Try different SQLite timestamp formats
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00", // Full format with timezone
		"2006-01-02 15:04:05.999999999",       // Without timezone
		"2006-01-02 15:04:05",                 // Simple format
		time.RFC3339,                          // ISO format
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestampStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", timestampStr)
}

// Error types for specific database errors
type ConfigAlreadyExistsError struct {
	ConfigName string
}

func (e *ConfigAlreadyExistsError) Error() string {
	return fmt.Sprintf("CONFIG_ALREADY_EXISTS: Configuration '%s' already exists", e.ConfigName)
}

type ConfigNotFoundError struct {
	ConfigName string
}

func (e *ConfigNotFoundError) Error() string {
	return fmt.Sprintf("CONFIG_NOT_FOUND: Configuration '%s' not found", e.ConfigName)
}

type VersionNotFoundError struct {
	ConfigName string
	Version    int
}

func (e *VersionNotFoundError) Error() string {
	return fmt.Sprintf("VERSION_NOT_FOUND: Version %d not found for configuration '%s'", e.Version, e.ConfigName)
}

// isUniqueConstraintError checks if the error is due to unique constraint violation
func isUniqueConstraintError(err error) bool {
	return err != nil &&
		(err.Error() == "UNIQUE constraint failed: configurations.name" ||
			err.Error() == "constraint failed: UNIQUE constraint failed: configurations.name")
}
