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

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = "jsonschema.extensions.yaml"

// ConfigLoader handles loading and parsing YAML configuration
type ConfigLoader struct {
	configPath string
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(configPath string) *ConfigLoader {
	if configPath == "" {
		configPath = DefaultConfigFile
	}
	return &ConfigLoader{configPath: configPath}
}

// LoadConfig loads configuration from YAML file
func (cl *ConfigLoader) LoadConfig() (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(cl.configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist (backward compatibility)
		return &Config{
			Version:         "1.0",
			SchemaOverrides: make(map[string]SchemaOverride),
			OptionProcessors: []OptionProcessor{},
		}, nil
	}

	data, err := os.ReadFile(cl.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", cl.configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	// Initialize empty maps/slices if not present
	if config.SchemaOverrides == nil {
		config.SchemaOverrides = make(map[string]SchemaOverride)
	}
	if config.OptionProcessors == nil {
		config.OptionProcessors = []OptionProcessor{}
	}

	// Basic validation
	if err := cl.validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// validateConfig performs basic validation of loaded configuration
func (cl *ConfigLoader) validateConfig(config *Config) error {
	// Version check
	if config.Version == "" {
		config.Version = "1.0" // Default version
	} else if config.Version != "1.0" {
		return fmt.Errorf("unsupported config version: %s (expected: 1.0)", config.Version)
	}

	// Validate schema overrides
	for fqn, override := range config.SchemaOverrides {
		if len(override.Schema) == 0 {
			return fmt.Errorf("schema override for %s has empty schema", fqn)
		}
		// Validate that schema is a valid object
		if _, ok := override.Schema["type"]; !ok {
			// Allow schemas without explicit type (they might be references or complex schemas)
		}
	}

	// Validate option processors
	for i, processor := range config.OptionProcessors {
		if processor.Match.Option == "" {
			return fmt.Errorf("option processor at index %d has empty option path", i)
		}
		if len(processor.Schema) == 0 {
			return fmt.Errorf("option processor %s has empty schema", processor.Name)
		}
		if processor.Name == "" {
			return fmt.Errorf("option processor at index %d has empty name", i)
		}
	}

	return nil
}