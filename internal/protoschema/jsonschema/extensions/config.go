// Copyright 2024-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extensions

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = "jsonschema.extensions.yaml"

// ConfigLoader handles loading and parsing YAML configuration
type ConfigLoader struct {
	configPath string
	logger     *Logger
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(configPath string) *ConfigLoader {
	if configPath == "" {
		configPath = DefaultConfigFile
	}
	return &ConfigLoader{
		configPath: configPath,
		logger:     GetGlobalLogger().With("config"),
	}
}

// LoadConfig loads configuration from YAML file
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	startTime := time.Now()
	cl.logger.LogConfigLoad(cl.configPath, "start", map[string]any{"timestamp": startTime})

	// Check if file exists
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		cl.logger.Info("Configuration file not found, using empty config: %s", cl.configPath)

		config := &Config{
			Version:         "1.0",
			SchemaOverrides: make(map[string]SchemaOverride),
			OptionProcessors: []OptionProcessor{},
		}

		cl.logger.LogConfigLoad(cl.configPath, "empty_config", map[string]any{
			"duration": time.Since(startTime),
		})

		return config, nil
	}

	cl.logger.Debug("Reading configuration file: %s", cl.configPath)

	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		cl.logger.LogError("read_config", err, map[string]any{
			"config_path": cl.configPath,
			"duration":    time.Since(startTime),
		})

		context := map[string]string{
			"config_file": cl.configPath,
			"operation":   "read file",
		}
		suggestions := []string{
			"Check if the file exists and is readable",
			"Verify the file path is correct",
			"Ensure proper file permissions",
		}
		errorWithContext := WrapError(err, ErrorTypeConfiguration,
			fmt.Sprintf("Cannot read configuration file '%s'", cl.configPath), context)
		errorWithContext.Suggestions = suggestions
		errorWithContext.Location = cl.configPath
		return nil, errorWithContext
	}

	cl.logger.LogConfigLoad(cl.configPath, "file_read", map[string]any{
		"file_size": len(data),
		"duration":  time.Since(startTime),
	})

	cl.logger.Debug("Parsing YAML configuration")

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		cl.logger.LogError("parse_yaml", err, map[string]any{
			"config_path": cl.configPath,
			"file_size":   len(data),
			"duration":    time.Since(startTime),
		})

		context := map[string]string{
			"config_file": cl.configPath,
			"operation":   "parse YAML",
		}
		suggestions := []string{
			"Check YAML syntax (indentation, colons, quotes)",
			"Use a YAML validator to check syntax",
			"Refer to documentation for proper YAML structure",
		}
		errorWithContext := WrapError(err, ErrorTypeConfiguration,
			"Invalid YAML syntax in configuration file", context)
		errorWithContext.Suggestions = suggestions
		errorWithContext.Location = cl.configPath
		return nil, errorWithContext
	}

	// Initialize empty maps/slices if not present
	if config.SchemaOverrides == nil {
		config.SchemaOverrides = make(map[string]SchemaOverride)
	}
	if config.OptionProcessors == nil {
		config.OptionProcessors = []OptionProcessor{}
	}

	cl.logger.LogConfigLoad(cl.configPath, "yaml_parsed", map[string]any{
		"schema_overrides":  len(config.SchemaOverrides),
		"option_processors": len(config.OptionProcessors),
		"version":           config.Version,
		"duration":          time.Since(startTime),
	})

	// Enhanced validation
	validationStart := time.Now()
	if err := cl.validateConfig(&config); err != nil {
		cl.logger.LogError("validate_config", err, map[string]any{
			"config_path": cl.configPath,
			"duration":    time.Since(startTime),
		})
		return nil, err
	}

	cl.logger.LogConfigLoad(cl.configPath, "validation_complete", map[string]any{
		"validation_duration": time.Since(validationStart),
		"total_duration":      time.Since(startTime),
	})

	cl.logger.Info("Configuration loaded successfully: %s (overrides: %d, processors: %d)",
		cl.configPath, len(config.SchemaOverrides), len(config.OptionProcessors))

	return &config, nil
}

// validateConfig performs enhanced validation of loaded configuration
func (cl *ConfigLoader) validateConfig(config *Config) error {
	errorBuilder := NewValidationErrorBuilder(cl.configPath)

	// Version check
	if config.Version == "" {
		config.Version = "1.0" // Default version
	} else if config.Version != "1.0" {
		errorBuilder.AddError(fmt.Sprintf("unsupported config version: %s", config.Version))
		errorBuilder.AddContext("version", config.Version)
		errorBuilder.AddContext("expected_version", "1.0")
		errorBuilder.AddSuggestion("Change version to '1.0' in your configuration file")
		errorBuilder.AddSuggestion("Check documentation for supported versions")
	}

	// Validate schema overrides
	for fqn, override := range config.SchemaOverrides {
		if len(override.Schema) == 0 {
			errorBuilder.AddError(fmt.Sprintf("schema override for '%s' has empty schema", fqn))
			errorBuilder.AddContext("empty_schema_fqn", fqn)
			errorBuilder.AddSuggestion(fmt.Sprintf("Add schema content for '%s' override", fqn))
			errorBuilder.AddSuggestion("Example: schema: { type: 'object', properties: {...} }")
		} else {
			// Validate basic schema structure
			cl.validateSchemaStructure(fqn, override.Schema, errorBuilder)
		}

		// Validate references
		for _, ref := range override.References {
			if ref == "" {
				errorBuilder.AddError(fmt.Sprintf("schema override '%s' has empty reference", fqn))
				errorBuilder.AddSuggestion("Remove empty references or provide valid FQN")
			} else if ref == fqn {
				errorBuilder.AddError(fmt.Sprintf("schema override '%s' references itself", fqn))
				errorBuilder.AddSuggestion("Remove self-reference or reference a different schema")
			}
		}
	}

	// Validate option processors
	for i, processor := range config.OptionProcessors {
		processorContext := fmt.Sprintf("option processor #%d", i+1)
		if processor.Name != "" {
			processorContext = fmt.Sprintf("option processor '%s'", processor.Name)
		}

		if processor.Match.Option == "" {
			errorBuilder.AddError(fmt.Sprintf("%s has empty option path", processorContext))
			errorBuilder.AddContext("processor_index", fmt.Sprintf("%d", i))
			errorBuilder.AddSuggestion("Add option path (e.g., 'validator.field.required')")
			errorBuilder.AddSuggestion("Common options: validator.field.required, validator.field.pattern, validator.field.min_length")
		}

		if len(processor.Schema) == 0 {
			errorBuilder.AddError(fmt.Sprintf("%s has empty schema", processorContext))
			errorBuilder.AddContext("processor_index", fmt.Sprintf("%d", i))
			errorBuilder.AddSuggestion("Add schema content to apply to matching fields")
			errorBuilder.AddSuggestion("Example: schema: { x-required: true }")
		}

		if processor.Name == "" {
			errorBuilder.AddError(fmt.Sprintf("option processor at index %d has empty name", i))
			errorBuilder.AddContext("processor_index", fmt.Sprintf("%d", i))
			errorBuilder.AddSuggestion("Add descriptive name for the processor")
			errorBuilder.AddSuggestion("Example: name: 'required_fields'")
		}

		// Validate priority
		if processor.Priority < 0 {
			errorBuilder.AddError(fmt.Sprintf("%s has negative priority: %d", processorContext, processor.Priority))
			errorBuilder.AddContext("priority", fmt.Sprintf("%d", processor.Priority))
			errorBuilder.AddSuggestion("Use positive priority values (higher numbers = higher priority)")
		}

		// Validate match criteria
		if processor.Match.Value == nil {
			errorBuilder.AddError(fmt.Sprintf("%s has no match value", processorContext))
			errorBuilder.AddSuggestion("Add match value (e.g., value: true or value: '*')")
		}
	}

	if errorBuilder.HasErrors() {
		return errorBuilder.Build()
	}

	return nil
}

// validateSchemaStructure performs basic validation of schema structure
func (cl *ConfigLoader) validateSchemaStructure(fqn string, schema map[string]any, errorBuilder *ValidationErrorBuilder) {
	// Check for common schema issues
	if typeVal, hasType := schema["type"]; hasType {
		if typeStr, ok := typeVal.(string); ok {
			switch typeStr {
			case "object":
				// For object types, suggest having properties
				if _, hasProperties := schema["properties"]; !hasProperties {
					if _, hasAllOf := schema["allOf"]; !hasAllOf {
						if _, hasRef := schema["$ref"]; !hasRef {
							errorBuilder.AddSuggestion(fmt.Sprintf("Object schema '%s' should have 'properties', 'allOf', or '$ref'", fqn))
						}
					}
				}
			case "array":
				// For array types, suggest having items
				if _, hasItems := schema["items"]; !hasItems {
					errorBuilder.AddSuggestion(fmt.Sprintf("Array schema '%s' should have 'items' definition", fqn))
				}
			}
		}
	}

	// Validate template references in values
	cl.validateTemplateReferences(schema, fqn, errorBuilder)
}

// validateTemplateReferences checks for template syntax issues
func (cl *ConfigLoader) validateTemplateReferences(value any, context string, errorBuilder *ValidationErrorBuilder) {
	switch v := value.(type) {
	case string:
		// Check for template syntax
		if strings.Contains(v, "${") && !strings.Contains(v, "}") {
			errorBuilder.AddError(fmt.Sprintf("unclosed template variable in %s: '%s'", context, v))
			errorBuilder.AddSuggestion("Ensure template variables are properly closed: ${option.name}")
		}
	case map[string]any:
		for key, val := range v {
			cl.validateTemplateReferences(val, fmt.Sprintf("%s.%s", context, key), errorBuilder)
		}
	case []any:
		for i, val := range v {
			cl.validateTemplateReferences(val, fmt.Sprintf("%s[%d]", context, i), errorBuilder)
		}
	}
}