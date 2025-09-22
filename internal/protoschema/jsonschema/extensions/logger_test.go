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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_Levels(t *testing.T) {
	logger := NewLogger(LogLevelDebug, "test")

	// Test level checking
	assert.True(t, logger.IsEnabled(LogLevelError))
	assert.True(t, logger.IsEnabled(LogLevelWarn))
	assert.True(t, logger.IsEnabled(LogLevelInfo))
	assert.True(t, logger.IsEnabled(LogLevelDebug))
	assert.False(t, logger.IsEnabled(LogLevelTrace))

	// Test level setting
	logger.SetLevel(LogLevelTrace)
	assert.True(t, logger.IsEnabled(LogLevelTrace))

	// Test level getting
	assert.Equal(t, LogLevelTrace, logger.GetLevel())
}

func TestLogger_With(t *testing.T) {
	parent := NewLogger(LogLevelInfo, "parent")
	child := parent.With("child")

	assert.Equal(t, LogLevelInfo, child.GetLevel())
	assert.Contains(t, child.prefix, "parent.child")
}

func TestLogger_LogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelSilent, "SILENT"},
		{LogLevelError, "ERROR"},
		{LogLevelWarn, "WARN"},
		{LogLevelInfo, "INFO"},
		{LogLevelDebug, "DEBUG"},
		{LogLevelTrace, "TRACE"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestNewLoggerFromEnv(t *testing.T) {
	// Test default (silent) level
	logger := NewLoggerFromEnv("test")
	assert.Equal(t, LogLevelSilent, logger.GetLevel())

	// Test setting level via environment variable
	os.Setenv("PROTOC_GEN_JSONSCHEMA_LOG_LEVEL", "DEBUG")
	defer os.Unsetenv("PROTOC_GEN_JSONSCHEMA_LOG_LEVEL")

	logger = NewLoggerFromEnv("test")
	assert.Equal(t, LogLevelDebug, logger.GetLevel())

	// Test legacy debug flag
	os.Unsetenv("PROTOC_GEN_JSONSCHEMA_LOG_LEVEL")
	os.Setenv("PROTOC_GEN_JSONSCHEMA_DEBUG", "true")
	defer os.Unsetenv("PROTOC_GEN_JSONSCHEMA_DEBUG")

	logger = NewLoggerFromEnv("test")
	assert.Equal(t, LogLevelDebug, logger.GetLevel())
}

func TestLogger_SpecializedMethods(t *testing.T) {
	logger := NewLogger(LogLevelTrace, "test")

	// Test that we can call these methods without panicking
	// (actual output testing is difficult without capturing log output)
	logger.LogConfigLoad("test.yaml", "start", map[string]any{"key": "value"})
	logger.LogProcessorMatch("processor1", "field1", true, "matched option")
	logger.LogSchemaProcessing("schema1", "process", map[string]any{"step": 1})
	logger.LogOptionExtraction("field1", "option.path", true, "value")
	logger.LogTemplateSubstitution("${template}", "result", map[string]any{"var": "value"})
	logger.LogPerformance("operation", time.Millisecond*100, map[string]any{"count": 5})
	logger.LogError("operation", assert.AnError, map[string]any{"context": "test"})
}

func TestGlobalLogger(t *testing.T) {
	// Test getting global logger
	globalLogger := GetGlobalLogger()
	require.NotNil(t, globalLogger)

	// Test setting global log level
	originalLevel := globalLogger.GetLevel()
	SetGlobalLogLevel(LogLevelInfo)
	assert.Equal(t, LogLevelInfo, globalLogger.GetLevel())

	// Restore original level
	SetGlobalLogLevel(originalLevel)
}

func TestGlobalLogFunctions(t *testing.T) {
	// Test that global functions work without panicking
	// (actual output testing is difficult without capturing log output)
	Error("test error message")
	Warn("test warning message")
	Info("test info message")
	Debug("test debug message")
	Trace("test trace message")
}

func TestLogger_Integration_WithConfigLoading(t *testing.T) {
	// Enable logging for this test
	SetGlobalLogLevel(LogLevelDebug)
	defer SetGlobalLogLevel(LogLevelSilent) // Restore silence

	// Create a temporary config file
	configContent := `version: "1.0"
schema_overrides:
  "test.Message":
    schema:
      type: "object"
    references: []
option_processors: []
`
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Load configuration with logging enabled
	loader := NewConfigLoader(tmpFile.Name())
	config, err := loader.LoadConfig()
	require.NoError(t, err)
	require.NotNil(t, config)

	// Create registry with logging enabled
	registry, err := NewRegistry(config)
	require.NoError(t, err)
	require.NotNil(t, registry)

	// Verify that logging didn't break functionality
	assert.Equal(t, "1.0", config.Version)
	assert.Len(t, config.SchemaOverrides, 1)
	assert.Len(t, config.OptionProcessors, 0)
}

func TestLogger_ConfigLoadPhases(t *testing.T) {
	logger := NewLogger(LogLevelTrace, "test")

	// Test different config load phases
	phases := []string{"start", "file_read", "yaml_parsed", "validation_complete", "empty_config"}

	for _, phase := range phases {
		logger.LogConfigLoad("test.yaml", phase, map[string]any{
			"phase":    phase,
			"duration": time.Millisecond * 50,
		})
	}
}

func TestLogger_ProcessorMatching(t *testing.T) {
	logger := NewLogger(LogLevelTrace, "test")

	// Test processor matching scenarios
	testCases := []struct {
		processorName string
		fieldName     string
		matched       bool
		reason        string
	}{
		{"required_processor", "name", true, "option value matched"},
		{"pattern_processor", "email", false, "option not found"},
		{"debug_processor", "id", true, "wildcard match"},
	}

	for _, tc := range testCases {
		logger.LogProcessorMatch(tc.processorName, tc.fieldName, tc.matched, tc.reason)
	}
}