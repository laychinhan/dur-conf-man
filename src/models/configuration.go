package models

import (
	"time"
)

// Configuration represents a named configuration with versioning
type Configuration struct {
	Name           string    `json:"name" db:"name"`
	CurrentVersion int       `json:"current_version" db:"current_version"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Version represents a specific version of configuration data
type Version struct {
	ID                int       `json:"id" db:"id"`
	ConfigurationName string    `json:"configuration_name" db:"configuration_name"`
	VersionNumber     int       `json:"version_number" db:"version_number"`
	JsonData          string    `json:"json_data" db:"json_data"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// ConfigData represents the validated configuration data that must conform to the hardcoded JSON schema
type ConfigData struct {
	MaxLimit int  `json:"max_limit"`
	Enabled  bool `json:"enabled"`
}

// SuccessResponse represents the standard success response format
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

// ErrorDetail contains detailed error information
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// ConfigurationCreated represents the response data for configuration creation
type ConfigurationCreated struct {
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// ConfigurationUpdated represents the response data for configuration updates
type ConfigurationUpdated struct {
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConfigurationRollback represents the response data for configuration rollbacks
type ConfigurationRollback struct {
	Name          string    `json:"name"`
	NewVersion    int       `json:"new_version"`
	TargetVersion int       `json:"target_version"`
	RolledBackAt  time.Time `json:"rolled_back_at"`
}

// ConfigurationData represents the response data for configuration retrieval
type ConfigurationData struct {
	Name       string     `json:"name"`
	Version    int        `json:"version"`
	ConfigData ConfigData `json:"config_data"`
	CreatedAt  time.Time  `json:"created_at"`
}

// VersionList represents the response data for listing versions
type VersionList struct {
	Name           string        `json:"name"`
	CurrentVersion int           `json:"current_version"`
	Versions       []VersionInfo `json:"versions"`
}

// VersionInfo represents version metadata for listing
type VersionInfo struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}
