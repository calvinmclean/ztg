package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	dicepb "ztg/gen/go/dice/v1"
	factorfightpb "ztg/gen/go/factorfight/v1"
	gamepb "ztg/gen/go/game/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
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
	Addr string

	FactorFight FactorFightConfig
}

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer

	cfg Config
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
		return playFactorFight(ctx, conn, s.cfg.FactorFight)
	case "highroll":
		return playHighRoll(ctx, conn)
	default:
		return nil, fmt.Errorf("unknown game: %q", req.GameName)
	}
}

// NewServer initializes a new GRPC server.
func NewServer(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", cfg.Addr, err)
	}

	server := grpc.NewServer()
	gamepb.RegisterGameServiceServer(server, &gameService{cfg: cfg})
	factorfightpb.RegisterFactorFightServiceServer(server, &factorfightService{cfg: cfg.FactorFight})
	dicepb.RegisterDiceServiceServer(server, &diceService{})

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
