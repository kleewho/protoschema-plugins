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
	"strings"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema/extensions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestGenerator_WithExtensions_SchemaOverride(t *testing.T) {
	// Create test configuration with schema override
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{
			"google.protobuf.Timestamp": {
				Schema: map[string]any{
					"type":        "string",
					"format":      "date-time",
					"pattern":     "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$",
					"description": "Custom timestamp format",
					"examples":    []any{"2023-12-25T10:30:00Z"},
				},
				References: []string{},
			},
		},
		OptionProcessors: []extensions.OptionProcessor{},
	}

	// Create registry
	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Create generator with extensions
	generator := NewGenerator(WithExtensions(registry))

	// Create a mock timestamp message descriptor
	timestampDesc := createMockTimestampDescriptor(t)

	// Generate schema
	err = generator.Add(timestampDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify that the override was applied
	timestampSchema, exists := schemas["google.protobuf.Timestamp"]
	require.True(t, exists)

	assert.Equal(t, "string", timestampSchema["type"])
	assert.Equal(t, "date-time", timestampSchema["format"])
	assert.Equal(t, "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$", timestampSchema["pattern"])
	assert.Equal(t, "Custom timestamp format", timestampSchema["description"])
}

func TestGenerator_WithoutExtensions_DefaultBehavior(t *testing.T) {
	// Create generator without extensions
	generator := NewGenerator()

	// Create a mock timestamp message descriptor
	timestampDesc := createMockTimestampDescriptor(t)

	// Generate schema
	err := generator.Add(timestampDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify that default WKT handler was used (not our override)
	timestampSchema, exists := schemas["google.protobuf.Timestamp"]
	require.True(t, exists)

	// The default WKT handler should generate different schema than our override
	assert.Equal(t, "string", timestampSchema["type"])
	assert.Equal(t, "date-time", timestampSchema["format"])
	// Should not have our custom pattern or description
	assert.NotContains(t, timestampSchema, "pattern")
	assert.NotContains(t, timestampSchema, "examples")
}

func TestGenerator_WithExtensions_MultipleOverrides(t *testing.T) {
	// Create configuration with multiple overrides
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{
			"test.Money": {
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"amount": map[string]any{
							"type":    "string",
							"pattern": "^\\d+\\.\\d{2}$",
						},
						"currency": map[string]any{
							"type": "string",
							"enum": []any{"USD", "EUR", "GBP"},
						},
					},
					"required":             []any{"amount", "currency"},
					"additionalProperties": false,
				},
				References: []string{},
			},
			"test.User": {
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":      "string",
							"minLength": 1,
						},
						"email": map[string]any{
							"type":   "string",
							"format": "email",
						},
					},
					"required": []any{"name", "email"},
				},
				References: []string{},
			},
		},
		OptionProcessors: []extensions.OptionProcessor{},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))

	// Create mock descriptors
	moneyDesc := createMockMessageDescriptor(t, "test.Money")
	userDesc := createMockMessageDescriptor(t, "test.User")

	// Generate schemas
	err = generator.Add(moneyDesc)
	require.NoError(t, err)
	err = generator.Add(userDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify Money override
	moneySchema, exists := schemas["test.Money"]
	require.True(t, exists)
	assert.Equal(t, "object", moneySchema["type"])
	properties := moneySchema["properties"].(map[string]any)
	assert.Contains(t, properties, "amount")
	assert.Contains(t, properties, "currency")

	// Verify User override
	userSchema, exists := schemas["test.User"]
	require.True(t, exists)
	assert.Equal(t, "object", userSchema["type"])
	userProperties := userSchema["properties"].(map[string]any)
	assert.Contains(t, userProperties, "name")
	assert.Contains(t, userProperties, "email")
}

func TestGenerator_WithExtensions_NoOverride(t *testing.T) {
	// Create configuration with no overrides for the message we're generating
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{
			"other.Message": {
				Schema:     map[string]any{"type": "object"},
				References: []string{},
			},
		},
		OptionProcessors: []extensions.OptionProcessor{},
	}

	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	generator := NewGenerator(WithExtensions(registry))

	// Create a message that doesn't have an override
	testDesc := createMockMessageDescriptor(t, "test.NoOverride")

	// Generate schema
	err = generator.Add(testDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Should use default generation (not our override)
	testSchema, exists := schemas["test.NoOverride"]
	require.True(t, exists)

	// Should have default schema structure
	assert.Contains(t, testSchema, "$schema")
	assert.Equal(t, "No Override", testSchema["title"]) // Default title generation (camelCase -> space separated)
}

func TestGenerator_WithExtensions_OptionProcessors(t *testing.T) {
	// Create test configuration with option processors
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "required_fields",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "validator.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-required": true,
				},
			},
			{
				Name:     "pattern_validation",
				Priority: 90,
				Match: extensions.OptionMatch{
					Option: "validator.field.pattern",
					Value:  "*", // Match any pattern value
				},
				Schema: map[string]any{
					"x-custom-pattern": "${option.pattern}",
				},
			},
			{
				Name:     "string_length",
				Priority: 80,
				Match: extensions.OptionMatch{
					Option: "validator.field.min_length",
					Value:  "*", // Match any min_length value
				},
				Schema: map[string]any{
					"x-min-length": "${option.min_length}",
				},
			},
		},
	}

	// Create registry
	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	// Create generator with extensions
	generator := NewGenerator(WithExtensions(registry))

	// Create a mock message descriptor with fields that would have validator options
	// Note: Since we're using mock descriptors without actual validator options,
	// we need to test the infrastructure more than the actual option extraction
	testDesc := createMockMessageDescriptor(t, "test.User")

	// Generate schema
	err = generator.Add(testDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Verify that the message schema was generated
	testSchema, exists := schemas["test.User"]
	require.True(t, exists)
	assert.Equal(t, "object", testSchema["type"])

	// Since our mock descriptors don't have actual validator options,
	// the option processors won't be applied. But we can verify the system
	// integrates correctly by checking that no errors occurred.
	// In a real scenario with actual protobuf files containing validator options,
	// the processors would be applied.
}

func TestGenerator_WithExtensions_PriorityProcessing(t *testing.T) {
	// Test that processors are applied in priority order
	config := &extensions.Config{
		Version: "1.0",
		SchemaOverrides: map[string]extensions.SchemaOverride{},
		OptionProcessors: []extensions.OptionProcessor{
			{
				Name:     "low_priority",
				Priority: 50,
				Match: extensions.OptionMatch{
					Option: "validator.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-priority": "low",
				},
			},
			{
				Name:     "high_priority",
				Priority: 100,
				Match: extensions.OptionMatch{
					Option: "validator.field.required",
					Value:  true,
				},
				Schema: map[string]any{
					"x-priority": "high",
				},
			},
		},
	}

	// Create registry and verify processors are sorted by priority
	registry, err := extensions.NewRegistry(config)
	require.NoError(t, err)

	processorNames := registry.GetProcessorNames()
	require.Len(t, processorNames, 2)
	// High priority should come first
	assert.Equal(t, "high_priority", processorNames[0])
	assert.Equal(t, "low_priority", processorNames[1])

	// Create generator with extensions
	generator := NewGenerator(WithExtensions(registry))

	// Test that the system integrates without errors
	testDesc := createMockMessageDescriptor(t, "test.PriorityTest")
	err = generator.Add(testDesc)
	require.NoError(t, err)

	schemas := generator.Generate()
	testSchema, exists := schemas["test.PriorityTest"]
	require.True(t, exists)
	assert.Equal(t, "object", testSchema["type"])
}

// Helper function to create a mock Timestamp descriptor
func createMockTimestampDescriptor(t *testing.T) protoreflect.MessageDescriptor {
	// Create a minimal descriptor for google.protobuf.Timestamp
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    stringPtr("google/protobuf/timestamp.proto"),
		Package: stringPtr("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: stringPtr("Timestamp"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   stringPtr("seconds"),
						Number: int32Ptr(1),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT64),
					},
					{
						Name:   stringPtr("nanos"),
						Number: int32Ptr(2),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT32),
					},
				},
			},
		},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	require.NoError(t, err)

	return file.Messages().ByName("Timestamp")
}

// Helper function to create a mock message descriptor
func createMockMessageDescriptor(t *testing.T, fullName string) protoreflect.MessageDescriptor {
	// Parse the full name to get package and message name
	pkg := ""
	msgName := fullName
	if lastDot := strings.LastIndex(fullName, "."); lastDot >= 0 {
		pkg = fullName[:lastDot]
		msgName = fullName[lastDot+1:]
	}

	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    stringPtr(pkg + "/test.proto"),
		Package: stringPtr(pkg),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: stringPtr(msgName),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   stringPtr("test_field"),
						Number: int32Ptr(1),
						Type:   typePtr(descriptorpb.FieldDescriptorProto_TYPE_STRING),
					},
				},
			},
		},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	require.NoError(t, err)

	return file.Messages().ByName(protoreflect.Name(msgName))
}

// Helper functions for proto descriptor creation
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}