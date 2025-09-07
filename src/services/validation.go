package services

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// ValidationService handles JSON schema validation for configuration data
type ValidationService struct {
	schema *gojsonschema.Schema
}

// ConfigDataSchema Hardcoded JSON schema that all configuration data must conform to
const ConfigDataSchema = `{
  "type": "object",
  "properties": {
    "max_limit": {"type": "integer", "minimum": 0},
    "enabled": {"type": "boolean"}
  },
  "required": ["max_limit", "enabled"],
  "additionalProperties": false
}`

// NewValidationService creates a new validation service with the hardcoded schema
func NewValidationService() (*ValidationService, error) {
	schemaLoader := gojsonschema.NewStringLoader(ConfigDataSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON schema: %w", err)
	}

	return &ValidationService{
		schema: schema,
	}, nil
}

// ValidateConfigData validates the provided JSON data against the hardcoded schema
func (vs *ValidationService) ValidateConfigData(jsonData string) error {
	documentLoader := gojsonschema.NewStringLoader(jsonData)
	result, err := vs.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("schema validation error: %w", err)
	}

	if !result.Valid() {
		var validationErrors []ValidationError
		for _, desc := range result.Errors() {
			validationErrors = append(validationErrors, ValidationError{
				Field: desc.Field(),
				Error: desc.Description(),
			})
		}

		return &SchemaValidationError{
			Message: "Configuration data does not match required schema",
			Errors:  validationErrors,
		}
	}

	return nil
}

// ValidateAndParseConfigData validates and parses JSON data into ConfigData struct
func (vs *ValidationService) ValidateAndParseConfigData(jsonData string) (map[string]interface{}, error) {
	// First validate against schema
	if err := vs.ValidateConfigData(jsonData); err != nil {
		return nil, err
	}

	// Parse JSON into generic map for flexibility
	var configData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &configData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON data: %w", err)
	}

	return configData, nil
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// SchemaValidationError represents schema validation failure with details
type SchemaValidationError struct {
	Message string            `json:"message"`
	Errors  []ValidationError `json:"validation_errors"`
}

func (e *SchemaValidationError) Error() string {
	return e.Message
}

// IsSchemaValidationError checks if an error is a schema validation error
func IsSchemaValidationError(err error) bool {
	_, ok := err.(*SchemaValidationError)
	return ok
}
