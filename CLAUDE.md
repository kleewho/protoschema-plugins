# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Claude files

All claude related files except this CLAUDE.md should go to .claude directory

## Project Overview

This repository contains Protobuf plugins that generate different types of schema from protobuf files:

- **protoc-gen-jsonschema**: Generates JSON Schema (Draft 2020-12) from protobuf files
- **protoc-gen-pubsub**: Generates PubSub schemas (single self-contained messages normalized to proto2)

## Architecture

The codebase follows a plugin-based architecture:

- `cmd/protoc-gen-*/`: Main entry points for the protoc plugins
- `internal/protoschema/plugin/`: Plugin-specific implementation logic
  - `pluginjsonschema/`: JSON Schema generation logic
  - `pluginpubsub/`: PubSub schema generation logic
- `internal/protoschema/`: Core schema generation functionality
  - `jsonschema/`: JSON Schema specific transformations
  - `pubsub/`: PubSub specific transformations
  - `normalize/`: Common protobuf normalization utilities
- `internal/proto/`: Test protobuf definitions
- `internal/testdata/`: Generated test data and golden files
- `internal/gen/`: Generated Go code from protobuf definitions

Both plugins use the `github.com/bufbuild/protoplugin` framework and share common version handling through `internal/protoschema/protoschema.go`.

## Build Commands

```bash
# Build all packages
make build

# Run tests (includes JSON Schema validation)
make test

# Run linting (includes buf lint, go vet, golangci-lint)
make lint

# Auto-fix lint issues
make lintfix

# Install binaries
make install

# Generate code and update golden test files
make generate
make golden
```

## Development Commands

```bash
# Run unit tests only
go test -vet=off -race -cover ./...

# Build specific plugin
go build ./cmd/protoc-gen-jsonschema
go build ./cmd/protoc-gen-pubsub

# Run single test
go test ./internal/protoschema/jsonschema -run TestSpecificTest

# Clean generated files
make clean
```

## Code Generation

The project uses Buf for protobuf code generation (see `buf.gen.yaml`):
- Generated Go code goes to `internal/gen/proto/`
- Test JSON schemas are generated to `internal/gen/jsonschema/`
- Golden test data is regenerated with `make golden`

Test data generation requires building the current protoc plugin binaries and running them against test proto files in `internal/proto/`.

## Extension System

The JSON Schema generator supports a configuration-based extension system that allows:

- **Custom Type Handlers**: Override generation for specific protobuf message types
- **Field Processors**: Modify individual field schemas during generation
- **Schema Processors**: Post-process complete message schemas

See `.claude/EXTENSION_SYSTEM.md` for detailed documentation and examples.

Example usage:
```bash
# Build the extended plugin
go build ./cmd/protoc-gen-jsonschema-extended

# Use with YAML configuration (recommended)
protoc --jsonschema-extended_out=config=config.yaml:. myfile.proto

# Use with built-in example processors
protoc --jsonschema-extended_out=use_examples=true:. myfile.proto
```

YAML configuration allows declarative extension setup without Go code. See `.claude/YAML_CONFIG_GUIDE.md` for complete configuration options.

## Testing

Tests include both unit tests and validation of generated JSON schemas against test data files using the `jv` JSON Schema validator tool.

Golden test files in `internal/testdata/` must be regenerated when the schema generation logic changes using `make golden`.
