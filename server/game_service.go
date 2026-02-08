package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/calvinmclean/ztg/config"
	gamepb "github.com/calvinmclean/ztg/gen/go/proto/game/v1"
	"github.com/calvinmclean/ztg/grpcutil"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	serverConfig config.ServerConfig
	keyManager   *identity.KeyManager
	serverAddr   string
	registry     *registry
	logger       *slog.Logger
	store        store.Store // SQL store for persistent identity storage
}

func (s *gameService) Challenge(ctx context.Context, req *gamepb.SignedChallengeRequest) (*gamepb.ChallengeResponse, error) {
	logger := s.logger
	logger.Debug("handling challenge request", "game_id", req.Challenge.GameId, "target", req.Challenge.Target)

	if req.Signature == nil || len(req.Signature.Signature) == 0 {
		return nil, fmt.Errorf("owner signature required for Challenge")
	}

	v := NewVerifier(0, s.store)
	v.AddPeerIdentity("owner", s.keyManager.OwnerPublicKey())

	err := v.VerifySignatureProto(req.Challenge, req.Signature)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, fmt.Errorf("signature verification failed: %w", err).Error())
	}

	logger.Debug("connecting to target server", "target", req.Challenge.Target)
	conn, err := grpcutil.NewClient(req.Challenge.Target)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Errorf("failed to connect to target %q: %w", req.Challenge.Target, err).Error())
	}

	challenge, ok := s.registry.challenge(req.Challenge.GameId)
	if !ok {
		return nil, status.Error(codes.NotFound, fmt.Errorf("unknown game: %q %v", req.Challenge.GameId, s.registry.GameIDs()).Error())
	}

	logger.Debug("executing challenge", "game_id", req.Challenge.GameId)
	return challenge(ctx, conn)
}
