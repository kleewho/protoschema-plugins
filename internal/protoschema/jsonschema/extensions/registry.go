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
	"sort"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Registry holds loaded extensions and provides lookup methods
type Registry struct {
	config          *Config
	schemaOverrides map[string]*SchemaOverride
	optionProcessors []OptionProcessor
	resolver        *ReferenceResolver
}

// NewRegistry creates a new extension registry
func NewRegistry(config *Config) (*Registry, error) {
	registry := &Registry{
		config:          config,
		schemaOverrides: make(map[string]*SchemaOverride),
		optionProcessors: config.OptionProcessors,
	}

	// Copy schema overrides for faster lookup
	for fqn, override := range config.SchemaOverrides {
		overrideCopy := override // Create copy
		registry.schemaOverrides[fqn] = &overrideCopy
	}

	// Sort option processors by priority (higher priority first)
	sort.Slice(registry.optionProcessors, func(i, j int) bool {
		return registry.optionProcessors[i].Priority > registry.optionProcessors[j].Priority
	})

	// Create and initialize reference resolver
	resolver, err := NewReferenceResolver(registry.schemaOverrides)
	if err != nil {
		return nil, fmt.Errorf("failed to create reference resolver: %w", err)
	}
	registry.resolver = resolver

	// Resolve all references at load time
	if err := registry.resolver.ResolveAll(); err != nil {
		return nil, fmt.Errorf("failed to resolve references: %w", err)
	}

	return registry, nil
}

// LoadRegistry loads configuration and creates a registry
func LoadRegistry(configPath string) (*Registry, error) {
	loader := NewConfigLoader(configPath)
	config, err := loader.LoadConfig()
	if err != nil {
		return nil, err
	}

	return NewRegistry(config)
}

// GetSchemaOverride returns schema override for given FQN, or nil if none exists
func (r *Registry) GetSchemaOverride(fqn string) map[string]any {
	if override, exists := r.schemaOverrides[fqn]; exists {
		// Return a deep copy to prevent modifications to the original
		return copySchema(override.Schema)
	}
	return nil
}

// GetOptionProcessors returns option processors that match the given field
func (r *Registry) GetOptionProcessors(field protoreflect.FieldDescriptor) []OptionProcessor {
	var matches []OptionProcessor

	// For MVP, simplified matching based on field options
	for _, processor := range r.optionProcessors {
		if r.matchesField(processor, field) {
			matches = append(matches, processor)
		}
	}

	return matches
}

// matchesField checks if an option processor matches a field (simplified for MVP)
func (r *Registry) matchesField(processor OptionProcessor, field protoreflect.FieldDescriptor) bool {
	// For MVP: simplified exact matching
	// This is a placeholder implementation - actual implementation will depend on
	// how protobuf options are accessed and structured.
	// For now, we'll return false to avoid errors, but this will be implemented
	// when we have actual protobuf option parsing.

	// TODO: Implement actual option matching based on protobuf field options
	// This would involve:
	// 1. Extracting options from the field descriptor
	// 2. Matching the option path (e.g., "validator.field.required")
	// 3. Comparing the option value with the expected value

	// Example structure (to be implemented):
	// optionPath := processor.Match.Option
	// expectedValue := processor.Match.Value
	//
	// Extract option value from field based on optionPath
	// actualValue := extractOptionValue(field, optionPath)
	//
	// return compareValues(expectedValue, actualValue)

	return false
}

// copySchema creates a deep copy of a schema map
func copySchema(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}

	result := make(map[string]any)
	for k, v := range schema {
		result[k] = copyValue(v)
	}
	return result
}

// copyValue creates a deep copy of any value
func copyValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return copySchema(v)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = copyValue(item)
		}
		return result
	case map[any]any:
		result := make(map[any]any)
		for k, val := range v {
			result[k] = copyValue(val)
		}
		return result
	default:
		// For primitive types (string, int, bool, etc.), return as-is
		// since they are immutable in Go
		return v
	}
}

// ApplyOptionProcessors applies all matching option processors to a field schema
func (r *Registry) ApplyOptionProcessors(field protoreflect.FieldDescriptor, schema map[string]any) error {
	processors := r.GetOptionProcessors(field)

	for _, processor := range processors {
		// Merge processor schema with field schema
		if err := mergeSchemas(schema, processor.Schema); err != nil {
			return fmt.Errorf("failed to apply processor %s: %w", processor.Name, err)
		}
	}

	return nil
}

// mergeSchemas merges source schema into target schema
func mergeSchemas(target, source map[string]any) error {
	for key, value := range source {
		if existing, exists := target[key]; exists {
			// If both are maps, merge recursively
			if existingMap, ok := existing.(map[string]any); ok {
				if sourceMap, ok := value.(map[string]any); ok {
					if err := mergeSchemas(existingMap, sourceMap); err != nil {
						return err
					}
					continue
				}
			}
			// If both are slices, append (for arrays like required fields)
			if existingSlice, ok := existing.([]any); ok {
				if sourceSlice, ok := value.([]any); ok {
					target[key] = append(existingSlice, sourceSlice...)
					continue
				}
			}
		}
		// Override or add new key
		target[key] = copyValue(value)
	}
	return nil
}

// HasOverrides returns true if the registry has any schema overrides
func (r *Registry) HasOverrides() bool {
	return len(r.schemaOverrides) > 0
}

// HasOptionProcessors returns true if the registry has any option processors
func (r *Registry) HasOptionProcessors() bool {
	return len(r.optionProcessors) > 0
}

// GetOverrideNames returns a list of all FQNs that have schema overrides
func (r *Registry) GetOverrideNames() []string {
	names := make([]string, 0, len(r.schemaOverrides))
	for fqn := range r.schemaOverrides {
		names = append(names, fqn)
	}
	sort.Strings(names)
	return names
}

// GetProcessorNames returns a list of all option processor names
func (r *Registry) GetProcessorNames() []string {
	names := make([]string, 0, len(r.optionProcessors))
	for _, processor := range r.optionProcessors {
		names = append(names, processor.Name)
	}
	return names
}

// String returns a string representation of the registry for debugging
func (r *Registry) String() string {
	var parts []string

	if len(r.schemaOverrides) > 0 {
		parts = append(parts, fmt.Sprintf("Schema Overrides: %d", len(r.schemaOverrides)))
	}

	if len(r.optionProcessors) > 0 {
		parts = append(parts, fmt.Sprintf("Option Processors: %d", len(r.optionProcessors)))
	}

	if len(parts) == 0 {
		return "Registry: empty"
	}

	return "Registry: " + strings.Join(parts, ", ")
}