package server

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/calvinmclean/ztg/config"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// identityService implements the IdentityService RPC defined in identity.proto.
type identityService struct {
	identitypb.UnimplementedIdentityServiceServer

	keyManager   *identity.KeyManager
	serverConfig config.ServerConfig
	registry     *registry
	store        store.Store // Optional SQL store for persistent identity storage
}

func (s *identityService) GetIdentity(ctx context.Context, req *emptypb.Empty) (*identitypb.Identity, error) {
	publicKey := s.keyManager.PublicKey()
	capabilities := s.registry.GameIDs()

	return &identitypb.Identity{
		PublicKey:     publicKey,
		ServerAddress: s.keyManager.ServerAddress(),
		ServerName:    s.keyManager.ServerName(),
		OwnerName:     s.keyManager.OwnerName(),
		Capabilities:  capabilities,
		CreatedAt:     timestamppb.Now(),
	}, nil
}

// AddIdentity adds a new identity to the store
func (s *identityService) AddIdentity(ctx context.Context, req *identitypb.AddIdentityRequest) (*emptypb.Empty, error) {
	// Check if store is available
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "identity store not available")
	}

	// Check if store is available
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "identity store not available")
	}

	if req.Identity.CreatedAt != nil {
		req.Identity.CreatedAt = timestamppb.New(time.Now())
	}

	// Create identity with trust
	identityWithTrust := store.NewIdentityWithTrust(req.Identity)
	identityWithTrust.SetTrust(false, time.Now()) // Default to not trusted

	// Insert into store
	if err := s.store.InsertIdentity(ctx, identityWithTrust); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert identity: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// ListIdentities returns a paginated list of identities
func (s *identityService) ListIdentities(ctx context.Context, req *identitypb.ListIdentitiesRequest) (*identitypb.ListIdentitiesResponse, error) {
	// Check if store is available
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "identity store not available")
	}

	// Get identities from store
	identities, err := s.store.ListIdentities(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list identities: %v", err)
	}

	// Convert to protobuf format
	response := &identitypb.ListIdentitiesResponse{
		Identities: make([]*identitypb.IdentityWithTrust, len(identities)),
	}

	for i, identity := range identities {
		response.Identities[i] = &identitypb.IdentityWithTrust{
			Identity:       identity.Identity,
			IsTrusted:      identity.IsTrusted,
			TrustUpdatedAt: timestamppb.New(identity.TrustUpdatedAt),
		}
	}

	return response, nil
}

// SetTrust updates the trust status of an identity (owner only)
func (s *identityService) SetTrust(ctx context.Context, req *identitypb.SetTrustRequest) (*emptypb.Empty, error) {
	// Verify owner signature by creating a simple message with the request data
	message := fmt.Appendf(nil, "set-trust:%x:%v", req.PublicKey, req.Trusted)
	if err := s.verifyOwnerSignatureBytes(message, req.Signature); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "owner signature verification failed: %v", err)
	}

	// Check if store is available
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "identity store not available")
	}

	// Update trust status
	if err := s.store.SetTrustStatus(ctx, req.PublicKey, req.Trusted, time.Now()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set trust status: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// verifyOwnerSignatureBytes verifies that a signature was created by the server owner
func (s *identityService) verifyOwnerSignatureBytes(message []byte, signature *identitypb.Signature) error {
	if signature == nil {
		return fmt.Errorf("signature is required")
	}

	// Hash the message
	hash := sha256.Sum256(message)

	// Verify with owner's private key (derived from server's public key)
	ownerPublicKey := s.keyManager.PublicKey()
	if !ed25519.Verify(ownerPublicKey, hash[:], signature.Signature) {
		return fmt.Errorf("signature verification failed")
	}

	// Verify the signer address matches our server address
	if signature.SignerAddress != s.keyManager.ServerAddress() {
		return fmt.Errorf("signer address mismatch: expected %s, got %s",
			s.keyManager.ServerAddress(), signature.SignerAddress)
	}

	return nil
}
