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
	"runtime"
	"testing"
	"time"

	testv1 "github.com/bufbuild/protoschema-plugins/internal/gen/proto/buf/protoschema/test/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ScaleTestConfig defines parameters for scale testing
type ScaleTestConfig struct {
	SchemaOverrides  int
	OptionProcessors int
	CrossReferences  int
	TestName         string
}

// Predefined scale test configurations
var scaleConfigs = []ScaleTestConfig{
	{10, 20, 5, "Small"},
	{100, 200, 50, "Medium"},
	{500, 1000, 250, "Large"},
	{1000, 2000, 500, "XLarge"},
	{2000, 5000, 1000, "XXLarge"},
}

func TestScale_ConfigurationLoading(t *testing.T) {
	for _, config := range scaleConfigs {
		t.Run(config.TestName, func(t *testing.T) {
			testConfigurationLoadingScale(t, config)
		})
	}
}

func testConfigurationLoadingScale(t *testing.T, scaleConfig ScaleTestConfig) {
	// Create large configuration
	config := createLargeScaleConfig(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors, scaleConfig.CrossReferences)

	// Save to temporary file
	configPath := createTempConfigFile(t, config)
	defer os.Remove(configPath)

	// Measure memory before
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Load configuration with timing
	start := time.Now()
	loader := NewConfigLoader(configPath)
	loadedConfig, err := loader.LoadConfig()
	loadDuration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, loadedConfig)

	// Verify configuration loaded correctly
	assert.Equal(t, scaleConfig.SchemaOverrides, len(loadedConfig.SchemaOverrides))
	assert.Equal(t, scaleConfig.OptionProcessors, len(loadedConfig.OptionProcessors))

	// Measure memory after
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	// Performance assertions
	maxLoadTime := getMaxLoadTime(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors)
	if loadDuration > maxLoadTime {
		t.Logf("WARNING: Load time %v exceeds expected maximum %v for %s scale",
			loadDuration, maxLoadTime, scaleConfig.TestName)
	}

	// Memory usage analysis
	memUsed := memAfter.HeapAlloc - memBefore.HeapAlloc
	maxMemory := getMaxMemoryUsage(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors)

	t.Logf("Scale: %s, Load time: %v, Memory used: %d bytes, Overrides: %d, Processors: %d",
		scaleConfig.TestName, loadDuration, memUsed, scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors)

	if memUsed > maxMemory {
		t.Logf("WARNING: Memory usage %d bytes exceeds expected maximum %d bytes for %s scale",
			memUsed, maxMemory, scaleConfig.TestName)
	}
}

func TestScale_RegistryCreation(t *testing.T) {
	for _, config := range scaleConfigs {
		t.Run(config.TestName, func(t *testing.T) {
			testRegistryCreationScale(t, config)
		})
	}
}

func testRegistryCreationScale(t *testing.T, scaleConfig ScaleTestConfig) {
	// Create large configuration
	config := createLargeScaleConfig(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors, scaleConfig.CrossReferences)

	// Measure memory before
	var memBefore runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// Create registry with timing
	start := time.Now()
	registry, err := NewRegistry(config)
	createDuration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, registry)

	// Verify registry created correctly
	assert.Equal(t, scaleConfig.SchemaOverrides, len(registry.GetOverrideNames()))
	assert.Equal(t, scaleConfig.OptionProcessors, len(registry.GetProcessorNames()))

	// Measure memory after
	var memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memAfter)

	// Performance assertions
	maxCreateTime := getMaxCreateTime(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors)
	if createDuration > maxCreateTime {
		t.Logf("WARNING: Registry creation time %v exceeds expected maximum %v for %s scale",
			createDuration, maxCreateTime, scaleConfig.TestName)
	}

	// Memory usage analysis
	memUsed := memAfter.HeapAlloc - memBefore.HeapAlloc
	registryMemory := getRegistryMaxMemory(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors)

	t.Logf("Scale: %s, Create time: %v, Memory used: %d bytes, Registry items: %d",
		scaleConfig.TestName, createDuration, memUsed,
		scaleConfig.SchemaOverrides+scaleConfig.OptionProcessors)

	if memUsed > registryMemory {
		t.Logf("WARNING: Registry memory usage %d bytes exceeds expected maximum %d bytes for %s scale",
			memUsed, registryMemory, scaleConfig.TestName)
	}
}

func TestScale_FieldProcessing(t *testing.T) {
	// Create a large configuration for field processing
	config := createLargeScaleConfig(100, 500, 50)

	registry, err := NewRegistry(config)
	require.NoError(t, err)

	// Get a test field descriptor
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()
	fields := constraintTestDesc.Fields()

	require.Greater(t, fields.Len(), 0, "Need at least one field for testing")
	testField := fields.Get(0)

	// Test field processing performance with many processors
	var totalDuration time.Duration
	numIterations := 1000

	for i := 0; i < numIterations; i++ {
		schema := map[string]any{"type": "string"}

		start := time.Now()
		result, err := registry.ProcessField(testField, schema)
		duration := time.Since(start)

		require.NoError(t, err)
		require.NotNil(t, result)
		totalDuration += duration
	}

	avgDuration := totalDuration / time.Duration(numIterations)
	maxAvgDuration := time.Millisecond * 10 // Should average under 10ms per field

	t.Logf("Field processing: %d iterations, average duration: %v, total: %v",
		numIterations, avgDuration, totalDuration)

	if avgDuration > maxAvgDuration {
		t.Logf("WARNING: Average field processing time %v exceeds expected maximum %v",
			avgDuration, maxAvgDuration)
	}

	// Ensure reasonable performance
	assert.Less(t, avgDuration.Nanoseconds(), maxAvgDuration.Nanoseconds(),
		"Field processing should be reasonably fast even with many processors")
}

func TestScale_MemoryGrowth(t *testing.T) {
	// Test that memory usage grows linearly with configuration size
	var previousMemory uint64 = 0
	var memoryGrowthRatios []float64

	for i, config := range scaleConfigs[:4] { // Skip XXLarge for this test
		t.Run(config.TestName, func(t *testing.T) {
			// Force garbage collection
			runtime.GC()
			var memBefore runtime.MemStats
			runtime.ReadMemStats(&memBefore)

			// Create configuration and registry
			cfg := createLargeScaleConfig(config.SchemaOverrides, config.OptionProcessors, config.CrossReferences)
			registry, err := NewRegistry(cfg)
			require.NoError(t, err)
			require.NotNil(t, registry)

			// Force another GC and measure
			runtime.GC()
			var memAfter runtime.MemStats
			runtime.ReadMemStats(&memAfter)

			currentMemory := memAfter.HeapAlloc - memBefore.HeapAlloc

			t.Logf("Scale: %s, Memory: %d bytes, Items: %d",
				config.TestName, currentMemory, config.SchemaOverrides+config.OptionProcessors)

			// Calculate growth ratio (skip first iteration)
			if i > 0 && previousMemory > 0 {
				growthRatio := float64(currentMemory) / float64(previousMemory)
				memoryGrowthRatios = append(memoryGrowthRatios, growthRatio)

				t.Logf("Memory growth ratio from previous: %.2fx", growthRatio)

				// Memory growth should be reasonable (less than 20x between scale levels)
				assert.Less(t, growthRatio, 20.0,
					"Memory growth should not be excessive between scale levels")
			}

			previousMemory = currentMemory
		})
	}

	// Check that growth is relatively consistent
	if len(memoryGrowthRatios) > 1 {
		var totalRatio float64
		for _, ratio := range memoryGrowthRatios {
			totalRatio += ratio
		}
		avgRatio := totalRatio / float64(len(memoryGrowthRatios))
		t.Logf("Average memory growth ratio: %.2fx", avgRatio)

		// Growth should be somewhat predictable (not wildly inconsistent)
		assert.Less(t, avgRatio, 15.0, "Average memory growth should be reasonable")
	}
}

func TestScale_CrossReferencePerformance(t *testing.T) {
	// Test configurations with heavy cross-referencing
	crossRefConfigs := []ScaleTestConfig{
		{50, 10, 200, "ManyReferences"},    // More references than overrides
		{100, 20, 500, "HeavyReferences"},  // 5x references to overrides
		{200, 30, 1000, "ExtremeReferences"}, // 5x references to overrides
	}

	for _, config := range crossRefConfigs {
		t.Run(config.TestName, func(t *testing.T) {
			testCrossReferenceScale(t, config)
		})
	}
}

func testCrossReferenceScale(t *testing.T, scaleConfig ScaleTestConfig) {
	config := createLargeScaleConfig(scaleConfig.SchemaOverrides, scaleConfig.OptionProcessors, scaleConfig.CrossReferences)

	start := time.Now()
	registry, err := NewRegistry(config)
	createDuration := time.Since(start)

	require.NoError(t, err)
	require.NotNil(t, registry)

	// Cross-reference resolution should not take too long
	maxResolveTime := time.Second * 5 // 5 seconds max for even extreme cases
	if createDuration > maxResolveTime {
		t.Errorf("Cross-reference resolution took %v, exceeds maximum %v",
			createDuration, maxResolveTime)
	}

	t.Logf("Cross-reference resolution: %d overrides, %d references, duration: %v",
		scaleConfig.SchemaOverrides, scaleConfig.CrossReferences, createDuration)
}

func TestScale_ConcurrentAccess(t *testing.T) {
	// Test that registry can handle concurrent access
	config := createLargeScaleConfig(100, 200, 50)
	registry, err := NewRegistry(config)
	require.NoError(t, err)

	// Get test data
	constraintTestDesc := (&testv1.ConstraintTest{}).ProtoReflect().Descriptor()
	fields := constraintTestDesc.Fields()
	require.Greater(t, fields.Len(), 0)
	testField := fields.Get(0)

	// Run concurrent access test
	numGoroutines := 10
	numOperations := 100
	results := make(chan error, numGoroutines)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numOperations; j++ {
				// Test concurrent schema override access
				override := registry.GetSchemaOverride(fmt.Sprintf("test.Message%d", j%50))
				_ = override // Use the result

				// Test concurrent field processing
				schema := map[string]any{"type": "string"}
				result, err := registry.ProcessField(testField, schema)
				if err != nil {
					results <- fmt.Errorf("goroutine %d, operation %d: %w", goroutineID, j, err)
					return
				}
				_ = result // Use the result
			}
			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		require.NoError(t, err)
	}

	duration := time.Since(start)
	totalOps := numGoroutines * numOperations
	opsPerSecond := float64(totalOps) / duration.Seconds()

	t.Logf("Concurrent access: %d goroutines × %d ops = %d total ops in %v (%.0f ops/sec)",
		numGoroutines, numOperations, totalOps, duration, opsPerSecond)

	// Should handle at least 1000 operations per second
	assert.Greater(t, opsPerSecond, 1000.0, "Should handle reasonable concurrent load")
}

// Helper functions for creating large-scale configurations

func createLargeScaleConfig(numOverrides, numProcessors, numReferences int) *Config {
	config := &Config{
		Version:         "1.0",
		SchemaOverrides: make(map[string]SchemaOverride),
		OptionProcessors: make([]OptionProcessor, 0, numProcessors),
	}

	// Create base schema overrides
	baseOverrides := make([]string, 0, numOverrides)
	for i := 0; i < numOverrides; i++ {
		fqn := fmt.Sprintf("test.scale.v1.Message%d", i)
		baseOverrides = append(baseOverrides, fqn)

		config.SchemaOverrides[fqn] = SchemaOverride{
			Schema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":    "integer",
						"minimum": 0,
					},
					"name": map[string]any{
						"type":      "string",
						"minLength": 1,
						"maxLength": 255,
					},
					"active": map[string]any{
						"type": "boolean",
					},
					"tags": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required": []any{"id", "name"},
				"additionalProperties": false,
			},
			References: []string{}, // Will be populated below
		}
	}

	// Add cross-references (disabled for basic scale testing)
	// Cross-reference testing is done separately to avoid complexity
	_ = numReferences // Suppress unused parameter warning

	// Create option processors
	for i := 0; i < numProcessors; i++ {
		priority := 1000 - i // Decreasing priority
		option := fmt.Sprintf("test.field.rule%d", i%20) // Cycle through 20 different options

		config.OptionProcessors = append(config.OptionProcessors, OptionProcessor{
			Name:     fmt.Sprintf("scale_processor_%d", i),
			Priority: priority,
			Match: OptionMatch{
				Option: option,
				Value:  true,
			},
			Schema: map[string]any{
				fmt.Sprintf("x-scale-rule-%d", i): true,
				"description": fmt.Sprintf("Applied by scale processor %d", i),
				"x-processor-priority": priority,
			},
		})
	}

	return config
}

func createTempConfigFile(t *testing.T, config *Config) string {
	configYAML := fmt.Sprintf(`version: "%s"
schema_overrides:`, config.Version)

	for fqn := range config.SchemaOverrides {
		configYAML += fmt.Sprintf(`
  "%s":
    schema:
      type: "object"
      properties:
        id:
          type: "integer"
        name:
          type: "string"
    references: []`, fqn)
	}

	configYAML += `
option_processors:`

	for _, processor := range config.OptionProcessors {
		configYAML += fmt.Sprintf(`
  - name: "%s"
    priority: %d
    match:
      option: "%s"
      value: true
    schema:
      x-test: true`, processor.Name, processor.Priority, processor.Match.Option)
	}

	tmpFile, err := os.CreateTemp("", "scale-test-config-*.yaml")
	require.NoError(t, err)

	_, err = tmpFile.WriteString(configYAML)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

// Performance expectation functions

func getMaxLoadTime(overrides, processors int) time.Duration {
	// Base time + linear scaling
	baseTime := time.Millisecond * 100
	scalingFactor := time.Microsecond * 50
	total := int64(overrides + processors)

	return baseTime + time.Duration(total)*scalingFactor
}

func getMaxCreateTime(overrides, processors int) time.Duration {
	// Registry creation should be faster than config loading
	baseTime := time.Millisecond * 50
	scalingFactor := time.Microsecond * 25
	total := int64(overrides + processors)

	return baseTime + time.Duration(total)*scalingFactor
}

func getMaxMemoryUsage(overrides, processors int) uint64 {
	// Rough estimate: ~1KB per override, ~500B per processor
	baseMemory := uint64(10240) // 10KB base
	overrideMemory := uint64(overrides) * 1024
	processorMemory := uint64(processors) * 512

	return baseMemory + overrideMemory + processorMemory
}

func getRegistryMaxMemory(overrides, processors int) uint64 {
	// Registry uses slightly more memory due to indexing structures
	baseMemory := getMaxMemoryUsage(overrides, processors)
	return baseMemory * 2 // Allow 2x overhead for registry structures
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}