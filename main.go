package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"
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

	addr := os.Getenv("ADDR")

	cfg := server.Config{
		Addr:       addr,
		ServerName: "ztg-server",
		OwnerName:  "ztg-user",
		Version:    "1.0.0",
		SignedMode: *signedMode,
		KeyConfig: identity.KeyConfig{
			ServerAddress: addr,
		},
		FactorFight: server.FactorFightConfig{
			Strategy: factorfight.DefaultStrategy,
			OnGameComplete: func(win bool, log factorfight.GameLog) {
				fmt.Println("Win:", win)
				fmt.Println(log)
			},
		},
	}
	grpcServer, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Server initialization failed: %v", err)
	}

	grpcServer.Run()
}
