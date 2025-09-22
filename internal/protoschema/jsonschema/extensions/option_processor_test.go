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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestFieldOptionExtractor_ExtractOptions(t *testing.T) {
	extractor := NewFieldOptionExtractor()

	// Create a simple field descriptor without validation rules
	fieldDesc := createMockFieldDescriptor(t, "test_field", descriptorpb.FieldDescriptorProto_TYPE_STRING)

	options, err := extractor.ExtractOptions(fieldDesc)
	require.NoError(t, err)
	assert.NotNil(t, options)
	assert.Empty(t, options) // No validation rules should result in empty options
}

func TestOptionMatch_Matches(t *testing.T) {
	tests := []struct {
		name         string
		match        OptionMatch
		optionPath   string
		optionValue  interface{}
		shouldMatch  bool
	}{
		{
			name: "exact boolean match true",
			match: OptionMatch{
				Option: "validator.field.required",
				Value:  true,
			},
			optionPath:  "validator.field.required",
			optionValue: true,
			shouldMatch: true,
		},
		{
			name: "exact boolean match false",
			match: OptionMatch{
				Option: "validator.field.required",
				Value:  true,
			},
			optionPath:  "validator.field.required",
			optionValue: false,
			shouldMatch: false,
		},
		{
			name: "wildcard match any value",
			match: OptionMatch{
				Option: "validator.field.pattern",
				Value:  "*",
			},
			optionPath:  "validator.field.pattern",
			optionValue: "^[a-z]+$",
			shouldMatch: true,
		},
		{
			name: "exact string match",
			match: OptionMatch{
				Option: "validator.field.pattern",
				Value:  "^[a-z]+$",
			},
			optionPath:  "validator.field.pattern",
			optionValue: "^[a-z]+$",
			shouldMatch: true,
		},
		{
			name: "string mismatch",
			match: OptionMatch{
				Option: "validator.field.pattern",
				Value:  "^[a-z]+$",
			},
			optionPath:  "validator.field.pattern",
			optionValue: "^[0-9]+$",
			shouldMatch: false,
		},
		{
			name: "numeric match int32",
			match: OptionMatch{
				Option: "validator.field.min",
				Value:  10,
			},
			optionPath:  "validator.field.min",
			optionValue: int32(10),
			shouldMatch: true,
		},
		{
			name: "numeric match with type coercion",
			match: OptionMatch{
				Option: "validator.field.min",
				Value:  "10",
			},
			optionPath:  "validator.field.min",
			optionValue: 10,
			shouldMatch: true,
		},
		{
			name: "option path mismatch",
			match: OptionMatch{
				Option: "validator.field.required",
				Value:  true,
			},
			optionPath:  "validator.field.pattern",
			optionValue: true,
			shouldMatch: false,
		},
		{
			name: "float match",
			match: OptionMatch{
				Option: "validator.field.min",
				Value:  10.5,
			},
			optionPath:  "validator.field.min",
			optionValue: float32(10.5),
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.match.Matches(tt.optionPath, tt.optionValue)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

func TestOptionMatch_toFloat64(t *testing.T) {
	match := OptionMatch{}

	tests := []struct {
		name     string
		input    interface{}
		expected *float64
	}{
		{
			name:     "int",
			input:    42,
			expected: func() *float64 { f := 42.0; return &f }(),
		},
		{
			name:     "int32",
			input:    int32(42),
			expected: func() *float64 { f := 42.0; return &f }(),
		},
		{
			name:     "int64",
			input:    int64(42),
			expected: func() *float64 { f := 42.0; return &f }(),
		},
		{
			name:     "uint32",
			input:    uint32(42),
			expected: func() *float64 { f := 42.0; return &f }(),
		},
		{
			name:     "float32",
			input:    float32(42.5),
			expected: func() *float64 { f := 42.5; return &f }(),
		},
		{
			name:     "float64",
			input:    42.5,
			expected: func() *float64 { f := 42.5; return &f }(),
		},
		{
			name:     "string numeric",
			input:    "42.5",
			expected: func() *float64 { f := 42.5; return &f }(),
		},
		{
			name:     "string non-numeric",
			input:    "not-a-number",
			expected: nil,
		},
		{
			name:     "invalid type",
			input:    map[string]string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := match.toFloat64(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestProcessorMatcher_FindMatchingProcessors(t *testing.T) {
	matcher := NewProcessorMatcher()

	// Create a simple field descriptor
	fieldDesc := createMockFieldDescriptor(t, "test_field", descriptorpb.FieldDescriptorProto_TYPE_STRING)

	processors := []OptionProcessor{
		{
			Name:     "required_processor",
			Priority: 100,
			Match: OptionMatch{
				Option: "validator.field.required",
				Value:  true,
			},
			Schema: map[string]any{"x-required": true},
		},
		{
			Name:     "pattern_processor",
			Priority: 90,
			Match: OptionMatch{
				Option: "validator.field.pattern",
				Value:  "*",
			},
			Schema: map[string]any{"x-has-pattern": true},
		},
		{
			Name:     "non_matching_processor",
			Priority: 80,
			Match: OptionMatch{
				Option: "validator.field.email",
				Value:  true,
			},
			Schema: map[string]any{"x-email": true},
		},
	}

	// Since we're using a mock field without actual validation rules,
	// no processors should match
	matching, err := matcher.FindMatchingProcessors(fieldDesc, processors)
	require.NoError(t, err)
	assert.Empty(t, matching)
}

func TestSchemaProcessor_ApplyProcessors(t *testing.T) {
	processor := NewSchemaProcessor()

	fieldSchema := map[string]interface{}{
		"type": "string",
	}

	processors := []OptionProcessor{
		{
			Name:     "required_processor",
			Priority: 100,
			Schema: map[string]any{
				"x-required": true,
				"minLength":  1,
			},
		},
		{
			Name:     "pattern_processor",
			Priority: 90,
			Schema: map[string]any{
				"pattern": "^[a-z]+$",
			},
		},
	}

	fieldOptions := map[string]interface{}{
		"validator.field.required": true,
		"validator.field.pattern":  "^[a-z]+$",
	}

	err := processor.ApplyProcessors(fieldSchema, processors, fieldOptions)
	require.NoError(t, err)

	// Check that all processor schemas were applied
	assert.Equal(t, "string", fieldSchema["type"])
	assert.Equal(t, true, fieldSchema["x-required"])
	assert.Equal(t, 1, fieldSchema["minLength"])
	assert.Equal(t, "^[a-z]+$", fieldSchema["pattern"])
}

func TestSchemaProcessor_ProcessTemplateValue(t *testing.T) {
	processor := NewSchemaProcessor()

	fieldOptions := map[string]interface{}{
		"validator.field.pattern":    "^[a-z]+$",
		"validator.field.min_length": 5,
		"validator.field.required":   true,
	}

	tests := []struct {
		name          string
		input         interface{}
		expected      interface{}
		shouldError   bool
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "template pattern substitution",
			input:    "${option.pattern}",
			expected: "^[a-z]+$",
		},
		{
			name:     "template min_length substitution",
			input:    "${option.min_length}",
			expected: int64(5),
		},
		{
			name:     "template boolean substitution",
			input:    "${option.required}",
			expected: true,
		},
		{
			name:     "partial template",
			input:    "Pattern: ${option.pattern}",
			expected: "Pattern: ^[a-z]+$",
		},
		{
			name:     "numeric value",
			input:    42,
			expected: 42,
		},
		{
			name: "nested object",
			input: map[string]interface{}{
				"pattern": "${option.pattern}",
				"minLength": "${option.min_length}",
			},
			expected: map[string]interface{}{
				"pattern":   "^[a-z]+$",
				"minLength": int64(5),
			},
		},
		{
			name:     "array",
			input:    []interface{}{"${option.pattern}", "static"},
			expected: []interface{}{"^[a-z]+$", "static"},
		},
		{
			name:        "unknown template variable",
			input:       "${option.unknown}",
			shouldError: true,
		},
		{
			name:        "unterminated template",
			input:       "${option.pattern",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.processTemplateValue(tt.input, fieldOptions)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaProcessor_SubstituteTemplate(t *testing.T) {
	processor := NewSchemaProcessor()

	fieldOptions := map[string]interface{}{
		"validator.field.pattern":    "^[a-z]+$",
		"validator.field.min_length": 5,
		"validator.field.max_length": 100,
		"validator.field.required":   true,
	}

	tests := []struct {
		name        string
		template    string
		expected    interface{}
		shouldError bool
	}{
		{
			name:     "pattern substitution",
			template: "${option.pattern}",
			expected: "^[a-z]+$",
		},
		{
			name:     "numeric substitution returns number",
			template: "${option.min_length}",
			expected: int64(5),
		},
		{
			name:     "boolean substitution",
			template: "${option.required}",
			expected: true,
		},
		{
			name:     "partial substitution",
			template: "Length between ${option.min_length} and ${option.max_length}",
			expected: "Length between 5 and 100",
		},
		{
			name:     "no template",
			template: "plain string",
			expected: "plain string",
		},
		{
			name:        "unknown option",
			template:    "${option.unknown}",
			shouldError: true,
		},
		{
			name:        "invalid template syntax",
			template:    "${invalid",
			shouldError: true,
		},
		{
			name:        "unsupported variable type",
			template:    "${unsupported.var}",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.substituteTemplate(tt.template, fieldOptions)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Helper function to create a mock field descriptor
func createMockFieldDescriptor(t *testing.T, name string, fieldType descriptorpb.FieldDescriptorProto_Type) protoreflect.FieldDescriptor {
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    stringPtr("test/mock.proto"),
		Package: stringPtr("test"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: stringPtr("TestMessage"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   stringPtr(name),
						Number: int32Ptr(1),
						Type:   &fieldType,
					},
				},
			},
		},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	require.NoError(t, err)

	message := file.Messages().ByName("TestMessage")
	return message.Fields().ByName(protoreflect.Name(name))
}

// Helper functions for proto descriptor creation
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}