package services

import (
	"encoding/json"
	"fmt"

	"config-manager/src/models"
	"config-manager/src/storage"
)

// ConfigService handles all configuration management business logic
type ConfigService struct {
	store             *storage.SQLiteStore
	validationService *ValidationService
}

// NewConfigService creates a new configuration service
func NewConfigService(store *storage.SQLiteStore, validationService *ValidationService) *ConfigService {
	return &ConfigService{
		store:             store,
		validationService: validationService,
	}
}

// CreateConfig creates a new configuration with validation (FR-001, FR-002, FR-003)
//
// CreateConfig handles the creation of a new configuration.
// It validates the input JSON against the hardcoded schema and stores the configuration
// with version 1 in the database.
//
// Returns the created Configuration model or an error if validation/storage fails.
func (cs *ConfigService) CreateConfig(name string, jsonData string) (*models.Configuration, error) {
	// Validate JSON against hardcoded schema
	if err := cs.validationService.ValidateConfigData(jsonData); err != nil {
		return nil, err
	}

	// Create configuration with version 1
	config, err := cs.store.CreateConfiguration(name, jsonData)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// UpdateConfig updates an existing configuration with new data (FR-004, FR-005)
//
// UpdateConfig validates the new configuration data against the schema and updates
// the configuration, incrementing the version number.
//
// Returns the updated Configuration model or an error if validation/storage fails.
func (cs *ConfigService) UpdateConfig(name string, jsonData string) (*models.Configuration, error) {
	// Validate JSON against hardcoded schema
	if err := cs.validationService.ValidateConfigData(jsonData); err != nil {
		return nil, err
	}

	// Update configuration (creates new version)
	config, err := cs.store.UpdateConfiguration(name, jsonData)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// RollbackConfig rolls back configuration to a previous version (FR-008, FR-009)
//
// RollbackConfig reverts the configuration to the specified previous version and
// creates a new version entry in the database.
//
// Returns the rolled-back Configuration model or an error if the version is invalid or not found.
func (cs *ConfigService) RollbackConfig(name string, targetVersion int) (*models.Configuration, error) {
	if targetVersion < 1 {
		return nil, fmt.Errorf("INVALID_VERSION_NUMBER: Version number must be positive integer")
	}

	// Rollback configuration (creates new version with target data)
	config, err := cs.store.RollbackConfiguration(name, targetVersion)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetLatestConfig retrieves the latest version of a configuration (FR-006)
//
// GetLatestConfig fetches the most recent configuration data for the given name.
// Returns a ConfigurationData struct containing the latest config and metadata.
func (cs *ConfigService) GetLatestConfig(name string) (*models.ConfigurationData, error) {
	config, version, err := cs.store.GetLatestConfiguration(name)
	if err != nil {
		return nil, err
	}

	// Parse JSON data into ConfigData struct
	var configData models.ConfigData
	if err := json.Unmarshal([]byte(version.JsonData), &configData); err != nil {
		return nil, fmt.Errorf("failed to parse configuration data: %w", err)
	}

	return &models.ConfigurationData{
		Name:       config.Name,
		Version:    config.CurrentVersion,
		ConfigData: configData,
		CreatedAt:  version.CreatedAt,
	}, nil
}

// GetConfigVersion retrieves a specific version of a configuration (FR-007)
//
// GetConfigVersion fetches the configuration data for the specified version number.
// Returns a ConfigurationData struct for the requested version or an error if not found.
func (cs *ConfigService) GetConfigVersion(name string, versionNumber int) (*models.ConfigurationData, error) {
	if versionNumber < 1 {
		return nil, fmt.Errorf("INVALID_VERSION_NUMBER: Version number must be positive integer")
	}

	version, err := cs.store.GetConfigurationVersion(name, versionNumber)
	if err != nil {
		return nil, err
	}

	// Parse JSON data into ConfigData struct
	var configData models.ConfigData
	if err := json.Unmarshal([]byte(version.JsonData), &configData); err != nil {
		return nil, fmt.Errorf("failed to parse configuration data: %w", err)
	}

	return &models.ConfigurationData{
		Name:       version.ConfigurationName,
		Version:    version.VersionNumber,
		ConfigData: configData,
		CreatedAt:  version.CreatedAt,
	}, nil
}

// ListVersions lists all versions of a configuration (FR-010)
//
// ListVersions returns a list of all version numbers and their creation timestamps
// for the specified configuration name.
// Returns a VersionList struct or an error if the configuration is not found.
func (cs *ConfigService) ListVersions(name string) (*models.VersionList, error) {
	config, versions, err := cs.store.ListVersions(name)
	if err != nil {
		return nil, err
	}

	// Convert to VersionInfo structs
	versionInfos := make([]models.VersionInfo, len(versions))
	for i, version := range versions {
		versionInfos[i] = models.VersionInfo{
			Version:   version.VersionNumber,
			CreatedAt: version.CreatedAt,
		}
	}

	return &models.VersionList{
		Name:           config.Name,
		CurrentVersion: config.CurrentVersion,
		Versions:       versionInfos,
	}, nil
}
