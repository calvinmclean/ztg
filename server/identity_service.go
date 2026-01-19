package server

import (
	"context"

	"ztg/config"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/protobuf/types/known/emptypb"
)

// identityService implements the IdentityService RPC defined in identity.proto.
type identityService struct {
	identitypb.UnimplementedIdentityServiceServer

	keyManager   *identity.KeyManager
	serverConfig config.ServerConfig
	registry     *registry
}

func (s *identityService) GetIdentity(ctx context.Context, req *emptypb.Empty) (*identitypb.Identity, error) {
	publicKey := s.keyManager.PublicKey()
	capabilities := s.registry.GameIDs()

	return &identitypb.Identity{
		PublicKey:     publicKey,
		ServerAddress: s.serverConfig.Address,
		ServerName:    s.serverConfig.ServerName,
		OwnerName:     s.serverConfig.OwnerName,
		Version:       s.serverConfig.Version,
		Capabilities:  capabilities,
		CreatedAt:     0,
	}, nil
}
