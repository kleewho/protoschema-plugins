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
	"strings"
)

// ReferenceResolver handles @FQN reference resolution
type ReferenceResolver struct {
	schemas   map[string]*SchemaOverride
	resolved  map[string]bool
	resolving map[string]bool // For circular dependency detection
}

// NewReferenceResolver creates a new reference resolver
func NewReferenceResolver(schemas map[string]*SchemaOverride) (*ReferenceResolver, error) {
	return &ReferenceResolver{
		schemas:   schemas,
		resolved:  make(map[string]bool),
		resolving: make(map[string]bool),
	}, nil
}

// ResolveAll resolves all @FQN references in schemas
func (r *ReferenceResolver) ResolveAll() error {
	// First pass: resolve all declared dependencies
	for fqn := range r.schemas {
		if err := r.resolveDependencies(fqn); err != nil {
			return fmt.Errorf("failed to resolve dependencies for %s: %w", fqn, err)
		}
	}

	// Second pass: resolve @FQN references in schemas
	for fqn := range r.schemas {
		if err := r.resolveSchema(fqn); err != nil {
			return fmt.Errorf("failed to resolve references in schema %s: %w", fqn, err)
		}
	}

	return nil
}

// resolveDependencies resolves dependencies declared in the references array
func (r *ReferenceResolver) resolveDependencies(fqn string) error {
	schema := r.schemas[fqn]
	if schema == nil {
		return fmt.Errorf("schema not found: %s", fqn)
	}

	// Check that all declared references exist
	for _, refFQN := range schema.References {
		if _, exists := r.schemas[refFQN]; !exists {
			return fmt.Errorf("referenced schema not found: %s (referenced by %s)", refFQN, fqn)
		}
	}

	return nil
}

// resolveSchema resolves references in a specific schema
func (r *ReferenceResolver) resolveSchema(fqn string) error {
	// Check for circular dependencies
	if r.resolving[fqn] {
		return fmt.Errorf("circular reference detected involving %s", fqn)
	}
	if r.resolved[fqn] {
		return nil // Already resolved
	}

	r.resolving[fqn] = true
	defer delete(r.resolving, fqn)

	schema := r.schemas[fqn]
	if schema == nil {
		return fmt.Errorf("schema not found: %s", fqn)
	}

	// Find and resolve @FQN references
	if err := r.resolveReferences(schema.Schema); err != nil {
		return err
	}

	r.resolved[fqn] = true
	return nil
}

// resolveReferences finds and resolves @FQN patterns in a schema
func (r *ReferenceResolver) resolveReferences(schema map[string]any) error {
	return r.walkSchemaForReferences(schema)
}

// walkSchemaForReferences recursively walks schema and resolves @FQN references
func (r *ReferenceResolver) walkSchemaForReferences(obj any) error {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			if key == "$ref" {
				// Found a $ref, check if it's an @FQN reference
				if strValue, ok := value.(string); ok && strings.HasPrefix(strValue, "@") {
					targetFQN := strings.TrimPrefix(strValue, "@")

					// Ensure target schema exists
					if _, exists := r.schemas[targetFQN]; !exists {
						return fmt.Errorf("referenced schema not found: %s", targetFQN)
					}

					// For MVP: simple file reference (non-bundle mode)
					// TODO: Handle bundle mode and proper ID generation based on generator options
					resolvedRef := r.generateSchemaReference(targetFQN)
					v[key] = resolvedRef
				}
			} else {
				// Recursively process nested objects
				if err := r.walkSchemaForReferences(value); err != nil {
					return err
				}
			}
		}
	case []any:
		for _, item := range v {
			if err := r.walkSchemaForReferences(item); err != nil {
				return err
			}
		}
	}
	return nil
}

// generateSchemaReference generates the appropriate reference for a target FQN
func (r *ReferenceResolver) generateSchemaReference(targetFQN string) string {
	// For MVP: use simple schema file reference
	// This will need to be enhanced to handle different generator options:
	// - Bundle vs non-bundle mode
	// - JSON vs proto naming
	// - Strict vs non-strict mode

	// For now, use a simple .schema.json reference
	return targetFQN + ".schema.json"
}

// validateReferences validates that all references can be resolved
func (r *ReferenceResolver) validateReferences() error {
	// Check for any unresolved @FQN references
	for fqn, schema := range r.schemas {
		if err := r.findUnresolvedReferences(fqn, schema.Schema); err != nil {
			return err
		}
	}
	return nil
}

// findUnresolvedReferences finds any @FQN references that haven't been resolved
func (r *ReferenceResolver) findUnresolvedReferences(schemaFQN string, obj any) error {
	switch v := obj.(type) {
	case map[string]any:
		for key, value := range v {
			if key == "$ref" {
				if strValue, ok := value.(string); ok && strings.HasPrefix(strValue, "@") {
					return fmt.Errorf("unresolved reference %s in schema %s", strValue, schemaFQN)
				}
			} else {
				if err := r.findUnresolvedReferences(schemaFQN, value); err != nil {
					return err
				}
			}
		}
	case []any:
		for _, item := range v {
			if err := r.findUnresolvedReferences(schemaFQN, item); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetResolvedSchema returns a resolved schema (for debugging/testing)
func (r *ReferenceResolver) GetResolvedSchema(fqn string) (map[string]any, bool) {
	if schema, exists := r.schemas[fqn]; exists && r.resolved[fqn] {
		return schema.Schema, true
	}
	return nil, false
}

// IsResolved returns true if a schema has been resolved
func (r *ReferenceResolver) IsResolved(fqn string) bool {
	return r.resolved[fqn]
}

// GetDependencies returns the list of dependencies for a schema
func (r *ReferenceResolver) GetDependencies(fqn string) []string {
	if schema, exists := r.schemas[fqn]; exists {
		return schema.References
	}
	return nil
}

// GetUnresolvedSchemas returns a list of schemas that haven't been resolved yet
func (r *ReferenceResolver) GetUnresolvedSchemas() []string {
	var unresolved []string
	for fqn := range r.schemas {
		if !r.resolved[fqn] {
			unresolved = append(unresolved, fqn)
		}
	}
	return unresolved
}