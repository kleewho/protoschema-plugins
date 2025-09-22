# YAML Configuration Guide

This guide shows how to use YAML configuration files to extend the JSON Schema generator without writing Go code.

## Overview

The YAML configuration system allows you to:
- **Configure custom type handlers** for specific protobuf message types
- **Add field processors** that modify individual field schemas
- **Add schema processors** that enhance complete message schemas
- **Set metadata** added to all generated schemas

## Configuration Loading

The YAML config is loaded through the extended plugin using the `config` parameter:

```bash
# Load configuration from a file
protoc --jsonschema-extended_out=config=my_config.yaml:. myfile.proto

# Combine with other parameters
protoc --jsonschema-extended_out=config=config.yaml,target=json-strict,additional_properties=true:. myfile.proto
```

## Complete Configuration Reference

### Top-Level Structure

```yaml
# Metadata added to all generated schemas
metadata:
  generator: "my-jsonschema-generator"
  version: "v2.0.0"
  additional:
    organization: "acme-corp"
    team: "platform"

# Custom type handlers override default message generation
custom_types:
  "google.protobuf.Timestamp":
    handler: "custom_timestamp"

# Field processors modify individual field schemas
field_processors:
  - name: "sensitive_data"
    priority: 100
    config: {...}

# Schema processors modify complete message schemas
schema_processors:
  - name: "metadata"
    config: {...}
```

## Custom Type Handlers

Override generation for specific protobuf message types:

```yaml
custom_types:
  "google.protobuf.Timestamp":
    handler: "custom_timestamp"
    config: {}

  "google.protobuf.Any":
    handler: "pass_through"  # Disables default handling
```

**Available Handlers:**
- `custom_timestamp`: Enhanced timestamp with strict RFC 3339 pattern
- `pass_through`: No-op handler that disables default processing

## Field Processors

Field processors modify schemas for individual fields during generation.

### Sensitive Data Processor

Identifies and marks sensitive fields with security metadata:

```yaml
field_processors:
  - name: "sensitive_data"
    priority: 100  # Higher numbers run first
    config:
      # Custom regex patterns for field types
      patterns:
        "credit_card": "^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{4}$"
        "ssn": "^\\d{3}-\\d{2}-\\d{4}$"
        "tax_id": "^[0-9]{2}-[0-9]{7}$"

      # Categories for different sensitive data types
      categories:
        "credit_card": "financial"
        "ssn": "pii"
        "tax_id": "government_id"
        "api_key": "credential"

      # Additional field name patterns to mark as sensitive
      additional_sensitive:
        - "private_key"
        - "auth_token"
        - "bearer_token"
```

**What it does:**
- Adds `x-sensitive: true` to matching fields
- Adds `x-category` with the configured category
- Adds `writeOnly: true` for credential fields
- Sets custom regex patterns for validation

### Validation Enhancer Processor

Adds intelligent validation rules based on field names and types:

```yaml
field_processors:
  - name: "validation_enhancer"
    priority: 50
    config:
      enable_builtin: true  # Enable built-in rules

      # Custom rules for age fields
      age_rules:
        min: 0
        max: 120

      # Custom rules for percentage fields
      percent_rules:
        min: 0.0
        max: 100.0

      # Custom validation for field name patterns
      custom_rules:
        "score":          # Any field containing "score"
          number:
            min: 0
            max: 10

        "rating":         # Any field containing "rating"
          number:
            min: 1
            max: 5

        "website":        # Any field containing "website"
          format: "uri"
          pattern: "^https?://.*"

        "zipcode":        # Any field containing "zipcode"
          pattern: "^[0-9]{5}(-[0-9]{4})?$"

        "phone":          # Any field containing "phone"
          pattern: "^[+]?[0-9\\s\\-\\(\\)]+$"
```

**Built-in rules** (when `enable_builtin: true`):
- Fields containing "age" get min/max validation
- Fields containing "percentage" get 0-100% range
- Fields containing "url" get URI format
- Fields containing "phone" get phone pattern

## Schema Processors

Schema processors modify complete message schemas after generation.

### Metadata Processor

Adds comprehensive metadata to schemas:

```yaml
schema_processors:
  - name: "metadata"
    config:
      # Override global metadata settings
      generator: "custom-generator-name"
      version: "v1.5.0"

      # Custom keys to add to every schema
      custom_keys:
        "x-api-version": "v1"
        "x-stability": "stable"
        "x-team": "platform-team"
        "x-docs": "https://docs.example.com"
```

**Automatically added:**
- `x-generated-by`: Generator name
- `x-generator-version`: Generator version
- `x-proto-message`: Full protobuf message name
- `x-generated-at`: ISO 8601 timestamp

### Documentation Processor

Enhances schema documentation and examples:

```yaml
schema_processors:
  - name: "documentation"
    config:
      add_examples: true
      add_draft_info: true

      # Template for auto-generated examples (use {message} placeholder)
      example_template: "This schema represents {message} in our API v1"

      # Additional fields to add to every schema
      additional_fields:
        "x-usage-notes": "Generated from protobuf definitions"
        "x-support": "https://example.com/support"
        "x-changelog": "https://example.com/changelog"
```

## Usage Examples

### Minimal Configuration

```yaml
# minimal.yaml
metadata:
  generator: "my-generator"
  version: "v1.0.0"

field_processors:
  - name: "sensitive_data"
    config:
      additional_sensitive: ["password", "secret"]
```

```bash
protoc --jsonschema-extended_out=config=minimal.yaml:. myfile.proto
```

### Security-Focused Configuration

```yaml
# security.yaml
metadata:
  generator: "secure-jsonschema"
  version: "v1.0.0"
  additional:
    security_policy: "strict"

field_processors:
  - name: "sensitive_data"
    priority: 100
    config:
      patterns:
        "ssn": "^\\d{3}-\\d{2}-\\d{4}$"
        "credit": "^[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{4}$"
      categories:
        "ssn": "pii"
        "credit": "financial"
      additional_sensitive:
        - "password"
        - "secret"
        - "token"
        - "key"

schema_processors:
  - name: "metadata"
    config:
      custom_keys:
        "x-security-reviewed": true
        "x-data-classification": "restricted"
```

### API Documentation Configuration

```yaml
# api_docs.yaml
metadata:
  generator: "api-jsonschema-generator"
  version: "v2.0.0"
  additional:
    api_version: "v1"
    organization: "acme-corp"

field_processors:
  - name: "validation_enhancer"
    config:
      enable_builtin: true
      custom_rules:
        "email":
          format: "email"
        "url":
          format: "uri"
        "phone":
          pattern: "^[+]?[0-9\\s\\-\\(\\)]+$"

schema_processors:
  - name: "documentation"
    config:
      add_examples: true
      example_template: "Example usage of {message} in Acme Corp API"
      additional_fields:
        "x-api-docs": "https://docs.acme.com"
        "x-support": "https://support.acme.com"

  - name: "metadata"
    config:
      custom_keys:
        "x-api-stability": "stable"
        "x-rate-limit": "1000/hour"
```

## File Loading Locations

The config file path can be:
- **Absolute path**: `/path/to/config.yaml`
- **Relative path**: `./config.yaml` (relative to where protoc is run)
- **Environment variable**: Use `${HOME}/.protoc-jsonschema.yaml`

## Error Handling

If the config file:
- **Cannot be read**: Extension loading fails silently (fallback to no extensions)
- **Has invalid YAML**: Error is returned and protoc fails
- **References unknown processors**: Error is returned and protoc fails
- **Has invalid processor config**: Error is returned and protoc fails

## Integration with Existing Parameters

YAML config works alongside existing plugin parameters:

```bash
# All of these work together:
protoc --jsonschema-extended_out=config=myconfig.yaml,target=json-strict,additional_properties=true:. myfile.proto
```

**Parameter precedence:**
1. YAML config file (if specified)
2. Built-in examples (if `use_examples=true`)
3. Other command-line parameters

## Configuration Validation

The system validates:
- ✅ YAML syntax and structure
- ✅ Processor names exist
- ✅ Handler names exist
- ✅ Configuration parameters are valid types
- ✅ Priority values are integers
- ✅ Patterns compile as valid regex

## Migration from Code to YAML

To migrate existing Go code configurations:

1. **Identify your processors**: Look at your `ExtensionConfig`
2. **Map to YAML structure**: Use the examples above
3. **Test with simple proto**: Verify output matches expectations
4. **Validate all edge cases**: Ensure behavior is preserved

## Best Practices

1. **Start simple**: Begin with minimal config and add complexity
2. **Use meaningful names**: Choose clear generator and version names
3. **Test thoroughly**: Validate with representative proto files
4. **Document your config**: Add comments explaining custom rules
5. **Version your config**: Track changes to configuration files
6. **Validate patterns**: Test regex patterns before deploying

## Debugging

To debug configuration issues:

1. **Check YAML syntax**: Use a YAML validator
2. **Verify file path**: Ensure config file is accessible
3. **Start minimal**: Remove complex config and add back gradually
4. **Check logs**: Look for error messages from the plugin
5. **Test processors individually**: Enable one at a time