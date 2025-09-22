# JSON Schema Extensions - Quick Reference

## Common Configuration Patterns

### Well-Known Types Overrides

```yaml
schema_overrides:
  # Timestamp as ISO 8601 string
  "google.protobuf.Timestamp":
    schema:
      type: "string"
      format: "date-time"
      pattern: "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.\\d{3})?Z$"
    references: []

  # Duration as seconds string
  "google.protobuf.Duration":
    schema:
      type: "string"
      pattern: "^\\d+(\\.\\d+)?s$"
    references: []

  # Any type with @type discriminator
  "google.protobuf.Any":
    schema:
      type: "object"
      properties:
        "@type": { type: "string", format: "uri" }
      additionalProperties: true
      required: ["@type"]
    references: []
```

### Validation Processors

```yaml
option_processors:
  # Required fields
  - name: "required_fields"
    priority: 100
    match:
      option: "validate.field.required"
      value: true
    schema:
      x-required: true

  # String length constraints
  - name: "string_min_length"
    priority: 90
    match:
      option: "validate.field.string.min_len"
      value: "*"
    schema:
      minLength: "${option.string.min_len}"

  # Email format
  - name: "email_format"
    priority: 95
    match:
      option: "validate.field.string.email"
      value: true
    schema:
      format: "email"

  # Numeric ranges
  - name: "int_minimum"
    priority: 85
    match:
      option: "validate.field.int32.gte"
      value: "*"
    schema:
      minimum: "${option.int32.gte}"
```

## Command Line Usage

```bash
# Validate configuration
./jsonschema-validate-config --config=config.yaml

# Verbose validation with warnings
./jsonschema-validate-config --config=config.yaml --verbose

# JSON output for automation
./jsonschema-validate-config --config=config.yaml --json

# Skip reference checking for faster validation
./jsonschema-validate-config --config=config.yaml --check-refs=false
```

## Priority Guidelines

| Range | Purpose | Examples |
|-------|---------|----------|
| 95-100 | Security & Critical Validation | Required fields, authentication |
| 85-94 | Format & Type Constraints | Email, UUID, phone formats |
| 75-84 | Business Logic Validation | Min/max values, string lengths |
| 65-74 | Documentation & Metadata | Descriptions, examples |
| 1-64 | Cosmetic & Optional | Debug markers, formatting |

## Template Variables

Common template patterns for option processors:

```yaml
# String validation
schema:
  minLength: "${option.string.min_len}"
  maxLength: "${option.string.max_len}"
  pattern: "${option.string.pattern}"

# Numeric validation
schema:
  minimum: "${option.int32.gte}"
  maximum: "${option.int32.lte}"
  exclusiveMinimum: "${option.int32.gt}"
  exclusiveMaximum: "${option.int32.lt}"

# Custom descriptions
schema:
  description: "Must be ${option.string.format} format"
  const: "${option.string.const}"
```

## Cross-References

```yaml
schema_overrides:
  "api.v1.User":
    schema:
      type: "object"
      properties:
        address: { $ref: "@api.v1.Address" }
    references: ["@api.v1.Address"]

  "api.v1.Address":
    schema:
      type: "object"
      properties:
        country: { type: "string" }
    references: []
```

## Error Messages Quick Fix

| Error | Solution |
|-------|----------|
| `referenced schema not found` | Add schema to `schema_overrides` or check spelling |
| `circular dependency detected` | Redesign schema relationships |
| `invalid template variable` | Check option path spelling and structure |
| `processor not applied` | Verify option path and value matching |

## Debugging Checklist

1. ✅ Configuration validates: `./jsonschema-validate-config --config=config.yaml`
2. ✅ Option paths match protobuf structure
3. ✅ Processor priorities are correct
4. ✅ Template variables exist in option structure
5. ✅ All references are declared and defined
6. ✅ No circular dependencies in references