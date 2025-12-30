package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ztg/dice"
	protodice "ztg/proto/dice"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	sides := uint8(10)

	addr := os.Getenv("ADDR")
	peerAddr := os.Getenv("PEER_ADDR")

	if addr != "" && peerAddr != "" {
		if os.Getenv("MODE") == "grpc" {
			runGRPC(sides, addr, peerAddr)
		} else {
			runHTTP(sides, addr, peerAddr)
		}
	} else {
		runSimple(sides)
	}
}

func runHTTP(sides uint8, addr, peerAddr string) {
	tp := dice.NewHTTPPeer(peerAddr)
	d, _ := dice.NewRoller(addr, sides, tp)

	s := http.Server{Addr: addr, Handler: tp}
	go s.ListenAndServe()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Running... press Ctrl+C to stop")

	ctx, bigCancel := context.WithCancel(context.Background())
	defer bigCancel()

	defer s.Close()

	go func() {
		<-sigCh
		fmt.Println("\nCtrl+C received, exiting loop")
		bigCancel()
	}()

	for {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)

		roll := d.Roll(ctx)
		out, err := roll.Get(8)
		cancel()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(out)

		time.Sleep(1 * time.Second)
	}
}

func runGRPC(sides uint8, addr, peerAddr string) {
	conn, err := grpc.NewClient(
		peerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := protodice.NewRollerServiceClient(conn)

	peer, err := dice.NewGRPCPeer(client)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create gRPC peer: %v\n", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer()
	protodice.RegisterRollerServiceServer(grpcServer, peer)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("[ERROR] Failed to bind gRPC server: %v\n", err)
		fmt.Printf("[DEBUG] [ERROR] Failed to bind gRPC server on %s. Error details: %v\n", addr, err)
		return
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			fmt.Printf("[ERROR] Failed to serve gRPC: %v\n", err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d, err := dice.NewRoller(addr, sides, peer)
	if err != nil {
		fmt.Printf("[ERROR] Failed to create Roller: %v\n", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ctx, bigCancel := context.WithCancel(context.Background())
	defer bigCancel()

	go func() {
		<-sigCh
		bigCancel()
		grpcServer.GracefulStop()
	}()

	for {
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		roll := d.Roll(ctx)
		out, err := roll.Get(8)
		cancel()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(out)
		time.Sleep(1 * time.Second)
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
