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

// ErrorType represents different categories of errors
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeTemplate     ErrorType = "TEMPLATE_ERROR"
	ErrorTypeReference    ErrorType = "REFERENCE_ERROR"
	ErrorTypeConfiguration ErrorType = "CONFIG_ERROR"
	ErrorTypeProcessing   ErrorType = "PROCESSING_ERROR"
)

// ExtensionError provides structured error information with context and suggestions
type ExtensionError struct {
	Type        ErrorType
	Message     string
	Context     map[string]string
	Suggestions []string
	Cause       error
	Location    string // File path, line number, or configuration section
}

// Error implements the error interface
func (e *ExtensionError) Error() string {
	var parts []string

	// Add error type and message
	parts = append(parts, fmt.Sprintf("[%s] %s", e.Type, e.Message))

	// Add location if available
	if e.Location != "" {
		parts = append(parts, fmt.Sprintf("Location: %s", e.Location))
	}

	// Add context information
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s: %s", key, value))
		}
		parts = append(parts, fmt.Sprintf("Context: %s", strings.Join(contextParts, ", ")))
	}

	// Add suggestions
	if len(e.Suggestions) > 0 {
		parts = append(parts, fmt.Sprintf("Suggestions: %s", strings.Join(e.Suggestions, "; ")))
	}

	// Add underlying cause
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("Caused by: %v", e.Cause))
	}

	return strings.Join(parts, "\n")
}

// Unwrap returns the underlying cause for error wrapping compatibility
func (e *ExtensionError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a validation error with context and suggestions
func NewValidationError(message string, context map[string]string, suggestions []string) *ExtensionError {
	return &ExtensionError{
		Type:        ErrorTypeValidation,
		Message:     message,
		Context:     context,
		Suggestions: suggestions,
	}
}

// NewTemplateError creates a template processing error with context
func NewTemplateError(template, reason string, availableOptions []string) *ExtensionError {
	context := map[string]string{
		"template": template,
		"reason":   reason,
	}

	var suggestions []string
	if len(availableOptions) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("Available options: [%s]", strings.Join(availableOptions, ", ")))
	}
	suggestions = append(suggestions, "Check template syntax: ${option.fieldname}")

	return &ExtensionError{
		Type:        ErrorTypeTemplate,
		Message:     fmt.Sprintf("Template processing failed: %s", reason),
		Context:     context,
		Suggestions: suggestions,
	}
}

// NewReferenceError creates a reference resolution error
func NewReferenceError(reference, schemaName string, availableSchemas []string) *ExtensionError {
	context := map[string]string{
		"reference":   reference,
		"schema":      schemaName,
	}

	var suggestions []string
	if len(availableSchemas) > 0 {
		// Find closest matches
		closest := findClosestMatches(reference, availableSchemas, 2)
		if len(closest) > 0 {
			suggestions = append(suggestions, fmt.Sprintf("Did you mean: %s", strings.Join(closest, " or ")))
		}
		suggestions = append(suggestions, fmt.Sprintf("Available schemas: [%s]", strings.Join(availableSchemas, ", ")))
	}
	suggestions = append(suggestions, "Use @FQN syntax for references (e.g., @example.v1.Message)")

	return &ExtensionError{
		Type:        ErrorTypeReference,
		Message:     fmt.Sprintf("Reference '%s' not found", reference),
		Context:     context,
		Suggestions: suggestions,
	}
}

// NewConfigurationError creates a configuration error with specific context
func NewConfigurationError(configPath, section, issue string, suggestions []string) *ExtensionError {
	context := map[string]string{
		"config_file": configPath,
		"section":     section,
		"issue":       issue,
	}

	return &ExtensionError{
		Type:        ErrorTypeConfiguration,
		Message:     fmt.Sprintf("Configuration error in '%s'", section),
		Context:     context,
		Suggestions: suggestions,
		Location:    configPath,
	}
}

// NewProcessingError creates a processing error during field/schema processing
func NewProcessingError(processorName, fieldName, reason string) *ExtensionError {
	context := map[string]string{
		"processor": processorName,
		"field":     fieldName,
		"reason":    reason,
	}

	suggestions := []string{
		"Check processor configuration and match criteria",
		"Verify field has required options",
		"Review template syntax in processor schema",
	}

	return &ExtensionError{
		Type:        ErrorTypeProcessing,
		Message:     fmt.Sprintf("Failed to process field '%s' with processor '%s'", fieldName, processorName),
		Context:     context,
		Suggestions: suggestions,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, message string, context map[string]string) *ExtensionError {
	return &ExtensionError{
		Type:    errorType,
		Message: message,
		Context: context,
		Cause:   err,
	}
}

// findClosestMatches finds the closest string matches using simple distance
func findClosestMatches(target string, candidates []string, limit int) []string {
	type match struct {
		value    string
		distance int
	}

	var matches []match
	for _, candidate := range candidates {
		distance := levenshteinDistance(target, candidate)
		// Only consider matches within reasonable distance
		if distance <= len(target)/2+1 {
			matches = append(matches, match{candidate, distance})
		}
	}

	// Sort by distance
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[i].distance > matches[j].distance {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	// Return up to limit closest matches
	var result []string
	for i := 0; i < len(matches) && i < limit; i++ {
		result = append(result, matches[i].value)
	}

	return result
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	// Initialize first row and column
	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// min returns the minimum of three integers
func min(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

// ValidationErrorBuilder helps build validation errors with context
type ValidationErrorBuilder struct {
	errors      []string
	context     map[string]string
	suggestions []string
	location    string
}

// NewValidationErrorBuilder creates a new validation error builder
func NewValidationErrorBuilder(location string) *ValidationErrorBuilder {
	return &ValidationErrorBuilder{
		context:  make(map[string]string),
		location: location,
	}
}

// AddError adds a validation error message
func (b *ValidationErrorBuilder) AddError(message string) {
	b.errors = append(b.errors, message)
}

// AddContext adds contextual information
func (b *ValidationErrorBuilder) AddContext(key, value string) {
	b.context[key] = value
}

// AddSuggestion adds a suggestion for fixing the issue
func (b *ValidationErrorBuilder) AddSuggestion(suggestion string) {
	b.suggestions = append(b.suggestions, suggestion)
}

// HasErrors returns true if any errors were added
func (b *ValidationErrorBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Build creates the final validation error
func (b *ValidationErrorBuilder) Build() *ExtensionError {
	if !b.HasErrors() {
		return nil
	}

	message := fmt.Sprintf("Configuration validation failed: %s", strings.Join(b.errors, "; "))

	return &ExtensionError{
		Type:        ErrorTypeValidation,
		Message:     message,
		Context:     b.context,
		Suggestions: b.suggestions,
		Location:    b.location,
	}
}