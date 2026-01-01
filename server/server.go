package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	dicepb "ztg/gen/proto/dice"
	factorfightpb "ztg/gen/proto/factorfight"
	gamepb "ztg/gen/proto/game"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// GRPCServer represents a gRPC server instance.
type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
}

// GRPCServerConfig holds configuration for initializing a gRPC server.
type GRPCServerConfig struct {
	Addr string
}

// gameService implements the GameService RPC defined in game.proto.
type gameService struct {
	gamepb.UnimplementedGameServiceServer
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
		return playFactorFight(ctx, conn)
	case "highroll":
		return playHighRoll(ctx, conn)
	default:
		return nil, fmt.Errorf("unknown game: %q", req.GameName)
	}
}

// NewGRPCServer initializes a new GRPC server.
func NewGRPCServer(cfg GRPCServerConfig) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", cfg.Addr, err)
	}

	server := grpc.NewServer()
	gamepb.RegisterGameServiceServer(server, &gameService{})
	factorfightpb.RegisterFactorFightServiceServer(server, &factorfightService{})
	dicepb.RegisterDiceServiceServer(server, &diceService{})

	reflection.Register(server)

	ctx, cancel := context.WithCancel(context.Background())

	return &GRPCServer{
		server:   server,
		listener: listener,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Run starts the gRPC server.
func (g *GRPCServer) Run() error {
	return g.server.Serve(g.listener)
}

// Stop gracefully stops the gRPC server.
func (g *GRPCServer) Stop() {
	g.cancel()
	g.server.GracefulStop()
}
