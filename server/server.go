package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"ztg/config"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	gamepb.RegisterGameServiceServer(server, &gameService{
		serverConfig: serverConfig,
		keyManager:   keyManager,
		serverAddr:   serverConfig.Address,
		signedMode:   serverConfig.Signed,
		registry:     registry,
	})
	identitypb.RegisterIdentityServiceServer(server, &identityService{
		keyManager:   keyManager,
		serverConfig: serverConfig,
		registry:     registry,
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
