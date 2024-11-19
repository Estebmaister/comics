# Comics Service Protocol Buffers

This directory contains the Protocol Buffer definitions for the Comics gRPC service.

## Overview

The service is defined using Protocol Buffers version 3 (proto3) and consists of two main proto files:

1. `comics.proto`: Contains message definitions for comics data structures
2. `comics_service.proto`: Defines the gRPC service interface and operations

## Message Definitions

### Comics Proto

The `comics.proto` file defines the core data structures:

- `Comic`: Represents a comic book with fields for:
  - Multiple titles (primary and alternative titles)
  - Author information
  - Current and viewed chapter numbers
  - Status and type information
  - Genre classifications
  - Timestamps for creation and updates

### Service Proto

The `comics_service.proto` file defines the following operations:

- `CreateComic`: Creates a new comic entry
- `GetComicById`: Retrieves a comic by its unique identifier
- `GetComics`: Lists comics with pagination support
- `SearchComics`: Searches comics based on query parameters
- `UpdateComic`: Updates an existing comic's information
- `DeleteComic`: Removes a comic from the system
- `GetComicByTitle`: Finds comics by title match

## Package Options

The proto files include the following package options:

```protobuf
option go_package = "go_server/proto";
option java_package = "com.comics";
option java_multiple_files = true;
```

## Generating Code

To generate code from these proto definitions:

1. Install Protocol Buffer compiler (protoc):
```bash
brew install protobuf
```

2. Install language-specific plugins:
```bash
# Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install protoc-gen-validate
go install github.com/envoyproxy/protoc-gen-validate@latest

# Get validate.proto
git clone https://github.com/envoyproxy/protoc-gen-validate.git
cp -r protoc-gen-validate/validate ./proto
```

3. Generate the code:
```bash
# From the project root directory
protoc \
  --go_out=./go_server/pb      --go_opt=paths=source_relative \
  --go-grpc_out=./go_server/pb --go-grpc_opt=paths=source_relative \
  --validate_out="lang=go,paths=source_relative:./go_server/pb" \
  --proto_path=./proto proto/*.proto
```

## Dependencies

- `google/protobuf/timestamp.proto`: Used for timestamp fields
- `validate/validate.proto`: Used for field validation rules

## Best Practices

1. Message Evolution:
   - Use optional fields for backward compatibility
   - Never remove or reuse field numbers
   - Add new fields as optional

2. Error Handling:
   - Use standard gRPC status codes
   - Include detailed error messages in responses

3. Validation:
   - Use protoc-gen-validate rules for field validation
   - Implement server-side validation for all fields
   - Use appropriate field types (int32, string, etc.)
   - Define clear constraints (e.g., non-negative chapter numbers)

## Generated Code Location

The generated Go code will be placed in the `go_server/proto` directory, maintaining the same directory structure as the proto files.