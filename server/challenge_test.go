package server

import (
	"context"
	"testing"
	"time"

	"ztg/factorfight"
	gamepb "ztg/gen/go/game/v1"
	"ztg/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestTwoServerChallenge(t *testing.T) {
	keyConfig1 := identity.KeyConfig{
		ServerAddress:  "localhost:50052",
		PrivateKeyPath: "../keys/example_ed25519.pem",
	}
	keyConfig2 := identity.KeyConfig{
		ServerAddress:  "localhost:50053",
		PrivateKeyPath: "../keys/example_ed25519.pem",
	}

	// Create server configs
	var p1WinResult *bool
	cfg1 := Config{
		Addr:       ":50052",
		ServerName: "test-server-1",
		OwnerName:  "test-user-1",
		Version:    "1.0.0",
		SignedMode: false,
		KeyConfig:  keyConfig1,
		FactorFight: FactorFightConfig{
			Strategy: factorfight.DefaultStrategy,
			OnGameComplete: func(win bool, log factorfight.GameLog) {
				t.Logf("Server 1 - Win: %v, Log: %v", win, log)
				p1WinResult = &win
			},
		},
	}

	var p2WinResult *bool
	cfg2 := Config{
		Addr:       ":50053",
		ServerName: "test-server-2",
		OwnerName:  "test-user-2",
		Version:    "1.0.0",
		SignedMode: false,
		KeyConfig:  keyConfig2,
		FactorFight: FactorFightConfig{
			Strategy: factorfight.DefaultStrategy,
			OnGameComplete: func(win bool, log factorfight.GameLog) {
				t.Logf("Server 2 - Win: %v, Log: %v", win, log)
				p2WinResult = &win
			},
		},
	}

	// Create servers
	server1, err := NewServer(cfg1)
	if err != nil {
		t.Fatalf("Failed to create server 1: %v", err)
	}

	server2, err := NewServer(cfg2)
	if err != nil {
		t.Fatalf("Failed to create server 2: %v", err)
	}

	// Start servers in goroutines
	go func() {
		if err := server1.Run(); err != nil {
			t.Errorf("Server 1 failed: %v", err)
		}
	}()

	go func() {
		if err := server2.Run(); err != nil {
			t.Errorf("Server 2 failed: %v", err)
		}
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Cleanup function to stop servers
	defer func() {
		server1.Stop()
		server2.Stop()
	}()

	// Create client connection to server 2 (the one that will receive the challenge)
	conn, err := grpc.NewClient(
		"localhost:50053",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server 2: %v", err)
	}
	defer conn.Close()

	// Create game service client
	client := gamepb.NewGameServiceClient(conn)

	// Create challenge request (matching the Taskfile.yml command)
	req := &gamepb.ChallengeRequest{
		Target:   "localhost:50052",
		GameName: "factorfight",
	}

	// Issue challenge request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Challenge(ctx, req)
	if err != nil {
		t.Fatalf("Challenge request failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	t.Logf("Challenge response - Win: %v, Message: %s", resp.Win, resp.Message)

	if *p1WinResult == *p2WinResult {
		t.Error("p1WinResult == p2WinResult and they should not match")
	}
}
