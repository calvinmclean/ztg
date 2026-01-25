---
name: config-editing
description: Comprehensive guide for editing configuration in ztg project including Go structs, CUE schema, validation, and testing workflows
license: MIT
compatibility: opencode
metadata:
  project: ztg
  category: configuration
  version: "1.0"
---

# Configuration Editing Guide

This skill provides step-by-step instructions for editing configuration in the ztg project. The configuration system uses dual files (`config.go` and `config.cue`) that must stay synchronized.

## Configuration System Overview

### Files to Edit
- **`config/config.go`** - Go structs defining the configuration schema and defaults
- **`config/config.cue`** - CUE validation schema with strict constraints

### Key Relationships
- CUE provides validation rules that are stricter than Go structs
- Test data files in `config/testdata/` must validate against CUE schema
- Environment variable mapping must match struct tags

## Editing Workflow

### Step 1: Modify Go Configuration (`config/config.go`)

1. **Add/Edit Fields** in the appropriate struct:
   ```go
   type ServerConfig struct {
       Port     int    `json:"port" envconfig:"ZTG_PORT"`
       LogLevel string `json:"log_level" envconfig:"ZTG_LOG_LEVEL"`
       // Add new fields here
   }
   ```

2. **Update Default Values** in `DefaultConfig()` function:
   ```go
   func DefaultConfig() *Config {
       return &Config{
           Server: ServerConfig{
               Port:     50052,
               LogLevel: "info",
               // Update defaults here
           },
       }
   }
   ```

3. **Tag Requirements**:
   - `json:"field_name"` - JSON/YAML field mapping
   - `envconfig:"ENV_VAR_NAME"` - Environment variable mapping
   - Follow snake_case for JSON, UPPER_SNAKE for env vars

### Step 2: Update CUE Schema (`config/config.cue`)

1. **Mirror Go struct changes** in CUE definitions:
   ```cue
   #Server: {
       port:      int | *50052
       log_level: string | *"info"
       // Add new fields here with appropriate constraints
   }
   ```

2. **Type Consistency**: Ensure CUE field types match Go struct types exactly:
   - `int` in Go ↔ `int` in CUE
   - `string` in Go ↔ `string` in CUE
   - `bool` in Go ↔ `bool` in CUE

3. **Validation Syntax**:
   - `field!` - Required field
   - `value | *default` - Optional with default
   - `{field: type}` - Exactly-one constraints
   - `field: string | *"default"` - String with default

4. **Common Validation Patterns**:
   ```cue
   // Required field with no default
   name!: string
   
   // Optional with default
   enabled: bool | *false
   
   // Exactly one of two fields must be set
   {api_key: string} | {api_key_file: string}
   
   // Numeric constraints
   port: int & >=1 & <=65535 | *50052
   ```

### Step 3: Validate Configuration

Run the validation task to ensure CUE schema matches test data:
```bash
task validate_config_testdata
```

**Expected Output**: No output (success) or validation errors

**Common Validation Errors**:
- Missing required fields in test data
- Incorrect field types
- Violation of exactly-one constraints
- Mismatched field names between Go and CUE
- **CRITICAL: CUE types must match Go struct types exactly**

### Step 4: Update Test Data (if needed)

If validation fails, update example files in `config/testdata/`:

**`config.example.json`**:
```json
{
  "server": {
    "port": 8080,
    "log_level": "debug",
    "new_field": "value"
  },
  "identity": {
    // ... existing fields
  }
}
```

**`config.example.yaml`**:
```yaml
server:
  port: 8080
  log_level: debug
  new_field: "value"

identity:
  # ... existing fields
```

### Step 5: Update Dependent Code

Search for code using the old configuration structure:

```bash
# Find all code that uses config
grep -r "config\." --include="*.go" .

# Find specific field usage
grep -r "config.Server.Port" --include="*.go" .
```

**Update locations**:
- Server initialization (`cmd/server/server.go`)
- Service configuration (`server/*/service.go`)
- Key management (`identity/key.go`)
- Test files

## Configuration Field Reference

### ServerConfig Fields

| Go Field | JSON Field | Env Var | Type | Default | Validation |
|----------|------------|---------|------|---------|------------|
| `Port` | `port` | `ZTG_PORT` | int | 50052 | N/A |
| `LogLevel` | `log_level` | `ZTG_LOG_LEVEL` | string | "info" | N/A |

### IdentityConfig Fields

| Go Field | JSON Field | Env Var | Type | Default | Validation |
|----------|------------|---------|------|---------|------------|
| `ServerName` | `server_name` | `ZTG_SERVER_NAME` | string | "ztg-server" | Required |
| `OwnerName` | `owner_name` | `ZTG_OWNER_NAME` | string | "ztg-user" | Required |
| `OwnerPublicKey` | `owner_public_key` | `ZTG_OWNER_PUBLIC_KEY` | string | "" | Exactly one with file |
| `OwnerPublicKeyFile` | `owner_public_key_file` | `ZTG_OWNER_PUBLIC_KEY_FILE` | string | "" | Exactly one with key |
| `PrivateKey` | `private_key` | `ZTG_PRIVATE_KEY` | string | "" | Exactly one with path |
| `PrivateKeyPath` | `private_key_file` | `ZTG_KEY_FILE` | string | "keys/example_ed25519.pem" | Exactly one with key |
| `ServerAddress` | `server_address` | `ZTG_IDENTITY_SERVER_ADDRESS` | string | "" | Required |
| `ForceExample` | `force_example` | `ZTG_FORCE_EXAMPLE` | bool | false | N/A |

## Common Editing Scenarios

### Adding a New Field

1. **Add to Go struct**:
   ```go
   type ServerConfig struct {
       Port     int    `json:"port" envconfig:"ZTG_PORT"`
       LogLevel string `json:"log_level" envconfig:"ZTG_LOG_LEVEL"`
       Timeout  int    `json:"timeout" envconfig:"ZTG_TIMEOUT"`
   }
   ```

2. **Add default**:
   ```go
   Server: ServerConfig{
       Port:     50052,
       LogLevel: "info",
       Timeout:  30, // seconds
   },
   ```

3. **Add to CUE**:
   ```cue
   #Server: {
       port:      int | *50052
       log_level: string | *"info"
       timeout:   int | *30
   }
   ```

4. **Update test data** and **validate**

### Making a Field Required vs Optional

**Optional with default**:
```cue
field: string | *"default"
```

**Required field**:
```cue
field!: string
```

**Optional without default**:
```cue
field?: string
```

### Adding Exactly-One Constraints

```cue
// Exactly one must be present
{api_key: string} | {api_key_file: string}

// At least one must be present  
{api_key: string} & {api_key_file: string} | {api_key: string} | {api_key_file: string}
```

## Testing Configuration Changes

### Run Config Tests
```bash
go test ./config/...
```

### Run All Tests
```bash
go test ./...
```

### Test with Environment Variables
```bash
ZTG_PORT=9000 ZTG_LOG_LEVEL=debug go test ./config/...
```

### Test Server with New Config
```bash
# Create test config
cat > test-config.json <<EOF
{
  "server": {
    "port": 9000,
    "log_level": "debug"
  },
  "identity": {
    "server_name": "test-server",
    "owner_name": "test-user",
    "owner_public_key": "test-key",
    "private_key": "test-private-key",
    "server_address": ":9000",
    "force_example": false
  }
}
EOF

# Test server startup
go run cmd/ztg/main.go server --config test-config.json
```

## Troubleshooting

### CUE Validation Fails
1. Check field names match exactly between Go and CUE
2. Verify required fields are present in test data
3. Ensure exactly-one constraints are satisfied
4. Check type compatibility (int vs string) - **CRITICAL: CUE types must match Go struct types exactly**

### Environment Variables Not Working
1. Verify `envconfig` tags match exactly
2. Ensure environment variables are exported
3. Check that field types are compatible with string values

### JSON/YAML Loading Fails
1. Validate JSON/YAML syntax
2. Check that field names match Go struct tags
3. Ensure file encoding is UTF-8

### Server Won't Start
1. Check configuration validation passed
2. Verify key file paths exist and are readable
3. Check port availability
4. Review error logs for specific issues

## Security Considerations

### Key Management
- Never commit real private keys to the repository
- Use `ForceExample: false` in production
- Validate key formats and permissions
- Prefer key files over inline keys

### Configuration Validation
- Always validate CUE schema against test data
- Use required fields (`!`) for security-critical values
- Implement exactly-one constraints for key configurations

## Best Practices

1. **Add new fields with defaults** to maintain backward compatibility
2. **Update CUE immediately** after changing Go structs
3. **Run validation after every change** to catch issues early
4. **Keep test examples realistic** and up-to-date
5. **Document field purposes** in Go struct comments
6. **Use descriptive environment variable names**
7. **Consider adding validation for numeric ranges** and string patterns

## Quick Reference Commands

```bash
# Validate config schema
task validate_config_testdata

# Run config tests
go test ./config/...

# Find config usage
grep -r "config\." --include="*.go" .

# Test with custom config
go run cmd/ztg/main.go server --config my-config.json

# Test with environment variables
ZTG_PORT=9000 go run cmd/ztg/main.go server
```

## Project-Specific Notes

This skill is tailored for the ztg project which implements:
- Go-based distributed dice rolling and factor fight game
- Dual configuration system (Go structs + CUE validation)
- Environment variable overrides using `envconfig` library
- Cryptographic key management for secure communication

The skill maintains consistency with the project's existing conventions and testing patterns.
