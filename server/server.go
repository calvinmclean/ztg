package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	dicepb "ztg/gen/go/dice/v1"
	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server represents a gRPC server instance.
type Server struct {
	server   *grpc.Server
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

// Config holds configuration for initializing a gRPC server.
type Config struct {
	Addr       string
	ServerName string
	OwnerName  string
	Version    string
	SignedMode bool

	KeyConfig   identity.KeyConfig
	FactorFight FactorFightConfig
}

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	cfg        Config
	keyManager *identity.KeyManager
	serverAddr string
	signedMode bool
}

// identityService implements the IdentityService RPC defined in identity.proto.
type identityService struct {
	identitypb.UnimplementedIdentityServiceServer

	keyManager *identity.KeyManager
	config     Config
}

func (s *gameService) Challenge(ctx context.Context, req *gamepb.ChallengeRequest) (*gamepb.ChallengeResponse, error) {
	conn, err := grpc.NewClient(
		req.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	switch strings.ToLower(req.GameName) {
	case "factorfight":
		return playFactorFight(ctx, conn, s.cfg.FactorFight, s.keyManager, s.serverAddr, s.signedMode)
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
		ServerAddress: s.config.Addr,
		ServerName:    s.config.ServerName,
		OwnerName:     s.config.OwnerName,
		Version:       s.config.Version,
		Capabilities:  capabilities,
		CreatedAt:     0,
	}, nil
}

// NewServer initializes a new GRPC server.
func NewServer(cfg Config) (*Server, error) {
	keyManager, err := identity.NewKeyManager(cfg.KeyConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key manager: %w", err)
	}

	if err := identity.ValidateKeyUsage(keyManager.PublicKey(), os.Getenv("ENV")); err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", cfg.Addr, err)
	}

	server := grpc.NewServer()
	gamepb.RegisterGameServiceServer(server, &gameService{
		cfg:        cfg,
		keyManager: keyManager,
		serverAddr: cfg.Addr,
		signedMode: cfg.SignedMode,
	})
	factorfightpb.RegisterFactorFightServiceServer(server, &factorfightService{
		cfg:        cfg.FactorFight,
		keyManager: keyManager,
		serverAddr: cfg.Addr,
		signedMode: cfg.SignedMode,
	})
	dicepb.RegisterDiceServiceServer(server, &diceService{
		keyManager: keyManager,
		serverAddr: cfg.Addr,
		signedMode: cfg.SignedMode,
	})
	identitypb.RegisterIdentityServiceServer(server, &identityService{
		keyManager: keyManager,
		config:     cfg,
	})

	reflection.Register(server)

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		server:   server,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Run starts the gRPC server.
func (g *Server) Run() error {
	return g.server.Serve(g.listener)
}

// Stop gracefully stops the gRPC server.
func (g *Server) Stop() {
	g.cancel()
	g.server.GracefulStop()
}
