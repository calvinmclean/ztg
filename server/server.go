package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"ztg/config"
	dicepb "ztg/gen/go/dice/v1"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	// DefaultVerifierTTL is the default time-to-live for verifier caches
	DefaultVerifierTTL = 5 * time.Minute
	// DefaultDieSides is the default number of sides for dice rolling
	DefaultDieSides = 10
)

// Server represents a gRPC server instance.
type Server struct {
	server   *grpc.Server
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	registry *registry
}

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	serverConfig config.ServerConfig
	keyManager   *identity.KeyManager
	serverAddr   string
	signedMode   bool
	registry     *registry
}

// identityService implements the IdentityService RPC defined in identity.proto.
type identityService struct {
	identitypb.UnimplementedIdentityServiceServer

	keyManager   *identity.KeyManager
	serverConfig config.ServerConfig
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

func (s *identityService) GetIdentity(ctx context.Context, req *emptypb.Empty) (*identitypb.Identity, error) {
	publicKey := s.keyManager.PublicKey()

	capabilities := []string{
		"dice.roll",
		"factorfight.play",
		"game.challenge",
	}

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

// NewServer initializes a new GRPC server.
func NewServer(serverConfig config.ServerConfig, keyManager *identity.KeyManager) (*Server, error) {
	if err := identity.ValidateKeyUsage(keyManager.PublicKey(), os.Getenv("ENV")); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", serverConfig.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", serverConfig.Address, err)
	}

	registry := newRegistry()
	server := grpc.NewServer()
	// TODO: merge gameService and Server
	gamepb.RegisterGameServiceServer(server, &gameService{
		serverConfig: serverConfig,
		keyManager:   keyManager,
		serverAddr:   serverConfig.Address,
		signedMode:   serverConfig.Signed,
		registry:     registry,
	})
	dicepb.RegisterDiceServiceServer(server, &diceService{
		keyManager: keyManager,
		serverAddr: serverConfig.Address,
		signedMode: serverConfig.Signed,
	})
	identitypb.RegisterIdentityServiceServer(server, &identityService{
		keyManager:   keyManager,
		serverConfig: serverConfig,
	})

	reflection.Register(server)

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		server:   server,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
		registry: registry,
	}, nil
}

func (s *Server) Register(g GameService) {
	g.Register(s.server)
	s.registry.registerGame(g)
}

// Run starts the gRPC server.
func (s *Server) Run() error {
	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.cancel()
	s.server.GracefulStop()
}
