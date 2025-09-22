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
	"strconv"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"buf.build/go/protovalidate"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FieldOptionExtractor extracts protobuf options from field descriptors
type FieldOptionExtractor struct{}

// NewFieldOptionExtractor creates a new field option extractor
func NewFieldOptionExtractor() *FieldOptionExtractor {
	return &FieldOptionExtractor{}
}

// ExtractOptions extracts protobuf options from a field descriptor
func (e *FieldOptionExtractor) ExtractOptions(field protoreflect.FieldDescriptor) (map[string]interface{}, error) {
	options := make(map[string]interface{})

	// Extract validation rules using the existing protovalidate infrastructure
	rules, err := protovalidate.ResolveFieldRules(field)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve field rules: %w", err)
	}

	if rules == nil {
		return options, nil
	}

	// Extract standard validator.field options
	if err := e.extractValidatorFieldOptions(rules, options); err != nil {
		return nil, fmt.Errorf("failed to extract validator field options: %w", err)
	}

	return options, nil
}

// extractValidatorFieldOptions extracts known validator.field options from FieldRules
func (e *FieldOptionExtractor) extractValidatorFieldOptions(rules *validate.FieldRules, options map[string]interface{}) error {
	baseKey := "validator.field"

	// Extract common validation rules
	if rules.GetRequired() {
		options[baseKey+".required"] = true
	}

	// Extract string-specific rules
	if stringRules := rules.GetString_(); stringRules != nil {
		if pattern := stringRules.GetPattern(); pattern != "" {
			options[baseKey+".pattern"] = pattern
		}
		if minLen := stringRules.GetMinLen(); minLen > 0 {
			options[baseKey+".min_length"] = minLen
		}
		if maxLen := stringRules.GetMaxLen(); maxLen > 0 {
			options[baseKey+".max_length"] = maxLen
		}
	}

	// Extract numeric rules (int32)
	if int32Rules := rules.GetInt32(); int32Rules != nil {
		if min := int32Rules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := int32Rules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := int32Rules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := int32Rules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	// Extract numeric rules (int64)
	if int64Rules := rules.GetInt64(); int64Rules != nil {
		if min := int64Rules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := int64Rules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := int64Rules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := int64Rules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	// Extract numeric rules (uint32)
	if uint32Rules := rules.GetUint32(); uint32Rules != nil {
		if min := uint32Rules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := uint32Rules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := uint32Rules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := uint32Rules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	// Extract numeric rules (uint64)
	if uint64Rules := rules.GetUint64(); uint64Rules != nil {
		if min := uint64Rules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := uint64Rules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := uint64Rules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := uint64Rules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	// Extract float rules
	if floatRules := rules.GetFloat(); floatRules != nil {
		if min := floatRules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := floatRules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := floatRules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := floatRules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	// Extract double rules
	if doubleRules := rules.GetDouble(); doubleRules != nil {
		if min := doubleRules.GetGt(); min != 0 {
			options[baseKey+".min"] = min
		}
		if min := doubleRules.GetGte(); min != 0 {
			options[baseKey+".min"] = min
		}
		if max := doubleRules.GetLt(); max != 0 {
			options[baseKey+".max"] = max
		}
		if max := doubleRules.GetLte(); max != 0 {
			options[baseKey+".max"] = max
		}
	}

	return nil
}

// OptionMatcher handles matching of option values against configured patterns
type OptionMatcher struct{}

// NewOptionMatcher creates a new option matcher
func NewOptionMatcher() *OptionMatcher {
	return &OptionMatcher{}
}

// Matches checks if the given option path and value match the configured pattern
func (om OptionMatch) Matches(optionPath string, optionValue interface{}) bool {
	// First check if the option path matches
	if om.Option != optionPath {
		return false
	}

	// If configured value is "*", it matches any value for this option path
	if om.Value == "*" {
		return true
	}

	// Otherwise, check for exact value match
	return om.matchesValue(optionValue)
}

// matchesValue performs type-aware value matching
func (om OptionMatch) matchesValue(optionValue interface{}) bool {
	configValue := om.Value

	// Handle different value types
	switch v := optionValue.(type) {
	case bool:
		if configBool, ok := configValue.(bool); ok {
			return v == configBool
		}
		// Try to parse string as bool
		if configStr, ok := configValue.(string); ok {
			if configBool, err := strconv.ParseBool(configStr); err == nil {
				return v == configBool
			}
		}
	case string:
		if configStr, ok := configValue.(string); ok {
			return v == configStr
		}
	case int, int32, int64:
		return om.matchesNumeric(v, configValue)
	case uint, uint32, uint64:
		return om.matchesNumeric(v, configValue)
	case float32, float64:
		return om.matchesNumeric(v, configValue)
	}

	// Fallback to string representation comparison
	return fmt.Sprintf("%v", optionValue) == fmt.Sprintf("%v", configValue)
}

// matchesNumeric handles numeric value matching with type coercion
func (om OptionMatch) matchesNumeric(optionValue interface{}, configValue interface{}) bool {
	// Convert both values to float64 for comparison
	optionFloat := om.toFloat64(optionValue)
	configFloat := om.toFloat64(configValue)

	if optionFloat == nil || configFloat == nil {
		return false
	}

	return *optionFloat == *configFloat
}

// toFloat64 safely converts numeric values to float64
func (om OptionMatch) toFloat64(value interface{}) *float64 {
	switch v := value.(type) {
	case int:
		f := float64(v)
		return &f
	case int32:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case uint:
		f := float64(v)
		return &f
	case uint32:
		f := float64(v)
		return &f
	case uint64:
		f := float64(v)
		return &f
	case float32:
		f := float64(v)
		return &f
	case float64:
		return &v
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

// ProcessorMatcher finds option processors that match given field options
type ProcessorMatcher struct {
	extractor *FieldOptionExtractor
	matcher   *OptionMatcher
}

// NewProcessorMatcher creates a new processor matcher
func NewProcessorMatcher() *ProcessorMatcher {
	return &ProcessorMatcher{
		extractor: NewFieldOptionExtractor(),
		matcher:   NewOptionMatcher(),
	}
}

// FindMatchingProcessors finds all option processors that match the given field
func (pm *ProcessorMatcher) FindMatchingProcessors(field protoreflect.FieldDescriptor, processors []OptionProcessor) ([]OptionProcessor, error) {
	// Extract options from the field
	fieldOptions, err := pm.extractor.ExtractOptions(field)
	if err != nil {
		return nil, fmt.Errorf("failed to extract field options: %w", err)
	}

	var matchingProcessors []OptionProcessor

	// Check each processor against the field options
	for _, processor := range processors {
		if pm.processorMatches(processor, fieldOptions) {
			matchingProcessors = append(matchingProcessors, processor)
		}
	}

	return matchingProcessors, nil
}

// processorMatches checks if a processor matches any of the field options
func (pm *ProcessorMatcher) processorMatches(processor OptionProcessor, fieldOptions map[string]interface{}) bool {
	// Check if any field option matches the processor's match criteria
	for optionPath, optionValue := range fieldOptions {
		if processor.Match.Matches(optionPath, optionValue) {
			return true
		}
	}
	return false
}

// SchemaProcessor applies processor schemas to field schemas
type SchemaProcessor struct{}

// NewSchemaProcessor creates a new schema processor
func NewSchemaProcessor() *SchemaProcessor {
	return &SchemaProcessor{}
}

// ApplyProcessors applies matching processors to a field schema in priority order
func (sp *SchemaProcessor) ApplyProcessors(fieldSchema map[string]interface{}, processors []OptionProcessor, fieldOptions map[string]interface{}) error {
	// Processors should already be sorted by priority (highest first)
	for _, processor := range processors {
		if err := sp.applyProcessor(fieldSchema, processor, fieldOptions); err != nil {
			return NewProcessingError(processor.Name, "unknown", err.Error())
		}
	}
	return nil
}

// applyProcessor applies a single processor to the field schema
func (sp *SchemaProcessor) applyProcessor(fieldSchema map[string]interface{}, processor OptionProcessor, fieldOptions map[string]interface{}) error {
	// Apply the processor schema to the field schema
	// This is similar to mergeSchemas but for field-level schemas
	for key, value := range processor.Schema {
		// Handle special processing for certain keys
		switch key {
		case "x-required":
			// This is a special marker that should be handled at the message level
			// For now, just add it to the field schema as metadata
			fieldSchema[key] = value
		default:
			// For regular schema properties, perform template substitution and merge
			processedValue, err := sp.processTemplateValue(value, fieldOptions)
			if err != nil {
				return fmt.Errorf("failed to process template value for key %s: %w", key, err)
			}
			fieldSchema[key] = processedValue
		}
	}
	return nil
}

// processTemplateValue processes template variables in processor schema values
func (sp *SchemaProcessor) processTemplateValue(value interface{}, fieldOptions map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		// Handle template substitution like ${option.pattern}
		if strings.Contains(v, "${") {
			return sp.substituteTemplate(v, fieldOptions)
		}
		return v, nil
	case map[string]interface{}:
		// Recursively process nested objects
		result := make(map[string]interface{})
		for k, nestedValue := range v {
			processedValue, err := sp.processTemplateValue(nestedValue, fieldOptions)
			if err != nil {
				return nil, err
			}
			result[k] = processedValue
		}
		return result, nil
	case []interface{}:
		// Recursively process arrays
		result := make([]interface{}, len(v))
		for i, item := range v {
			processedValue, err := sp.processTemplateValue(item, fieldOptions)
			if err != nil {
				return nil, err
			}
			result[i] = processedValue
		}
		return result, nil
	default:
		// Return non-string values as-is
		return v, nil
	}
}

// substituteTemplate performs simple template substitution
func (sp *SchemaProcessor) substituteTemplate(template string, fieldOptions map[string]interface{}) (interface{}, error) {
	result := template

	// Find all ${...} patterns
	for strings.Contains(result, "${") {
		start := strings.Index(result, "${")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}")
		if end == -1 {
			return nil, NewTemplateError(template, "unterminated template variable", nil)
		}
		end += start

		// Extract the variable name
		varName := result[start+2 : end]

		// Look up the value in field options
		var replacement string
		if strings.HasPrefix(varName, "option.") {
			// Extract option value
			optionKey := strings.TrimPrefix(varName, "option.")
			if value, exists := fieldOptions[optionKey]; exists {
				replacement = fmt.Sprintf("%v", value)
			} else {
				// Look for the option key with validator.field prefix
				fullKey := "validator.field." + optionKey
				if value, exists := fieldOptions[fullKey]; exists {
					replacement = fmt.Sprintf("%v", value)
				} else {
					// Get list of available options for suggestions
					var availableOptions []string
					for key := range fieldOptions {
						availableOptions = append(availableOptions, key)
					}
					return nil, NewTemplateError(template, fmt.Sprintf("option '%s' not found", varName), availableOptions)
				}
			}
		} else {
			return nil, NewTemplateError(template, fmt.Sprintf("unsupported variable type: %s", varName), nil)
		}

		// Replace the template variable
		result = result[:start] + replacement + result[end+1:]
	}

	// If the entire string was a template and resulted in a numeric value, try to parse it
	if template != result && result != "" {
		// Try to parse as number if it looks like one
		if f, err := strconv.ParseFloat(result, 64); err == nil {
			// If it's a whole number, return as int
			if f == float64(int64(f)) {
				return int64(f), nil
			}
			return f, nil
		}
		// Try to parse as bool
		if b, err := strconv.ParseBool(result); err == nil {
			return b, nil
		}
	}

	return result, nil
}