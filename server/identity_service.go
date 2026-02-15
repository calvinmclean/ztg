package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"fmt"
	"net/url"
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

	// Validate public key
	if len(req.Identity.PublicKey) != ed25519.PublicKeySize {
		return nil, status.Error(codes.InvalidArgument, "invalid public key size")
	}

	url, err := url.Parse(req.Identity.ServerAddress)
	// if there is no error (valid URL), then get the server's Identity and compare
	if err == nil && url.Host != "" {
		peerID, err := getPeerIdentity(req.Identity.ServerAddress)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting peer identity: %v", err)
		}

		if !bytes.Equal(req.Identity.PublicKey, peerID.PublicKey) {
			return nil, status.Error(codes.InvalidArgument, "provided public key does not match key provided by server")
		}
	}

	if req.Identity.CreatedAt != nil {
		req.Identity.CreatedAt = timestamppb.New(time.Now())
	}

	if req.Identity.IsTrusted {
		return nil, status.Error(codes.InvalidArgument, "cannot created trusted identity")
	}
	req.Identity.TrustUpdatedAt = req.Identity.CreatedAt

	if err := s.store.InsertIdentity(ctx, req.Identity); err != nil {
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

	return &identitypb.ListIdentitiesResponse{
		Identities: identities,
	}, nil
}

// SetTrust updates the trust status of an identity (owner only)
func (s *identityService) SetTrust(ctx context.Context, req *identitypb.SetTrustRequest) (*emptypb.Empty, error) {
	// Verify owner signature by creating a simple message with the request data
	message := fmt.Appendf(nil, "set-trust:%s:%v", req.ServerAddress, req.Trusted)
	if err := s.verifyOwnerSignatureBytes(message, req.Signature); err != nil {
		return nil, status.Errorf(codes.PermissionDenied, "owner signature verification failed: %v", err)
	}

	// Check if store is available
	if s.store == nil {
		return nil, status.Error(codes.Unavailable, "identity store not available")
	}

	// Update trust status
	if err := s.store.SetTrustStatus(ctx, req.ServerAddress, req.Trusted, time.Now()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set trust status: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// verifyOwnerSignatureBytes verifies that a signature was created by the server owner
func (s *identityService) verifyOwnerSignatureBytes(message []byte, signature *identitypb.Signature) error {
	if signature == nil || len(signature.Signature) == 0 {
		return fmt.Errorf("owner signature required for Challenge")
	}

	v := NewVerifier(0, s.store)
	v.AddPeerIdentity("owner", s.keyManager.OwnerPublicKey())

	err := v.VerifyMessageSignature(message, signature)
	if err != nil {
		return status.Error(codes.PermissionDenied, fmt.Errorf("signature verification failed: %w", err).Error())
	}

	return nil
}
