# JSON Schema Extensions - Troubleshooting Guide

## Diagnostic Tools

### 1. Configuration Validation

Always start with configuration validation:

```bash
# Basic validation
./jsonschema-validate-config --config=config.yaml

# Verbose output with detailed information
./jsonschema-validate-config --config=config.yaml --verbose

# JSON output for scripting
./jsonschema-validate-config --config=config.yaml --json
```

### 2. Debug Configuration

Create a debug configuration to trace processing:

```yaml
version: "1.0"
schema_overrides: {}
option_processors:
  - name: "trace_all_fields"
    priority: 1
    match:
      option: "*"
      value: "*"
    schema:
      x-debug-processed: true
      x-debug-option-found: "YES"
```

This adds debug markers to any field that has options.

### 3. Option Inspection

Use integration tests to inspect actual option structures:

```go
// In a test file
func TestInspectOptions(t *testing.T) {
    desc := (&YourMessage{}).ProtoReflect().Descriptor()
    fields := desc.Fields()

    for i := 0; i < fields.Len(); i++ {
        field := fields.Get(i)
        if field.Options() != nil {
            t.Logf("Field %s options: %+v", field.Name(), field.Options())
        }
    }
}
```

## Common Problems and Solutions

### Problem: Processor Not Applied

**Symptoms:**
- Expected schema modifications don't appear
- Debug markers not added to fields
- Option processor seems ignored

**Diagnostic Steps:**

1. **Check option path matching:**
   ```yaml
   # Test with wildcard matching first
   match:
     option: "*"
     value: "*"
   ```

2. **Verify option structure:**
   ```bash
   # Run with debug processor to see what options exist
   go test -v ./path/to/tests -run TestInspectOptions
   ```

3. **Check processor priority:**
   ```yaml
   # Ensure priority is high enough
   - name: "test_processor"
     priority: 100  # Higher numbers processed first
   ```

**Common Solutions:**

- **Incorrect option path:** Protobuf options may have different paths than expected
  ```yaml
  # Wrong
  match:
    option: "validate.field.string.min_len"

  # Might be correct
  match:
    option: "buf.validate.field"  # Top-level option with nested structure
  ```

- **Value type mismatch:** Option values may be different types
  ```yaml
  # Try wildcard matching first
  match:
    value: "*"
  ```

- **Case sensitivity:** Option names are case-sensitive
  ```yaml
  match:
    option: "buf.validate.field"  # Not "Buf.Validate.Field"
  ```

### Problem: Schema Override Not Applied

**Symptoms:**
- Default schema generated instead of custom override
- Override configuration validates but doesn't take effect

**Diagnostic Steps:**

1. **Verify message name:**
   ```bash
   # Check the exact fully qualified name
   protoc --descriptor_set_out=temp.desc --include_imports *.proto
   protoc --decode_raw < temp.desc | grep -A 5 -B 5 "YourMessageName"
   ```

2. **Check message is being processed:**
   ```go
   // In generator code, add logging
   schemas := generator.Generate()
   for name := range schemas {
       log.Printf("Generated schema: %s", name)
   }
   ```

**Common Solutions:**

- **Incorrect fully qualified name:**
  ```yaml
  # Wrong
  schema_overrides:
    "MessageName":  # Missing package

  # Correct
  schema_overrides:
    "your.package.v1.MessageName":
  ```

- **Message not included in generation:**
  ```bash
  # Ensure the .proto file containing the message is being processed
  protoc --jsonschema-extended_out=config=config.yaml:. your_file.proto
  ```

### Problem: Template Substitution Fails

**Symptoms:**
- Literal `${option.field}` appears in output
- Template variables not replaced with actual values

**Diagnostic Steps:**

1. **Check template variable path:**
   ```yaml
   # Add debug info to see option structure
   schema:
     x-debug-option: "${option}"  # Shows entire option structure
     x-debug-field: "${option.field}"
   ```

2. **Verify option value type:**
   ```yaml
   # Test with simple template first
   schema:
     x-test: "${option}"
   ```

**Common Solutions:**

- **Incorrect template path:**
  ```yaml
  # Wrong - nested field access
  schema:
    minLength: "${option.string.min_len}"

  # Might be correct - depending on option structure
  schema:
    minLength: "${option.min_len}"
  ```

- **Missing option value:**
  ```yaml
  # Provide fallback or make template conditional
  schema:
    description: "Validation: ${option.description:default description}"
  ```

### Problem: Reference Resolution Errors

**Symptoms:**
- Configuration validation fails with "referenced schema not found"
- Circular dependency errors

**Diagnostic Steps:**

1. **Check reference syntax:**
   ```yaml
   # Correct syntax
   properties:
     field: { $ref: "@package.v1.MessageName" }
   references: ["@package.v1.MessageName"]
   ```

2. **Verify all references exist:**
   ```bash
   # List all override names
   ./jsonschema-validate-config --config=config.yaml --verbose | grep -A 10 "Summary"
   ```

**Common Solutions:**

- **Typo in reference name:**
  ```yaml
  # Check spelling carefully
  references: ["@package.v1.MessageName"]  # Not "MessageNam"
  ```

- **Circular dependencies:**
  ```yaml
  # Redesign to avoid cycles
  # Instead of A -> B -> A, use A -> C, B -> C
  ```

- **Missing schema definition:**
  ```yaml
  # Ensure referenced schema is defined
  schema_overrides:
    "package.v1.ReferencedMessage":
      schema: { ... }
      references: []
  ```

### Problem: Performance Issues

**Symptoms:**
- Slow configuration loading
- High memory usage
- Timeout during generation

**Diagnostic Steps:**

1. **Run benchmarks:**
   ```bash
   go test -bench=. ./internal/protoschema/jsonschema/extensions
   ```

2. **Profile configuration:**
   ```bash
   go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=BenchmarkConfigLoading
   ```

**Common Solutions:**

- **Too many broad processors:**
  ```yaml
  # Avoid overly broad matching
  match:
    option: "*"  # Processes every field
    value: "*"

  # Be more specific
  match:
    option: "validate.field.required"
    value: true
  ```

- **Complex regex patterns:**
  ```yaml
  # Simplify patterns where possible
  pattern: "^[a-zA-Z0-9_]+$"  # Simple
  # vs
  pattern: "^(?=.*[A-Z])(?=.*[a-z])(?=.*\\d)..."  # Complex
  ```

## Environment-Specific Issues

### Development vs Production

**Development Issues:**
- Frequent configuration changes
- Temporary debug processors
- Performance not critical

**Production Issues:**
- Configuration must be stable
- Performance is critical
- Error handling must be robust

**Best Practices:**
```yaml
# Use environment-specific configurations
version: "1.0"

# Development: Include debug processors
option_processors:
  - name: "debug_trace"
    priority: 1
    match: { option: "*", value: "*" }
    schema: { x-debug: true }

# Production: Remove debug processors and optimize
option_processors:
  - name: "essential_validation"
    priority: 100
    match: { option: "validate.field.required", value: true }
    schema: { x-required: true }
```

### CI/CD Pipeline Issues

**Common Problems:**
- Configuration validation failures in CI
- Different behavior between local and CI environments
- Missing dependencies

**Solutions:**

1. **Add validation to CI:**
   ```yaml
   # In .github/workflows/test.yml
   - name: Validate Extension Config
     run: |
       go build ./cmd/jsonschema-validate-config
       ./jsonschema-validate-config --config=config.yaml --verbose
   ```

2. **Pin dependency versions:**
   ```go
   // In go.mod, use specific versions
   require (
       github.com/bufbuild/protoplugin v0.1.0
       google.golang.org/protobuf v1.31.0
   )
   ```

3. **Test with real proto files:**
   ```bash
   # Generate schemas in CI to catch integration issues
   go build ./cmd/protoc-gen-jsonschema-extended
   protoc --jsonschema-extended_out=config=config.yaml:./output *.proto
   ```

## Getting Help

### 1. Gather Information

Before asking for help, collect:

- Configuration file (`config.yaml`)
- Error messages (full output)
- Proto files (relevant excerpts)
- Go version and dependencies
- Command line used

### 2. Create Minimal Reproduction

```yaml
# Minimal config that reproduces the issue
version: "1.0"
schema_overrides:
  "simple.v1.Message":
    schema:
      type: "object"
      properties:
        field: { type: "string" }
    references: []
option_processors: []
```

### 3. Check Existing Issues

- Review test files for similar patterns
- Check documentation for configuration examples
- Search for similar error messages in logs

### 4. Debug Systematically

1. Start with minimal configuration
2. Add complexity incrementally
3. Test each change
4. Document what works and what doesn't

## Prevention

### Configuration Best Practices

1. **Always validate configurations:**
   ```bash
   ./jsonschema-validate-config --config=config.yaml
   ```

2. **Use version control:**
   ```bash
   git add config.yaml
   git commit -m "Update schema extensions"
   ```

3. **Test with real data:**
   ```bash
   # Generate and validate actual schemas
   protoc --jsonschema-extended_out=config=config.yaml:./test *.proto
   ```

4. **Monitor performance:**
   ```bash
   # Regular benchmark checks
   go test -bench=. ./internal/protoschema/jsonschema/extensions
   ```

### Development Workflow

1. **Start simple:** Begin with basic overrides and processors
2. **Test incrementally:** Add one processor at a time
3. **Validate early:** Check configuration after each change
4. **Document decisions:** Comment complex configurations
5. **Review regularly:** Update configurations as requirements change