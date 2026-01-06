package main

import (
	"fmt"
	"log"
	"os"

	"ztg/factorfight"
	"ztg/server"
)

func main() {
	addr := os.Getenv("ADDR")

	cfg := server.Config{
		Addr: addr,
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
