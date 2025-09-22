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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnyConstraints_SingleAnyField(t *testing.T) {
	t.Parallel()

	// Read the generated schema file
	schemaPath := "../../gen/jsonschema/buf.protoschema.test.v1.EventEnvelope.schema.json"
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should be able to read generated schema file")

	var eventSchema map[string]any
	err = json.Unmarshal(data, &eventSchema)
	require.NoError(t, err, "should be able to parse JSON schema")

	// Check the payload field
	properties, ok := eventSchema["properties"].(map[string]any)
	require.True(t, ok, "schema should have properties")

	payloadField, ok := properties["payload"].(map[string]any)
	require.True(t, ok, "payload field should exist")

	// Should have oneOf instead of type/properties
	oneOf, exists := payloadField["oneOf"]
	assert.True(t, exists, "payload should have oneOf constraint")

	if exists {
		oneOfArray, ok := oneOf.([]any)
		require.True(t, ok, "oneOf should be an array")
		assert.Len(t, oneOfArray, 2, "should have exactly 2 options")

		// Check that both MessageA and MessageB are referenced
		refs := make([]string, 0, len(oneOfArray))
		for _, option := range oneOfArray {
			if optionMap, ok := option.(map[string]any); ok {
				if ref, ok := optionMap["$ref"].(string); ok {
					refs = append(refs, ref)
				}
			}
		}

		assert.Contains(t, refs, "buf.protoschema.test.v1.MessageA.schema.json", "should reference MessageA")
		assert.Contains(t, refs, "buf.protoschema.test.v1.MessageB.schema.json", "should reference MessageB")
	}

	// Should NOT have type: object or @type property (default Any behavior)
	assert.NotContains(t, payloadField, "type", "constrained Any should not have type field")
	assert.NotContains(t, payloadField, "properties", "constrained Any should not have properties field")
}

func TestAnyConstraints_RepeatedAnyField(t *testing.T) {
	t.Parallel()

	// Read the generated schema file
	schemaPath := "../../gen/jsonschema/buf.protoschema.test.v1.EventBatch.schema.json"
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should be able to read generated schema file")

	var eventBatchSchema map[string]any
	err = json.Unmarshal(data, &eventBatchSchema)
	require.NoError(t, err, "should be able to parse JSON schema")

	properties, ok := eventBatchSchema["properties"].(map[string]any)
	require.True(t, ok, "schema should have properties")

	eventsField, ok := properties["events"].(map[string]any)
	require.True(t, ok, "events field should exist")

	// Should be an array
	assert.Equal(t, "array", eventsField["type"], "events should be an array")

	// Check the items constraint
	items, ok := eventsField["items"].(map[string]any)
	require.True(t, ok, "events should have items definition")

	oneOf, exists := items["oneOf"]
	assert.True(t, exists, "items should have oneOf constraint")

	if exists {
		oneOfArray, ok := oneOf.([]any)
		require.True(t, ok, "oneOf should be an array")
		assert.Len(t, oneOfArray, 3, "should have exactly 3 options (MessageA, MessageB, MessageC)")
	}
}

func TestAnyConstraints_UnconstrainedAny(t *testing.T) {
	t.Parallel()

	// Read the generated schema file
	schemaPath := "../../gen/jsonschema/buf.protoschema.test.v1.GenericEnvelope.schema.json"
	data, err := os.ReadFile(schemaPath)
	require.NoError(t, err, "should be able to read generated schema file")

	var genericSchema map[string]any
	err = json.Unmarshal(data, &genericSchema)
	require.NoError(t, err, "should be able to parse JSON schema")

	properties, ok := genericSchema["properties"].(map[string]any)
	require.True(t, ok, "schema should have properties")

	dataField, ok := properties["data"].(map[string]any)
	require.True(t, ok, "data field should exist")

	// Should use default Any behavior - check if it has a $ref to google.protobuf.Any
	if ref, hasRef := dataField["$ref"]; hasRef {
		assert.Equal(t, "google.protobuf.Any.schema.json", ref, "unconstrained Any should reference google.protobuf.Any schema")
	} else {
		// If not using $ref, should have default Any behavior (type: object, @type property)
		assert.Equal(t, "object", dataField["type"], "unconstrained Any should have object type")

		props, ok := dataField["properties"].(map[string]any)
		assert.True(t, ok, "unconstrained Any should have properties")

		if ok {
			typeField, exists := props["@type"].(map[string]any)
			assert.True(t, exists, "should have @type field")
			if exists {
				assert.Equal(t, "string", typeField["type"], "@type should be a string")
			}
		}
	}

	// Should NOT have oneOf
	assert.NotContains(t, dataField, "oneOf", "unconstrained Any should not have oneOf")
}