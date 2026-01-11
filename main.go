package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ztg/factorfight"
	"ztg/identity"
	"ztg/server"
)

func main() {
	generateKey := flag.Bool("generate-key", false, "Generate new Ed25519 key pair")
	outputPath := flag.String("output", "", "Output path for generated key")
	serverAddress := flag.String("server-address", "", "Server address to embed in key")
	signedMode := flag.Bool("signed", false, "Enable signed server mode (with identity verification)")
	flag.Parse()

	if *generateKey {
		if *outputPath == "" {
			*outputPath = "keys/server_ed25519.pem"
		}

		if err := generateKeyPair(*outputPath, *serverAddress); err != nil {
			log.Fatalf("Key generation failed: %v", err)
		}
		return
	}

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

func generateKeyPair(outputPath, serverAddress string) error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, privateKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	publicKeyPath := outputPath + ".pub"
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	if err := os.WriteFile(publicKeyPath, publicKeyPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	fmt.Printf("Generated key pair:\n")
	fmt.Printf("Private key: %s\n", outputPath)
	fmt.Printf("Public key:  %s\n", publicKeyPath)

	if serverAddress != "" {
		fmt.Printf("Server address: %s\n", serverAddress)
	}

	return nil
}
