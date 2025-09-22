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
	err := generator.generateConstrainedAnyValidation(nil, schema)
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

	err := generator.generateConstrainedAnyValidation(rules, schema)
	require.NoError(t, err)

	// Should fall back to default Any behavior
	assert.Equal(t, jsObject, schema["type"])
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	typeProperty, ok := properties["@type"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", typeProperty["type"])
}
