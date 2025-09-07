package handlers

import (
	"net/http"
	"strconv"

	"config-manager/src/models"
	"config-manager/src/services"
	"config-manager/src/storage"

	"github.com/labstack/echo/v4"
)

// ConfigHandler handles HTTP requests for configuration management
type ConfigHandler struct {
	configService *services.ConfigService
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(configService *services.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// CreateConfig handles POST /api/v1/configs
//
//	@Summary		Create a new configuration
//	@Description	Validates and creates a new configuration with version 1. The request must include a name and JSON data matching the schema.
//	@Tags			configurations
//	@Accept			json
//	@Produce		json
//	@Param			body	body		models.CreateConfigRequest	true	"Configuration data"
//	@Success		201		{object}	models.SuccessResponse	"Created"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		409		{object}	models.ErrorResponse
//	@Failure		422		{object}	models.ErrorResponse
//	@Router			/api/v1/configs [post]
//
//	@Example request
//	{
//	  "name": "feature-toggle",
//	  "data": {"max_limit": 100, "enabled": true}
//	}
//	@Example response 201
//	{
//	  "success": true,
//	  "message": "Configuration created successfully",
//	  "data": {
//	    "name": "feature-toggle",
//	    "version": 1,
//	    "created_at": "2025-09-07T12:00:00Z"
//	  }
//	}
func (ch *ConfigHandler) CreateConfig(c echo.Context) error {
	var req models.CreateConfigRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST_FORMAT",
				Message: "Request body must be valid JSON",
				Details: map[string]string{"parse_error": err.Error()},
			},
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "MISSING_REQUIRED_FIELD",
				Message: "Missing required field: name",
				Details: map[string][]string{
					"required_fields": {"name", "data"},
				},
			},
		})
	}

	// Validate configuration name pattern
	if !isValidConfigName(req.Name) {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_CONFIG_NAME",
				Message: "Configuration name contains invalid characters",
				Details: map[string]string{
					"provided_name":   req.Name,
					"allowed_pattern": "^[a-zA-Z0-9_-]+$",
				},
			},
		})
	}

	// Create configuration
	config, err := ch.configService.CreateConfig(req.Name, string(req.Data))
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusCreated, models.SuccessResponse{
		Success: true,
		Message: "Configuration created successfully",
		Data: models.ConfigurationCreated{
			Name:      config.Name,
			Version:   config.CurrentVersion,
			CreatedAt: config.CreatedAt,
		},
	})
}

// UpdateConfig handles PUT /api/v1/configs/{name}
//
//	@Summary		Update an existing configuration
//	@Description	Updates the configuration data and increments the version number.
//	@Tags			configurations
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string	true	"Configuration name"
//	@Param			body	body		models.UpdateConfigRequest	true	"Updated configuration data"
//	@Success		200		{object}	models.SuccessResponse	"OK"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Failure		422		{object}	models.ErrorResponse
//	@Router			/api/v1/configs/{name} [put]
//
//	@Example request
//	{
//	  "data": {"max_limit": 200, "enabled": false}
//	}
//	@Example response 200
//	{
//	  "success": true,
//	  "message": "Configuration updated successfully",
//	  "data": {
//	    "name": "feature-toggle",
//	    "version": 2,
//	    "updated_at": "2025-09-07T12:05:00Z"
//	  }
//	}
func (ch *ConfigHandler) UpdateConfig(c echo.Context) error {
	name := c.Param("name")

	var req models.UpdateConfigRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST_FORMAT",
				Message: "Request body must be valid JSON",
				Details: map[string]string{"parse_error": err.Error()},
			},
		})
	}

	// Update configuration
	config, err := ch.configService.UpdateConfig(name, string(req.Data))
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Configuration updated successfully",
		Data: models.ConfigurationUpdated{
			Name:      config.Name,
			Version:   config.CurrentVersion,
			UpdatedAt: config.UpdatedAt,
		},
	})
}

// RollbackConfig handles POST /api/v1/configs/{name}/rollback
//
//	@Summary		Rollback configuration to a previous version
//	@Description	Reverts the configuration to the specified version and increments the current version.
//	@Tags			configurations
//	@Accept			json
//	@Produce		json
//	@Param			name	path		string	true	"Configuration name"
//	@Param			body	body		models.RollbackConfigRequest	true	"Target version to rollback to"
//	@Success		200		{object}	models.SuccessResponse	"OK"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Router			/api/v1/configs/{name}/rollback [post]
//
//	@Example request
//	{
//	  "target_version": 1
//	}
//	@Example response 200
//	{
//	  "success": true,
//	  "message": "Configuration rolled back successfully",
//	  "data": {
//	    "name": "feature-toggle",
//	    "new_version": 3,
//	    "target_version": 1,
//	    "rolled_back_at": "2025-09-07T12:10:00Z"
//	  }
//	}
func (ch *ConfigHandler) RollbackConfig(c echo.Context) error {
	name := c.Param("name")

	var req models.RollbackConfigRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_REQUEST_FORMAT",
				Message: "Request body must be valid JSON",
				Details: map[string]string{"parse_error": err.Error()},
			},
		})
	}

	if req.TargetVersion < 1 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_VERSION_NUMBER",
				Message: "Version number must be positive integer",
				Details: map[string]int{
					"provided_version": req.TargetVersion,
					"minimum_version":  1,
				},
			},
		})
	}

	// Rollback configuration
	config, err := ch.configService.RollbackConfig(name, req.TargetVersion)
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Message: "Configuration rolled back successfully",
		Data: models.ConfigurationRollback{
			Name:          config.Name,
			NewVersion:    config.CurrentVersion,
			TargetVersion: req.TargetVersion,
			RolledBackAt:  config.UpdatedAt,
		},
	})
}

// GetLatestConfig handles GET /api/v1/configs/{name}
//
//	@Summary		Get the latest version of a configuration
//	@Description	Returns the latest configuration data for the given name.
//	@Tags			configurations
//	@Produce		json
//	@Param			name	path		string	true	"Configuration name"
//	@Success		200		{object}	models.SuccessResponse	"OK"
//	@Failure		404		{object}	models.ErrorResponse
//	@Router			/api/v1/configs/{name} [get]
//
//	@Example response 200
//	{
//	  "success": true,
//	  "data": {
//	    "name": "feature-toggle",
//	    "version": 3,
//	    "data": {"max_limit": 200, "enabled": false},
//	    "updated_at": "2025-09-07T12:10:00Z"
//	  }
//	}
func (ch *ConfigHandler) GetLatestConfig(c echo.Context) error {
	name := c.Param("name")

	configData, err := ch.configService.GetLatestConfig(name)
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    configData,
	})
}

// GetConfigVersion handles GET /api/v1/configs/{name}/versions/{version}
//
//	@Summary		Get a specific version of a configuration
//	@Description	Returns the configuration data for the specified version.
//	@Tags			configurations
//	@Produce		json
//	@Param			name	path		string	true	"Configuration name"
//	@Param			version	path		int		true	"Version number"
//	@Success		200		{object}	models.SuccessResponse	"OK"
//	@Failure		400		{object}	models.ErrorResponse
//	@Failure		404		{object}	models.ErrorResponse
//	@Router			/api/v1/configs/{name}/versions/{version} [get]
//
//	@Example response 200
//	{
//	  "success": true,
//	  "data": {
//	    "name": "feature-toggle",
//	    "version": 1,
//	    "data": {"max_limit": 100, "enabled": true},
//	    "created_at": "2025-09-07T12:00:00Z"
//	  }
//	}
func (ch *ConfigHandler) GetConfigVersion(c echo.Context) error {
	name := c.Param("name")
	versionStr := c.Param("version")

	version, err := strconv.Atoi(versionStr)
	if err != nil || version < 1 {
		return c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INVALID_VERSION_NUMBER",
				Message: "Version number must be positive integer",
			},
		})
	}

	configData, err := ch.configService.GetConfigVersion(name, version)
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    configData,
	})
}

// ListVersions handles GET /api/v1/configs/{name}/versions
//
//	@Summary		List all versions of a configuration
//	@Description	Returns a list of all version numbers and their creation timestamps for the specified configuration name.
//	@Tags			configurations
//	@Produce		json
//	@Param			name	path		string	true	"Configuration name"
//	@Success		200		{object}	models.SuccessResponse	"OK"
//	@Failure		404		{object}	models.ErrorResponse
//	@Router			/api/v1/configs/{name}/versions [get]
//
//	@Example response 200
//	{
//	  "success": true,
//	  "data": [
//	    {"version": 1, "created_at": "2025-09-07T12:00:00Z"},
//	    {"version": 2, "created_at": "2025-09-07T12:05:00Z"},
//	    {"version": 3, "created_at": "2025-09-07T12:10:00Z"}
//	  ]
//	}
func (ch *ConfigHandler) ListVersions(c echo.Context) error {
	name := c.Param("name")

	versionList, err := ch.configService.ListVersions(name)
	if err != nil {
		return ch.handleError(c, err)
	}

	return c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    versionList,
	})
}

// handleError converts service errors to appropriate HTTP responses
func (ch *ConfigHandler) handleError(c echo.Context, err error) error {
	switch {
	case isConfigAlreadyExistsError(err):
		return c.JSON(http.StatusConflict, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "CONFIG_ALREADY_EXISTS",
				Message: err.Error(),
			},
		})
	case isConfigNotFoundError(err):
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "CONFIG_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case isVersionNotFoundError(err):
		return c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "VERSION_NOT_FOUND",
				Message: err.Error(),
			},
		})
	case services.IsSchemaValidationError(err):
		schemaErr := err.(*services.SchemaValidationError)
		return c.JSON(http.StatusUnprocessableEntity, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "SCHEMA_VALIDATION_FAILED",
				Message: schemaErr.Message,
				Details: map[string][]services.ValidationError{
					"validation_errors": schemaErr.Errors,
				},
			},
		})
	default:
		return c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.ErrorDetail{
				Code:    "INTERNAL_SERVER_ERROR",
				Message: "An unexpected error occurred",
			},
		})
	}
}

// Helper functions for error type checking
func isConfigAlreadyExistsError(err error) bool {
	_, ok := err.(*storage.ConfigAlreadyExistsError)
	return ok
}

func isConfigNotFoundError(err error) bool {
	_, ok := err.(*storage.ConfigNotFoundError)
	return ok
}

func isVersionNotFoundError(err error) bool {
	_, ok := err.(*storage.VersionNotFoundError)
	return ok
}

// isValidConfigName validates configuration name pattern
func isValidConfigName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}

	for _, char := range name {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' && char != '-' {
			return false
		}
	}

	return true
}
