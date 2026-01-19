package factorfight_test

import (
	"context"
	"testing"
	"time"

	"ztg/config"
	"ztg/factorfight"
	gamepb "ztg/gen/go/game/v1"
	identitypb "ztg/gen/go/identity/v1"
	"ztg/identity"
	"ztg/server"
	ffserver "ztg/server/factorfight"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestTwoServerChallenge(t *testing.T) {
	keyConfig1 := config.KeyConfig{
		ServerAddress:      "localhost:50052",
		PrivateKeyPath:     "../../keys/example_ed25519.pem",
		OwnerPublicKeyFile: "../../keys/example_ed25519.pub.pem",
	}
	keyConfig2 := config.KeyConfig{
		ServerAddress:      "localhost:50053",
		PrivateKeyPath:     "../../keys/example_ed25519.pem",
		OwnerPublicKeyFile: "../../keys/example_ed25519.pub.pem",
	}

	km1, err := identity.NewKeyManager(keyConfig1)
	if err != nil {
		t.Fatalf("Failed to create KeyManager 1: %v", err)
	}
	km2, err := identity.NewKeyManager(keyConfig2)
	if err != nil {
		t.Fatalf("Failed to create KeyManager 2: %v", err)
	}

	// Create server configs
	var p1WinResult *bool
	serverConfig1 := config.ServerConfig{
		Address:    ":50052",
		ServerName: "test-server-1",
		OwnerName:  "test-user-1",
		Version:    "1.0.0",
		Signed:     false,
	}
	factorFightConfig1 := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, log factorfight.GameLog) {
			t.Logf("Server 1 - Win: %v, Log: %v", win, log)
			p1WinResult = &win
		},
	}
	ffserver1 := ffserver.NewService(factorFightConfig1, km1, "localhost:50052", false)

	var p2WinResult *bool
	serverConfig2 := config.ServerConfig{
		Address:    ":50053",
		ServerName: "test-server-2",
		OwnerName:  "test-user-2",
		Version:    "1.0.0",
		Signed:     false,
	}
	factorFightConfig2 := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, log factorfight.GameLog) {
			t.Logf("Server 2 - Win: %v, Log: %v", win, log)
			p2WinResult = &win
		},
	}
	ffserver2 := ffserver.NewService(factorFightConfig2, km2, "localhost:50053", false)

	// Create servers
	server1, err := server.NewServer(serverConfig1, km1)
	if err != nil {
		t.Fatalf("Failed to create server 1: %v", err)
	}
	server1.Register(ffserver1)

	server2, err := server.NewServer(serverConfig2, km2)
	if err != nil {
		t.Fatalf("Failed to create server 2: %v", err)
	}
	server2.Register(ffserver2)

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
		Target: "localhost:50052",
		GameId: ffserver.GameID,
	}

	// Signer can use either KM because they just use the same owner key
	signer := server.NewSigner(km1, ":50052")
	signature, err := signer.SignProto(req)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	signedReq := &gamepb.SignedChallengeRequest{
		Challenge: req,
		Signature: signature,
	}

	// Issue challenge request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Challenge(ctx, signedReq)
	if err != nil {
		t.Fatalf("Challenge request failed: %v", err)
	}

	// Verify response
	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	t.Logf("Challenge response - Win: %v, Message: %s", resp.Win, resp.Message)

	if p1WinResult != nil && p2WinResult != nil && *p1WinResult == *p2WinResult {
		t.Error("p1WinResult == p2WinResult and they should not match")
	}
}

func TestChallengeAuthorization(t *testing.T) {
	// Server config with owner public key
	keyConfig := config.KeyConfig{
		ServerAddress:      "localhost:50054",
		PrivateKeyPath:     "../../keys/example_ed25519.pem",
		OwnerPublicKeyFile: "../../keys/example_ed25519.pub.pem",
	}

	km, err := identity.NewKeyManager(keyConfig)
	if err != nil {
		t.Fatalf("Failed to create KeyManager: %v", err)
	}

	serverConfig := config.ServerConfig{
		Address:    ":50054",
		ServerName: "auth-test-server",
		OwnerName:  "test-user",
		Version:    "1.0.0",
		Signed:     true, // Enable signed mode
	}

	factorFightConfig := ffserver.Config{
		Strategy: factorfight.DefaultStrategy,
		OnGameComplete: func(win bool, log factorfight.GameLog) {
			t.Logf("Auth Test - Win: %v, Log: %v", win, log)
		},
	}
	ffserver := ffserver.NewService(factorFightConfig, km, "localhost:50054", true)

	srv, err := server.NewServer(serverConfig, km)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	srv.Register(ffserver)

	// Start server in goroutine
	go func() {
		if err := srv.Run(); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Cleanup function to stop server
	defer func() {
		srv.Stop()
	}()

	// Create client connection
	conn, err := grpc.NewClient(
		"localhost:50054",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := gamepb.NewGameServiceClient(conn)

	// Test 1: Invalid signature should fail
	req := &gamepb.ChallengeRequest{
		Target: "localhost:50055",
		GameId: "ztg.FactorFight.v1",
	}

	// Create a signature with correct signer address but invalid signature content
	signer := server.NewSigner(km, "localhost:50054")

	// Get a proper signature first, then corrupt it
	validSig, _ := signer.SignProto(req)

	// Create invalid signature by corrupting the signature bytes
	invalidSignature := &identitypb.Signature{
		Signature:     []byte("corrupted-signature-data"),
		SignerAddress: validSig.SignerAddress, // Use correct address
	}

	invalidSignedReq := &gamepb.SignedChallengeRequest{
		Challenge: req,
		Signature: invalidSignature,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Challenge(ctx, invalidSignedReq)
	if err == nil {
		t.Error("Invalid signature challenge should fail, but it succeeded")
		t.Logf("Response: %v", resp)
	} else {
		// Check for PermissionDenied status code
		st, ok := status.FromError(err)
		if !ok {
			t.Errorf("Expected gRPC status error, got: %v", err)
		} else if st.Code() != codes.PermissionDenied {
			t.Errorf("Expected PermissionDenied status code, got %v", st.Code())
		} else {
			t.Logf("Invalid signature challenge correctly failed with PermissionDenied: %v", st.Message())
		}
	}
}
