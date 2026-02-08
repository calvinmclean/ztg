# Identity Store Implementation Plan

This feature establishes a trusted identity store using SQLite with Turso integration. We'll cache identities in a SQL database instead of just in-memory, allowing the server owner to set identities as known/trusted. We'll continue adding any challengers to the DB but the owner can mark as trusted or manually add them.

## Technology Stack

- **Database**: SQLite with Turso cloud integration
- **Driver**: `turso.tech/database/tursogo` (official Turso Go driver)
- **Query Generation**: SQLC for type-safe SQL operations
- **Authentication**: Owner signature verification for protected operations

## Phase 1: Database Schema & SQLC Setup

### 1.1 Dependencies
- [ ] Add `turso.tech/database/tursogo` to go.mod
- [ ] Add SQLC tooling and generate scripts
- [ ] Update Taskfile with SQLC generation targets

### 1.2 Database Schema
```sql
-- identities table
CREATE TABLE identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    public_key BLOB UNIQUE NOT NULL,
    server_address TEXT NOT NULL,
    server_name TEXT NOT NULL,
    owner_name TEXT NOT NULL,
    capabilities TEXT, -- JSON array
    created_at INTEGER NOT NULL,
    last_seen INTEGER NOT NULL,
    is_trusted BOOLEAN DEFAULT FALSE,
    trust_updated_at INTEGER
);

-- Indexes for performance
CREATE INDEX idx_identities_public_key ON identities(public_key);
CREATE INDEX idx_identities_server_address ON identities(server_address);
```

### 1.3 SQLC Queries
- [ ] `InsertIdentity` - Add new identity
- [ ] `GetIdentityByKey` - Retrieve by public key
- [ ] `GetIdentityByAddress` - Retrieve by server address
- [ ] `UpdateLastSeen` - Update activity timestamp
- [ ] `ListIdentities` - List all with pagination
- [ ] `SetTrustStatus` - Mark identity as trusted/untrusted

## Phase 2: Database Integration

### 2.1 New Package: `identity/store/`
- [ ] `Store` interface for database operations
- [ ] `SQLStore` implementation using SQLC-generated code
- [ ] Connection management with Turso using `turso.tech/database/tursogo`
- [ ] Migration system for schema updates

### 2.2 Turso Connection Setup
```go
import (
    "context"
    "database/sql"
    turso "turso.tech/database/tursogo"
)

// For embedded replica with sync (local + remote)
db, err := turso.NewTursoSyncDb(ctx, turso.TursoSyncDbConfig{
    Path:              "./identity.db",                    // local path
    RemoteUrl:         "libsql://[DATABASE].turso.io",     // remote URL
    RemoteAuthToken:   authToken,                         // authentication token
    LongPollTimeoutMs: 10_000,                            // optional: server waits before replying to pull
    BootstrapIfEmpty: true,                              // bootstrap from remote on first run
})
if err != nil {
    return err
}

// Get *sql.DB instance for database operations
conn, err := db.Connect(ctx)
if err != nil {
    return err
}
defer conn.Close()

// For local-only connection (no sync)
conn, err := sql.Open("turso", "./identity.db")
if err != nil {
    return err
}
defer conn.Close()

// For remote-only connection
conn, err := sql.Open("turso", "libsql://[DATABASE].turso.io?authToken=[TOKEN]")
if err != nil {
    return err
}
defer conn.Close()
```

### 2.3 Sync Operations & Cache Integration
```go
// Push local changes to remote
if err := db.Push(ctx); err != nil {
    return err
}

// Pull remote changes to local
changed, err := db.Pull(ctx)
if err != nil {
    return err
}
log.Println("pulled changes:", changed)

// Compact local WAL to bound disk usage
if err := db.Checkpoint(ctx); err != nil {
    return err
}

// Monitor sync behavior
stats, err := db.Stats(ctx)
if err != nil {
    return err
}
log.Printf("stats: cdc=%v, main=%d revert=%d rx=%d tx=%d pull=%d push=%d revision=%s",
    stats.CdcOperations,
    stats.MainWalSize,
    stats.RevertWalSize,
    stats.NetworkReceivedBytes,
    stats.NetworkSentBytes,
    stats.LastPullUnixTime,
    stats.LastPushUnixTime,
    stats.Revision,
)
```

- [ ] Modify `IdentityCacheManager` to check SQL store on cache miss
- [ ] Implement write-through caching (new identities saved to both)
- [ ] Add background sync for trusted status changes using `db.Pull()` and `db.Push()`

## Phase 3: Enhanced gRPC Service

### 3.1 Extended Identity Proto
```protobuf
message AddIdentityRequest {
  Identity identity = 1;
  Signature signature = 2; // Owner signature for authorization
}

message ListIdentitiesRequest {
  bool trusted_only = 1;
  int32 page_size = 2;
  string page_token = 3;
}

message ListIdentitiesResponse {
  repeated IdentityWithTrust identities = 1;
  string next_page_token = 2;
}

message SetTrustRequest {
  bytes public_key = 1;
  bool trusted = 2;
  Signature signature = 3; // Owner signature
}

message IdentityWithTrust {
  Identity identity = 1;
  bool is_trusted = 2;
  int64 trust_updated_at = 3;
}

service IdentityService {
  rpc GetIdentity(google.protobuf.Empty) returns (Identity);
  rpc AddIdentity(AddIdentityRequest) returns (google.protobuf.Empty);
  rpc ListIdentities(ListIdentitiesRequest) returns (ListIdentitiesResponse);
  rpc SetTrust(SetTrustRequest) returns (google.protobuf.Empty);
}
```

### 3.2 Owner Authentication
- [ ] Verify request signature matches server's private key
- [ ] Add `OwnerAuthInterceptor` for protected methods
- [ ] Only allow `AddIdentity`, `ListIdentities`, `SetTrust` for owner

## Phase 4: Configuration & Integration

### 4.1 Configuration Updates
```cue
database_url: string | *"libsql://[DATABASE].turso.io"
database_auth_token: string | *""
database_path: string | *":memory:"
database_long_poll_timeout_ms: int | *10000
database_bootstrap_if_empty: bool | *true
```

### 4.2 Server Integration
- [ ] Initialize SQL store in server startup
- [ ] Wire store into existing `Verifier` and `IdentityCacheManager`
- [ ] Add graceful shutdown for database connections

## Phase 5: Testing

### 5.1 Testing Strategy
- [ ] Unit tests for SQL store operations
- [ ] Integration tests with in-memory SQLite
- [ ] gRPC service tests with owner authentication
- [ ] Cache miss/fallthrough tests

## Key Implementation Details

### Database Driver Choice:
- Use `turso.tech/database/tursogo` for embedded replicas (recommended)
- Supports both local SQLite and remote Turso with bidirectional sync
- No CGO dependency required (uses purego)
- Provides explicit sync control with `Push()`, `Pull()`, `Checkpoint()`, and `Stats()`

### Connection Patterns:
- **Embedded Replica with Sync**: Local SQLite + remote Turso with explicit sync control
  - Use `turso.NewTursoSyncDb()` for full sync capabilities
  - Automatic bootstrap from remote on first run (configurable)
  - Manual sync with `db.Push()` and `db.Pull()`
  - Long polling support for efficient sync
- **Remote Only**: Direct Turso connection using `sql.Open("turso", remoteUrl)`
- **Local Only**: SQLite file without remote sync using `sql.Open("turso", localPath)`

### Sync Strategy:
- **Bootstrap**: Automatically bootstrap from remote on first run (configurable)
- **Push**: Send local identity changes to remote using `db.Push(ctx)`
- **Pull**: Fetch remote changes using `db.Pull(ctx)` (returns boolean if changed)
- **Long Polling**: Configure `LongPollTimeoutMs` for efficient remote change detection
- **Maintenance**: Use `db.Checkpoint(ctx)` to compact WAL and bound disk usage
- **Monitoring**: Use `db.Stats(ctx)` to observe sync behavior and network usage

### Cache Strategy:
- **Read-through**: Check cache first, fallback to SQL store on miss
- **Write-through**: Save to both cache and SQL store for new identities
- **Trust updates**: Persist immediately to SQL store, update cache

### Security Model:
- Owner-only operations require signature verification with server's private key
- Trust status is persistent and survives server restarts
- All existing peer operations remain unchanged with current zero-trust model

## Implementation Order
1. Database schema and SQLC setup
2. SQL store implementation with basic CRUD
3. Cache integration (read-through on cache miss)
4. gRPC service extensions with owner auth
5. Configuration and server integration
6. Testing

## Original Requirements
- [x] Create SQLite DB for identities
- [x] Store identities here
- [x] Allow owner to manually add with protected GPRC method
- [x] Allow owner to list identities and mark some as trusted
