package server

import (
	"context"
	"fmt"

	"ztg/config"
	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"

	ffserver "ztg/server/factorfight"
	highrollserver "ztg/server/highroll"

	"github.com/urfave/cli/v3"
)

var Command = &cli.Command{
	Name:  "server",
	Usage: "Zero Trust Gaming server",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "config",
			Usage: "Path to configuration file",
		},
	},
	Action: func(ctx context.Context, cmd *cli.Command) error {
		configPath := cmd.String("config")

		cfg := config.DefaultConfig()

		// Load config file if provided
		if configPath != "" {
			fileCfg, err := config.LoadFromFile(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config from %s: %w", configPath, err)
			}
			cfg = fileCfg
		}

		// Override with environment variables
		config.LoadFromEnv(cfg)

		fmt.Println("🔐 Starting server with identity verification")
		fmt.Println("   - All RPCs require valid signatures")
		fmt.Println("   - Server will sign all responses with its private key")
		fmt.Println("   - Hash chain verification enabled for FactorFight")

		// Set default private key path if not specified
		if cfg.Identity.PrivateKeyPath == "" && cfg.Identity.PrivateKey == "" {
			cfg.Identity.PrivateKeyPath = "keys/example_ed25519.pem"
		}

		keyManager, err := identity.NewKeyManager(cfg.Identity)
		if err != nil {
			return fmt.Errorf("failed to initialize key manager: %w", err)
		}

		grpcServer, err := server.NewServer(cfg.Server, keyManager)
		if err != nil {
			return fmt.Errorf("server initialization failed: %w", err)
		}

		ffCfg := ffserver.Config{
			Strategy: factorfight.DefaultStrategy,
			OnGameComplete: func(win bool, log factorfight.GameLog) {
				fmt.Println("Win:", win)
				fmt.Println(log)
			},
		}
		ffService := ffserver.NewService(ffCfg, keyManager, cfg.Identity.ServerAddress)
		grpcServer.Register(ffService)

		highrollService := highrollserver.NewService(keyManager, cfg.Identity.ServerAddress)
		grpcServer.Register(highrollService)

		grpcServer.Run()
		return nil
	},
}
