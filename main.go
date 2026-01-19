package main

import (
	"flag"
	"fmt"
	"log"

	"ztg/config"
	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"

	ffserver "ztg/server/factorfight"
	highrollserver "ztg/server/highroll"

	"github.com/kelseyhightower/envconfig"
)

func main() {
	signedMode := flag.Bool("signed", false, "Enable signed server mode (with identity verification)")
	flag.Parse()

	if *signedMode {
		fmt.Println("🔐 Starting server in SIGNED mode with identity verification")
		fmt.Println("   - All signed RPCs will require valid signatures")
		fmt.Println("   - Server will sign all responses with its private key")
		fmt.Println("   - Hash chain verification enabled for FactorFight")
	} else {
		fmt.Println("🎲 Starting server in REGULAR mode")
		fmt.Println("   - Unsigned RPCs are available")
		fmt.Println("   - Use --signed flag to enable identity verification")
	}

	cfg := config.DefaultConfig()
	_ = envconfig.Process("", cfg)
	cfg.Key.PrivateKeyPath = "keys/example_ed25519.pem"

	keyManager, err := identity.NewKeyManager(cfg.Key)
	if err != nil {
		log.Fatalf("failed to initialize key manager: %v", err)
	}

	cfg.Server.Signed = *signedMode

	grpcServer, err := server.NewServer(cfg.Server, keyManager)
	if err != nil {
		log.Fatalf("Server initialization failed: %v", err)
	}

	ffCfg := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, log factorfight.GameLog) {
			fmt.Println("Win:", win)
			fmt.Println(log)
		},
	}
	ffService := ffserver.NewService(ffCfg, keyManager, cfg.Server.Address, *signedMode)
	grpcServer.Register(ffService)

	highrollService := highrollserver.NewService(keyManager, cfg.Server.Address, *signedMode)
	grpcServer.Register(highrollService)

	grpcServer.Run()
}
