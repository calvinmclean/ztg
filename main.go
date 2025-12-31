package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ztg/dice"
	"ztg/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	sides := uint8(10)

	addr := os.Getenv("ADDR")
	peerAddr := os.Getenv("PEER_ADDR")

	if addr != "" && peerAddr != "" {
		runGRPCServer(addr, peerAddr)
	} else {
		runSimple(sides)
	}
}

// runSimple runs a single roll using two in-memory Rollers with ChannelPeers
func runSimple(sides uint8) {
	peer1, peer2 := dice.NewChannelPeers()

	d1, _ := dice.NewRoller("One", sides, peer1)
	d2, _ := dice.NewRoller("Two", sides, peer2)

	ctx := context.Background()
	roll1 := d1.Roll(ctx)
	roll2 := d2.Roll(ctx)

	r1, _ := roll1.GetOne()
	r2, _ := roll2.GetOne()

	if r1 != r2 {
		panic("invalid roll")
	}

	fmt.Println(r1)
}

func runGRPCServer(addr, peerAddr string) {
	conn, err := grpc.NewClient(
		peerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	cfg := server.GRPCServerConfig{
		Addr: addr,
	}
	grpcServer, err := server.NewGRPCServer(cfg)
	if err != nil {
		log.Fatalf("Server initialization failed: %v", err)
	}

	grpcServer.Run()
}
