# JSON Schema Extension System

This document explains how to use the configuration-based extension system for the protoc-gen-jsonschema plugin.

## Overview

The extension system allows you to:

1. **Custom Type Handlers**: Override how specific protobuf message types are converted to JSON schema
2. **Field Processors**: Modify or enhance individual field schemas during generation
3. **Schema Processors**: Post-process complete message schemas after generation

## Architecture

The extension system uses three main interfaces:

- `TypeHandler`: Function type for handling specific message types
- `FieldProcessor`: Interface for processing individual fields
- `SchemaProcessor`: Interface for processing complete schemas

Extensions are configured via the `ExtensionConfig` struct and applied using the `WithExtension()` GeneratorOption.

## Usage

### YAML Configuration (Recommended)

The easiest way to use extensions is through YAML configuration files:

```yaml
# config.yaml
metadata:
  generator: "my-jsonschema-generator"
  version: "v1.0.0"

field_processors:
  - name: "sensitive_data"
    config:
      additional_sensitive: ["password", "api_key"]

schema_processors:
  - name: "metadata"
    config:
      custom_keys:
        "x-api-version": "v1"
```

```bash
# Use with protoc
protoc --jsonschema-extended_out=config=config.yaml:. myfile.proto
```

See `.claude/YAML_CONFIG_GUIDE.md` for complete YAML configuration documentation.

### Programmatic Configuration

You can also configure extensions programmatically in Go:

```go
import "github.com/bufbuild/protoschema-plugins/internal/protoschema/jsonschema"

config := jsonschema.ExtensionConfig{
    CustomTypes: map[string]func(...) error{
        "google.protobuf.Timestamp": myTimestampHandler,
    },
    FieldProcessors: []jsonschema.FieldProcessor{
        myFieldProcessor,
    },
    SchemaProcessors: []jsonschema.SchemaProcessor{
        mySchemaProcessor,
    },
}

generator := jsonschema.NewGenerator(
    jsonschema.WithExtension(config),
    // other options...
)
```

### Custom Type Handler Example

```go
func customTimestampHandler(desc protoreflect.MessageDescriptor, rules *validate.FieldRules, schema map[string]any) error {
    schema["type"] = "string"
    schema["format"] = "date-time"
    schema["pattern"] = "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$"
    return nil
}
```

### Field Processor Example

```go
type SensitiveDataProcessor struct{}

func (p SensitiveDataProcessor) ProcessField(field protoreflect.FieldDescriptor, schema map[string]any) error {
    if strings.Contains(string(field.Name()), "password") {
        schema["x-sensitive"] = true
        schema["writeOnly"] = true
    }
    return nil
}

func (p SensitiveDataProcessor) Priority() int {
    return 100 // Higher priority runs first
}
```

### Schema Processor Example

```go
type MetadataProcessor struct {
    Version string
}

func (m MetadataProcessor) ProcessSchema(messageName protoreflect.FullName, schema map[string]any) error {
    schema["x-generated-by"] = "protoc-gen-jsonschema-extended"
    schema["x-proto-message"] = string(messageName)
    if m.Version != "" {
        schema["x-version"] = m.Version
    }
    return nil
}
```

## Extended Plugin Usage

The `protoc-gen-jsonschema-extended` binary demonstrates how to use the extension system:

### Build the Extended Plugin

```bash
go build ./cmd/protoc-gen-jsonschema-extended
```

### Use with protoc

```bash
# Basic usage with extensions enabled
protoc --jsonschema-extended_out=use_examples=true:. myfile.proto

# With custom version
protoc --jsonschema-extended_out=use_examples=true,generator_version=v2.0.0:. myfile.proto

# Combined with existing options
protoc --jsonschema-extended_out=use_examples=true,additional_properties=true,target=json-strict:. myfile.proto
```

### Extended Plugin Parameters

The extended plugin supports all original parameters plus:

- `config=<path>`: Load extensions from YAML configuration file
- `use_examples=true`: Enable the built-in example processors
- `generator_version=<version>`: Set a custom version in metadata

**Parameter precedence** (highest to lowest):
1. YAML config file (if `config=` specified)
2. Built-in examples (if `use_examples=true`)
3. Other command-line parameters

## Creating Custom Extensions

### Step 1: Implement Processors

Create your custom processors by implementing the interfaces:

```go
// Custom field processor
type MyFieldProcessor struct{}

func (p MyFieldProcessor) ProcessField(field protoreflect.FieldDescriptor, schema map[string]any) error {
    // Your custom logic here
    return nil
}

func (p MyFieldProcessor) Priority() int {
    return 50
}

// Custom schema processor
type MySchemaProcessor struct{}

func (m MySchemaProcessor) ProcessSchema(messageName protoreflect.FullName, schema map[string]any) error {
    // Your custom logic here
    return nil
}
```

### Step 2: Create Configuration

```go
func createMyExtensionConfig() jsonschema.ExtensionConfig {
    return jsonschema.ExtensionConfig{
        FieldProcessors: []jsonschema.FieldProcessor{
            MyFieldProcessor{},
        },
        SchemaProcessors: []jsonschema.SchemaProcessor{
            MySchemaProcessor{},
        },
    }
}
```

### Step 3: Use with Generator

```go
generator := jsonschema.NewGenerator(
    jsonschema.WithExtension(createMyExtensionConfig()),
)
```

### Step 4: Create Wrapper Plugin (Optional)

Create your own plugin binary by copying and modifying `cmd/protoc-gen-jsonschema-extended/main.go`.

## Processor Priority and Ordering

- **Field Processors**: Run in priority order (highest first). Use this to ensure critical processors run before others.
- **Schema Processors**: Run in the order they appear in the configuration slice.
- **Custom Type Handlers**: Override default message generation entirely.

## Upstream Compatibility

The extension system is designed to maintain upstream compatibility:

1. **No Core Modifications**: Extensions don't modify core generation logic
2. **Additive Only**: Extensions can only add or enhance, not remove existing functionality
3. **Backward Compatible**: Existing code continues to work unchanged
4. **Optional**: Extensions are opt-in and don't affect default behavior

## Examples in Code

See `internal/protoschema/jsonschema/examples.go` for complete working examples of:

- Sensitive data processing
- Validation enhancement
- Metadata addition
- Custom timestamp handling
- Documentation enhancement

## Best Practices

1. **Use Appropriate Priorities**: Set field processor priorities to control execution order
2. **Error Handling**: Always check for and handle errors in processors
3. **Minimal Changes**: Make targeted changes rather than wholesale replacements
4. **Test Extensions**: Verify your extensions work with various proto schemas
5. **Document Behavior**: Document what your extensions do and when to use them

## Migration Path

To move extensions upstream:

1. Implement extensions using this system
2. Test thoroughly with your use cases
3. Create well-documented examples
4. Propose specific extensions for upstream inclusion
5. The extension system makes it easy to transition from external to built-in extensions