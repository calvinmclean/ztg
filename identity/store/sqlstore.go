package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/calvinmclean/ztg/config"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/gen/go/sqlc"
	"google.golang.org/protobuf/types/known/timestamppb"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// SQLStore implements the Store interface using SQLC-generated code with PostgreSQL
type SQLStore struct {
	db      *sql.DB
	queries *sqlc.Queries
}

// NewSQLStore creates a new SQLStore instance
func NewSQLStore(cfg config.DatabaseConfig) (*SQLStore, error) {
	// Use pgx driver with connection string
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &SQLStore{
		db:      db,
		queries: sqlc.New(db),
	}, nil
}

// Close closes the database connection
func (s *SQLStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// InsertIdentity adds a new identity to the store
func (s *SQLStore) InsertIdentity(ctx context.Context, identity *identitypb.Identity) error {
	// Convert capabilities to JSON string
	capJSON, err := json.Marshal(identity.GetCapabilities())
	if err != nil {
		return fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	params := sqlc.InsertIdentityParams{
		PublicKey:     identity.GetPublicKey(),
		ServerAddress: identity.GetServerAddress(),
		ServerName:    identity.GetServerName(),
		OwnerName:     identity.GetOwnerName(),
		Capabilities:  string(capJSON),
		IsTrusted:     identity.IsTrusted,
	}

	_, err = s.queries.InsertIdentity(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to insert identity: %w", err)
	}

	return nil
}

// GetIdentityByKey retrieves an identity by its public key
func (s *SQLStore) GetIdentityByKey(ctx context.Context, publicKey []byte) (*identitypb.Identity, error) {
	row, err := s.queries.GetIdentityByKey(ctx, publicKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity not found")
		}
		return nil, fmt.Errorf("failed to get identity by key: %w", err)
	}

	return s.rowToIdentity(&row)
}

// GetIdentityByAddress retrieves an identity by its server address
func (s *SQLStore) GetIdentityByAddress(ctx context.Context, serverAddress string) (*identitypb.Identity, error) {
	row, err := s.queries.GetIdentityByAddress(ctx, serverAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("identity not found")
		}
		return nil, fmt.Errorf("failed to get identity by address: %w", err)
	}

	return s.rowToIdentity(&row)
}

// UpdateLastSeen updates the last seen timestamp for an identity
func (s *SQLStore) UpdateLastSeen(ctx context.Context, serverAddress string, lastSeen time.Time) error {
	err := s.queries.UpdateLastSeen(ctx, serverAddress)
	if err != nil {
		return fmt.Errorf("failed to update last seen: %w", err)
	}

	return nil
}

// ListIdentities returns a paginated list of identities
func (s *SQLStore) ListIdentities(ctx context.Context) ([]*identitypb.Identity, error) {
	rows, err := s.queries.ListIdentities(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list identities: %w", err)
	}

	identities := make([]*identitypb.Identity, len(rows))
	for i, row := range rows {
		identity, err := s.rowToIdentity(&row)
		if err != nil {
			return nil, fmt.Errorf("failed to convert row to identity: %w", err)
		}
		identities[i] = identity
	}

	return identities, nil
}

// SetTrustStatus updates the trust status of an identity
func (s *SQLStore) SetTrustStatus(ctx context.Context, serverAddress string, trusted bool, updatedAt time.Time) error {
	params := sqlc.SetTrustStatusParams{
		ServerAddress: serverAddress,
		IsTrusted:     trusted,
	}

	err := s.queries.SetTrustStatus(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to set trust status: %w", err)
	}

	return nil
}

// DeleteIdentity removes an identity from the store
func (s *SQLStore) DeleteIdentity(ctx context.Context, publicKey []byte) error {
	err := s.queries.DeleteIdentity(ctx, publicKey)
	if err != nil {
		return fmt.Errorf("failed to delete identity: %w", err)
	}

	return nil
}

// CountIdentities returns the total count of identities (optionally filtered by trust status)
func (s *SQLStore) CountIdentities(ctx context.Context, trustedOnly bool) (int64, error) {
	count, err := s.queries.CountIdentities(ctx, trustedOnly)
	if err != nil {
		return 0, fmt.Errorf("failed to count identities: %w", err)
	}

	return count, nil
}

// rowToIdentity converts a SQLC row to IdentityWithTrust
func (s *SQLStore) rowToIdentity(row *sqlc.ZtgIdentity) (*identitypb.Identity, error) {
	identity := &identitypb.Identity{
		PublicKey:      row.PublicKey,
		ServerAddress:  row.ServerAddress,
		ServerName:     row.ServerName,
		OwnerName:      row.OwnerName,
		CreatedAt:      timestamppb.New(row.CreatedAt),
		IsTrusted:      row.IsTrusted,
		TrustUpdatedAt: timestamppb.New(row.TrustUpdatedAt),
	}

	// Parse capabilities JSON if present
	var capabilities []string
	if row.Capabilities != "" {
		if err := json.Unmarshal([]byte(row.Capabilities), &capabilities); err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
		identity.Capabilities = capabilities
	}

	return identity, nil
}
