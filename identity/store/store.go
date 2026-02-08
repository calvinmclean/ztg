package store

import (
	"context"
	"time"

	identitypb "github.com/calvinmclean/ztg/gen/go/identity/v1"
)

// Store defines the interface for identity persistence operations
type Store interface {
	// InsertIdentity adds a new identity to the store
	InsertIdentity(ctx context.Context, identity *IdentityWithTrust) error

	// GetIdentityByKey retrieves an identity by its public key
	GetIdentityByKey(ctx context.Context, publicKey []byte) (*IdentityWithTrust, error)

	// GetIdentityByAddress retrieves an identity by its server address
	GetIdentityByAddress(ctx context.Context, serverAddress string) (*IdentityWithTrust, error)

	// UpdateLastSeen updates the last seen timestamp for an identity
	UpdateLastSeen(ctx context.Context, serverAddress string, lastSeen time.Time) error

	// ListIdentities returns a paginated list of identities
	ListIdentities(ctx context.Context, trustedOnly bool, pageSize int, offset int) ([]*IdentityWithTrust, error)

	// SetTrustStatus updates the trust status of an identity
	SetTrustStatus(ctx context.Context, publicKey []byte, trusted bool, updatedAt time.Time) error

	// DeleteIdentity removes an identity from the store
	DeleteIdentity(ctx context.Context, publicKey []byte) error

	// CountIdentities returns the total count of identities (optionally filtered by trust status)
	CountIdentities(ctx context.Context, trustedOnly bool) (int64, error)

	// Close closes the database connection
	Close() error
}

// IdentityWithTrust represents an identity with trust information
type IdentityWithTrust struct {
	*identitypb.Identity
	IsTrusted      bool      `json:"is_trusted"`
	TrustUpdatedAt time.Time `json:"trust_updated_at"`
}

// NewIdentityWithTrust creates a new IdentityWithTrust from a protobuf Identity
func NewIdentityWithTrust(identity *identitypb.Identity) *IdentityWithTrust {
	return &IdentityWithTrust{
		Identity: identity,
	}
}

// SetTrust sets the trust status and updated timestamp
func (i *IdentityWithTrust) SetTrust(trusted bool, updatedAt time.Time) {
	i.IsTrusted = trusted
	i.TrustUpdatedAt = updatedAt
}
