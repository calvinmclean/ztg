---
name: buf-generate
description: Generate Go code from protobuf definitions using buf
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: protobuf
---

## What I do
- Run `buf generate` to compile protobuf files to Go code
- Generate both gRPC and standard Go bindings
- Use the project's buf.yaml and buf.gen.yaml configurations
- Generate code to the gen/go directory with source_relative paths

## When to use me
Use this when you have modified protobuf files (.proto) and need to regenerate the corresponding Go code.

Also use me when:
- Adding new protobuf services or messages
- Updating existing protobuf definitions
- Setting up a new development environment

## Configuration
This skill uses the existing configuration files:
- `buf.yaml` - Main buf configuration with linting rules
- `buf.gen.yaml` - Generation configuration for Go and gRPC plugins

## Generated files
The skill generates Go files in `gen/go/` following the source_relative path pattern:
- `gen/go/dice/v1/dice.pb.go` and `dice_grpc.pb.go`
- `gen/go/factorfight/v1/factorfight.pb.go` and `factorfight_grpc.pb.go`
- `gen/go/game/v1/game.pb.go` and `game_grpc.pb.go`
- `gen/go/identity/v1/identity.pb.go` and `identity_grpc.pb.go`