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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kleewho/protoschema-plugins/internal/protoschema"
	"github.com/kleewho/protoschema-plugins/internal/protoschema/jsonschema/extensions"
)

func main() {
	var (
		configPath   = flag.String("config", "", "Path to YAML configuration file to validate (required)")
		verbose      = flag.Bool("verbose", false, "Enable verbose output with detailed validation information")
		jsonOutput   = flag.Bool("json", false, "Output validation results in JSON format")
		quiet        = flag.Bool("quiet", false, "Only output errors, suppress informational messages")
		showVersion  = flag.Bool("version", false, "Show version information")
		checkRefs    = flag.Bool("check-refs", true, "Validate reference resolution (can be disabled for faster validation)")
		showSummary  = flag.Bool("summary", true, "Show validation summary (can be disabled with --no-summary)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "Validates YAML configuration files for protoc-gen-jsonschema extensions.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --config=config.yaml\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s --config=config.yaml --verbose\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s --config=config.yaml --json\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "\nExit codes:\n")
		fmt.Fprintf(os.Stderr, "  0: Validation successful\n")
		fmt.Fprintf(os.Stderr, "  1: Validation failed\n")
		fmt.Fprintf(os.Stderr, "  2: Invalid arguments or file not found\n")
	}

	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("jsonschema-validate-config version %s\n", protoschema.Version())
		os.Exit(0)
	}

	// Validate arguments
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config flag is required\n\n")
		flag.Usage()
		os.Exit(2)
	}

	// Check if config file exists
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Configuration file not found: %s\n", *configPath)
		os.Exit(2)
	}

	// Perform validation
	result := validateConfig(*configPath, ValidationOptions{
		CheckReferences: *checkRefs,
		Verbose:        *verbose,
	})

	// Output results
	if *jsonOutput {
		outputJSON(result)
	} else {
		outputText(result, *quiet, *showSummary, *verbose)
	}

	// Exit with appropriate code
	if result.Valid {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

type ValidationOptions struct {
	CheckReferences bool
	Verbose         bool
}

type ValidationResult struct {
	Valid       bool     `json:"valid"`
	ConfigPath  string   `json:"config_path"`
	Version     string   `json:"version,omitempty"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	Summary     Summary  `json:"summary"`
	ProcessTime string   `json:"process_time,omitempty"`
}

type Summary struct {
	SchemaOverrides  int `json:"schema_overrides"`
	OptionProcessors int `json:"option_processors"`
	References       int `json:"references"`
}

func validateConfig(configPath string, opts ValidationOptions) ValidationResult {
	result := ValidationResult{
		ConfigPath: configPath,
		Summary:    Summary{},
	}

	// Load configuration
	loader := extensions.NewConfigLoader(configPath)
	config, err := loader.LoadConfig()
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Configuration loading failed: %v", err))
		return result
	}

	result.Version = config.Version
	result.Summary.SchemaOverrides = len(config.SchemaOverrides)
	result.Summary.OptionProcessors = len(config.OptionProcessors)

	// Count references
	for _, override := range config.SchemaOverrides {
		result.Summary.References += len(override.References)
	}

	// Create registry to validate everything works together
	registry, err := extensions.NewRegistry(config)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Registry creation failed: %v", err))
		return result
	}

	// Additional validations
	if opts.CheckReferences {
		if warnings := validateReferences(config); len(warnings) > 0 {
			result.Warnings = append(result.Warnings, warnings...)
		}
	}

	// Check for common configuration issues
	if warnings := validateCommonIssues(config); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Success - ensure registry is used to avoid "unused variable" warning
	_ = registry
	result.Valid = true
	return result
}

func validateReferences(config *extensions.Config) []string {
	var warnings []string

	// Check for unused references
	allFQNs := make(map[string]bool)
	for fqn := range config.SchemaOverrides {
		allFQNs[fqn] = true
	}

	for fqn, override := range config.SchemaOverrides {
		for _, ref := range override.References {
			// Remove @FQN prefix if present
			cleanRef := strings.TrimPrefix(ref, "@")
			// Extract FQN part (before any JSON pointer)
			if dotIndex := strings.Index(cleanRef, "#"); dotIndex > 0 {
				cleanRef = cleanRef[:dotIndex]
			}

			if !allFQNs[cleanRef] {
				warnings = append(warnings, fmt.Sprintf("Reference '%s' in override '%s' points to undefined schema", ref, fqn))
			}
		}
	}

	return warnings
}

func validateCommonIssues(config *extensions.Config) []string {
	var warnings []string

	// Check for conflicting processors with same priority
	priorityMap := make(map[int][]string)
	for _, processor := range config.OptionProcessors {
		priorityMap[processor.Priority] = append(priorityMap[processor.Priority], processor.Name)
	}

	for priority, processors := range priorityMap {
		if len(processors) > 1 {
			warnings = append(warnings, fmt.Sprintf("Multiple processors with same priority %d: %s", priority, strings.Join(processors, ", ")))
		}
	}

	// Check for processors with very low priority (might not be applied)
	for _, processor := range config.OptionProcessors {
		if processor.Priority < 10 {
			warnings = append(warnings, fmt.Sprintf("Processor '%s' has very low priority (%d), might not be applied", processor.Name, processor.Priority))
		}
	}

	// Check for empty schemas
	for fqn, override := range config.SchemaOverrides {
		if len(override.Schema) == 0 {
			warnings = append(warnings, fmt.Sprintf("Schema override '%s' has empty schema", fqn))
		}
	}

	return warnings
}

func outputJSON(result ValidationResult) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON output: %v", err)
	}
	fmt.Println(string(data))
}

func outputText(result ValidationResult, quiet, showSummary, verbose bool) {
	if result.Valid {
		if !quiet {
			fmt.Printf("✓ Configuration validation successful: %s\n", result.ConfigPath)
		}

		if showSummary {
			if !quiet {
				fmt.Printf("\nSummary:\n")
				fmt.Printf("  Version: %s\n", result.Version)
				fmt.Printf("  Schema overrides: %d\n", result.Summary.SchemaOverrides)
				fmt.Printf("  Option processors: %d\n", result.Summary.OptionProcessors)
				if result.Summary.References > 0 {
					fmt.Printf("  Cross-references: %d\n", result.Summary.References)
				}
			}
		}

		if len(result.Warnings) > 0 {
			if !quiet {
				fmt.Printf("\nWarnings:\n")
			}
			for _, warning := range result.Warnings {
				fmt.Printf("  ⚠ %s\n", warning)
			}
		}
	} else {
		fmt.Printf("✗ Configuration validation failed: %s\n", result.ConfigPath)

		if len(result.Errors) > 0 {
			fmt.Printf("\nErrors:\n")
			for _, err := range result.Errors {
				fmt.Printf("  ✗ %s\n", err)
			}
		}

		if len(result.Warnings) > 0 && verbose {
			fmt.Printf("\nWarnings:\n")
			for _, warning := range result.Warnings {
				fmt.Printf("  ⚠ %s\n", warning)
			}
		}
	}
}