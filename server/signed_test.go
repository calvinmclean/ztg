package server

import (
	"crypto/ed25519"
	"testing"
	"time"

	"ztg/identity"

	identitypb "ztg/gen/go/identity/v1"
)

func TestSignedServer_VerifyMessageSignature(t *testing.T) {
	// Create two key managers for testing
	km1, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 1: %v", err)
	}

	km2, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create signed server
	signedServer := NewSignedServer(km1, "localhost:8080")

	// Add peer identity to cache
	signedServer.AddPeerIdentity("localhost:8081", km2.PublicKey())

	// Create a test message
	message := []byte("test message for signature verification")

	// Sign the message with km2 (peer's key)
	signer := identity.NewSigner(km2.PrivateKey(), "localhost:8081")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Create signature proto
	sigProto := &identitypb.Signature{
		Signature:     signature,
		SignerAddress: "localhost:8081",
	}

	// Verify the signature - should succeed
	err = signedServer.verifyMessageSignature(message, sigProto)
	if err != nil {
		t.Fatalf("Failed to verify valid signature: %v", err)
	}

	// Test with wrong message - should fail
	wrongMessage := []byte("wrong message")
	err = signedServer.verifyMessageSignature(wrongMessage, sigProto)
	if err == nil {
		t.Fatal("Should fail verification with wrong message")
	}

	// Test with nil signature - should fail
	err = signedServer.verifyMessageSignature(message, nil)
	if err == nil {
		t.Fatal("Should fail verification with nil signature")
	}

	// Test with empty signature - should fail
	emptySig := &identitypb.Signature{
		Signature:     []byte{},
		SignerAddress: "localhost:8081",
	}
	err = signedServer.verifyMessageSignature(message, emptySig)
	if err == nil {
		t.Fatal("Should fail verification with empty signature")
	}

	// Test with empty signer address - should fail
	noAddrSig := &identitypb.Signature{
		Signature:     signature,
		SignerAddress: "",
	}
	err = signedServer.verifyMessageSignature(message, noAddrSig)
	if err == nil {
		t.Fatal("Should fail verification with empty signer address")
	}
}

func TestSignedServer_VerifyOrderedSignature(t *testing.T) {
	// Create two key managers for testing
	km1, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 1: %v", err)
	}

	km2, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create signed server
	signedServer := NewSignedServer(km1, "localhost:8080")

	// Add peer identity to cache
	signedServer.AddPeerIdentity("localhost:8081", km2.PublicKey())

	// Create a test message
	message := []byte("test message for ordered signature verification")

	// Sign the message with km2 (peer's key)
	signer := identity.NewSigner(km2.PrivateKey(), "localhost:8081")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Create ordered signature proto
	orderedSig := &identitypb.OrderedSignature{
		Signature:     signature,
		SignerAddress: "localhost:8081",
		PreviousHash:  []byte("previous_hash"),
		Sequence:      1,
	}

	// Verify the ordered signature - should succeed
	err = signedServer.verifyOrderedSignature(message, orderedSig)
	if err != nil {
		t.Fatalf("Failed to verify valid ordered signature: %v", err)
	}

	// Test with sequence 0 - should fail
	zeroSeqSig := &identitypb.OrderedSignature{
		Signature:     signature,
		SignerAddress: "localhost:8081",
		PreviousHash:  []byte("previous_hash"),
		Sequence:      0,
	}
	err = signedServer.verifyOrderedSignature(message, zeroSeqSig)
	if err == nil {
		t.Fatal("Should fail verification with sequence 0")
	}
}

func TestSignedServer_AddPeerIdentity(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	signedServer := NewSignedServer(km, "localhost:8080")

	// Add peer identity
	signedServer.AddPeerIdentity("localhost:8081", km.PublicKey())

	// Verify the peer identity was added (by checking cache length)
	if signedServer.GetCacheSize() != 1 {
		t.Fatalf("Expected 1 identity in cache, got %d", signedServer.GetCacheSize())
	}

	// Verify the correct public key was stored by attempting to retrieve it
	cached, exists := signedServer.identityCache.Get("localhost:8081")
	if !exists {
		t.Fatal("Peer identity not found in cache")
	}

	if !ed25519.PublicKey.Equal(cached.PublicKey, km.PublicKey()) {
		t.Fatal("Cached public key doesn't match original")
	}
}

func TestIdentityCacheManager_Expiration(t *testing.T) {
	// Create cache manager with very short TTL for testing
	cache := NewIdentityCacheManager(10 * time.Millisecond)

	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Add an identity to cache
	cache.SetPublicKey("localhost:8081", km.PublicKey())

	// Should be found immediately
	_, exists := cache.Get("localhost:8081")
	if !exists {
		t.Fatal("Identity should be found immediately after caching")
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Should not be found after expiration
	_, exists = cache.Get("localhost:8081")
	if exists {
		t.Fatal("Identity should be expired after TTL")
	}
}

func TestIdentityCacheManager_Cleanup(t *testing.T) {
	cache := NewIdentityCacheManager(50 * time.Millisecond)

	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Add multiple identities
	cache.SetPublicKey("localhost:8081", km.PublicKey())
	cache.SetPublicKey("localhost:8082", km.PublicKey())

	// Should have 2 entries
	if cache.Size() != 2 {
		t.Fatalf("Expected 2 identities in cache, got %d", cache.Size())
	}

	// Wait for some entries to expire
	time.Sleep(60 * time.Millisecond)

	// Add a fresh entry
	cache.SetPublicKey("localhost:8083", km.PublicKey())

	// Run cleanup
	cache.Cleanup()

	// Should have only 1 entry (the fresh one)
	if cache.Size() != 1 {
		t.Fatalf("Expected 1 identity in cache after cleanup, got %d", cache.Size())
	}

	// Verify the fresh entry is still there
	_, exists := cache.Get("localhost:8083")
	if !exists {
		t.Fatal("Fresh identity should still be in cache after cleanup")
	}
}

func TestSignedServer_CacheManagement(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	signedServer := NewSignedServer(km, "localhost:8080")

	// Add multiple peer identities
	signedServer.AddPeerIdentity("localhost:8081", km.PublicKey())
	signedServer.AddPeerIdentity("localhost:8082", km.PublicKey())

	if signedServer.GetCacheSize() != 2 {
		t.Fatalf("Expected 2 identities in cache, got %d", signedServer.GetCacheSize())
	}

	// Clear cache
	signedServer.ClearIdentityCache()

	if signedServer.GetCacheSize() != 0 {
		t.Fatalf("Expected 0 identities in cache after clear, got %d", signedServer.GetCacheSize())
	}
}

func TestSignedServer_AddressMismatchSecurity(t *testing.T) {
	// Create two different key managers
	km1, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 1: %v", err)
	}

	// Note: km2 not used in this test but kept for consistency with other tests
	_, err = identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8082",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	signedServer := NewSignedServer(km1, "localhost:8080")

	// Create a fake identity that has mismatched address
	fakeIdentity := &identitypb.Identity{
		PublicKey:     km1.PublicKey(),
		ServerAddress: "different-address", // Intentionally mismatched
		ServerName:    "fake-server",
		OwnerName:     "fake-owner",
		Version:       "1.0.0",
		Capabilities:  []string{"test"},
		CreatedAt:     0,
	}

	// Cache the fake identity
	signedServer.identityCache.Set("localhost:8081", fakeIdentity)

	// Create a message signed with km1's private key
	message := []byte("test message")
	signer := identity.NewSigner(km1.PrivateKey(), "localhost:8081")
	signature, err := signer.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	// Create signature proto claiming localhost:8081
	sigProto := &identitypb.Signature{
		Signature:     signature,
		SignerAddress: "localhost:8081",
	}

	// This should fail due to address mismatch in cached identity
	err = signedServer.verifyMessageSignature(message, sigProto)
	if err == nil {
		t.Fatal("Should fail verification due to address mismatch")
	}
}

func TestIdentityCacheManager_GetWithAddressCheck(t *testing.T) {
	cache := NewIdentityCacheManager(5 * time.Minute)

	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create identity with matching address
	validIdentity := &identitypb.Identity{
		PublicKey:     km.PublicKey(),
		ServerAddress: "localhost:8081", // Matching address
		ServerName:    "test-server",
		OwnerName:     "test-owner",
		Version:       "1.0.0",
		Capabilities:  []string{"test"},
		CreatedAt:     0,
	}

	// Cache the valid identity
	cache.Set("localhost:8081", validIdentity)

	// Should be found with address check
	cached, exists := cache.GetWithAddressCheck("localhost:8081")
	if !exists {
		t.Fatal("Valid identity should be found with address check")
	}
	if cached.Identity.ServerAddress != "localhost:8081" {
		t.Fatal("Cached identity address should match")
	}

	// Create identity with mismatched address
	mismatchedIdentity := &identitypb.Identity{
		PublicKey:     km.PublicKey(),
		ServerAddress: "different-address", // Mismatched address
		ServerName:    "test-server",
		OwnerName:     "test-owner",
		Version:       "1.0.0",
		Capabilities:  []string{"test"},
		CreatedAt:     0,
	}

	// Cache the mismatched identity
	cache.Set("localhost:8082", mismatchedIdentity)

	// Should NOT be found due to address mismatch
	_, exists = cache.GetWithAddressCheck("localhost:8082")
	if exists {
		t.Fatal("Mismatched identity should not be found with address check")
	}

	// Verify the mismatched entry was removed from cache
	if cache.Size() != 1 {
		t.Fatalf("Expected 1 identity after address mismatch cleanup, got %d", cache.Size())
	}
}
