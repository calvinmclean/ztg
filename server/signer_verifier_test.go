package server

import (
	"crypto/ed25519"
	"testing"
	"time"

	"ztg/identity"

	identitypb "ztg/gen/go/identity/v1"
)

func TestSigner_BasicOperations(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	signer := NewSigner(km, "localhost:8080")

	// Test address
	if signer.Address() != "localhost:8080" {
		t.Fatalf("Expected address localhost:8080, got %s", signer.Address())
	}

	// Test message signing
	message := []byte("test message")
	signature, err := signer.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if signature.Signature == nil {
		t.Fatal("Signature should not be nil")
	}

	if signature.SignerAddress != "localhost:8080" {
		t.Fatalf("Expected signer address localhost:8080, got %s", signature.SignerAddress)
	}

	// Test ordered message signing
	orderedSig, err := signer.SignOrderedMessage(message, []byte("prev_hash"), 1)
	if err != nil {
		t.Fatalf("Failed to sign ordered message: %v", err)
	}

	if orderedSig.Signature == nil {
		t.Fatal("Ordered signature should not be nil")
	}

	if orderedSig.SignerAddress != "localhost:8080" {
		t.Fatalf("Expected signer address localhost:8080, got %s", orderedSig.SignerAddress)
	}

	if orderedSig.Sequence != 1 {
		t.Fatalf("Expected sequence 1, got %d", orderedSig.Sequence)
	}

	if string(orderedSig.PreviousHash) != "prev_hash" {
		t.Fatalf("Expected previous hash prev_hash, got %s", string(orderedSig.PreviousHash))
	}
}

func TestVerifier_VerifyMessageSignature(t *testing.T) {
	km2, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create verifier
	verifier := NewVerifier(5 * time.Minute)

	// Add peer identity to cache
	verifier.AddPeerIdentity("localhost:8081", km2.PublicKey())

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
	err = verifier.VerifyMessageSignature(message, sigProto)
	if err != nil {
		t.Fatalf("Failed to verify valid signature: %v", err)
	}

	// Test with wrong message - should fail
	wrongMessage := []byte("wrong message")
	err = verifier.VerifyMessageSignature(wrongMessage, sigProto)
	if err == nil {
		t.Fatal("Should fail verification with wrong message")
	}

	// Test with nil signature - should fail
	err = verifier.VerifyMessageSignature(message, nil)
	if err == nil {
		t.Fatal("Should fail verification with nil signature")
	}

	// Test with empty signature - should fail
	emptySig := &identitypb.Signature{
		Signature:     []byte{},
		SignerAddress: "localhost:8081",
	}
	err = verifier.VerifyMessageSignature(message, emptySig)
	if err == nil {
		t.Fatal("Should fail verification with empty signature")
	}

	// Test with empty signer address - should fail
	noAddrSig := &identitypb.Signature{
		Signature:     signature,
		SignerAddress: "",
	}
	err = verifier.VerifyMessageSignature(message, noAddrSig)
	if err == nil {
		t.Fatal("Should fail verification with empty signer address")
	}
}

func TestVerifier_VerifyOrderedSignature(t *testing.T) {
	km2, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create verifier
	verifier := NewVerifier(5 * time.Minute)

	// Add peer identity to cache
	verifier.AddPeerIdentity("localhost:8081", km2.PublicKey())

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
	err = verifier.VerifyOrderedSignature(message, orderedSig)
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
	err = verifier.VerifyOrderedSignature(message, zeroSeqSig)
	if err == nil {
		t.Fatal("Should fail verification with sequence 0")
	}
}

func TestVerifier_AddPeerIdentity(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	verifier := NewVerifier(5 * time.Minute)

	// Add peer identity
	verifier.AddPeerIdentity("localhost:8081", km.PublicKey())

	// Verify the peer identity was added (by checking cache length)
	if verifier.GetCacheSize() != 1 {
		t.Fatalf("Expected 1 identity in cache, got %d", verifier.GetCacheSize())
	}

	// Verify the correct public key was stored by attempting to retrieve it
	cached, exists := verifier.identityCache.Get("localhost:8081")
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

func TestVerifier_CacheManagement(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	verifier := NewVerifier(5 * time.Minute)

	// Add multiple peer identities
	verifier.AddPeerIdentity("localhost:8081", km.PublicKey())
	verifier.AddPeerIdentity("localhost:8082", km.PublicKey())

	if verifier.GetCacheSize() != 2 {
		t.Fatalf("Expected 2 identities in cache, got %d", verifier.GetCacheSize())
	}

	// Clear cache
	verifier.ClearIdentityCache()

	if verifier.GetCacheSize() != 0 {
		t.Fatalf("Expected 0 identities in cache after clear, got %d", verifier.GetCacheSize())
	}
}

func TestVerifier_AddressMismatchSecurity(t *testing.T) {
	// Create two different key managers
	km1, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8081",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 1: %v", err)
	}

	verifier := NewVerifier(5 * time.Minute)

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

	// Cache fake identity
	verifier.identityCache.Set("localhost:8081", fakeIdentity)

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
	err = verifier.VerifyMessageSignature(message, sigProto)
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

	// Cache valid identity
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

	// Cache mismatched identity
	cache.Set("localhost:8082", mismatchedIdentity)

	// Should NOT be found due to address mismatch
	_, exists = cache.GetWithAddressCheck("localhost:8082")
	if exists {
		t.Fatal("Mismatched identity should not be found with address check")
	}

	// Verify mismatched entry was removed from cache
	if cache.Size() != 1 {
		t.Fatalf("Expected 1 identity after address mismatch cleanup, got %d", cache.Size())
	}
}

func TestSigner_Verifier_Integration(t *testing.T) {
	km, err := identity.NewKeyManager(identity.KeyConfig{
		PrivateKeyPath: "../keys/example_ed25519.pem",
		ServerAddress:  "localhost:8080",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Test getting separate components
	signer := NewSigner(km, "localhost:8080")
	if signer.Address() != "localhost:8080" {
		t.Fatalf("Signer address should match server address")
	}

	verifier := NewVerifier(5 * time.Minute)
	if verifier.GetCacheSize() != 0 {
		t.Fatalf("New verifier should have empty cache")
	}

	// Test that components work independently
	message := []byte("test message")
	signature, err := signer.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign with component signer: %v", err)
	}

	// Manually add the identity to verifier for testing
	verifier.AddPeerIdentity("localhost:8080", km.PublicKey())

	// Verify signature with component verifier
	err = verifier.VerifyMessageSignature(message, signature)
	if err != nil {
		t.Fatalf("Failed to verify with component verifier: %v", err)
	}

	// Verify that cache operations work
	if verifier.GetCacheSize() != 1 {
		t.Fatalf("Expected 1 entry in verifier cache")
	}

	verifier.ClearIdentityCache()
	if verifier.GetCacheSize() != 0 {
		t.Fatalf("Expected empty cache after clear")
	}
}
