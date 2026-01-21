package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
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

// parseLogLevel converts a string log level to slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Server represents a gRPC server instance.
type Server struct {
	server   *grpc.Server
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	registry *registry
	logger   *slog.Logger
}

// NewServer initializes a new GRPC server.
func NewServer(serverConfig config.ServerConfig, keyManager *identity.KeyManager) (*Server, error) {
	logLevel := parseLogLevel(serverConfig.LogLevel)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	logger.Debug("initializing new server", "port", serverConfig.Port, "log_level", serverConfig.LogLevel)

	if err := identity.ValidateKeyUsage(keyManager.PublicKey()); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf(":%d", serverConfig.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind gRPC server on %s: %w", addr, err)
	}

	registry := newRegistry()
	server := grpc.NewServer()
	gamepb.RegisterGameServiceServer(server, &gameService{
		serverConfig: serverConfig,
		keyManager:   keyManager,
		serverAddr:   keyManager.ServerAddress(),
		registry:     registry,
		logger:       logger,
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
		logger:   logger,
	}, nil
}

func (s *Server) Register(g GameService) {
	g.Register(s.server)
	s.registry.registerGame(g)
}

// Run starts the gRPC server.
func (s *Server) Run() error {
	s.logger.Info("starting gRPC server", "address", s.listener.Addr())
	return s.server.Serve(s.listener)
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.cancel()
	s.server.GracefulStop()
}
