package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"time"

	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/gen/go/sqlc"
	"google.golang.org/protobuf/types/known/timestamppb"

	turso "turso.tech/database/tursogo"
)

//go:embed schema/*.sql
var schemaFS embed.FS

// SQLStore implements the Store interface using SQLC-generated code with Turso
type SQLStore struct {
	db      *sql.DB
	tursoDB *turso.TursoSyncDb
	queries *sqlc.Queries
}

// Config holds the configuration for the SQLStore
type Config struct {
	DatabaseURL               string
	DatabaseAuthToken         string
	DatabasePath              string
	DatabaseLongPollTimeoutMs int
	DatabaseBootstrapIfEmpty  bool
	UseEmbeddedReplica        bool
}

// NewSQLStore creates a new SQLStore instance
func NewSQLStore(cfg Config) (*SQLStore, error) {
	var (
		db      *sql.DB
		tursoDB *turso.TursoSyncDb
		err     error
	)

	ctx := context.Background()

	if cfg.UseEmbeddedReplica {
		// For embedded replica with sync (local + remote)
		syncCfg := turso.TursoSyncDbConfig{
			Path:              cfg.DatabasePath,
			LongPollTimeoutMs: cfg.DatabaseLongPollTimeoutMs,
			BootstrapIfEmpty:  &cfg.DatabaseBootstrapIfEmpty,
		}

		if cfg.DatabaseURL != "" {
			syncCfg.RemoteUrl = cfg.DatabaseURL
			syncCfg.AuthToken = cfg.DatabaseAuthToken
		}

		tursoDB, err = turso.NewTursoSyncDb(ctx, syncCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create turso sync db: %w", err)
		}

		db, err = tursoDB.Connect(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to turso sync db: %w", err)
		}
	} else {
		// For direct connection (local or remote)
		var dsn string
		if cfg.DatabaseURL != "" {
			// Remote connection
			dsn = fmt.Sprintf("%s?authToken=%s", cfg.DatabaseURL, cfg.DatabaseAuthToken)
		} else {
			// Local connection
			dsn = cfg.DatabasePath
		}

		db, err = sql.Open("turso", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open database: %w", err)
		}
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &SQLStore{
		db:      db,
		tursoDB: tursoDB,
		queries: sqlc.New(db),
	}, nil
}

// runMigrations ensures the database schema is up to date
func runMigrations(ctx context.Context, db *sql.DB) error {
	// Check if identities table exists
	var result struct {
		name string
	}
	err := db.QueryRowContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name='identities'").Scan(&result.name)
	if err == sql.ErrNoRows {
		schemaContent, err := schemaFS.ReadFile("schema/schema.sql")
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}

		if _, err := db.ExecContext(ctx, string(schemaContent)); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	return nil
}

// InsertIdentity adds a new identity to the store
func (s *SQLStore) InsertIdentity(ctx context.Context, identity *IdentityWithTrust) error {
	now := time.Now()

	// Convert capabilities to JSON string
	capJSON, err := json.Marshal(identity.GetCapabilities())
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	params := sqlc.InsertIdentityParams{
		PublicKey:      identity.GetPublicKey(),
		ServerAddress:  identity.GetServerAddress(),
		ServerName:     identity.GetServerName(),
		OwnerName:      identity.GetOwnerName(),
		Capabilities:   string(capJSON),
		CreatedAt:      now.Unix(),
		LastSeen:       now.Unix(),
		TrustUpdatedAt: now.Unix(),
		IsTrusted:      identity.IsTrusted,
	}

	_, err = s.queries.InsertIdentity(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to insert identity: %w", err)
	}

	return s.syncChanges(ctx)
}

// GetIdentityByKey retrieves an identity by its public key
func (s *SQLStore) GetIdentityByKey(ctx context.Context, publicKey []byte) (*IdentityWithTrust, error) {
	row, err := s.queries.GetIdentityByKey(ctx, publicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity not found")
		}
		return nil, fmt.Errorf("failed to get identity by key: %w", err)
	}

	return s.rowToIdentityWithTrust(&row)
}

// GetIdentityByAddress retrieves an identity by its server address
func (s *SQLStore) GetIdentityByAddress(ctx context.Context, serverAddress string) (*IdentityWithTrust, error) {
	row, err := s.queries.GetIdentityByAddress(ctx, serverAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity not found")
		}
		return nil, fmt.Errorf("failed to get identity by address: %w", err)
	}

	return s.rowToIdentityWithTrust(&row)
}

// UpdateLastSeen updates the last seen timestamp for an identity
func (s *SQLStore) UpdateLastSeen(ctx context.Context, serverAddress string, lastSeen time.Time) error {
	err := s.queries.UpdateLastSeen(ctx, sqlc.UpdateLastSeenParams{
		LastSeen:      lastSeen.Unix(),
		ServerAddress: serverAddress,
	})
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return s.syncChanges(ctx)
}

// ListIdentities returns a paginated list of identities
func (s *SQLStore) ListIdentities(ctx context.Context) ([]*IdentityWithTrust, error) {
	rows, err := s.queries.ListIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %w", err)
	}

	identities := make([]*IdentityWithTrust, len(rows))
	for i, row := range rows {
		identity, err := s.rowToIdentityWithTrust(&row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to identity: %w", err)
		}
		identities[i] = identity
	}

	return identities, nil
}

// SetTrustStatus updates the trust status of an identity
func (s *SQLStore) SetTrustStatus(ctx context.Context, publicKey []byte, trusted bool, updatedAt time.Time) error {
	params := sqlc.SetTrustStatusParams{
		PublicKey: publicKey,
		IsTrusted: trusted,
	}

	if trusted {
		params.TrustUpdatedAt = updatedAt.Unix()
	}

	err := s.queries.SetTrustStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to set trust status: %w", err)
	}

	return s.syncChanges(ctx)
}

// DeleteIdentity removes an identity from the store
func (s *SQLStore) DeleteIdentity(ctx context.Context, publicKey []byte) error {
	err := s.queries.DeleteIdentity(ctx, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete identity: %w", err)
	}

	return s.syncChanges(ctx)
}

// CountIdentities returns the total count of identities (optionally filtered by trust status)
func (s *SQLStore) CountIdentities(ctx context.Context, trustedOnly bool) (int64, error) {
	count, err := s.queries.CountIdentities(ctx, trustedOnly)
	if err != nil {
		return 0, fmt.Errorf("failed to count identities: %w", err)
	}

	return count, nil
}

// Close closes the database connection
func (s *SQLStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Sync Operations - only available with embedded replica

// Push pushes local changes to remote
func (s *SQLStore) Push(ctx context.Context) error {
	if s.tursoDB == nil {
		return fmt.Errorf("push not available: not using embedded replica")
	}

	if err := s.tursoDB.Push(ctx); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	return nil
}

// Pull fetches remote changes to local
func (s *SQLStore) Pull(ctx context.Context) (bool, error) {
	if s.tursoDB == nil {
		return false, fmt.Errorf("pull not available: not using embedded replica")
	}

	changed, err := s.tursoDB.Pull(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to pull changes: %w", err)
	}

	return changed, nil
}

// Checkpoint compacts local WAL to bound disk usage
func (s *SQLStore) Checkpoint(ctx context.Context) error {
	if s.tursoDB == nil {
		return fmt.Errorf("checkpoint not available: not using embedded replica")
	}

	if err := s.tursoDB.Checkpoint(ctx); err != nil {
		return fmt.Errorf("failed to checkpoint: %w", err)
	}

	return nil
}

// Stats returns sync statistics
func (s *SQLStore) Stats(ctx context.Context) (*turso.TursoSyncDbStats, error) {
	if s.tursoDB == nil {
		return nil, fmt.Errorf("stats not available: not using embedded replica")
	}

	stats, err := s.tursoDB.Stats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// syncChanges is a helper that syncs changes if using embedded replica
func (s *SQLStore) syncChanges(ctx context.Context) error {
	if s.tursoDB != nil {
		if err := s.tursoDB.Push(ctx); err != nil {
			log.Printf("Warning: failed to push changes: %v", err)
		}
	}
	return nil
}

// rowToIdentityWithTrust converts a SQLC row to IdentityWithTrust
func (s *SQLStore) rowToIdentityWithTrust(row *sqlc.Identity) (*IdentityWithTrust, error) {
	identity := &identitypb.Identity{
		PublicKey:     row.PublicKey,
		ServerAddress: row.ServerAddress,
		ServerName:    row.ServerName,
		OwnerName:     row.OwnerName,
		CreatedAt:     timestamppb.New(time.Unix(row.CreatedAt, 0)),
	}

	// Parse capabilities JSON if present
	var capabilities []string
	if row.Capabilities != "" {
		if err := json.Unmarshal([]byte(row.Capabilities), &capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
		identity.Capabilities = capabilities
	}

	result := NewIdentityWithTrust(identity)

	result.IsTrusted = row.IsTrusted
	result.TrustUpdatedAt = time.Unix(row.TrustUpdatedAt, 0)

	return result, nil
}
