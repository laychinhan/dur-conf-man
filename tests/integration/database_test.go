package integration

import (
	"config-manager/src/models"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	"config-manager/src/services"
	"config-manager/src/storage"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// DatabaseTestSuite provides integration testing with real SQLite database
type DatabaseTestSuite struct {
	suite.Suite
	db *sql.DB
}

// SetupTest creates a fresh test database before each test
func (suite *DatabaseTestSuite) SetupTest() {
	// Create temporary test database
	testDB := "./test_config.db"

	// Remove existing test database
	_ = os.Remove(testDB)

	db, err := sql.Open("sqlite3", testDB)
	suite.Require().NoError(err)

	// Create tables using the exact schema from migrations
	schema := `
	CREATE TABLE configurations (
		name TEXT PRIMARY KEY,
		current_version INTEGER NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE versions (
		id INTEGER PRIMARY KEY,
		configuration_name TEXT NOT NULL,
		version_number INTEGER NOT NULL,
		json_data TEXT NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (configuration_name) REFERENCES configurations(name),
		UNIQUE(configuration_name, version_number)
	);

	CREATE INDEX idx_configurations_name ON configurations(name);
	CREATE INDEX idx_versions_config_version ON versions(configuration_name, version_number);
	CREATE INDEX idx_versions_config_created ON versions(configuration_name, created_at DESC);
	`

	_, err = db.Exec(schema)
	suite.Require().NoError(err)

	suite.db = db
}

// TearDownTest cleans up the test database after each test
func (suite *DatabaseTestSuite) TearDownTest() {
	if suite.db != nil {
		_ = suite.db.Close()
		_ = os.Remove("./test_config.db")
	}
}

// TestCreateConfiguration tests the create configuration workflow
func (suite *DatabaseTestSuite) TestCreateConfiguration() {
	// This test will fail initially - no service implementation exists yet

	// Test data matching the hardcoded JSON schema
	configName := "test-config"
	jsonData := `{"max_limit": 1000, "enabled": true}`

	// Expected workflow:
	// 1. Insert configuration record
	// 2. Insert version 1 record
	// 3. Set current_version to 1

	// Create required dependencies for ConfigService
	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	// This will fail until ConfigService is implemented
	service := services.NewConfigService(store, validationService)
	config, err := service.CreateConfig(configName, jsonData)
	suite.NoError(err)
	suite.Equal(configName, config.Name)
	suite.Equal(1, config.CurrentVersion)

	// Temporary assertion to make test fail (RED phase)
	//suite.Fail("ConfigService not implemented yet")
}

// TestUpdateConfiguration tests the update configuration workflow
func (suite *DatabaseTestSuite) TestUpdateConfiguration() {
	// This test will fail initially - no service implementation exists yet

	// Expected workflow:
	// 1. Create initial configuration (version 1)
	// 2. Update with new data (creates version 2)
	// 3. Verify current_version incremented

	// This will fail until ConfigService is implemented
	configName := "test-config"
	initialData := `{"max_limit": 1000, "enabled": true}`
	updatedData := `{"max_limit": 2000, "enabled": false}`
	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)

	// Create initial config
	_, err = service.CreateConfig(configName, initialData)
	suite.NoError(err)

	// Update config
	updatedConfig, err := service.UpdateConfig(configName, updatedData)
	suite.NoError(err)
	suite.Equal(2, updatedConfig.CurrentVersion)

	// Temporary assertion to make test fail (RED phase)
	//suite.Fail("ConfigService.UpdateConfig not implemented yet")
}

// TestRollbackConfiguration tests the rollback workflow
func (suite *DatabaseTestSuite) TestRollbackConfiguration() {
	// This test will fail initially - no service implementation exists yet

	// Expected workflow:
	// 1. Create config (version 1)
	// 2. Update config (version 2)
	// 3. Rollback to version 1 (creates version 3 with version 1 data)

	// This will fail until ConfigService is implemented
	configName := "test-config"
	version1Data := `{"max_limit": 1000, "enabled": true}`
	version2Data := `{"max_limit": 2000, "enabled": false}`
	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)

	// Create and update config
	_, err = service.CreateConfig(configName, version1Data)
	suite.NoError(err)
	_, err = service.UpdateConfig(configName, version2Data)
	suite.NoError(err)

	// Rollback to version 1
	rolledBackConfig, err := service.RollbackConfig(configName, 1)
	suite.NoError(err)
	suite.Equal(3, rolledBackConfig.CurrentVersion)

	// Temporary assertion to make test fail (RED phase)
	//suite.Fail("ConfigService.RollbackConfig not implemented yet")
}

// TestRetrieveLatestConfiguration tests getting the latest config version
func (suite *DatabaseTestSuite) TestRetrieveLatestConfiguration() {
	// This will fail until ConfigService is implemented
	configName := "test-config"
	jsonData := `{"max_limit": 1000, "enabled": true}`

	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)
	_, err = service.CreateConfig(configName, jsonData)
	suite.NoError(err)
	config, err := service.GetLatestConfig(configName)
	suite.NoError(err)
	suite.Equal(configName, config.Name)

	//suite.Fail("ConfigService.GetLatestConfig not implemented yet")
}

// TestRetrieveSpecificVersion tests getting a specific version
func (suite *DatabaseTestSuite) TestRetrieveSpecificVersion() {
	// This will fail until ConfigService is implemented
	configName := "test-config"

	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)
	version1Data := `{"max_limit": 1000, "enabled": true}`
	version2Data := `{"max_limit": 2000, "enabled": false}`
	_, err = service.CreateConfig(configName, version1Data)
	suite.NoError(err)
	_, err = service.UpdateConfig(configName, version2Data)
	suite.NoError(err)
	config, err := service.GetConfigVersion(configName, 1)
	suite.NoError(err)
	var expected models.ConfigData
	err = json.Unmarshal([]byte(version1Data), &expected)
	suite.NoError(err)
	suite.Equal(expected, config.ConfigData)
	//suite.Fail("ConfigService.GetConfigVersion not implemented yet")
}

// TestListAllVersions tests listing all versions of a configuration
func (suite *DatabaseTestSuite) TestListAllVersions() {
	// This will fail until ConfigService is implemented
	configName := "test-config"

	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)
	version1Data := `{"max_limit": 1000, "enabled": true}`
	version2Data := `{"max_limit": 2000, "enabled": false}`
	_, err = service.CreateConfig(configName, version1Data)
	suite.NoError(err)
	_, err = service.UpdateConfig(configName, version2Data)
	suite.NoError(err)
	versions, err := service.ListVersions(configName)
	suite.NoError(err)
	suite.Len(versions.Versions, 2) // Assuming 2 versions exist

	//suite.Fail("ConfigService.ListVersions not implemented yet")
}

// TestConfigNotFoundError tests error handling for non-existent config
func (suite *DatabaseTestSuite) TestConfigNotFoundError() {
	// This will fail until error handling is implemented
	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)
	_, err = service.GetLatestConfig("non-existent")
	suite.Error(err)
	suite.Contains(err.Error(), "CONFIG_NOT_FOUND")

	//suite.Fail("Error handling not implemented yet")
}

// TestVersionNotFoundError tests error handling for non-existent version
func (suite *DatabaseTestSuite) TestVersionNotFoundError() {
	// This will fail until error handling is implemented
	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	configName := "test-config"
	service := services.NewConfigService(store, validationService)
	version1Data := `{"max_limit": 1000, "enabled": true}`
	_, _ = service.CreateConfig(version1Data, configName)
	_, err = service.GetConfigVersion(configName, 999)
	suite.Error(err)
	suite.Contains(err.Error(), "VERSION_NOT_FOUND")

	//suite.Fail("Version error handling not implemented yet")
}

// TestDatabasePerformance tests that operations meet performance requirements
func (suite *DatabaseTestSuite) TestDatabasePerformance() {
	// Performance target: <100ms per operation
	start := time.Now()

	// This will fail until ConfigService is implemented
	configName := "perf-test-config"
	jsonData := `{"max_limit": 1000, "enabled": true}`

	store := storage.NewSQLiteStore(suite.db)
	validationService, err := services.NewValidationService()
	suite.Require().NoError(err)

	service := services.NewConfigService(store, validationService)
	_, err = service.CreateConfig(configName, jsonData)
	suite.NoError(err)

	elapsed := time.Since(start)

	// Performance assertion - should complete in <100ms
	suite.Less(elapsed, 100*time.Millisecond, "Database operation exceeded performance target")

	// Temporary failure for RED phase
	//suite.Fail("ConfigService performance test not ready - no implementation")
}

// TestInTransaction runs the test suite
func TestDatabaseIntegration(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
