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

// Config represents the complete YAML configuration structure
type Config struct {
	Version         string                    `yaml:"version"`
	SchemaOverrides map[string]SchemaOverride `yaml:"schema_overrides"`
	OptionProcessors []OptionProcessor        `yaml:"option_processors"`
}

// SchemaOverride represents a complete JSON schema replacement for a type
type SchemaOverride struct {
	Schema     map[string]any `yaml:"schema"`     // The JSON schema definition
	References []string       `yaml:"references"` // FQNs this schema references
}

// OptionProcessor represents field processing based on protobuf options
type OptionProcessor struct {
	Name     string         `yaml:"name"`     // Processor name
	Priority int            `yaml:"priority"` // Processing priority (higher = first)
	Match    OptionMatch    `yaml:"match"`    // Matching criteria
	Schema   map[string]any `yaml:"schema"`   // Schema to merge
}

// OptionMatch defines how to match protobuf options
type OptionMatch struct {
	Option string `yaml:"option"` // Option path (e.g., "validator.field")
	Value  any    `yaml:"value"`  // Expected value (simplified for MVP)
}