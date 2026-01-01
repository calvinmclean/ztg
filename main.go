package main

import (
	"log"
	"os"

	"ztg/server"
)

func main() {
	addr := os.Getenv("ADDR")

	runGRPCServer(addr)
}

func runGRPCServer(addr string) {
	cfg := server.GRPCServerConfig{
		Addr: addr,
	}
	grpcServer, err := server.NewGRPCServer(cfg)
	if err != nil {
		log.Fatalf("Server initialization failed: %v", err)
	}

	grpcServer.Run()
}
