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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry_Empty(t *testing.T) {
	config := &Config{
		Version:         "1.0",
		SchemaOverrides: make(map[string]SchemaOverride),
		OptionProcessors: []OptionProcessor{},
	}

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	assert.False(t, registry.HasOverrides())
	assert.False(t, registry.HasOptionProcessors())
	assert.Empty(t, registry.GetOverrideNames())
	assert.Empty(t, registry.GetProcessorNames())
}

func TestNewRegistry_WithOverrides(t *testing.T) {
	config := &Config{
		Version: "1.0",
		SchemaOverrides: map[string]SchemaOverride{
			"test.Message": {
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{"type": "string"},
					},
				},
				References: []string{},
			},
			"test.AnotherMessage": {
				Schema: map[string]any{
					"type": "string",
				},
				References: []string{"test.Message"},
			},
		},
		OptionProcessors: []OptionProcessor{},
	}

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	assert.True(t, registry.HasOverrides())
	assert.False(t, registry.HasOptionProcessors())

	names := registry.GetOverrideNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "test.Message")
	assert.Contains(t, names, "test.AnotherMessage")
}

func TestNewRegistry_WithOptionProcessors(t *testing.T) {
	config := &Config{
		Version: "1.0",
		SchemaOverrides: make(map[string]SchemaOverride),
		OptionProcessors: []OptionProcessor{
			{
				Name:     "required_processor",
				Priority: 100,
				Match: OptionMatch{
					Option: "validator.field",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
				},
			},
			{
				Name:     "pattern_processor",
				Priority: 90,
				Match: OptionMatch{
					Option: "validator.pattern",
					Value:  "*",
				},
				Schema: map[string]any{
					"x-has-pattern": true,
				},
			},
		},
	}

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	assert.False(t, registry.HasOverrides())
	assert.True(t, registry.HasOptionProcessors())

	names := registry.GetProcessorNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "required_processor")
	assert.Contains(t, names, "pattern_processor")

	// Check that processors are sorted by priority
	assert.Equal(t, "required_processor", registry.optionProcessors[0].Name)
	assert.Equal(t, "pattern_processor", registry.optionProcessors[1].Name)
}

func TestRegistry_GetSchemaOverride(t *testing.T) {
	config := &Config{
		Version: "1.0",
		SchemaOverrides: map[string]SchemaOverride{
			"test.Message": {
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{
							"type": "string",
							"pattern": "^[a-z]+$",
						},
					},
					"required": []any{"field"},
				},
				References: []string{},
			},
		},
		OptionProcessors: []OptionProcessor{},
	}

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	// Test existing override
	schema := registry.GetSchemaOverride("test.Message")
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	// Verify it's a deep copy (modifying returned schema shouldn't affect original)
	schema["type"] = "modified"

	schema2 := registry.GetSchemaOverride("test.Message")
	assert.Equal(t, "object", schema2["type"]) // Should still be "object"

	// Test non-existing override
	schema = registry.GetSchemaOverride("nonexistent.Message")
	assert.Nil(t, schema)
}

func TestRegistry_MergeSchemas(t *testing.T) {
	target := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"existing": map[string]any{"type": "string"},
		},
		"required": []any{"existing"},
	}

	source := map[string]any{
		"properties": map[string]any{
			"new": map[string]any{"type": "number"},
		},
		"required": []any{"new"},
		"additionalProperties": false,
	}

	err := mergeSchemas(target, source)
	require.NoError(t, err)

	// Check merged properties
	properties := target["properties"].(map[string]any)
	assert.Contains(t, properties, "existing")
	assert.Contains(t, properties, "new")

	// Check merged required fields
	required := target["required"].([]any)
	assert.Contains(t, required, "existing")
	assert.Contains(t, required, "new")

	// Check new property
	assert.Equal(t, false, target["additionalProperties"])
}

func TestRegistry_CopyValue(t *testing.T) {
	original := map[string]any{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nested": map[string]any{
			"inner": "nested_value",
		},
		"array": []any{"item1", "item2"},
	}

	copied := copyValue(original)
	copiedMap := copied.(map[string]any)

	// Verify copy is correct
	assert.Equal(t, original["string"], copiedMap["string"])
	assert.Equal(t, original["number"], copiedMap["number"])
	assert.Equal(t, original["bool"], copiedMap["bool"])

	// Verify it's a deep copy by modifying nested values
	nested := copiedMap["nested"].(map[string]any)
	nested["inner"] = "modified"

	originalNested := original["nested"].(map[string]any)
	assert.Equal(t, "nested_value", originalNested["inner"]) // Should be unchanged
}

func TestNewRegistry_WithReferences(t *testing.T) {
	config := &Config{
		Version: "1.0",
		SchemaOverrides: map[string]SchemaOverride{
			"test.Base": {
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{"type": "string"},
					},
				},
				References: []string{},
			},
			"test.Extended": {
				Schema: map[string]any{
					"allOf": []any{
						map[string]any{"$ref": "@test.Base"},
						map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
							},
						},
					},
				},
				References: []string{"test.Base"},
			},
		},
		OptionProcessors: []OptionProcessor{},
	}

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	// Test that references were resolved
	extendedSchema := registry.GetSchemaOverride("test.Extended")
	require.NotNil(t, extendedSchema)

	allOf := extendedSchema["allOf"].([]any)
	require.Len(t, allOf, 2)

	firstItem := allOf[0].(map[string]any)
	ref := firstItem["$ref"].(string)
	// Reference should have been resolved from "@test.Base" to actual reference
	assert.NotEqual(t, "@test.Base", ref)
	assert.Contains(t, ref, "test.Base")
}

func TestRegistry_String(t *testing.T) {
	// Empty registry
	config := &Config{
		Version:         "1.0",
		SchemaOverrides: make(map[string]SchemaOverride),
		OptionProcessors: []OptionProcessor{},
	}
	registry, _ := NewRegistry(config)
	assert.Equal(t, "Registry: empty", registry.String())

	// Registry with overrides only
	config.SchemaOverrides["test.Message"] = SchemaOverride{
		Schema: map[string]any{"type": "object"},
	}
	registry, _ = NewRegistry(config)
	assert.Contains(t, registry.String(), "Schema Overrides: 1")

	// Registry with processors only
	config.SchemaOverrides = make(map[string]SchemaOverride)
	config.OptionProcessors = []OptionProcessor{
		{Name: "test", Schema: map[string]any{"x-test": true}},
	}
	registry, _ = NewRegistry(config)
	assert.Contains(t, registry.String(), "Option Processors: 1")

	// Registry with both
	config.SchemaOverrides["test.Message"] = SchemaOverride{
		Schema: map[string]any{"type": "object"},
	}
	registry, _ = NewRegistry(config)
	str := registry.String()
	assert.Contains(t, str, "Schema Overrides: 1")
	assert.Contains(t, str, "Option Processors: 1")
}

func TestLoadRegistry(t *testing.T) {
	configYAML := `version: "1.0"
schema_overrides:
  "test.Message":
    schema:
      type: "object"
    references: []
option_processors: []
`

	tmpFile := createTempYAMLFile(t, configYAML)
	defer func() {
		_ = os.Remove(tmpFile)
	}()

	registry, err := LoadRegistry(tmpFile)
	require.NoError(t, err)

	assert.True(t, registry.HasOverrides())
	assert.NotNil(t, registry.GetSchemaOverride("test.Message"))
}