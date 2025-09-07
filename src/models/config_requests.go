package models

import "encoding/json"

// CreateConfigRequest is the request body for creating a configuration
// swagger:model
type CreateConfigRequest struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// UpdateConfigRequest is the request body for updating a configuration
// swagger:model
type UpdateConfigRequest struct {
	Data json.RawMessage `json:"data"`
}

// RollbackConfigRequest is the request body for rolling back a configuration
// swagger:model
type RollbackConfigRequest struct {
	TargetVersion int `json:"target_version"`
}
