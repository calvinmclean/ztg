package server

import (
	"context"
	"fmt"
	"log"

	"ztg/config"
	gamepb "ztg/gen/go/game/v1"
	"ztg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	serverConfig config.ServerConfig
	keyManager   *identity.KeyManager
	serverAddr   string
	signedMode   bool
	registry     *registry
}

func (s *gameService) Challenge(ctx context.Context, req *gamepb.SignedChallengeRequest) (*gamepb.ChallengeResponse, error) {
	if req.Signature == nil || len(req.Signature.Signature) == 0 {
		return nil, fmt.Errorf("owner signature required for Challenge")
	}

	v := NewVerifier(0)
	v.AddPeerIdentity("owner", s.keyManager.PublicKey())
	v.VerifySignatureProto(req.Challenge, req.Signature)

	err := v.VerifySignatureProto(req.Challenge, req.Signature)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	conn, err := grpc.NewClient(
		req.Challenge.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	challenge, ok := s.registry.challenge(req.Challenge.GameId)
	if !ok {
		return nil, fmt.Errorf("unknown game: %q %v", req.Challenge.GameId, s.registry.GameIDs())
	}
	return challenge(ctx, conn)
}
