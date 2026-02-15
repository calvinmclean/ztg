package store

import (
	"context"
	"time"

	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
)

// Store defines the interface for identity persistence operations
type Store interface {
	// InsertIdentity adds a new identity to the store
	InsertIdentity(ctx context.Context, identity *identitypb.Identity) error
	// GetIdentityByAddress retrieves an identity by its server address
	GetIdentityByAddress(ctx context.Context, serverAddress string) (*identitypb.Identity, error)
	// UpdateLastSeen updates the last seen timestamp for an identity
	UpdateLastSeen(ctx context.Context, serverAddress string, lastSeen time.Time) error
	// ListIdentities returns a paginated list of identities
	ListIdentities(ctx context.Context) ([]*identitypb.Identity, error)
	// SetTrustStatus updates the trust status of an identity
	SetTrustStatus(ctx context.Context, serverAddress string, trusted bool, updatedAt time.Time) error
	// Close closes the database connection
	Close() error
}
