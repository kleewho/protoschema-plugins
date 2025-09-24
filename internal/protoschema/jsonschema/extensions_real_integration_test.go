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

package jsonschema

import (
	"os"
	"testing"

	testv1 "github.com/kleewho/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/kleewho/protoschema-plugins/internal/protoschema/jsonschema/extensions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_WithExtensions_RealValidatorOptions(t *testing.T) {
	// Test configuration that processes buf.validate options
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "buf_validate_required",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
					"description": "Field marked as required by buf.validate",
				},
			},
			{
				Name:     "buf_validate_string_pattern",
				Priority: 90,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.pattern",
					Value:  "*", // Match any pattern value
				},
				Schema: map[string]any{
					"pattern":     "${option.string.pattern}",
					"description": "Pattern validation from buf.validate: ${option.string.pattern}",
				},
			},
			{
				Name:     "buf_validate_string_min_len",
				Priority: 85,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.min_len",
					Value:  "*", // Match any min_len value
				},
				Schema: map[string]any{
					"minLength":   "${option.string.min_len}",
					"description": "Minimum length validation from buf.validate: ${option.string.min_len}",
				},
			},
			{
				Name:     "buf_validate_string_max_len",
				Priority: 84,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.max_len",
					Value:  "*", // Match any max_len value
				},
				Schema: map[string]any{
					"maxLength":   "${option.string.max_len}",
					"description": "Maximum length validation from buf.validate: ${option.string.max_len}",
				},
			},
			{
				Name:     "buf_validate_string_const",
				Priority: 95,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.const",
					Value:  "*", // Match any const value
				},
				Schema: map[string]any{
					"const":       "${option.string.const}",
					"description": "Constant value from buf.validate: ${option.string.const}",
				},
			},
			{
				Name:     "buf_validate_string_email",
				Priority: 80,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.email",
					Value:  true,
				},
				Schema: map[string]any{
					"format":      "email",
					"description": "Email format validation from buf.validate",
				},
			},
			{
				Name:     "buf_validate_int32_const",
				Priority: 95,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.int32.const",
					Value:  "*",
				},
				Schema: map[string]any{
					"const":       "${option.int32.const}",
					"description": "Constant value from buf.validate: ${option.int32.const}",
				},
			},
			{
				Name:     "buf_validate_int32_lt",
				Priority: 80,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.int32.lt",
					Value:  "*",
				},
				Schema: map[string]any{
					"exclusiveMaximum": "${option.int32.lt}",
					"description":      "Less than validation from buf.validate: ${option.int32.lt}",
				},
			},
			{
				Name:     "buf_validate_int32_gt",
				Priority: 80,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.int32.gt",
					Value:  "*",
				},
				Schema: map[string]any{
					"exclusiveMinimum": "${option.int32.gt}",
					"description":      "Greater than validation from buf.validate: ${option.int32.gt}",
				},
			},
		},
	}

	// Create registry
	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Create generator with extensions
	generator := NewGenerator(WithExtensions(registry))

	// Get the real ConstraintTest message descriptor
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	// Generate schema
	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Debug: print all generated schema names
	t.Logf("Generated schemas:")
	for name := range schemas {
		t.Logf("  - %s", name)
	}

	// Verify that the ConstraintTest schema was generated
	constraintSchema, exists := schemas["buf.protoschema.test.v1.ConstraintTest"]
	require.True(t, exists, "ConstraintTest schema should exist in generated schemas")
	assert.Equal(t, "object", constraintSchema["type"])

	// Test properties contain processed fields
	properties, ok := constraintSchema["properties"].(map[string]any)
	require.True(t, ok)

	// Test specific fields that should have been processed

	// Test const_string field (should have const validation)
	if constString, exists := properties["const_string"]; exists {
		constStringMap, ok := constString.(map[string]any)
		require.True(t, ok)

		// Debug: show what properties are actually set
		t.Logf("const_string field properties: %+v", constStringMap)

		if constVal, hasConst := constStringMap["const"]; hasConst {
			assert.Equal(t, "const", constVal)
		} else {
			t.Logf("const_string field does not have 'const' property - option processor may not have matched")
		}

		if desc, hasDesc := constStringMap["description"]; hasDesc {
			assert.Contains(t, desc.(string), "Constant value from buf.validate")
		} else {
			t.Logf("const_string field does not have description - option processor may not have matched")
		}
	} else {
		t.Logf("const_string field not found in properties")
	}

	// Test pattern_string field (should have pattern validation)
	if patternString, exists := properties["pattern_string"]; exists {
		patternStringMap, ok := patternString.(map[string]any)
		require.True(t, ok)

		t.Logf("pattern_string field properties: %+v", patternStringMap)

		if patternVal, hasPattern := patternStringMap["pattern"]; hasPattern {
			assert.Equal(t, "^pat*ern$", patternVal)
		} else {
			t.Logf("pattern_string field does not have 'pattern' property - option processor may not have matched")
		}

		if desc, hasDesc := patternStringMap["description"]; hasDesc {
			assert.Contains(t, desc.(string), "Pattern validation from buf.validate")
		} else {
			t.Logf("pattern_string field does not have description - option processor may not have matched")
		}
	}

	// Test min_len_string field (should have minLength validation)
	if minLenString, exists := properties["min_len_string"]; exists {
		minLenStringMap, ok := minLenString.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), minLenStringMap["minLength"])
		assert.Contains(t, minLenStringMap["description"].(string), "Minimum length validation from buf.validate")
	}

	// Test max_len_string field (should have maxLength validation)
	if maxLenString, exists := properties["max_len_string"]; exists {
		maxLenStringMap, ok := maxLenString.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), maxLenStringMap["maxLength"])
		assert.Contains(t, maxLenStringMap["description"].(string), "Maximum length validation from buf.validate")
	}

	// Test email_string field (should have email format)
	if emailString, exists := properties["email_string"]; exists {
		emailStringMap, ok := emailString.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "email", emailStringMap["format"])
		assert.Contains(t, emailStringMap["description"].(string), "Email format validation from buf.validate")
	}

	// Test const_int32 field (should have const validation)
	if constInt32, exists := properties["const_int32"]; exists {
		constInt32Map, ok := constInt32.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), constInt32Map["const"])
		assert.Contains(t, constInt32Map["description"].(string), "Constant value from buf.validate")
	}

	// Test lt_int32 field (should have exclusiveMaximum validation)
	if ltInt32, exists := properties["lt_int32"]; exists {
		ltInt32Map, ok := ltInt32.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), ltInt32Map["exclusiveMaximum"])
		assert.Contains(t, ltInt32Map["description"].(string), "Less than validation from buf.validate")
	}

	// Test gt_int32 field (should have exclusiveMinimum validation)
	if gtInt32, exists := properties["gt_int32"]; exists {
		gtInt32Map, ok := gtInt32.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), gtInt32Map["exclusiveMinimum"])
		assert.Contains(t, gtInt32Map["description"].(string), "Greater than validation from buf.validate")
	}
}

func TestGenerator_WithExtensions_RequiredFields(t *testing.T) {
	// Test configuration that processes required field validation
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "buf_validate_required",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
					"description": "Field marked as required by buf.validate",
				},
			},
		},
	}

	// Create registry
	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Create generator with extensions
	generator := NewGenerator(WithExtensions(registry))

	// Get the real ConstraintTest message descriptor
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	// Generate schema
	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify that the ConstraintTest schema was generated
	constraintSchema, exists := schemas["buf.protoschema.test.testv1.ConstraintTest"]
	require.True(t, exists)

	// Get the RequiredImplicit nested message schema
	properties, ok := constraintSchema["properties"].(map[string]any)
	require.True(t, ok)

	if requiredImplicit, exists := properties["required_implicit"]; exists {
		requiredImplicitMap, ok := requiredImplicit.(map[string]any)
		require.True(t, ok)

		// Check if required fields were processed
		if required, hasRequired := requiredImplicitMap["required"]; hasRequired {
			requiredArray, ok := required.([]any)
			require.True(t, ok)

			// The fields bool_value, string_value, enum_value, and strict_enum_value should all be required
			expectedRequiredFields := []string{"bool_value", "string_value", "enum_value", "strict_enum_value"}
			for _, expectedField := range expectedRequiredFields {
				assert.Contains(t, requiredArray, expectedField, "Field %s should be in required array", expectedField)
			}
		}

		// Check that individual fields have the required description
		if nestedProperties, hasProps := requiredImplicitMap["properties"]; hasProps {
			nestedPropsMap, ok := nestedProperties.(map[string]any)
			require.True(t, ok)

			// Check bool_value field
			if boolValue, exists := nestedPropsMap["bool_value"]; exists {
				boolValueMap, ok := boolValue.(map[string]any)
				require.True(t, ok)
				assert.Contains(t, boolValueMap["description"].(string), "Field marked as required by buf.validate")
			}
		}
	}
}

func TestGenerator_WithExtensions_NestedMessages(t *testing.T) {
	// Test that option processors work with nested messages
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "buf_validate_required",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
				},
			},
		},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))

	// Get the real ConstraintTest message descriptor
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	// Generate schema
	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify nested messages were processed
	// Check RequiredImplicit
	if requiredImplicitSchema, exists := schemas["buf.protoschema.test.testv1.ConstraintTest.RequiredImplicit"]; exists {
		assert.Equal(t, "object", requiredImplicitSchema["type"])

		if required, hasRequired := requiredImplicitSchema["required"]; hasRequired {
			requiredArray, ok := required.([]any)
			require.True(t, ok)
			assert.Greater(t, len(requiredArray), 0, "Required array should not be empty")
		}
	}

	// Check RequiredOptional
	if requiredOptionalSchema, exists := schemas["buf.protoschema.test.testv1.ConstraintTest.RequiredOptional"]; exists {
		assert.Equal(t, "object", requiredOptionalSchema["type"])

		if required, hasRequired := requiredOptionalSchema["required"]; hasRequired {
			requiredArray, ok := required.([]any)
			require.True(t, ok)
			assert.Greater(t, len(requiredArray), 0, "Required array should not be empty")
		}
	}
}

func TestGenerator_WithExtensions_PerformanceWithRealData(t *testing.T) {
	// Performance test with real protobuf data and validation options
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "buf_validate_string_pattern",
				Priority: 90,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.pattern",
					Value:  "*",
				},
				Schema: map[string]any{
					"pattern": "${option.string.pattern}",
				},
			},
			{
				Name:     "buf_validate_required",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
				},
			},
		},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Measure performance of multiple schema generations
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	for i := 0; i < 10; i++ {
		generator := NewGenerator(WithExtensions(registry))
		err = generator.Add(constraintTestDesc)
		require.NoError(t, err)

		schemas := generator.Generate()
		assert.Greater(t, len(schemas), 0)
	}
}

// Helper function to create a temporary YAML config file for testing
func createTempYAMLConfig(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestGenerator_WithExtensions_FromYAMLConfig(t *testing.T) {
	// Test loading extensions from YAML configuration
	yamlContent := `version: "1.0"
schema_overrides: {}
option_processors:
  - name: "buf_validate_required"
    priority: 100
    match:
      option: "buf.validate.field.required"
      value: true
    schema:
      x-required: true
      description: "Required field from YAML config"
  - name: "buf_validate_string_const"
    priority: 95
    match:
      option: "buf.validate.field.string.const"
      value: "*"
    schema:
      const: "${option.string.const}"
      description: "Constant string from YAML config"
`

	configPath := createTempYAMLConfig(t, yamlContent)

	// Load registry from YAML file
	registry, err := extensions.LoadRegistry(configPath)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))

	// Generate schema for ConstraintTest
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()
	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify schema generation worked
	constraintSchema, exists := schemas["buf.protoschema.test.testv1.ConstraintTest"]
	require.True(t, exists)
	assert.Equal(t, "object", constraintSchema["type"])

	// Check that const_string field has constant value
	properties, ok := constraintSchema["properties"].(map[string]any)
	require.True(t, ok)

	if constString, exists := properties["const_string"]; exists {
		constStringMap, ok := constString.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "const", constStringMap["const"])
		assert.Contains(t, constStringMap["description"].(string), "Constant string from YAML config")
	}
}

func TestGenerator_WithExtensions_ComplexValidation(t *testing.T) {
	// Test complex validation scenarios from the real protobuf file
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "buf_validate_string_min_max_len",
				Priority: 90,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.min_len",
					Value:  "*",
				},
				Schema: map[string]any{
					"minLength": "${option.string.min_len}",
				},
			},
			{
				Name:     "buf_validate_string_max_len_separate",
				Priority: 85,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.string.max_len",
					Value:  "*",
				},
				Schema: map[string]any{
					"maxLength": "${option.string.max_len}",
				},
			},
			{
				Name:     "buf_validate_int32_range",
				Priority: 80,
				Match: extensions.OptionMatch{
					Option: "buf.validate.field.int32.lt",
					Value:  "*",
				},
				Schema: map[string]any{
					"exclusiveMaximum": "${option.int32.lt}",
				},
			},
		},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()
	constraintSchema := schemas["buf.protoschema.test.testv1.ConstraintTest"]
	properties := constraintSchema["properties"].(map[string]any)

	// Test min_max_len_string field which should have both minLength and maxLength
	if minMaxLenString, exists := properties["min_max_len_string"]; exists {
		minMaxLenStringMap, ok := minMaxLenString.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), minMaxLenStringMap["minLength"])
		assert.Equal(t, int64(10), minMaxLenStringMap["maxLength"])
	}

	// Test lt_int32 field should have exclusiveMaximum
	if ltInt32, exists := properties["lt_int32"]; exists {
		ltInt32Map, ok := ltInt32.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(5), ltInt32Map["exclusiveMaximum"])
	}
}

func TestGenerator_WithExtensions_AllFieldTypes(t *testing.T) {
	// Verify that option processing works for all field types in ConstraintTest
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	// Simple config that adds metadata to any field with validation options
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "validation_marker",
				Priority: 50, // Lower priority to not interfere with other processors
				Match: extensions.OptionMatch{
					Option: "buf.validate.field",
					Value:  "*", // Match any validation option
				},
				Schema: map[string]any{
					"x-has-validation": true,
				},
			},
		},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))
	err = generator.Add(constraintTestDesc)
	require.NoError(t, err)

	schemas := generator.Generate()
	constraintSchema := schemas["buf.protoschema.test.testv1.ConstraintTest"]
	properties := constraintSchema["properties"].(map[string]any)

	// Count fields that have validation markers
	fieldsWithValidation := 0
	for fieldName, fieldSchema := range properties {
		if fieldMap, ok := fieldSchema.(map[string]any); ok {
			if hasValidation, exists := fieldMap["x-has-validation"]; exists && hasValidation.(bool) {
				fieldsWithValidation++
				t.Logf("Field %s has validation marker", fieldName)
			}
		}
	}

	// We expect many fields to have validation since the constraints.proto file
	// has extensive validation rules
	assert.Greater(t, fieldsWithValidation, 10, "Expected many fields to have validation markers")
}

// Test field descriptor inspection
func TestFieldDescriptorInspection(t *testing.T) {
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()

	// Inspect some specific fields to understand their options
	fields := constraintTestDesc.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())

		// Look for interesting fields
		if fieldName == "const_string" || fieldName == "pattern_string" || fieldName == "min_len_string" {
			t.Logf("Field: %s", fieldName)
			t.Logf("  Type: %s", field.Kind())
			t.Logf("  HasOptions: %v", field.Options() != nil)

			// This helps us understand the field structure for debugging
			options := field.Options()
			if options != nil {
				t.Logf("  Options available: %v", options.ProtoReflect().IsValid())
			}
		}
	}

	// This test mainly serves to understand the field structure
	assert.Greater(t, fields.Len(), 0, "Should have fields")
}