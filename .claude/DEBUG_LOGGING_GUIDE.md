# Debug Logging Guide

The JSON Schema Extensions system includes comprehensive debug logging to help troubleshoot configuration and processing issues.

## Enabling Debug Logging

### Environment Variables

```bash
# Enable debug-level logging
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=DEBUG

# Enable trace-level logging (very verbose)
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=TRACE

# Legacy debug flag (equivalent to DEBUG level)
export PROTOC_GEN_JSONSCHEMA_DEBUG=true
```

### Available Log Levels

- **SILENT** (default): No logging output
- **ERROR**: Only error messages
- **WARN**: Warnings and errors
- **INFO**: General information, warnings, and errors
- **DEBUG**: Detailed debugging information
- **TRACE**: Very verbose trace information

## What Gets Logged

### Configuration Loading
When loading YAML configuration files:

```bash
[12:53:09.477] [INFO] extensions.config: config_load phase=start path=config.yaml
[12:53:09.477] [DEBUG] extensions.config: Reading configuration file: config.yaml
[12:53:09.485] [INFO] extensions.config: config_load phase=file_read file_size=125 duration=8ms
[12:53:09.485] [DEBUG] extensions.config: Parsing YAML configuration
[12:53:09.485] [INFO] extensions.config: config_load phase=yaml_parsed schema_overrides=1 option_processors=0 version=1.0
[12:53:09.485] [INFO] extensions.config: Configuration loaded successfully (overrides: 1, processors: 0)
```

### Registry Creation
When creating the extension registry:

```bash
[12:53:09.485] [DEBUG] extensions.registry: Creating new registry with 1 overrides and 0 processors
[12:53:09.485] [DEBUG] extensions.registry: Copying 1 schema overrides
[12:53:09.485] [TRACE] extensions.registry: Added schema override: example.v1.User
[12:53:09.485] [DEBUG] extensions.registry: Sorting 2 option processors by priority
[12:53:09.485] [TRACE] extensions.registry: Processor 0: required_fields (priority=100)
[12:53:09.485] [TRACE] extensions.registry: Processor 1: email_validation (priority=95)
[12:53:09.485] [INFO] extensions.registry: Registry created successfully (overrides: 1, processors: 2)
```

### Field Processing
When processing fields with option processors:

```bash
[12:53:09.486] [TRACE] extensions.registry: Processing field: email
[12:53:09.486] [DEBUG] extensions.registry: processor_match processor=email_validation field=email status=MATCH reason=option value matched
[12:53:09.486] [DEBUG] extensions.registry: Field marked as required: email
[12:53:09.486] [TRACE] extensions.registry: Field processing complete: email (required=true, duration=245µs)
```

### Option Extraction and Template Substitution
When extracting protobuf options and substituting templates:

```bash
[12:53:09.486] [TRACE] extensions: option_extraction field=name option=validate.field.required status=FOUND value=true
[12:53:09.486] [TRACE] extensions: template_substitution template=${option.min_length} result=5 variables=[min_length=5]
```

### Performance Metrics
Performance timings for various operations:

```bash
[12:53:09.485] [INFO] extensions.registry: performance operation=create_resolver duration=1.5µs schema_count=1
[12:53:09.485] [INFO] extensions.registry: performance operation=resolve_references duration=14µs schema_count=1
[12:53:09.485] [INFO] extensions.registry: performance operation=create_registry duration=26µs schema_overrides=1 option_processors=0
```

### Error Logging
Detailed error information with context:

```bash
[12:53:09.486] [ERROR] extensions.config: error operation=parse_yaml error=yaml: line 5: did not find expected key context=[config_path=config.yaml, file_size=125]
[12:53:09.486] [ERROR] extensions.registry: error operation=apply_processors error=template variable not found context=[field_name=email, duration=1.2ms]
```

## Using Debug Logging

### 1. Troubleshooting Configuration Issues

```bash
# Enable debug logging and run validation
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=DEBUG
./jsonschema-validate-config --config=config.yaml

# You'll see detailed steps:
# - Configuration file reading
# - YAML parsing
# - Validation phases
# - Reference resolution
```

### 2. Understanding Option Processing

```bash
# Enable trace logging to see option extraction
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=TRACE

# Run schema generation (when extended generator is available)
protoc --jsonschema-extended_out=config=config.yaml:./output *.proto

# You'll see:
# - Which fields are being processed
# - Which processors match each field
# - Option values extracted from protobuf
# - Template substitutions
```

### 3. Performance Analysis

```bash
# Enable info-level logging for performance metrics
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=INFO

# All operations will show timing information
# Useful for identifying slow configurations or large-scale processing bottlenecks
```

## Common Debug Scenarios

### Configuration Not Loading

**Symptoms**: Empty or default configuration being used

**Debug approach**:
```bash
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=DEBUG
./jsonschema-validate-config --config=myconfig.yaml
```

**Look for**:
- "Configuration file not found" messages
- File reading errors
- YAML parsing errors

### Option Processors Not Matching

**Symptoms**: Expected schema modifications not appearing

**Debug approach**:
```bash
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=TRACE
# Run schema generation
```

**Look for**:
- `processor_match` messages with `status=NOMATCH`
- `option_extraction` messages with `status=NOTFOUND`
- Field names and processor names in trace logs

### Template Substitution Problems

**Symptoms**: Literal `${...}` text in output schemas

**Debug approach**:
```bash
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=TRACE
```

**Look for**:
- `template_substitution` messages
- `option_extraction` messages showing available variables
- Error messages about missing template variables

### Performance Issues

**Symptoms**: Slow configuration loading or processing

**Debug approach**:
```bash
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=INFO
```

**Look for**:
- `performance` messages with high durations
- Large counts in `schema_count` or `processor_count`
- Operations taking longer than expected

## Log Message Format

All log messages follow a consistent format:

```
[HH:MM:SS.mmm] [LEVEL] component: message details
```

- **Timestamp**: High-resolution timestamp
- **Level**: Log level (ERROR, WARN, INFO, DEBUG, TRACE)
- **Component**: Which part of the system generated the log
  - `extensions.config`: Configuration loading
  - `extensions.registry`: Registry operations
  - `extensions`: General extension processing
- **Message**: Structured message with key-value pairs

## Structured Logging Fields

Common fields in structured log messages:

- `phase`: Processing phase (start, file_read, yaml_parsed, etc.)
- `duration`: Time taken for operation
- `config_path`: Path to configuration file
- `schema_count`: Number of schema overrides
- `processor_count`: Number of option processors
- `field_name`: Name of field being processed
- `processor`: Name of processor being applied
- `status`: Result status (MATCH/NOMATCH, FOUND/NOTFOUND)
- `reason`: Human-readable explanation

## Integration with CI/CD

### GitHub Actions Example

```yaml
- name: Debug Extension Processing
  env:
    PROTOC_GEN_JSONSCHEMA_LOG_LEVEL: INFO
  run: |
    go build ./cmd/jsonschema-validate-config
    ./jsonschema-validate-config --config=config.yaml --verbose

    # Generate schemas with logging
    go build ./cmd/protoc-gen-jsonschema-extended
    protoc --jsonschema-extended_out=config=config.yaml:./output *.proto
```

### Collecting Logs

```bash
# Redirect logs to file for analysis
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=DEBUG
./jsonschema-validate-config --config=config.yaml 2>&1 | tee debug.log

# Filter for specific components or levels
grep "extensions.config" debug.log
grep "\[ERROR\]" debug.log
```

## Best Practices

1. **Start with INFO level**: Good balance of detail and noise
2. **Use DEBUG for troubleshooting**: When things aren't working as expected
3. **Use TRACE only when necessary**: Very verbose, use for deep debugging
4. **Combine with configuration validation**: Always validate configs with logging enabled
5. **Save logs for complex issues**: Capture complete logs for analysis

## Disabling Logging

To completely disable logging (useful for production):

```bash
unset PROTOC_GEN_JSONSCHEMA_LOG_LEVEL
unset PROTOC_GEN_JSONSCHEMA_DEBUG

# Or explicitly set to silent
export PROTOC_GEN_JSONSCHEMA_LOG_LEVEL=SILENT
```

The logging system is designed to have minimal performance impact when disabled (SILENT level).