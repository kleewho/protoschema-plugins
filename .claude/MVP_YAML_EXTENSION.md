# MVP YAML Extension System for protoc-gen-jsonschema

## 1. MVP Scope Definition

### What Gets Implemented
- **YAML Configuration Loading**: Single YAML file with schema overrides and option processors
- **Schema Override System**: Replace entire JSON Schema definitions for specific protobuf types
- **Option Processor System**: Transform fields based on protobuf options (e.g., `[(validator.field).required = true]`)
- **Basic Reference Resolution**: Support `@FQN` references between definitions
- **Integration Hook**: Single entry point in existing converter logic

### What's Explicitly Excluded
- Multiple configuration files or includes
- Complex matching patterns beyond exact FQN matching
- Plugin architecture or dynamic loading
- Migration tools or configuration validation CLI
- Performance optimizations or caching
- Extensive error recovery (fail fast on errors)

## 2. Simplified Architecture

```
┌─────────────────┐
│  YAML Config    │
│  (config.yaml)  │
└────────┬────────┘
         │ Load & Parse
         ▼
┌─────────────────┐
│ ConfigLoader    │
│ - LoadConfig()  │
│ - Validate()    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────┐
│      ExtensionRegistry          │
│ - schemaOverrides map           │
│ - optionProcessors []           │
│ - ResolveReferences()           │
└────────┬───────────────┬────────┘
         │               │
         ▼               ▼
┌─────────────────┐ ┌──────────────┐
│ SchemaOverride  │ │OptionProc   │
│ ApplyOverride() │ │ Process()    │
└─────────────────┘ └──────────────┘
         │               │
         └───────┬───────┘
                 ▼
        ┌─────────────────┐
        │ Existing        │
        │ Converter       │
        │ (Modified)      │
        └─────────────────┘
```

### Core Components

1. **ConfigLoader** (New)
   - Loads YAML from fixed location: `jsonschema.extensions.yaml`
   - Basic YAML structure validation
   - Returns parsed configuration

2. **ExtensionRegistry** (New)
   - Holds loaded configuration
   - Provides lookup methods for overrides and processors
   - Resolves `@FQN` references at load time

3. **Integration Points** (Modified)
   - Hook in `Converter.ConvertMessage()` for schema overrides
   - Hook in `Converter.ConvertField()` for option processors

## 3. Implementation Plan

### Week 1: Foundation (5 days)

**Day 1-2: YAML Schema & Loader**
- Define YAML structure (see section 4)
- Implement `ConfigLoader` with basic YAML parsing
- Add configuration validation
- **Files**: `extensions/config.go`, `extensions/types.go`

**Day 3-4: Schema Override System**
- Implement schema override registry
- Add FQN-based lookup
- Integrate into `ConvertMessage()`
- **Files**: `extensions/schema_override.go`, modify `converter.go`

**Day 5: Basic Testing**
- Unit tests for config loading
- Integration test with sample override
- **Files**: `extensions/config_test.go`, test fixtures

### Week 2: Option Processors (5 days)

**Day 6-7: Option Processor Framework**
- Implement option matcher logic
- Create processor interface
- Add built-in processors (required, pattern, min/max)
- **Files**: `extensions/option_processor.go`

**Day 8-9: Integration & Reference Resolution**
- Hook processors into `ConvertField()`
- Implement `@FQN` reference resolution
- Handle circular reference detection
- **Files**: Modify `converter.go`, `extensions/resolver.go`

**Day 10: Testing & Documentation**
- End-to-end tests with real protobuf files
- Basic usage documentation
- **Files**: Integration tests, README update

### Week 3: Polish & Edge Cases (3 days)

**Day 11-12: Error Handling & Validation**
- Improve error messages
- Add configuration validation
- Handle missing references gracefully

**Day 13: Final Testing**
- Test with existing test suite
- Ensure backward compatibility
- Performance smoke test

## 4. YAML Schema

```yaml
# jsonschema.extensions.yaml
version: "1.0"

schema_overrides:
  # Complete replacement for a type
  "example.v1.Money":
    type: "object"
    properties:
      amount:
        type: "string"
        pattern: "^-?\\d+\\.\\d{2}$"
      currency:
        type: "string"
        enum: ["USD", "EUR", "GBP"]
    required: ["amount", "currency"]
    additionalProperties: false

  # Using references to other definitions
  "example.v1.Price":
    allOf:
      - "@example.v1.Money"
      - type: "object"
        properties:
          tax_included:
            type: "boolean"

option_processors:
  # Process fields with specific options
  - option: "validator.field"
    conditions:
      required: true
    actions:
      - add_to_required: true

  - option: "validator.field"
    conditions:
      pattern: ".*"  # any pattern value
    actions:
      - set_property:
          name: "pattern"
          value: "${option.value}"  # use the option's value

  - option: "validator.field"
    conditions:
      min: ".*"
    actions:
      - set_property:
          name: "minimum"
          value: "${option.value}"
```

## 5. Integration Strategy

### Minimal Code Changes

1. **Converter Initialization**
```go
// In NewConverter() or similar
converter.extensions = extensions.LoadRegistry("jsonschema.extensions.yaml")
```

2. **Message Conversion Hook**
```go
// In ConvertMessage()
if override := c.extensions.GetSchemaOverride(messageFQN); override != nil {
    return override, nil
}
// ... existing logic
```

3. **Field Conversion Hook**
```go
// In ConvertField()
schema := // ... existing schema generation
for _, processor := range c.extensions.GetOptionProcessors(field) {
    schema = processor.Process(field, schema)
}
return schema
```

### File Structure
```
protoc-gen-jsonschema/
├── extensions/
│   ├── config.go          # YAML loading
│   ├── types.go           # Data structures
│   ├── registry.go        # ExtensionRegistry
│   ├── schema_override.go # Override logic
│   ├── option_processor.go# Option processing
│   └── resolver.go        # Reference resolution
├── converter.go           # Modified with hooks
└── jsonschema.extensions.yaml # User configuration
```

## 6. Success Criteria

### Functional Requirements
- ✅ Can override any message schema via YAML configuration
- ✅ Can process field options to modify JSON Schema properties
- ✅ References between definitions work (`@FQN` syntax)
- ✅ Existing tests continue to pass (backward compatibility)

### Technical Validation
1. **Schema Override Test**: Define custom schema for `google.protobuf.Timestamp`
2. **Option Processor Test**: Add `required` based on validator option
3. **Reference Test**: Create schema that references another custom schema
4. **Integration Test**: Run against existing protobuf files without configuration

### Performance Baseline
- Configuration loading < 100ms for typical file
- No measurable impact on conversion without configuration
- Linear scaling with number of overrides/processors

## 7. Future Extension Path

### Immediate Next Steps (Post-MVP)
1. **Configuration Validation CLI**: `--validate-config` flag
2. **Multiple Config Files**: Support includes/imports
3. **More Built-in Processors**: Common validation patterns

### Medium-term Enhancements
1. **Go Interface Extensions**: Allow Go plugins for complex logic
2. **Pattern Matching**: Wildcards and regex in FQN matching
3. **Configuration Generation**: From existing schemas

### Long-term Vision
1. **Plugin System**: Dynamic loading of extension modules
2. **Migration Tools**: Convert between configuration formats
3. **Schema Library**: Shared repository of common overrides

## Implementation Notes

### Key Decisions
1. **Single YAML file**: Simplifies loading and debugging
2. **Fixed filename**: No configuration flags initially
3. **Fail-fast errors**: Clear error messages, no partial processing
4. **Load-time resolution**: All references resolved on startup
5. **No hot reload**: Requires regeneration for config changes

### Risk Mitigation
- Keep existing code paths unchanged when no config present
- Use feature flag or build tag if needed for gradual rollout
- Extensive logging for debugging configuration issues
- Clear error messages pointing to configuration problems

### Testing Strategy
1. Unit tests for each component
2. Integration tests with sample protobuf files
3. Backward compatibility tests with existing suite
4. Manual testing with real-world schemas

## Example Usage

Given this protobuf:
```protobuf
message User {
  string email = 1 [(validator.field).required = true,
                     (validator.field).pattern = "^[a-z]+@[a-z]+\\.[a-z]+$"];
  Money balance = 2;
}
```

With the YAML configuration above, generates:
```json
{
  "type": "object",
  "properties": {
    "email": {
      "type": "string",
      "pattern": "^[a-z]+@[a-z]+\\.[a-z]+$"
    },
    "balance": {
      "type": "object",
      "properties": {
        "amount": {
          "type": "string",
          "pattern": "^-?\\d+\\.\\d{2}$"
        },
        "currency": {
          "type": "string",
          "enum": ["USD", "EUR", "GBP"]
        }
      },
      "required": ["amount", "currency"],
      "additionalProperties": false
    }
  },
  "required": ["email"]
}
```

This MVP provides a working foundation that can be extended incrementally while delivering immediate value for schema customization needs.