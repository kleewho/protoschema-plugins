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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"gopkg.in/yaml.v3"
)

func TestJSONSchemaGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/jsonschema")
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)
	generator := NewGenerator()
	for _, testDesc := range testDescs {
		err = generator.Add(testDesc)
		require.NoError(t, err)
	}

	schemas := generator.Generate()
	require.NoError(t, err)
	for _, jsonSchema := range schemas {
		// Serialize the JSON
		data, err := json.MarshalIndent(jsonSchema, "", "  ")
		require.NoError(t, err)

		identifier, ok := jsonSchema["$id"].(string)
		require.True(t, ok)
		require.NotEmpty(t, identifier)

		filePath := filepath.Join(dirPath, identifier)
		err = golden.CheckGolden(filePath, string(data)+"\n")
		require.NoError(t, err)
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Foo", nameToTitle("Foo"))
	require.Equal(t, "Foo Bar", nameToTitle("FooBar"))
	require.Equal(t, "foo Bar", nameToTitle("fooBar"))
	require.Equal(t, "Foo Bar Baz", nameToTitle("FooBarBaz"))
	require.Equal(t, "FOO Bar", nameToTitle("FOOBar"))
	require.Equal(t, "U Int64 Value", nameToTitle("UInt64Value"))
	require.Equal(t, "Uint64 Value", nameToTitle("Uint64Value"))
	require.Equal(t, "FOO", nameToTitle("FOO"))
}

func TestConstraints(t *testing.T) {
	t.Parallel()
	schemaPath := filepath.FromSlash("../../testdata/jsonschema/buf.protoschema.test.v1.ConstraintTests.schema.json")
	bundledSchemaPath := filepath.FromSlash("../../testdata/jsonschema/buf.protoschema.test.v1.ConstraintTests.schema.bundle.json")
	testPath := filepath.FromSlash("../../testdata/jsonschema-doc/test.ConstraintTests.yaml")
	expectedPath := filepath.FromSlash("../../testdata/jsonschema-doc/test.ConstraintTests.txt")
	expectedBundledPath := filepath.FromSlash("../../testdata/jsonschema-doc/test.ConstraintTests.bundle.txt")
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(schemaPath)
	require.NoError(t, err)
	bundledSchema, err := compiler.Compile(bundledSchemaPath)
	require.NoError(t, err)

	yamlData, err := os.ReadFile(testPath)
	require.NoError(t, err)
	var jsonData map[string]any
	err = yaml.Unmarshal(yamlData, &jsonData)
	require.NoError(t, err)

	assertValidation(t, schema, jsonData, expectedPath)
	assertValidation(t, bundledSchema, jsonData, expectedBundledPath)
}

func assertValidation(t *testing.T, schema *jsonschema.Schema, jsonData map[string]any, expectedPath string) {
	t.Helper()
	expectedData, err := os.ReadFile(expectedPath)
	require.NoError(t, err)
	err = schema.Validate(jsonData)
	require.Error(t, err)
	errStr := err.Error()
	// Remove the first line of the error message, which contains the path to the schema file.
	if pos := strings.Index(errStr, "\n"); pos != -1 {
		errStr = errStr[pos+1:]
	}
	expectedStr := string(expectedData)
	expectedStr = strings.TrimSpace(expectedStr)
	require.Equal(t, expectedStr, errStr, errStr)
}

func TestParseTypeUrl(t *testing.T) {
	t.Parallel()
	generator := NewGenerator()

	tests := []struct {
		name     string
		typeUrl  string
		expected string
	}{
		{
			name:     "standard googleapis format",
			typeUrl:  "type.googleapis.com/example.v1.Message",
			expected: "example.v1.Message",
		},
		{
			name:     "buf build format",
			typeUrl:  "type.buf.build/example.v1.Message",
			expected: "example.v1.Message",
		},
		{
			name:     "simple FQN",
			typeUrl:  "example.v1.Message",
			expected: "example.v1.Message",
		},
		{
			name:     "path format",
			typeUrl:  "path/to/proto/example.v1.Message",
			expected: "example.v1.Message",
		},
		{
			name:     "deep path format",
			typeUrl:  "very/deep/path/to/proto/files/example.v1.Message",
			expected: "example.v1.Message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.parseTypeUrl(tt.typeUrl)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateConstrainedAnyValidation_NoConstraints(t *testing.T) {
	t.Parallel()
	generator := NewGenerator()
	schema := make(map[string]any)

	// Test with nil rules
	_, err := generator.generateConstrainedAnyValidation(nil, schema)
	require.NoError(t, err)

	// Should fall back to default Any behavior
	assert.Equal(t, jsObject, schema["type"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	typeProperty, ok := properties["@type"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", typeProperty["type"])
}

func TestGenerateConstrainedAnyValidation_EmptyConstraints(t *testing.T) {
	t.Parallel()
	generator := NewGenerator()
	schema := make(map[string]any)

	// Create field rules with nil any rules (simulates no constraints)
	rules := &validate.FieldRules{}

	_, err := generator.generateConstrainedAnyValidation(rules, schema)
	require.NoError(t, err)

	// Should fall back to default Any behavior
	assert.Equal(t, jsObject, schema["type"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	typeProperty, ok := properties["@type"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", typeProperty["type"])
}

func TestCrossFileAnyConstraints(t *testing.T) {
	t.Parallel()

	// Load test descriptors that include cross-file Any constraints
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)

	// Create generator in bundle mode to test cross-file reference consistency
	generator := NewGenerator(WithBundle())

	for _, testDesc := range testDescs {
		err = generator.Add(testDesc)
		require.NoError(t, err)
	}

	schemas := generator.Generate()

	// Find the CrossFileEventEnvelope schema
	var crossFileSchema map[string]any
	for _, schema := range schemas {
		if id, ok := schema["$id"].(string); ok && strings.Contains(id, "CrossFileEventEnvelope") {
			crossFileSchema = schema
			break
		}
	}

	require.NotNil(t, crossFileSchema, "CrossFileEventEnvelope schema not found")

	// Check that $defs contains all referenced message types
	defs, ok := crossFileSchema["$defs"].(map[string]any)
	require.True(t, ok, "Expected $defs section in bundle schema")

	// Verify that external message schemas are included
	// Note: The actual keys use .schema.json suffix, not .jsonschema.strict.json
	expectedDefs := []string{
		"buf.protoschema.test.v1.LocalMessage.schema.json",
		"buf.protoschema.test.v1.ExternalMessageD.schema.json",
		"buf.protoschema.test.v1.ExternalMessageE.schema.json",
	}

	for _, expectedDef := range expectedDefs {
		_, exists := defs[expectedDef]
		assert.True(t, exists, "Expected referenced schema %s to be included in $defs but it was missing", expectedDef)
	}

	// Check that the payload field has proper oneOf structure referencing all types
	ref, ok := crossFileSchema["$ref"].(string)
	require.True(t, ok)

	mainSchemaKey := strings.TrimPrefix(ref, "#/$defs/")
	mainSchema, ok := defs[mainSchemaKey].(map[string]any)
	require.True(t, ok, "Main schema not found in $defs: %s", mainSchemaKey)

	properties, ok := mainSchema["properties"].(map[string]any)
	require.True(t, ok)

	payloadProperty, ok := properties["payload"].(map[string]any)
	require.True(t, ok, "payload property not found in main schema")

	oneOfRaw, exists := payloadProperty["oneOf"]
	require.True(t, exists, "oneOf field not found in payload property")

	var oneOf []any
	var oneOfOk bool

	// Try different possible slice types
	if oneOfSlice, ok2 := oneOfRaw.([]any); ok2 {
		oneOf = oneOfSlice
		oneOfOk = true
	} else if oneOfSlice, ok2 := oneOfRaw.([]interface{}); ok2 {
		oneOf = oneOfSlice
		oneOfOk = true
	} else if oneOfSlice, ok2 := oneOfRaw.([]map[string]interface{}); ok2 {
		// Convert []map[string]interface{} to []any
		oneOf = make([]any, len(oneOfSlice))
		for i, item := range oneOfSlice {
			oneOf[i] = item
		}
		oneOfOk = true
	}
	require.True(t, oneOfOk, "Expected payload field to have oneOf structure, got type %T", oneOfRaw)

	// THIS IS THE MAIN ISSUE: References in oneOf should match the keys in $defs
	// The bug is that references don't have .json suffix but the keys in $defs do
	expectedRefs := []string{
		"#/$defs/buf.protoschema.test.v1.LocalMessage.schema.json",
		"#/$defs/buf.protoschema.test.v1.ExternalMessageD.schema.json",
		"#/$defs/buf.protoschema.test.v1.ExternalMessageE.schema.json",
	}

	assert.Len(t, oneOf, len(expectedRefs), "Expected oneOf to have %d options", len(expectedRefs))

	for i, refSchema := range oneOf {
		refMap, ok := refSchema.(map[string]any)
		require.True(t, ok)

		actualRef, ok := refMap["$ref"].(string)
		require.True(t, ok)

		// Check if this reference actually exists in $defs
		refKey := strings.TrimPrefix(actualRef, "#/$defs/")
		_, refExists := defs[refKey]
		assert.True(t, refExists, "Reference %s in oneOf[%d] points to non-existent definition in $defs", actualRef, i)
	}
}

func TestCrossFileAnyConstraints_Realistic(t *testing.T) {
	t.Parallel()

	// Load all test descriptors to simulate having all FileDescriptors available
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)

	// Build a map of all descriptors by FullName for easy lookup
	allDescs := make(map[protoreflect.FullName]protoreflect.MessageDescriptor)
	var allFiles []protoreflect.FileDescriptor
	fileSet := make(map[string]protoreflect.FileDescriptor)

	for _, desc := range testDescs {
		allDescs[desc.FullName()] = desc
		// Collect unique file descriptors
		if _, exists := fileSet[desc.ParentFile().Path()]; !exists {
			fileSet[desc.ParentFile().Path()] = desc.ParentFile()
			allFiles = append(allFiles, desc.ParentFile())
		}
	}

	// Create generator in bundle mode
	generator := NewGenerator(WithBundle())
	generator.SetFileDescriptors(allFiles) // Provide all file descriptors for cross-file resolution

	// Simulate real-world scenario: only add the main CrossFileEventEnvelope message
	// (like how protoc would process only one file at a time)
	var crossFileDesc protoreflect.MessageDescriptor
	for _, desc := range testDescs {
		if desc.FullName() == "buf.protoschema.test.v1.CrossFileEventEnvelope" {
			crossFileDesc = desc
			break
		}
	}
	require.NotNil(t, crossFileDesc, "CrossFileEventEnvelope descriptor not found")

	// Add only the main message (not the external references)
	err = generator.Add(crossFileDesc)
	require.NoError(t, err)

	schemas := generator.Generate()

	// Find the generated bundle
	var crossFileSchema map[string]any
	for _, schema := range schemas {
		if id, ok := schema["$id"].(string); ok && strings.Contains(id, "CrossFileEventEnvelope") {
			crossFileSchema = schema
			break
		}
	}

	require.NotNil(t, crossFileSchema, "CrossFileEventEnvelope schema not found")

	// Check that $defs contains all referenced message types (including external ones)
	defs, ok := crossFileSchema["$defs"].(map[string]any)
	require.True(t, ok, "Expected $defs section in bundle schema")

	// Verify that external message schemas are included
	// These messages are from different proto files but should be discovered and included
	expectedDefs := []string{
		"buf.protoschema.test.v1.LocalMessage.schema.json",
		"buf.protoschema.test.v1.ExternalMessageD.schema.json",
		"buf.protoschema.test.v1.ExternalMessageE.schema.json",
	}

	for _, expectedDef := range expectedDefs {
		_, exists := defs[expectedDef]
		assert.True(t, exists, "Expected referenced schema %s to be included in $defs but it was missing", expectedDef)
	}

	// Verify the oneOf structure and that all references are valid
	ref, ok := crossFileSchema["$ref"].(string)
	require.True(t, ok)

	mainSchemaKey := strings.TrimPrefix(ref, "#/$defs/")
	mainSchema, ok := defs[mainSchemaKey].(map[string]any)
	require.True(t, ok, "Main schema not found in $defs: %s", mainSchemaKey)

	properties, ok := mainSchema["properties"].(map[string]any)
	require.True(t, ok)

	payloadProperty, ok := properties["payload"].(map[string]any)
	require.True(t, ok, "payload property not found in main schema")

	oneOfRaw, exists := payloadProperty["oneOf"]
	require.True(t, exists, "oneOf field not found in payload property")

	var oneOf []any
	var oneOfOk bool

	// Handle type conversion
	if oneOfSlice, ok2 := oneOfRaw.([]any); ok2 {
		oneOf = oneOfSlice
		oneOfOk = true
	} else if oneOfSlice, ok2 := oneOfRaw.([]interface{}); ok2 {
		oneOf = oneOfSlice
		oneOfOk = true
	} else if oneOfSlice, ok2 := oneOfRaw.([]map[string]interface{}); ok2 {
		oneOf = make([]any, len(oneOfSlice))
		for i, item := range oneOfSlice {
			oneOf[i] = item
		}
		oneOfOk = true
	}
	require.True(t, oneOfOk, "Expected payload field to have oneOf structure, got type %T", oneOfRaw)

	expectedRefs := []string{
		"#/$defs/buf.protoschema.test.v1.LocalMessage.schema.json",
		"#/$defs/buf.protoschema.test.v1.ExternalMessageD.schema.json",
		"#/$defs/buf.protoschema.test.v1.ExternalMessageE.schema.json",
	}

	assert.Len(t, oneOf, len(expectedRefs), "Expected oneOf to have %d options", len(expectedRefs))

	for i, refSchema := range oneOf {
		refMap, ok := refSchema.(map[string]any)
		require.True(t, ok)

		actualRef, ok := refMap["$ref"].(string)
		require.True(t, ok)

		// Check if this reference actually exists in $defs
		refKey := strings.TrimPrefix(actualRef, "#/$defs/")
		_, refExists := defs[refKey]
		assert.True(t, refExists, "Reference %s in oneOf[%d] points to non-existent definition in $defs", actualRef, i)
	}
}
