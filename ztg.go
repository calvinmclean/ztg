// Package ztg provides a simplified API for setting up ZTG game servers.
package ztg

import (
	"context"
	"fmt"

	"github.com/calvinmclean/ztg/config"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/server"
	ffserver "github.com/calvinmclean/ztg/server/factorfight"
	highrollserver "github.com/calvinmclean/ztg/server/highroll"
)

// Option is a functional option for configuring the server.
type Option func(*ztgServer)

// ztgServer wraps the underlying gRPC server with a simplified setup interface.
type ztgServer struct {
	*server.Server
	keyManager *identity.KeyManager
	cfg        config.Config
}

// LoadConfig loads configuration from environment variables and validates it.
// This is a convenience wrapper around config.LoadFromEnv and cfg.Validate().
//
// Example usage:
//
//	cfg, err := ztg.LoadConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadConfig() (config.Config, error) {
	cfg := config.Config{}
	config.LoadFromEnv(&cfg)

	if err := cfg.Validate(); err != nil {
		return config.Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Run creates and runs a fully initialized ZTG server from the provided configuration.
// It handles:
//   - Loading and validating the configuration
//   - Initializing the identity key manager
//   - Setting up the gRPC server with identity services
//   - Configuring the SQL store if a database URL is provided
//   - Registering game services via options
//
// The server runs until the context is cancelled or an error occurs.
//
// Example usage:
//
//	cfg, err := ztg.LoadConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	err = ztg.Run(ctx, cfg,
//	    ztg.WithFactorFight(ffserver.Config{
//	        Strategy: factorfight.DefaultStrategy,
//	    }),
//	    ztg.WithHighRoll(),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func Run(ctx context.Context, cfg config.Config, opts ...Option) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	keyManager, err := identity.NewKeyManager(cfg.Identity)
	if err != nil {
		return fmt.Errorf("failed to initialize key manager: %w", err)
	}

	grpcServer, err := server.NewServer(cfg.Server, cfg.Database, keyManager)
	if err != nil {
		return fmt.Errorf("server initialization failed: %w", err)
	}

	s := &ztgServer{
		Server:     grpcServer,
		keyManager: keyManager,
		cfg:        cfg,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s.Server.Run(ctx)
}

// WithFactorFight adds the FactorFight game service to the server.
// The config parameter allows customizing the strategy and game completion callback.
func WithFactorFight(cfg ffserver.Config) Option {
	return func(s *ztgServer) {
		service := ffserver.NewService(cfg, s.keyManager, s.cfg.Identity.ServerAddress, s.Server.GetStore(), s.Server.GetLogger())
		s.Register(service)
	}
}

// WithHighRoll adds the HighRoll dice game service to the server.
func WithHighRoll() Option {
	return func(s *ztgServer) {
		service := highrollserver.NewService(s.keyManager, s.cfg.Identity.ServerAddress, s.Server.GetStore())
		s.Register(service)
	}
}
