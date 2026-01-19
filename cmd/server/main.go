package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ztg/config"
	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"

	ffserver "ztg/server/factorfight"
	highrollserver "ztg/server/highroll"

	"github.com/kelseyhightower/envconfig"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "ztg-server",
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
			_ = envconfig.Process("", cfg)

			if cfg.Server.Signed {
				fmt.Println("🔐 Starting server in SIGNED mode with identity verification")
				fmt.Println("   - All signed RPCs will require valid signatures")
				fmt.Println("   - Server will sign all responses with its private key")
				fmt.Println("   - Hash chain verification enabled for FactorFight")
			} else {
				fmt.Println("🎲 Starting server in REGULAR mode")
				fmt.Println("   - Unsigned RPCs are available")
				fmt.Println("   - Use --signed flag to enable identity verification")
			}

			// Set default private key path if not specified
			if cfg.Key.PrivateKeyPath == "" && cfg.Key.PrivateKey == "" {
				cfg.Key.PrivateKeyPath = "keys/example_ed25519.pem"
			}

			keyManager, err := identity.NewKeyManager(cfg.Key)
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
			ffService := ffserver.NewService(ffCfg, keyManager, cfg.Server.Address, cfg.Server.Signed)
			grpcServer.Register(ffService)

			highrollService := highrollserver.NewService(keyManager, cfg.Server.Address, cfg.Server.Signed)
			grpcServer.Register(highrollService)

			grpcServer.Run()
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
