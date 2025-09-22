# Architectural Analysis: Extending protobuf JsonSchema Generation

## Executive Summary

This document provides a comprehensive architectural analysis of extension strategies for the protobuf JsonSchema generation project, with a focus on enhancing the YAML configuration system to support complete type replacement with cross-referencing and option-based field processing.

### Key Requirements Addressed

1. **Complete Type Replacement**: The enhanced architecture allows users to specify a protobuf type's FQN and provide a complete JSON schema replacement, stored in a schema registry with dependency tracking.

2. **Cross-Type References**: A two-phase resolution system (registration then resolution) enables schemas to reference each other, with automatic dependency ordering and circular dependency detection.

3. **Option-Based Field Processing**: A declarative rule engine processes fields based on protobuf options without requiring procedural code, using flexible match conditions and direct schema merging in YAML.

### Architectural Highlights

- **Schema Resolution System**: Synchronous, dependency-ordered resolution with JSON pointer reference support
- **Schema Merging Approach**: Direct schema merging instead of complex action systems for better consistency and simplicity
- **Flexible Value Matching**: Rich comparison patterns supporting numeric ranges, patterns, existence checks, and negation
- **Improved Naming**: `schema_overrides` instead of `type_replacements` for clarity
- **Backward Compatibility**: Existing Go-based configuration remains functional during migration
- **Estimated Implementation**: 6 weeks for full feature set with testing and documentation

### Key Architectural Improvements

1. **Schema Merging over Actions**: The option processors use direct schema merging instead of action-based transformations. This provides:
   - **Consistency**: Same pattern as `schema_overrides`
   - **Simplicity**: Direct JSON schema properties instead of action abstractions
   - **Flexibility**: Can define complex nested structures easily
   - **Predictability**: Simple merge semantics vs complex action execution

2. **Flexible Value Matching**: The match system supports rich comparison patterns:
   - Equality, existence checks, negation
   - Numeric comparisons (greater_than, less_than, in_range)
   - String pattern matching with regex
   - Multiple value matching (any_of)
   - Extensible matcher interface for future enhancements

3. **Improved Naming Convention**:
   - `schema_overrides` instead of `type_replacements` - more descriptive of the functionality
   - Emphasizes that these are complete schema definitions that override default generation

## Current Architecture Overview

### Key Components

1. **Entry Point** (`cmd/protoc-gen-jsonschema/main.go`)
   - Minimal main function using `protoplugin.Main()`
   - Delegates to `pluginjsonschema.Handle`

2. **Plugin Handler** (`pluginjsonschema.go`)
   - Parses command-line parameters
   - Creates Generator instances with options
   - Processes protobuf descriptors

3. **Core Generator** (`jsonschema.go`)
   - Contains the main generation logic
   - Has a `custom` map for Well-Known-Types (WKT)
   - Uses `GeneratorOption` functional pattern for configuration
   - Extensible via `makeWktGenerators()` method

### Current Extension Points

1. **GeneratorOption Pattern**: Functional options for configuring the Generator
2. **Custom Type Handlers**: Map of message type handlers for WKTs
3. **Field Visibility**: Comment-based directives (`jsonschema:ignore`, `jsonschema:hide`)

## Enhanced YAML Configuration Architecture

### Overview

The enhanced YAML configuration system extends the current configuration-based approach to support:
1. **Complete Type Replacement**: Define entire JSON schemas for protobuf types
2. **Cross-Type References**: Allow schemas to reference other custom schemas
3. **Option-Based Processing**: Process fields based on protobuf options/annotations
4. **Declarative Configuration**: Pure YAML without procedural code

### Key Architectural Components

#### 1. Schema Definition Registry

A two-phase schema resolution system:
- **Phase 1: Registration** - All custom type schemas are registered with their dependencies
- **Phase 2: Resolution** - Cross-references are resolved after all schemas are loaded

```yaml
# Example YAML configuration
schema_overrides:
  base.schemas.JsonNull:
    schema:
      type: "null"
      description: "Represents a JSON null value"

  base.schemas.JsonValue:
    schema:
      oneOf:
        - $ref: "@base.schemas.JsonNull"  # Runtime-resolved cross-reference
        - type: "string"
        - type: "number"
        - type: "boolean"
        - $ref: "@base.schemas.JsonObject"
        - $ref: "@base.schemas.JsonArray"
    references:
      - "base.schemas.JsonNull"      # Logical dependency (FQN)
      - "base.schemas.JsonObject"    # Logical dependency (FQN)
      - "base.schemas.JsonArray"     # Logical dependency (FQN)
```

#### 2. Option-Based Field Processing

A declarative rule engine for processing fields based on protobuf options with direct schema merging:

```yaml
option_processors:
  - name: "sensitive_field"
    # Match condition with flexible value matching
    match:
      option: "field_options.sensitive"
      value: true
    # Schema to merge with existing field schema
    schema:
      x-sensitive: true
      writeOnly: true
      x-category: "pii"
      pattern: "^[*]+$"  # Masked pattern

  - name: "deprecated_field"
    match:
      option: "field_options.deprecated"
      value:
        exists: true  # Match if option exists regardless of value
    schema:
      deprecated: true
      x-deprecation-message: "This field is deprecated"

  - name: "validation_level_strict"
    match:
      option: "field_options.validation_level"
      value:
        any_of: ["strict", "high"]  # Match multiple values
    schema:
      x-validation: "strict"
      minLength: 1

  - name: "max_length_constraint"
    match:
      option: "field_options.max_length"
      value:
        greater_than: 100  # Numeric comparison
    schema:
      x-large-field: true
      maxLength: 1000
```

#### 3. Flexible Value Matching

The match system supports rich comparison patterns beyond simple equality:

```yaml
# Simple equality
match:
  option: "field_options.sensitive"
  value: true

# Multiple possible values
match:
  option: "field_options.access_level"
  value:
    any_of: ["admin", "moderator"]

# Numeric comparisons
match:
  option: "field_options.max_length"
  value:
    greater_than: 50
    less_than: 200

# String pattern matching
match:
  option: "field_options.format_hint"
  value:
    pattern: "email.*"

# Existence checks
match:
  option: "field_options.custom_validation"
  value:
    exists: true

# Negation
match:
  option: "field_options.visibility"
  value:
    not: "hidden"

# Range matching
match:
  option: "field_options.priority"
  value:
    in_range: [1, 10]
```

#### 4. Reference Resolution Strategy

**Context-Aware Reference Resolution:**

The reference system uses a two-level approach:

1. **Logical Dependencies** (`references` array): Lists FQNs of dependent types for dependency ordering
2. **Runtime References** (`$ref` with `@FQN` syntax): Resolved to actual schema IDs based on generator options

**Reference Resolution Process:**

```go
type SchemaResolver struct {
    schemas         map[string]SchemaDefinition
    resolved        map[string]map[string]any
    resolving       map[string]bool // For circular dependency detection
    generatorOptions GeneratorOptions // Current generation context
}

func (r *SchemaResolver) resolveReferences(schema map[string]any) error {
    // Find all "@FQN" patterns in $ref fields
    refs := r.findRuntimeReferences(schema)

    for refPath, fqn := range refs {
        // Convert FQN to actual schema ID based on generator options
        actualID := r.generateSchemaID(fqn, r.generatorOptions)

        // Update the $ref to point to the actual schema
        if r.generatorOptions.Bundle {
            // Bundle mode: internal reference
            r.updateReference(schema, refPath, "#/$defs/"+actualID)
        } else {
            // Non-bundle mode: external file reference
            r.updateReference(schema, refPath, actualID)
        }
    }
    return nil
}

func (r *SchemaResolver) generateSchemaID(fqn string, opts GeneratorOptions) string {
    id := fqn
    if opts.UseJSONNames {
        id += ".jsonschema"
    } else {
        id += ".schema"
    }
    if opts.Strict {
        id += ".strict"
    }
    if opts.Bundle {
        id += ".bundle"
    }
    return id + ".json"
}
```

**Example Resolution:**

Given generator options `{UseJSONNames: true, Strict: true, Bundle: false}`:

```yaml
# Input YAML
schema_overrides:
  base.schemas.JsonValue:
    schema:
      oneOf:
        - $ref: "@base.schemas.JsonNull"
    references:
      - "base.schemas.JsonNull"
```

**Resolves to (Non-Bundle):**
```json
// In base.schemas.JsonValue.jsonschema.strict.json
{
  "oneOf": [
    { "$ref": "base.schemas.JsonNull.jsonschema.strict.json" }
  ]
}
```

**Resolves to (Bundle):**
```json
// In base.schemas.JsonValue.jsonschema.strict.bundle.json
{
  "$defs": {
    "base.schemas.JsonNull.jsonschema.strict.json": {
      "type": "null",
      "description": "Represents a JSON null value"
    },
    "base.schemas.JsonValue.jsonschema.strict.json": {
      "oneOf": [
        { "$ref": "#/$defs/base.schemas.JsonNull.jsonschema.strict.json" }
      ]
    }
  }
}
```

### Architectural Changes Required

#### 1. Generator Enhancement

```go
type Generator struct {
    // Existing fields...

    // New fields for enhanced YAML support
    schemaResolver   *SchemaResolver
    typeReplacements map[string]SchemaDefinition
    optionProcessors []OptionBasedProcessor
}
```

#### 2. YAML Configuration Structure

```yaml
version: "1.0"  # Configuration schema version

# Complete type schema overrides
schema_overrides:
  <FQN>:
    schema: <JSON Schema>
    references: [<FQN>, ...]  # Dependencies for cross-references

# Option-based field processors with schema merging
option_processors:
  - name: <processor_name>
    priority: <int>
    match:
      option: <option_path>
      value: <match_pattern>  # Supports flexible matching
    schema: <JSON Schema>  # Merged with existing field schema

# Existing configuration sections
field_processors: [...]
schema_processors: [...]
metadata: {...}
```

#### 3. Processing Pipeline

1. **Load Phase**: Parse YAML configuration
2. **Registration Phase**: Register all custom schemas and processors
3. **Resolution Phase**: Resolve cross-references between schemas
4. **Generation Phase**: Generate schemas with custom types and processors
5. **Post-Processing Phase**: Apply schema processors

### Trade-offs Analysis

#### YAML Configuration vs Go-based Approach

**Advantages of YAML:**
- **Accessibility**: Non-programmers can modify configurations
- **Portability**: Configuration can be shared across projects
- **Hot-reload potential**: Can reload without recompilation
- **Validation**: Schema validation for configuration files
- **Version control**: Easier to track changes in configuration

**Disadvantages of YAML:**
- **Limited Logic**: Complex transformations harder to express
- **Performance**: Runtime parsing overhead
- **Type Safety**: Loss of compile-time type checking
- **Debugging**: Harder to debug configuration errors
- **Expressiveness**: Some patterns impossible to express declaratively

#### Synchronous vs Asynchronous Resolution

**Synchronous (Recommended):**
- Simpler mental model
- Easier error handling
- Predictable resolution order
- Better for detecting circular dependencies

**Asynchronous:**
- Could parallelize independent resolutions
- More complex implementation
- Harder to debug
- Minimal performance benefit for typical schema counts

## Extension Strategy Options

### Option 1: Configuration-Based Extension (Recommended for Most Cases)

**Use Cases:**
- Complete type replacement with custom JSON schemas
- Cross-referencing between types in configuration
- Option-based field processing
- Adding new output formats or schema versions
- Configuring field naming strategies
- Setting validation rules
- Adjusting schema generation behavior

**Implementation Approach:**

```go
// New file: jsonschema_config.go
package jsonschema

// ExtensionConfig represents external configuration for schema generation
type ExtensionConfig struct {
    CustomTypes    map[string]func(desc protoreflect.MessageDescriptor, rules *validate.FieldRules, schema map[string]any) error
    FieldProcessors []FieldProcessor
    SchemaProcessors []SchemaProcessor
    OutputFormats  map[string]OutputFormatter

    // New: Complete type replacements with cross-referencing
    SchemaOverrides map[string]SchemaDefinition

    // New: Option-based field processors
    OptionProcessors []OptionBasedProcessor
}

// SchemaDefinition represents a complete JSON schema replacement
type SchemaDefinition struct {
    Schema     map[string]any // The complete JSON schema
    References []string       // FQNs of types this schema references
}

// OptionBasedProcessor processes fields based on protobuf options with schema merging
type OptionBasedProcessor interface {
    // Match returns true if this processor should handle a field based on its options
    Match(field protoreflect.FieldDescriptor, options protoreflect.ProtoMessage) (bool, error)

    // GetSchemaToMerge returns the schema fragment to merge with the existing field schema
    GetSchemaToMerge(field protoreflect.FieldDescriptor, options protoreflect.ProtoMessage) (map[string]any, error)

    Priority() int
}

// ValueMatcher represents different ways to match option values
type ValueMatcher interface {
    Match(value any) (bool, error)
}

// Simple implementations of ValueMatcher
type ExactMatcher struct{ Expected any }
type AnyOfMatcher struct{ Values []any }
type ExistsMatcher struct{}
type PatternMatcher struct{ Pattern string }
type NumericMatcher struct{
    GreaterThan *float64
    LessThan    *float64
    InRange     []float64
}
type NotMatcher struct{ Inner ValueMatcher }

// SchemaResolver handles cross-references between schemas
type SchemaResolver interface {
    // Register a schema definition
    RegisterSchema(fqn string, definition SchemaDefinition) error

    // Resolve references in a schema
    ResolveReferences(schema map[string]any) error

    // Get a resolved schema by FQN
    GetResolvedSchema(fqn string) (map[string]any, error)
}

// FieldProcessor modifies field schema generation
type FieldProcessor interface {
    ProcessField(field protoreflect.FieldDescriptor, schema map[string]any) error
    Priority() int // For ordering multiple processors
}

// SchemaProcessor post-processes complete schemas
type SchemaProcessor interface {
    ProcessSchema(messageName protoreflect.FullName, schema map[string]any) error
}

// OutputFormatter formats the final output
type OutputFormatter interface {
    Format(schemas map[protoreflect.FullName]map[string]any) ([]byte, error)
}

// WithExtension creates a GeneratorOption from ExtensionConfig
func WithExtension(config ExtensionConfig) GeneratorOption {
    return func(g *Generator) {
        // Initialize schema resolver for cross-references
        if len(config.SchemaOverrides) > 0 {
            g.schemaResolver = NewSchemaResolver()
            for fqn, def := range config.SchemaOverrides {
                g.schemaResolver.RegisterSchema(fqn, def)
            }
        }

        // Merge custom types
        for name, handler := range config.CustomTypes {
            g.custom[protoreflect.FullName(name)] = handler
        }

        // Add field processors
        g.fieldProcessors = append(g.fieldProcessors, config.FieldProcessors...)

        // Add option-based processors
        g.optionProcessors = append(g.optionProcessors, config.OptionProcessors...)

        // Add schema processors
        g.schemaProcessors = append(g.schemaProcessors, config.SchemaProcessors...)

        // Add output formats
        for name, formatter := range config.OutputFormats {
            g.outputFormats[name] = formatter
        }
    }
}
```

**Upstream Compatibility:**
- Extend Generator struct with new fields (backward compatible)
- Use existing GeneratorOption pattern
- No changes to main.go or plugin handler needed
- Can be loaded from JSON/YAML config file

### Option 2: Plugin-Based Extension (For Complex/Dynamic Requirements)

**Use Cases:**
- Customer-specific schema transformations
- Integration with external validation systems
- Dynamic loading of extensions at runtime
- Isolation of proprietary logic

**Implementation Approach:**

```go
// New file: jsonschema_plugin.go
package jsonschema

import "plugin"

// Plugin interface for dynamically loaded extensions
type SchemaPlugin interface {
    // Initialize is called once when plugin is loaded
    Initialize(config map[string]any) error

    // GetCustomTypes returns custom type handlers
    GetCustomTypes() map[string]TypeHandler

    // GetFieldProcessors returns field processors
    GetFieldProcessors() []FieldProcessor

    // GetSchemaProcessors returns schema processors
    GetSchemaProcessors() []SchemaProcessor

    // Version returns the plugin version for compatibility checks
    Version() string
}

// LoadPlugin loads a plugin from a .so file
func LoadPlugin(path string) (SchemaPlugin, error) {
    p, err := plugin.Open(path)
    if err != nil {
        return nil, err
    }

    symPlugin, err := p.Lookup("Plugin")
    if err != nil {
        return nil, err
    }

    return symPlugin.(SchemaPlugin), nil
}

// WithPlugins creates a GeneratorOption that loads plugins
func WithPlugins(pluginPaths ...string) GeneratorOption {
    return func(g *Generator) {
        for _, path := range pluginPaths {
            plugin, err := LoadPlugin(path)
            if err != nil {
                // Log error or handle appropriately
                continue
            }

            // Apply plugin extensions
            config := ExtensionConfig{
                CustomTypes: plugin.GetCustomTypes(),
                FieldProcessors: plugin.GetFieldProcessors(),
                SchemaProcessors: plugin.GetSchemaProcessors(),
            }

            WithExtension(config)(g)
        }
    }
}
```

### Option 3: Wrapper Binary Approach (For Complete Control)

**Use Cases:**
- Need to modify the protoc plugin protocol
- Want to add pre/post processing steps
- Need to integrate with existing toolchains
- Want to maintain separate release cycles

**Implementation:**

```go
// New file: cmd/protoc-gen-jsonschema-extended/main.go
package main

import (
    "github.com/bufbuild/protoplugin"
    "github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"
    "github.com/yourorg/extensions"
)

func main() {
    // Create extended generator with your customizations
    handler := createExtendedHandler()
    protoplugin.Main(handler)
}

func createExtendedHandler() protoplugin.Handler {
    return protoplugin.HandlerFunc(func(
        ctx context.Context,
        env protoplugin.PluginEnv,
        responseWriter protoplugin.ResponseWriter,
        request protoplugin.Request,
    ) error {
        // Parse extended parameters
        config := parseExtendedConfig(request.Parameter())

        // Create generators with extensions
        opts := []jsonschema.GeneratorOption{
            jsonschema.WithExtension(config.ToExtensionConfig()),
        }

        // Reuse existing generation logic
        return handleWithOptions(ctx, env, responseWriter, request, opts)
    })
}
```

## Specific Interface Capabilities

### TypeHandler Interface
**Capabilities:**
- Complete control over schema generation for specific message types
- Access to message descriptor and validation rules
- Can override default generation logic
- Can add custom JSON Schema properties

**Example:**
```go
func customTimestampHandler(desc protoreflect.MessageDescriptor, rules *validate.FieldRules, schema map[string]any) error {
    schema["type"] = "string"
    schema["format"] = "date-time"
    schema["pattern"] = "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$"
    return nil
}
```

### FieldProcessor Interface
**Capabilities:**
- Modify schema for individual fields
- Add custom validation rules
- Transform field names or types
- Priority-based ordering for multiple processors

**Example:**
```go
type SensitiveDataProcessor struct{}

func (p SensitiveDataProcessor) ProcessField(field protoreflect.FieldDescriptor, schema map[string]any) error {
    if strings.Contains(string(field.Name()), "ssn") {
        schema["pattern"] = "^\\d{3}-\\d{2}-\\d{4}$"
        schema["x-sensitive"] = true
    }
    return nil
}

func (p SensitiveDataProcessor) Priority() int { return 100 }
```

### SchemaProcessor Interface
**Capabilities:**
- Post-process complete message schemas
- Add schema-level metadata
- Apply organization-wide policies
- Cross-reference other schemas

**Example:**
```go
type MetadataProcessor struct{}

func (m MetadataProcessor) ProcessSchema(name protoreflect.FullName, schema map[string]any) error {
    schema["x-generated-by"] = "protoc-gen-jsonschema-extended"
    schema["x-proto-message"] = string(name)
    return nil
}
```

## Maintaining Upstream Compatibility

### Strategy 1: Extension-Only Approach
**How it works:**
- Never modify upstream files directly
- Use composition and delegation
- Extend via interfaces and configuration

**Implementation:**
```go
// Extend Generator without modifying it
type ExtendedGenerator struct {
    *jsonschema.Generator
    extensions ExtensionConfig
}

func NewExtendedGenerator(opts ...jsonschema.GeneratorOption) *ExtendedGenerator {
    base := jsonschema.NewGenerator(opts...)
    return &ExtendedGenerator{
        Generator: base,
        extensions: loadExtensions(),
    }
}
```

### Strategy 2: Fork with Merge Strategy
**How it works:**
- Maintain minimal fork with clear boundaries
- Regular upstream merges
- Extensions in separate packages

**Directory Structure:**
```
protoschema-plugins/
├── upstream/          # Unmodified upstream code
├── extensions/        # Your extensions
│   ├── config.go
│   ├── processors.go
│   └── handlers.go
└── cmd/
    └── protoc-gen-jsonschema-extended/
        └── main.go    # Your entry point
```

### Strategy 3: Configuration Injection
**How it works:**
- Use environment variables or config files
- Inject configuration at runtime
- No code changes needed

**Example:**
```bash
# Use environment variable for config
export JSONSCHEMA_CONFIG=/path/to/config.yaml
protoc --jsonschema_out=. myfile.proto

# Or use parameter
protoc --jsonschema_out=config=/path/to/config.yaml:. myfile.proto
```

## Recommended Approach

For most use cases, I recommend **Option 1 (Configuration-Based)** with the following implementation:

1. **Create Extension Package**
```go
// extensions/config.go
package extensions

import "github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"

func LoadFromFile(path string) (jsonschema.GeneratorOption, error) {
    // Load configuration from YAML/JSON
    config := parseConfigFile(path)

    // Create extension configuration
    extConfig := jsonschema.ExtensionConfig{
        CustomTypes: config.CustomTypes,
        FieldProcessors: config.FieldProcessors,
    }

    return jsonschema.WithExtension(extConfig), nil
}
```

2. **Create Wrapper Binary**
```go
// cmd/protoc-gen-jsonschema-custom/main.go
package main

func main() {
    // Load extensions
    ext, _ := extensions.LoadFromFile("config.yaml")

    // Create handler with extensions
    handler := createHandlerWithExtensions(ext)

    protoplugin.Main(handler)
}
```

3. **Use in Build Process**
```bash
# Install your extended version
go install ./cmd/protoc-gen-jsonschema-custom

# Use with protoc
protoc --jsonschema-custom_out=. myfile.proto
```

## Benefits of Recommended Approach

1. **Clear Separation**: Extensions are separate from core logic
2. **Upstream Compatible**: Easy to pull upstream changes
3. **Testable**: Each component can be tested independently
4. **Flexible**: Can switch between config and plugins
5. **Maintainable**: Clear boundaries and interfaces

## Implementation Complexity and Migration Strategy

### Implementation Complexity Analysis

#### 1. Schema Resolution System (Medium-High Complexity)

**Components to Build:**
- Dependency graph builder for topological sorting
- Circular dependency detector
- Reference resolver with JSON pointer support
- Schema validation engine

**Estimated Effort:** 2-3 weeks for robust implementation
- Core resolver: 1 week
- Error handling and validation: 1 week
- Testing and edge cases: 1 week

#### 2. Option-Based Processing Engine (Medium Complexity)

**Components to Build:**
- Option extractor from protobuf descriptors
- Declarative rule matcher
- Action executor framework
- Template variable resolver

**Estimated Effort:** 1-2 weeks
- Rule engine: 3-4 days
- Action framework: 3-4 days
- Testing: 3-4 days

#### 3. YAML Configuration Parser (Low-Medium Complexity)

**Components to Build:**
- Enhanced YAML schema definition
- Validation against configuration schema
- Error reporting with line numbers

**Estimated Effort:** 3-5 days
- Parser enhancement: 2 days
- Validation: 2 days
- Testing: 1 day

### Migration Strategy

#### Phase 1: Foundation (Week 1-2)
**Goal:** Establish core infrastructure without breaking existing functionality

1. **Implement SchemaResolver Interface**
   ```go
   // New file: internal/protoschema/jsonschema/resolver.go
   type SchemaResolver struct {
       schemas    map[string]SchemaDefinition
       resolved   map[string]map[string]any
       dependency *DependencyGraph
   }
   ```

2. **Extend YAMLConfig Structure**
   ```go
   // Update: internal/protoschema/jsonschema/config.go
   type YAMLConfig struct {
       Version          string                         `yaml:"version"`
       SchemaOverrides map[string]SchemaOverrideConfig `yaml:"schema_overrides"`
       OptionProcessors []OptionProcessorConfig         `yaml:"option_processors"`
       // ... existing fields
   }
   ```

3. **Create Backward-Compatible Generator Update**
   - Add new fields to Generator struct
   - Ensure existing configurations still work

#### Phase 2: Type Replacement System (Week 3-4)
**Goal:** Implement complete type replacement with cross-referencing

1. **Build Dependency Resolution**
   - Implement topological sort for resolution order
   - Add circular dependency detection
   - Create reference resolution logic

2. **Integration with Generator**
   - Hook SchemaResolver into generation pipeline
   - Override default type handlers with replacements
   - Add fallback to existing handlers

3. **Testing Infrastructure**
   - Unit tests for resolver
   - Integration tests with sample schemas
   - Performance benchmarks

#### Phase 3: Option-Based Processing (Week 5)
**Goal:** Add declarative option-based field processing

1. **Implement Rule Engine**
   - Create option extraction utilities
   - Build rule matching system
   - Implement action executors

2. **YAML Configuration Support**
   - Parse option processor configurations
   - Convert to runtime processors
   - Add validation

3. **Testing**
   - Test with common protobuf options
   - Verify priority handling
   - Edge case testing

#### Phase 4: Production Readiness (Week 6)
**Goal:** Polish, document, and prepare for deployment

1. **Error Handling Enhancement**
   - Comprehensive error messages
   - Configuration validation CLI tool
   - Debug mode for troubleshooting

2. **Documentation**
   - Configuration schema documentation
   - Migration guide for existing users
   - Example configurations

3. **Performance Optimization**
   - Cache resolved schemas
   - Optimize resolution algorithm
   - Profile and optimize hot paths

### Migration Path for Existing Users

#### Step 1: Audit Current Customizations
```bash
# Identify current custom type handlers
grep -r "CustomTypes\[" .
grep -r "WithExtension" .

# Document existing processors
find . -name "*.go" | xargs grep "FieldProcessor\|SchemaProcessor"
```

#### Step 2: Create YAML Configuration
```yaml
# Migrate Go-based handlers to YAML
version: "1.0"

# Convert existing custom type handlers to schema overrides
schema_overrides:
  google.protobuf.Timestamp:
    schema:
      type: "string"
      format: "date-time"
      pattern: "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$"
      x-custom-timestamp: true

  # Example with cross-references
  my.types.UserProfile:
    schema:
      type: "object"
      properties:
        user:
          $ref: "#/definitions/User"
        preferences:
          $ref: "#/definitions/UserPreferences"
      required: ["user"]
    references:
      - "my.types.User"
      - "my.types.UserPreferences"

# Convert existing field processors to option-based processors
option_processors:
  - name: "sensitive_data"
    priority: 100
    match:
      option: "field_options.sensitive"
      value: true
    schema:
      x-sensitive: true
      writeOnly: true
      pattern: "^[*]+$"  # Masked display pattern

  - name: "validation_level"
    priority: 90
    match:
      option: "field_options.validation"
      value:
        any_of: ["strict", "high"]
    schema:
      x-validation-level: "strict"
      minLength: 1

  - name: "large_text_fields"
    priority: 50
    match:
      option: "field_options.max_length"
      value:
        greater_than: 1000
    schema:
      x-large-field: true
      format: "textarea"
      maxLength: 10000
```

#### Step 3: Gradual Migration
1. **Run in parallel mode** (both Go and YAML configs)
2. **Validate outputs match** between old and new systems
3. **Gradually move logic** from Go to YAML
4. **Deprecate Go-based** configuration over 2-3 releases

#### Step 4: Tooling Support
```bash
# Validation tool
protoc-gen-jsonschema-validate config.yaml

# Migration assistant
protoc-gen-jsonschema-migrate --from-go custom.go --to-yaml config.yaml

# Schema generator for IDE support
protoc-gen-jsonschema-schema > config-schema.json
```

### Risk Mitigation

1. **Feature Flag Control**
   - `--enable-yaml-config` flag for opt-in
   - Gradual rollout to specific teams

2. **Comprehensive Testing**
   - Snapshot testing for schema outputs
   - Regression test suite
   - Performance benchmarks

3. **Rollback Strategy**
   - Keep Go-based system functional
   - Version configuration files
   - Clear deprecation timeline

### Success Metrics

1. **Adoption Rate**: 50% of users migrated within 6 months
2. **Configuration Errors**: <5% error rate in YAML configs
3. **Performance**: <10% overhead vs Go-based approach
4. **User Satisfaction**: Positive feedback on ease of use

## Migration Path

1. **Phase 1**: Implement ExtensionConfig and interfaces in jsonschema package
2. **Phase 2**: Create extension package with custom handlers
3. **Phase 3**: Create wrapper binary if needed
4. **Phase 4**: Migrate existing customizations to new system

## Conclusion

The configuration-based approach with clear interfaces provides the best balance of flexibility, maintainability, and upstream compatibility. It allows for extensive customization without modifying core files, making it easy to track upstream changes while maintaining your extensions.
