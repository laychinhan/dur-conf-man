package models

import "encoding/json"

// CreateConfigRequest is the request body for creating a configuration
type CreateConfigRequest struct {
	Name string          `json:"name" example:"feature_toggle"`
	Data json.RawMessage `json:"data" swaggertype:"object" example:"{\"max_limit\": 100, \"enabled\": true}"`
}

// UpdateConfigRequest is the request body for updating a configuration
type UpdateConfigRequest struct {
	Data json.RawMessage `json:"data" example: {"max_limit": 100, "enabled": true}`
}

// RollbackConfigRequest is the request body for rolling back a configuration
type RollbackConfigRequest struct {
	TargetVersion int `json:"target_version" example:"1"`
}
