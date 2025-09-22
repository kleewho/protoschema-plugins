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
	"testing"

	testv1 "github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema/extensions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestGenerator_WithExtensions_DebugOptionExtraction(t *testing.T) {
	// Very simple test to see what options are actually extracted
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "debug_any_option",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "*", // Match any option
					Value:  "*", // Match any value
				},
				Schema: map[string]any{
					"x-debug-processed": true,
					"x-debug-message":   "This field was processed by the extension system",
				},
			},
		},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Test the registry directly first
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()
	fields := constraintTestDesc.Fields()

	// Find the const_string field
	var constStringField protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if string(field.Name()) == "const_string" {
			constStringField = field
			break
		}
	}

	require.NotNil(t, constStringField, "const_string field should exist")

	// Test option extraction directly
	processors, err := registry.GetOptionProcessors(constStringField)
	require.NoError(t, err)

	t.Logf("Found %d processors for const_string field", len(processors))
	for i, processor := range processors {
		t.Logf("  Processor %d: %s", i, processor.Name)
	}

	// Test ProcessField directly
	fieldSchema := map[string]any{"type": "string"}
	result, err := registry.ProcessField(constStringField, fieldSchema)
	require.NoError(t, err)
	require.NotNil(t, result)

	t.Logf("ProcessField result: RequiredField=%v", result.RequiredField)
	t.Logf("Field schema after processing: %+v", fieldSchema)

	// Check if the debug processor was applied
	if debugProcessed, exists := fieldSchema["x-debug-processed"]; exists {
		assert.True(t, debugProcessed.(bool), "Debug processor should have been applied")
		t.Logf("✓ Debug processor successfully applied to const_string field")
	} else {
		t.Logf("✗ Debug processor was not applied - no options found or matched")
	}
}

func TestGenerator_WithExtensions_OptionsInspection(t *testing.T) {
	// Test to inspect what options are actually available on constraint fields
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()
	fields := constraintTestDesc.Fields()

	interestingFields := []string{"const_string", "pattern_string", "min_len_string", "email_string"}

	for _, fieldName := range interestingFields {
		for i := 0; i < fields.Len(); i++ {
			field := fields.Get(i)
			if string(field.Name()) == fieldName {
				t.Logf("Field: %s", fieldName)
				t.Logf("  Type: %s", field.Kind())
				t.Logf("  HasOptions: %v", field.Options() != nil)

				if field.Options() != nil {
					options := field.Options()
					t.Logf("  Options: %+v", options)

					// Try to examine the options using reflection
					optionsReflect := options.ProtoReflect()
					t.Logf("  Options descriptor: %s", optionsReflect.Descriptor().FullName())

					// List all fields in options
					optionsFields := optionsReflect.Descriptor().Fields()
					for j := 0; j < optionsFields.Len(); j++ {
						optionField := optionsFields.Get(j)
						if optionsReflect.Has(optionField) {
							value := optionsReflect.Get(optionField)
							t.Logf("    Option field: %s = %v", optionField.Name(), value)
						}
					}
				}
				break
			}
		}
	}
}