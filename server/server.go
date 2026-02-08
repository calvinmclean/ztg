package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"time"

	"github.com/calvinmclean/ztg/config"
	gamepb "github.com/calvinmclean/ztg/gen/go/proto/game/v1"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"

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
	store    store.Store // SQL store for persistent identity storage
}

// NewServer initializes a new GRPC server.
func NewServer(serverConfig config.ServerConfig, databaseConfig config.DatabaseConfig, keyManager *identity.KeyManager) (*Server, error) {
	logLevel := parseLogLevel(serverConfig.LogLevel)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	logger.Debug("initializing new server", "port", serverConfig.Port, "log_level", serverConfig.LogLevel)

	if err := identity.ValidateKeyUsage(keyManager.PublicKey()); err != nil {
		return nil, err
	}

	// Initialize SQL store if database is configured
	var sqlStore store.Store
	if databaseConfig.DatabasePath != "" || databaseConfig.DatabaseURL != "" {
		storeConfig := store.Config{
			DatabaseURL:               databaseConfig.DatabaseURL,
			DatabaseAuthToken:         databaseConfig.DatabaseAuthToken,
			DatabasePath:              databaseConfig.DatabasePath,
			DatabaseLongPollTimeoutMs: databaseConfig.DatabaseLongPollTimeoutMs,
			DatabaseBootstrapIfEmpty:  databaseConfig.DatabaseBootstrapIfEmpty,
			UseEmbeddedReplica:        databaseConfig.DatabaseUseEmbeddedReplica,
		}

		var err error
		sqlStore, err = store.NewSQLStore(storeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SQL store: %w", err)
		}
		logger.Info("initialized SQL identity store", "path", databaseConfig.DatabasePath, "url", databaseConfig.DatabaseURL)
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
		store:        sqlStore,
	})
	identitypb.RegisterIdentityServiceServer(server, &identityService{
		keyManager:   keyManager,
		serverConfig: serverConfig,
		registry:     registry,
		store:        sqlStore,
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
		store:    sqlStore,
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

// GetStore returns the SQL store for use by game services
func (s *Server) GetStore() store.Store {
	return s.store
}

// Stop gracefully stops the gRPC server.
func (s *Server) Stop() {
	s.logger.Info("stopping gRPC server")
	s.cancel()

	// Close SQL store if initialized
	if s.store != nil {
		if err := s.store.Close(); err != nil {
			s.logger.Error("error closing SQL store", "error", err)
		}
	}

	s.server.GracefulStop()
}
