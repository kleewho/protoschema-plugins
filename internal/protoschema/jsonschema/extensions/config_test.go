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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_LoadConfig_NonExistentFile(t *testing.T) {
	loader := NewConfigLoader("non-existent-file.yaml")
	config, err := loader.LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "1.0", config.Version)
	assert.Empty(t, config.SchemaOverrides)
	assert.Empty(t, config.OptionProcessors)
}

func TestConfigLoader_LoadConfig_ValidConfig(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides:
  "test.Message":
    schema:
      type: "object"
      properties:
        field1:
          type: "string"
    references: []
  "test.AnotherMessage":
    schema:
      type: "string"
    references: ["test.Message"]
option_processors:
  - name: "required_processor"
    priority: 100
    match:
      option: "validator.field"
      value: true
    schema:
      x-required: true
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	config, err := loader.LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "1.0", config.Version)
	assert.Len(t, config.SchemaOverrides, 2)
	assert.Len(t, config.OptionProcessors, 1)

	// Check schema override
	messageOverride := config.SchemaOverrides["test.Message"]
	assert.Equal(t, "object", messageOverride.Schema["type"])
	assert.Empty(t, messageOverride.References)

	anotherOverride := config.SchemaOverrides["test.AnotherMessage"]
	assert.Equal(t, "string", anotherOverride.Schema["type"])
	assert.Equal(t, []string{"test.Message"}, anotherOverride.References)

	// Check option processor
	processor := config.OptionProcessors[0]
	assert.Equal(t, "required_processor", processor.Name)
	assert.Equal(t, 100, processor.Priority)
	assert.Equal(t, "validator.field", processor.Match.Option)
	assert.Equal(t, true, processor.Match.Value)
	assert.Equal(t, true, processor.Schema["x-required"])
}

func TestConfigLoader_LoadConfig_InvalidYAML(t *testing.T) {
	invalidYAML := `
version: "1.0"
schema_overrides:
  invalid: yaml: syntax: error
`

	tmpFile := createTempYAMLFile(t, invalidYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML config")
}

func TestConfigLoader_LoadConfig_UnsupportedVersion(t *testing.T) {
	configYAML := `version: "2.0"
schema_overrides: {}
option_processors: []
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config version: 2.0")
}

func TestConfigLoader_LoadConfig_EmptySchema(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides:
  "test.Message":
    schema: {}
    references: []
option_processors: []
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has empty schema")
}

func TestConfigLoader_LoadConfig_EmptyOptionPath(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides: {}
option_processors:
  - name: "test_processor"
    priority: 100
    match:
      option: ""
      value: true
    schema:
      x-test: true
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has empty option path")
}

func TestConfigLoader_LoadConfig_EmptyProcessorName(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides: {}
option_processors:
  - name: ""
    priority: 100
    match:
      option: "validator.field"
      value: true
    schema:
      x-test: true
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has empty name")
}

func TestConfigLoader_LoadConfig_EmptyProcessorSchema(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides: {}
option_processors:
  - name: "test_processor"
    priority: 100
    match:
      option: "validator.field"
      value: true
    schema: {}
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	_, err := loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has empty schema")
}

func TestConfigLoader_LoadConfig_MissingVersion(t *testing.T) {
	configYAML := `schema_overrides: {}
option_processors: []
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	config, err := loader.LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "1.0", config.Version) // Should default to 1.0
}

func TestConfigLoader_LoadConfig_ComplexSchema(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides:
  "example.v1.Money":
    schema:
      type: "object"
      properties:
        amount:
          type: "string"
          pattern: "^-?\\d+\\.\\d{2}$"
        currency:
          type: "string"
          enum: ["USD", "EUR", "GBP"]
      required: ["amount", "currency"]
      additionalProperties: false
    references: []
  "example.v1.Price":
    schema:
      allOf:
        - $ref: "@example.v1.Money"
        - type: "object"
          properties:
            tax_included:
              type: "boolean"
    references: ["example.v1.Money"]
option_processors:
  - name: "required_fields"
    priority: 100
    match:
      option: "validator.field"
      value: true
    schema:
      x-required: true
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer os.Remove(tmpFile)

	loader := NewConfigLoader(tmpFile)
	config, err := loader.LoadConfig()

	require.NoError(t, err)
	assert.Equal(t, "1.0", config.Version)
	assert.Len(t, config.SchemaOverrides, 2)
	assert.Len(t, config.OptionProcessors, 1)

	// Check Money schema
	moneySchema := config.SchemaOverrides["example.v1.Money"]
	assert.Equal(t, "object", moneySchema.Schema["type"])
	properties := moneySchema.Schema["properties"].(map[string]any)
	assert.NotNil(t, properties["amount"])
	assert.NotNil(t, properties["currency"])

	// Check Price schema with reference
	priceSchema := config.SchemaOverrides["example.v1.Price"]
	allOf := priceSchema.Schema["allOf"].([]any)
	assert.Len(t, allOf, 2)
	firstItem := allOf[0].(map[string]any)
	assert.Equal(t, "@example.v1.Money", firstItem["$ref"])
	assert.Equal(t, []string{"example.v1.Money"}, priceSchema.References)
}

// Helper function to create temporary YAML files for testing
func createTempYAMLFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "config-test-*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func TestNewConfigLoader_DefaultPath(t *testing.T) {
	loader := NewConfigLoader("")
	assert.Equal(t, DefaultConfigFile, loader.configPath)

	loader = NewConfigLoader("custom.yaml")
	assert.Equal(t, "custom.yaml", loader.configPath)
}

func TestConfigLoader_ReadError(t *testing.T) {
	// Create a file with no read permissions
	tmpDir, err := os.MkdirTemp("", "config-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "no-read.yaml")
	err = os.WriteFile(tmpFile, []byte("test"), 0000) // No permissions
	require.NoError(t, err)

	loader := NewConfigLoader(tmpFile)
	_, err = loader.LoadConfig()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}