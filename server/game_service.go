package server

import (
	"context"
	"fmt"
	"log"
	"strings"

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

func (s *gameService) Challenge(ctx context.Context, req *gamepb.ChallengeRequest) (*gamepb.ChallengeResponse, error) {
	conn, err := grpc.NewClient(
		req.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	challenge, ok := s.registry.challenge(strings.ToLower(req.GameName))
	if ok {
		return challenge(ctx, conn)
	}
	// TODO: Remove the switch and register the HighRoll

	switch strings.ToLower(req.GameName) {
	case "highroll":
		return playHighRoll(ctx, conn, s.keyManager, s.serverAddr, s.signedMode)
	default:
		return nil, fmt.Errorf("unknown game: %q", req.GameName)
	}
}
