package contract

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"config-manager/src/handlers"
	"config-manager/src/services"
	"config-manager/src/storage"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

// setupTestServer creates a test server with real database
func setupTestServer(t *testing.T) (*echo.Echo, func()) {
	// Create temporary test database
	testDB := "./test_contract.db"

	// Remove existing test database
	if err := os.Remove(testDB); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove test database: %v", err)
	}

	db, err := sql.Open("sqlite3", testDB)
	if err != nil {
		t.Fatal("Failed to open test database:", err)
	}

	// Create tables using the exact schema from migrations
	schema := `
	CREATE TABLE configurations (
		name TEXT PRIMARY KEY,
		current_version INTEGER NOT NULL,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
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
	if err != nil {
		t.Fatal("Failed to create schema:", err)
	}

	// Initialize services
	validationService, err := services.NewValidationService()
	if err != nil {
		t.Fatal("Failed to create validation service:", err)
	}

	sqliteStore := storage.NewSQLiteStore(db)
	configService := services.NewConfigService(sqliteStore, validationService)
	configHandler := handlers.NewConfigHandler(configService)

	// Create Echo instance and register routes
	e := echo.New()
	api := e.Group("/api/v1")

	api.POST("/configs", configHandler.CreateConfig)
	api.PUT("/configs/:name", configHandler.UpdateConfig)
	api.POST("/configs/:name/rollback", configHandler.RollbackConfig)
	api.GET("/configs/:name", configHandler.GetLatestConfig)
	api.GET("/configs/:name/versions/:version", configHandler.GetConfigVersion)
	api.GET("/configs/:name/versions", configHandler.ListVersions)

	// Return cleanup function
	cleanup := func() {
		_ = db.Close()
		_ = os.Remove(testDB)
	}

	return e, cleanup
}

// TestCreateConfigEndpoint tests POST /api/v1/configs
func TestCreateConfigEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Test case: Valid configuration creation
	reqBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 201 Created
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Response should contain success and proper data structure
	response := rec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"name":"app-settings"`)
	assert.Contains(t, response, `"version":1`)
	assert.Contains(t, response, `"message":"Configuration created successfully"`)
}

// TestCreateConfigMissingNameError tests POST /api/v1/configs with missing name
func TestCreateConfigMissingNameError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Test case: Missing name field
	reqBody := `{
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Response should contain proper error format
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"MISSING_REQUIRED_FIELD"`)
	assert.Contains(t, response, `"Missing required field: name"`)
}

// TestCreateConfigInvalidNameError tests POST /api/v1/configs with invalid name
func TestCreateConfigInvalidNameError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Test case: Invalid name with special characters
	reqBody := `{
		"name": "app@settings!",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Response should contain proper error format
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"INVALID_CONFIG_NAME"`)
	assert.Contains(t, response, `"Configuration name contains invalid characters"`)
}

// TestUpdateConfigEndpoint tests PUT /api/v1/configs/{name}
func TestUpdateConfigEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// First create a configuration
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)

	// Verify creation was successful
	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Now update the configuration - request body should only contain data field
	updateBody := `{
		"data": {
			"max_limit": 2000,
			"enabled": false
		}
	}`

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/configs/app-settings", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()

	e.ServeHTTP(updateRec, updateReq)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Response should show version 2 and success message
	response := updateRec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"version":2`)
	assert.Contains(t, response, `"message":"Configuration updated successfully"`)
	assert.Contains(t, response, `"name":"app-settings"`)
}

// TestUpdateConfigNotFoundError tests PUT /api/v1/configs/{name} with non-existent config
func TestUpdateConfigNotFoundError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Try to update non-existent configuration
	updateBody := `{
		"data": {
			"max_limit": 2000,
			"enabled": false
		}
	}`

	req := httptest.NewRequest(http.MethodPut, "/api/v1/configs/non-existent", strings.NewReader(updateBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 404 Not Found
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Response should contain proper error format
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"CONFIG_NOT_FOUND"`)
}

// TestRollbackConfigEndpoint tests POST /api/v1/configs/{name}/rollback
func TestRollbackConfigEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Create initial configuration (version 1)
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Update configuration (version 2)
	updateBody := `{
		"data": {
			"max_limit": 2000,
			"enabled": false
		}
	}`

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/configs/app-settings", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()
	e.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Now rollback to version 1
	rollbackBody := `{
		"target_version": 1
	}`

	rollbackReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs/app-settings/rollback", strings.NewReader(rollbackBody))
	rollbackReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rollbackRec := httptest.NewRecorder()

	e.ServeHTTP(rollbackRec, rollbackReq)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, rollbackRec.Code)

	// Response should show new version 3 with target version 1
	response := rollbackRec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"new_version":3`)
	assert.Contains(t, response, `"target_version":1`)
	assert.Contains(t, response, `"message":"Configuration rolled back successfully"`)
}

// TestRollbackConfigInvalidVersionError tests POST /api/v1/configs/{name}/rollback with invalid version
func TestRollbackConfigInvalidVersionError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a configuration first
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Try to rollback to invalid version (0)
	rollbackBody := `{
		"target_version": 0
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/configs/app-settings/rollback", strings.NewReader(rollbackBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Response should contain proper error format
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"INVALID_VERSION_NUMBER"`)
	assert.Contains(t, response, `"Version number must be positive integer"`)
}

// TestGetLatestConfigEndpoint tests GET /api/v1/configs/{name}
func TestGetLatestConfigEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// First create a configuration
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)

	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Now get the latest configuration
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/configs/app-settings", nil)
	getRec := httptest.NewRecorder()

	e.ServeHTTP(getRec, getReq)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, getRec.Code)

	// Response should contain the configuration data
	response := getRec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"name":"app-settings"`)
	assert.Contains(t, response, `"max_limit":1000`)
	assert.Contains(t, response, `"enabled":true`)
}

// TestGetConfigVersionEndpoint tests GET /api/v1/configs/{name}/versions/{version}
func TestGetConfigVersionEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Create initial configuration (version 1)
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Update configuration (version 2)
	updateBody := `{
		"data": {
			"max_limit": 2000,
			"enabled": false
		}
	}`

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/configs/app-settings", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()
	e.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// Get version 1
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/configs/app-settings/versions/1", nil)
	getRec := httptest.NewRecorder()

	e.ServeHTTP(getRec, getReq)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, getRec.Code)

	// Response should contain version 1 data
	response := getRec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"version":1`)
	assert.Contains(t, response, `"max_limit":1000`)
	assert.Contains(t, response, `"enabled":true`)
}

// TestListVersionsEndpoint tests GET /api/v1/configs/{name}/versions
func TestListVersionsEndpoint(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Create initial configuration (version 1)
	createBody := `{
		"name": "app-settings",
		"data": {
			"max_limit": 1000,
			"enabled": true
		}
	}`

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(createBody))
	createReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	createRec := httptest.NewRecorder()
	e.ServeHTTP(createRec, createReq)
	assert.Equal(t, http.StatusCreated, createRec.Code)

	// Update configuration (version 2)
	updateBody := `{
		"data": {
			"max_limit": 2000,
			"enabled": false
		}
	}`

	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/configs/app-settings", strings.NewReader(updateBody))
	updateReq.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	updateRec := httptest.NewRecorder()
	e.ServeHTTP(updateRec, updateReq)
	assert.Equal(t, http.StatusOK, updateRec.Code)

	// List all versions
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/configs/app-settings/versions", nil)
	listRec := httptest.NewRecorder()

	e.ServeHTTP(listRec, listReq)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, listRec.Code)

	// Response should contain both versions
	response := listRec.Body.String()
	assert.Contains(t, response, `"success":true`)
	assert.Contains(t, response, `"version":1`)
	assert.Contains(t, response, `"version":2`)
	assert.Contains(t, response, `"created_at"`)
}

// TestConfigNotFoundError tests 404 error scenario
func TestConfigNotFoundError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/configs/non-existent", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 404 Not Found
	assert.Equal(t, http.StatusNotFound, rec.Code)

	// Response should contain proper error format
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"CONFIG_NOT_FOUND"`)
}

// TestSchemaValidationError tests 422 error scenario
func TestSchemaValidationError(t *testing.T) {
	e, cleanup := setupTestServer(t)
	defer cleanup()

	// Invalid request body - wrong type for max_limit
	reqBody := `{
		"name": "test-config",
		"data": {
			"max_limit": "invalid-type",
			"enabled": true
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/configs", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 422 Unprocessable Entity
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// Response should contain schema validation error
	response := rec.Body.String()
	assert.Contains(t, response, `"success":false`)
	assert.Contains(t, response, `"SCHEMA_VALIDATION_FAILED"`)
}
