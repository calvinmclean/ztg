package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestEnv(t *testing.T) {
	wd, _ := os.Getwd()
	projectRoot := filepath.Join(wd, "..")
	os.Chdir(projectRoot)
	t.Cleanup(func() {
		os.Chdir(wd)
	})
}

func TestKeyManager_LoadExampleKey(t *testing.T) {
	setupTestEnv(t)

	privKey, pubKey, err := loadKeyFromFile("keys/example_ed25519.pem")
	km := &KeyManager{privateKey: privKey, publicKey: pubKey, isExample: true}
	if err != nil {
		t.Fatalf("Failed to load example key: %v", err)
	}

	if km.PrivateKey() == nil {
		t.Fatal("Private key is nil")
	}

	if km.PublicKey() == nil {
		t.Fatal("Public key is nil")
	}

	if !km.IsExample() {
		t.Fatal("Key manager should detect example key")
	}
}

func TestIsExampleKey(t *testing.T) {
	setupTestEnv(t)

	privKey, pubKey, err := loadKeyFromFile("keys/example_ed25519.pem")
	km := &KeyManager{privateKey: privKey, publicKey: pubKey, isExample: true}
	if err != nil {
		t.Fatalf("Failed to load example key: %v", err)
	}

	if !isExampleKey(km.PublicKey()) {
		t.Fatal("Should detect example key")
	}
}

func TestValidateKeyUsage(t *testing.T) {
	setupTestEnv(t)

	privKey, pubKey, err := loadKeyFromFile("keys/example_ed25519.pem")
	km := &KeyManager{privateKey: privKey, publicKey: pubKey, isExample: true}
	if err != nil {
		t.Fatalf("Failed to load example key: %v", err)
	}

	// Test production environment
	err = ValidateKeyUsage(km.PublicKey(), "production")
	if err == nil {
		t.Fatal("Should error on example key in production")
	}

	// Test development environment
	err = ValidateKeyUsage(km.PublicKey(), "development")
	if err != nil {
		t.Fatalf("Should not error in development: %v", err)
	}
}

func TestSigner_BasicOperations(t *testing.T) {
	setupTestEnv(t)

	privKey, pubKey, err := loadKeyFromFile("keys/example_ed25519.pem")
	km := &KeyManager{privateKey: privKey, publicKey: pubKey, isExample: true}
	if err != nil {
		t.Fatalf("Failed to load example key: %v", err)
	}

	signer := NewSigner(km.PrivateKey(), "localhost:8080")

	message := []byte("test message")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	err = signer.Verify(message, signature, "localhost:8080")
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	// Test verification with wrong address
	err = signer.Verify(message, signature, "wrong:address")
	if err == nil {
		t.Fatal("Should fail verification with wrong address")
	}
}

func TestHashChain_BasicOperations(t *testing.T) {
	hc := NewHashChain()

	if hc.Sequence() != 0 {
		t.Fatalf("Expected sequence 0, got %d", hc.Sequence())
	}

	data1 := []byte("first data")
	hash1 := hc.NextHash(data1)

	if hc.Sequence() != 1 {
		t.Fatalf("Expected sequence 1, got %d", hc.Sequence())
	}

	data2 := []byte("second data")
	hash2 := hc.NextHash(data2)

	if hc.Sequence() != 2 {
		t.Fatalf("Expected sequence 2, got %d", hc.Sequence())
	}

	// Hashes should be different
	if string(hash1) == string(hash2) {
		t.Fatal("Hashes should be different for different data")
	}
}

func TestNewIdentity(t *testing.T) {
	setupTestEnv(t)

	privKey, pubKey, err := loadKeyFromFile("keys/example_ed25519.pem")
	km := &KeyManager{privateKey: privKey, publicKey: pubKey, isExample: true}
	if err != nil {
		t.Fatalf("Failed to load example key: %v", err)
	}

	capabilities := []string{"dice.roll", "factorfight.play"}
	identity := NewIdentity(km.PublicKey(), "localhost:8080", "test-server", "test-owner", "1.0.0", capabilities)

	if identity.ServerAddr != "localhost:8080" {
		t.Fatalf("Expected server address localhost:8080, got %s", identity.ServerAddr)
	}

	if identity.ServerName != "test-server" {
		t.Fatalf("Expected server name test-server, got %s", identity.ServerName)
	}

	if identity.OwnerName != "test-owner" {
		t.Fatalf("Expected owner name test-owner, got %s", identity.OwnerName)
	}

	if len(identity.Capabilities) != 2 {
		t.Fatalf("Expected 2 capabilities, got %d", len(identity.Capabilities))
	}

	if !identity.HasCapability("dice.roll") {
		t.Fatal("Should have dice.roll capability")
	}

	if identity.HasCapability("nonexistent") {
		t.Fatal("Should not have nonexistent capability")
	}
}
