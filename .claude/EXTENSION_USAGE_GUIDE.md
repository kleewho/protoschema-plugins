# JSON Schema Extensions Usage Guide

This guide provides comprehensive examples and best practices for using the protoc-gen-jsonschema extension system to customize JSON Schema generation from Protocol Buffers.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Overview](#configuration-overview)
- [Schema Overrides](#schema-overrides)
- [Option Processors](#option-processors)
- [Real-World Examples](#real-world-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Quick Start

### 1. Basic Setup

Create a configuration file `jsonschema-extensions.yaml`:

```yaml
version: "1.0"
schema_overrides:
  "google.protobuf.Timestamp":
    schema:
      type: "string"
      format: "date-time"
      pattern: "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.\\d{3})?Z$"
      description: "RFC 3339 date-time format"
      examples: ["2023-12-25T10:30:00.123Z"]
    references: []

option_processors: []
```

### 2. Generate Schemas

Use the extended generator with your configuration:

```bash
# Build the validation tool
go build ./cmd/jsonschema-validate-config

# Validate your configuration
./jsonschema-validate-config --config=jsonschema-extensions.yaml

# Generate schemas (when extended generator is implemented)
protoc --jsonschema-extended_out=config=jsonschema-extensions.yaml:./output *.proto
```

## Configuration Overview

The extension system supports two main customization approaches:

1. **Schema Overrides**: Complete replacement of generated schemas for specific message types
2. **Option Processors**: Field-level modifications based on protobuf options

### Configuration Structure

```yaml
version: "1.0"                    # Configuration format version
schema_overrides:                 # Complete schema replacements
  "fully.qualified.MessageName":
    schema: { ... }              # JSON Schema object
    references: ["@other.Type"]  # Cross-references to other schemas

option_processors:               # Field-level processors
  - name: "processor_name"       # Unique processor name
    priority: 100                # Processing order (higher = first)
    match:                       # Matching criteria
      option: "option.path"      # Protobuf option to match
      value: "expected_value"    # Expected option value (* for wildcard)
    schema:                      # Schema modifications to apply
      property: "value"
      template: "${option.field}" # Template substitution
```

## Schema Overrides

Schema overrides provide complete control over the JSON Schema generated for specific message types.

### Basic Override Example

```yaml
schema_overrides:
  "example.v1.Money":
    schema:
      type: "object"
      properties:
        amount:
          type: "string"
          pattern: "^\\d+\\.\\d{2}$"
          description: "Monetary amount in decimal format"
        currency:
          type: "string"
          enum: ["USD", "EUR", "GBP", "JPY"]
          description: "ISO 4217 currency code"
      required: ["amount", "currency"]
      additionalProperties: false
    references: []
```

### Cross-References Between Schemas

```yaml
schema_overrides:
  "example.v1.Address":
    schema:
      type: "object"
      properties:
        street: { type: "string", minLength: 1 }
        city: { type: "string", minLength: 1 }
        country: { type: "string", pattern: "^[A-Z]{2}$" }
      required: ["street", "city", "country"]
    references: []

  "example.v1.User":
    schema:
      type: "object"
      properties:
        name: { type: "string", minLength: 1 }
        email: { type: "string", format: "email" }
        address:
          $ref: "@example.v1.Address"  # Reference to another override
      required: ["name", "email"]
    references: ["@example.v1.Address"]  # Declare the reference
```

### Well-Known Types

Common overrides for Google's Well-Known Types:

```yaml
schema_overrides:
  "google.protobuf.Timestamp":
    schema:
      type: "string"
      format: "date-time"
      pattern: "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.\\d{3})?Z$"
      description: "RFC 3339 date-time string"
    references: []

  "google.protobuf.Duration":
    schema:
      type: "string"
      pattern: "^\\d+(\\.\\d+)?s$"
      description: "Duration in seconds with optional fractional part"
      examples: ["1.5s", "300s"]
    references: []

  "google.protobuf.Any":
    schema:
      type: "object"
      properties:
        "@type": { type: "string", format: "uri" }
      additionalProperties: true
      required: ["@type"]
    references: []
```

## Option Processors

Option processors modify field schemas based on protobuf field options, enabling validation-driven schema generation.

### Required Fields Processor

```yaml
option_processors:
  - name: "required_fields"
    priority: 100
    match:
      option: "validate.field.required"  # Adjust path based on your validation library
      value: true
    schema:
      x-required: true                   # Special marker for required fields
      description: "This field is required"
```

### String Validation Processors

```yaml
option_processors:
  - name: "string_min_length"
    priority: 90
    match:
      option: "validate.field.string.min_len"
      value: "*"                         # Match any min_len value
    schema:
      minLength: "${option.string.min_len}"  # Template substitution

  - name: "string_max_length"
    priority: 90
    match:
      option: "validate.field.string.max_len"
      value: "*"
    schema:
      maxLength: "${option.string.max_len}"

  - name: "string_pattern"
    priority: 85
    match:
      option: "validate.field.string.pattern"
      value: "*"
    schema:
      pattern: "${option.string.pattern}"
      description: "Must match pattern: ${option.string.pattern}"
```

### Numeric Validation Processors

```yaml
option_processors:
  - name: "numeric_minimum"
    priority: 80
    match:
      option: "validate.field.int32.gte"
      value: "*"
    schema:
      minimum: "${option.int32.gte}"

  - name: "numeric_maximum"
    priority: 80
    match:
      option: "validate.field.int32.lte"
      value: "*"
    schema:
      maximum: "${option.int32.lte}"

  - name: "exclusive_range"
    priority: 75
    match:
      option: "validate.field.int32.gt"
      value: "*"
    schema:
      exclusiveMinimum: "${option.int32.gt}"
```

### Email and Format Processors

```yaml
option_processors:
  - name: "email_validation"
    priority: 95
    match:
      option: "validate.field.string.email"
      value: true
    schema:
      format: "email"
      description: "Must be a valid email address"

  - name: "uuid_validation"
    priority: 95
    match:
      option: "validate.field.string.uuid"
      value: true
    schema:
      format: "uuid"
      pattern: "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
```

## Real-World Examples

### E-commerce API Configuration

```yaml
version: "1.0"

schema_overrides:
  # Custom money type with strict validation
  "ecommerce.v1.Money":
    schema:
      type: "object"
      properties:
        amount:
          type: "string"
          pattern: "^\\d+\\.\\d{2}$"
          description: "Amount in decimal format (e.g., '19.99')"
        currency_code:
          type: "string"
          enum: ["USD", "EUR", "GBP", "CAD"]
          description: "ISO 4217 currency code"
      required: ["amount", "currency_code"]
      additionalProperties: false
    references: []

  # Address with country-specific validation
  "ecommerce.v1.Address":
    schema:
      type: "object"
      properties:
        street_address:
          type: "string"
          minLength: 5
          maxLength: 100
        city:
          type: "string"
          minLength: 2
          maxLength: 50
        state_province:
          type: "string"
          minLength: 2
          maxLength: 50
        postal_code:
          type: "string"
          pattern: "^[A-Z0-9]{3,10}$"
        country_code:
          type: "string"
          pattern: "^[A-Z]{2}$"
          description: "ISO 3166-1 alpha-2 country code"
      required: ["street_address", "city", "country_code"]
    references: []

option_processors:
  # Product ID validation
  - name: "product_id_format"
    priority: 100
    match:
      option: "ecommerce.field.product_id"
      value: true
    schema:
      pattern: "^PROD-[A-Z0-9]{8}$"
      description: "Product ID format: PROD-XXXXXXXX"

  # Price validation
  - name: "price_validation"
    priority: 95
    match:
      option: "ecommerce.field.price_cents"
      value: true
    schema:
      minimum: 0
      maximum: 999999999
      description: "Price in cents (integer)"

  # SKU validation
  - name: "sku_format"
    priority: 90
    match:
      option: "ecommerce.field.sku"
      value: true
    schema:
      pattern: "^[A-Z]{2,4}-[0-9]{4,8}$"
      description: "SKU format: PREFIX-NUMBERS"
```

### User Management System

```yaml
version: "1.0"

schema_overrides:
  # User profile with comprehensive validation
  "users.v1.UserProfile":
    schema:
      type: "object"
      properties:
        user_id:
          type: "string"
          format: "uuid"
          description: "Unique user identifier"
        username:
          type: "string"
          pattern: "^[a-zA-Z0-9_]{3,30}$"
          description: "Username: 3-30 alphanumeric characters and underscores"
        email:
          type: "string"
          format: "email"
          description: "Primary email address"
        profile:
          $ref: "@users.v1.Profile"
        preferences:
          $ref: "@users.v1.UserPreferences"
      required: ["user_id", "username", "email"]
      additionalProperties: false
    references: ["@users.v1.Profile", "@users.v1.UserPreferences"]

option_processors:
  # Username validation
  - name: "username_rules"
    priority: 100
    match:
      option: "users.field.username"
      value: true
    schema:
      minLength: 3
      maxLength: 30
      pattern: "^[a-zA-Z0-9_]+$"
      description: "Username must be 3-30 characters, alphanumeric and underscore only"

  # Password strength requirements
  - name: "password_strength"
    priority: 100
    match:
      option: "users.field.password"
      value: true
    schema:
      minLength: 12
      pattern: "^(?=.*[a-z])(?=.*[A-Z])(?=.*\\d)(?=.*[@$!%*?&])[A-Za-z\\d@$!%*?&]+$"
      description: "Password must contain uppercase, lowercase, number, and special character"

  # Age validation
  - name: "age_limits"
    priority: 90
    match:
      option: "users.field.age"
      value: true
    schema:
      minimum: 13
      maximum: 120
      description: "Age must be between 13 and 120 years"
```

## Best Practices

### 1. Configuration Organization

- **Use descriptive names**: Choose clear, descriptive names for processors and overrides
- **Group related processors**: Organize processors by domain or validation type
- **Document your configuration**: Include descriptions for complex validations

```yaml
# Good: Descriptive and organized
option_processors:
  - name: "user_email_validation"      # Clear purpose
    priority: 100
    # ... config

  - name: "user_phone_validation"      # Grouped with user validations
    priority: 95
    # ... config

# Avoid: Generic or unclear names
option_processors:
  - name: "processor_1"                # Unclear purpose
    # ... config
```

### 2. Priority Management

- **Use priority ranges**: Reserve priority ranges for different types of processors
  - 90-100: Security and validation processors
  - 70-89: Format and constraint processors
  - 50-69: Documentation and metadata processors
  - 1-49: Cosmetic or optional processors

```yaml
option_processors:
  - name: "security_validation"
    priority: 95                       # High priority for security

  - name: "format_constraint"
    priority: 75                       # Medium priority for formatting

  - name: "documentation_enhancement"
    priority: 55                       # Lower priority for docs
```

### 3. Schema Override Guidelines

- **Be specific with patterns**: Use precise regex patterns for validation
- **Include examples**: Provide example values to clarify expected formats
- **Document edge cases**: Explain any special validation rules

```yaml
schema_overrides:
  "api.v1.PhoneNumber":
    schema:
      type: "string"
      pattern: "^\\+[1-9]\\d{1,14}$"   # E.164 format
      description: "International phone number in E.164 format"
      examples: ["+1234567890", "+441234567890"]
    references: []
```

### 4. Template Usage

- **Validate templates**: Ensure template variables exist in the option structure
- **Provide fallbacks**: Consider what happens when template values are missing
- **Escape special characters**: Properly escape regex patterns and JSON strings

```yaml
# Good: Safe template usage
schema:
  pattern: "${option.string.pattern}"
  description: "Must match pattern: ${option.string.pattern}"

# Be careful with: Direct template insertion in arrays or complex structures
schema:
  enum: "${option.enum.values}"  # May not work as expected
```

### 5. Cross-Reference Management

- **Declare all references**: Always list references in the `references` array
- **Avoid circular dependencies**: Design schemas to prevent circular references
- **Use meaningful reference names**: Make reference relationships clear

```yaml
schema_overrides:
  "order.v1.Order":
    schema:
      type: "object"
      properties:
        customer: { $ref: "@user.v1.Customer" }
        items:
          type: "array"
          items: { $ref: "@order.v1.OrderItem" }
    references: ["@user.v1.Customer", "@order.v1.OrderItem"]
```

### 6. Validation and Testing

- **Validate configurations**: Always use the validation CLI before deployment
- **Test with real data**: Verify generated schemas work with actual API responses
- **Version your configurations**: Track changes to extension configurations

```bash
# Always validate before use
./jsonschema-validate-config --config=extensions.yaml --verbose

# Test generated schemas
protoc --jsonschema-extended_out=config=extensions.yaml:./test-output *.proto
```

### 7. Performance Considerations

- **Limit processor complexity**: Keep option matching logic simple and fast
- **Use specific matchers**: Avoid overly broad wildcard matchers when possible
- **Monitor processing time**: Use benchmarks to verify configuration performance

```yaml
# Good: Specific matching
match:
  option: "validate.field.string.email"
  value: true

# Less efficient: Overly broad matching
match:
  option: "*"  # Matches everything
  value: "*"
```

## Troubleshooting

### Common Issues

#### 1. Processor Not Applied

**Symptoms**: Expected schema modifications not appearing in generated output.

**Solutions**:
- Verify option paths match actual protobuf options
- Check processor priority ordering
- Ensure option values match exactly (or use wildcards)
- Use debug configuration to trace option extraction

```bash
# Debug with verbose validation
./jsonschema-validate-config --config=extensions.yaml --verbose
```

#### 2. Reference Resolution Errors

**Symptoms**: Configuration validation fails with reference errors.

**Solutions**:
- Ensure all referenced schemas are defined
- Check reference syntax (`@fully.qualified.name`)
- Verify no circular dependencies exist
- Use `--check-refs=false` to disable reference checking during development

#### 3. Template Substitution Failures

**Symptoms**: Template variables not replaced, literal `${...}` in output.

**Solutions**:
- Verify template variable paths match option structure
- Check for typos in template variable names
- Ensure option values are the expected type (string, number, etc.)

#### 4. Schema Override Not Applied

**Symptoms**: Original schema generated instead of override.

**Solutions**:
- Verify fully qualified message name is correct
- Check for typos in message name
- Ensure the message is actually being processed

### Debug Configuration

For troubleshooting, create a debug configuration that logs processing:

```yaml
version: "1.0"
schema_overrides: {}
option_processors:
  - name: "debug_all_options"
    priority: 1
    match:
      option: "*"
      value: "*"
    schema:
      x-debug-processed: true
      x-debug-timestamp: "2024-01-01T00:00:00Z"
```

This will add debug markers to any field that has options, helping identify which fields are being processed by the extension system.

### Getting Help

1. **Use verbose validation**: Run configuration validation with `--verbose` flag
2. **Check integration tests**: Review test files for working examples
3. **Inspect protobuf options**: Use protobuf reflection tools to examine option structures
4. **Simplify and isolate**: Start with minimal configurations and add complexity gradually

### Performance Monitoring

Monitor extension processing performance:

```bash
# Run benchmarks
go test -bench=. ./internal/protoschema/jsonschema/extensions

# Profile configuration loading
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=BenchmarkConfigLoading
```

## Conclusion

The extension system provides powerful customization capabilities for JSON Schema generation. Start with simple overrides and processors, then gradually add complexity as needed. Always validate configurations and test with real data to ensure the generated schemas meet your requirements.

For the most up-to-date examples and advanced usage patterns, refer to the test files in the `internal/protoschema/jsonschema/extensions*_test.go` files.