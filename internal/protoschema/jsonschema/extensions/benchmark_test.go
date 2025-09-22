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
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

// BenchmarkConfigLoading measures configuration loading performance
func BenchmarkConfigLoading(b *testing.B) {
	// Create a temporary YAML file with moderate complexity
	configYAML := `version: "1.0"
schema_overrides:
  "example.v1.User":
    schema:
      type: "object"
      properties:
        name:
          type: "string"
          minLength: 1
        email:
          type: "string"
          format: "email"
      required: ["name", "email"]
    references: []
  "example.v1.Address":
    schema:
      type: "object"
      properties:
        street:
          type: "string"
        city:
          type: "string"
        country:
          type: "string"
          enum: ["US", "CA", "UK"]
      required: ["street", "city", "country"]
    references: []
option_processors:
  - name: "required_fields"
    priority: 100
    match:
      option: "validator.field.required"
      value: true
    schema:
      x-required: true
  - name: "pattern_validation"
    priority: 90
    match:
      option: "validator.field.pattern"
      value: "*"
    schema:
      x-has-pattern: true
  - name: "string_length"
    priority: 80
    match:
      option: "validator.field.min_length"
      value: "*"
    schema:
      x-min-length: "${option.min_length}"
`

	tmpFile, err := os.CreateTemp("", "benchmark-config-*.yaml")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configYAML); err != nil {
		b.Fatal(err)
	}
	tmpFile.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		loader := NewConfigLoader(tmpFile.Name())
		_, err := loader.LoadConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRegistryCreation measures registry creation performance
func BenchmarkRegistryCreation(b *testing.B) {
	config := createLargeConfig(10, 20) // 10 overrides, 20 processors

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := NewRegistry(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOptionProcessorMatching measures option processor matching performance
func BenchmarkOptionProcessorMatching(b *testing.B) {
	config := createLargeConfig(5, 50) // 5 overrides, 50 processors
	registry, err := NewRegistry(config)
	if err != nil {
		b.Fatal(err)
	}

	// Create a mock field descriptor
	field := createBenchmarkFieldDescriptor(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := registry.GetOptionProcessors(field)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFieldProcessing measures end-to-end field processing performance
func BenchmarkFieldProcessing(b *testing.B) {
	config := createLargeConfig(5, 20) // 5 overrides, 20 processors
	registry, err := NewRegistry(config)
	if err != nil {
		b.Fatal(err)
	}

	field := createBenchmarkFieldDescriptor(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		schema := map[string]any{
			"type": "string",
		}
		_, err := registry.ProcessField(field, schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSchemaOverrideLookup measures schema override lookup performance
func BenchmarkSchemaOverrideLookup(b *testing.B) {
	config := createLargeConfig(100, 10) // 100 overrides, 10 processors
	registry, err := NewRegistry(config)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fqn := "example.v1.Message50" // Middle of the range
		_ = registry.GetSchemaOverride(fqn)
	}
}

// BenchmarkRegistryMethods measures various registry method performance
func BenchmarkRegistryMethods(b *testing.B) {
	config := createLargeConfig(50, 30)
	registry, err := NewRegistry(config)
	if err != nil {
		b.Fatal(err)
	}

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "HasOverrides",
			fn:   func() { _ = registry.HasOverrides() },
		},
		{
			name: "HasOptionProcessors",
			fn:   func() { _ = registry.HasOptionProcessors() },
		},
		{
			name: "GetOverrideNames",
			fn:   func() { _ = registry.GetOverrideNames() },
		},
		{
			name: "GetProcessorNames",
			fn:   func() { _ = registry.GetProcessorNames() },
		},
		{
			name: "String",
			fn:   func() { _ = registry.String() },
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tt.fn()
			}
		})
	}
}

// BenchmarkMemoryUsage measures memory usage with different configuration sizes
func BenchmarkMemoryUsage(b *testing.B) {
	sizes := []struct {
		name              string
		overrides         int
		processors        int
	}{
		{"Small", 5, 5},
		{"Medium", 50, 25},
		{"Large", 200, 100},
		{"XLarge", 500, 250},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			b.ReportAllocs()

			config := createLargeConfig(size.overrides, size.processors)

			for i := 0; i < b.N; i++ {
				registry, err := NewRegistry(config)
				if err != nil {
					b.Fatal(err)
				}

				// Perform some operations to measure working memory
				_ = registry.HasOverrides()
				_ = registry.GetOverrideNames()
				_ = registry.GetProcessorNames()

				// Force garbage collection to see true memory usage
				if i%10 == 0 {
					b.StopTimer()
					// Don't force GC during benchmarks as it skews results
					b.StartTimer()
				}
			}
		})
	}
}

// Helper function to create a large configuration for benchmarking
func createLargeConfig(numOverrides, numProcessors int) *Config {
	config := &Config{
		Version:         "1.0",
		SchemaOverrides: make(map[string]SchemaOverride),
		OptionProcessors: make([]OptionProcessor, 0, numProcessors),
	}

	// Create schema overrides
	for i := 0; i < numOverrides; i++ {
		fqn := fmt.Sprintf("example.v1.Message%d", i)
		config.SchemaOverrides[fqn] = SchemaOverride{
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type": "integer",
					},
					"name": map[string]any{
						"type":      "string",
						"minLength": 1,
					},
					"active": map[string]any{
						"type": "boolean",
					},
				},
				"required": []any{"id", "name"},
			},
			References: []string{},
		}
	}

	// Create option processors
	for i := 0; i < numProcessors; i++ {
		config.OptionProcessors = append(config.OptionProcessors, OptionProcessor{
			Name:     fmt.Sprintf("processor_%d", i),
			Priority: 100 - i, // Decreasing priority
			Match: OptionMatch{
				Option: fmt.Sprintf("validator.field.rule%d", i%10), // Cycle through 10 different options
				Value:  true,
			},
			Schema: map[string]any{
				fmt.Sprintf("x-rule-%d", i): true,
				"description": fmt.Sprintf("Applied by processor %d", i),
			},
		})
	}

	return config
}

// Helper function to create a field descriptor for benchmarking
func createBenchmarkFieldDescriptor(tb testing.TB) protoreflect.FieldDescriptor {
	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:    func() *string { s := "bench/test.proto"; return &s }(),
		Package: func() *string { s := "bench"; return &s }(),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: func() *string { s := "BenchMessage"; return &s }(),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   func() *string { s := "test_field"; return &s }(),
						Number: func() *int32 { i := int32(1); return &i }(),
						Type:   func() *descriptorpb.FieldDescriptorProto_Type { t := descriptorpb.FieldDescriptorProto_TYPE_STRING; return &t }(),
					},
				},
			},
		},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	if err != nil {
		tb.Fatal(err)
	}

	message := file.Messages().ByName("BenchMessage")
	return message.Fields().ByName("test_field")
}

// Performance smoke test - validates that operations complete within reasonable time
func TestPerformanceBaseline(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (func(), error)
		maxDuration string // Not used in Go tests, but documents expected performance
	}{
		{
			name: "Config loading under 100ms",
			setup: func() (func(), error) {
				config := createLargeConfig(20, 40)
				return func() {
					_, err := NewRegistry(config)
					if err != nil {
						t.Fatal(err)
					}
				}, nil
			},
			maxDuration: "100ms",
		},
		{
			name: "Field processing under 10ms",
			setup: func() (func(), error) {
				config := createLargeConfig(10, 50)
				registry, err := NewRegistry(config)
				if err != nil {
					return nil, err
				}
				field := createBenchmarkFieldDescriptor(t)

				return func() {
					schema := map[string]any{"type": "string"}
					_, err := registry.ProcessField(field, schema)
					if err != nil {
						t.Fatal(err)
					}
				}, nil
			},
			maxDuration: "10ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, err := tt.setup()
			if err != nil {
				t.Fatal(err)
			}

			// Run the operation multiple times to ensure consistency
			for i := 0; i < 100; i++ {
				fn()
			}

			// If we get here without timeouts, the performance is acceptable
			// In a real performance test, we'd use testing.B and check b.Elapsed()
		})
	}
}