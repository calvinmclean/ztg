package server

import (
	"crypto/ed25519"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/calvinmclean/ztg/config"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"

	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
)

func TestSigner_BasicOperations(t *testing.T) {
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
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
	orderedSig, err := signer.SignOrderedMessage(message, []byte{}, 1)
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

	if len(orderedSig.PreviousHash) != 0 {
		t.Fatalf("Expected empty previous hash, got %s", string(orderedSig.PreviousHash))
	}
}

func TestVerifier_VerifyMessageSignature(t *testing.T) {
	km2, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8081",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create verifier
	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)

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
	km2, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8081",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 2: %v", err)
	}

	// Create verifier
	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)

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
		PreviousHash:  []byte{}, // Empty previous hash for sequence 1
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
		PreviousHash:  []byte{}, // Empty previous hash for sequence 0 (should still fail)
		Sequence:      0,
	}
	err = verifier.VerifyOrderedSignature(message, zeroSeqSig)
	if err == nil {
		t.Fatal("Should fail verification with sequence 0")
	}
}

func TestVerifier_AddPeerIdentity(t *testing.T) {
	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)

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

	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
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

	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
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
	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)

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
	km1, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8081",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager 1: %v", err)
	}

	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)

	// Create a fake identity that has mismatched address

	fakeIdentity := &identitypb.Identity{
		PublicKey:     km1.PublicKey(),
		ServerAddress: "different-address", // Intentionally mismatched
		ServerName:    "fake-server",
		OwnerName:     "fake-owner",
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

	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
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
	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Test getting separate components
	signer := NewSigner(km, "localhost:8080")
	if signer.Address() != "localhost:8080" {
		t.Fatalf("Signer address should match server address")
	}

	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)
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

func TestVerifier_HashChainVerification(t *testing.T) {
	km, err := identity.NewKeyManager(config.IdentityConfig{
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	var sqlStore store.Store = nil
	verifier := NewVerifier(5*time.Minute, sqlStore)
	verifier.AddPeerIdentity("localhost:8081", km.PublicKey())

	signer := identity.NewSigner(km.PrivateKey(), "localhost:8081")

	// Test valid hash chain sequence
	message1 := []byte("message 1")
	signature1, err := signer.Sign(message1)
	if err != nil {
		t.Fatalf("Failed to sign message 1: %v", err)
	}
	hash1 := sha256.Sum256(message1)

	orderedSig1 := &identitypb.OrderedSignature{
		Signature:     signature1,
		SignerAddress: "localhost:8081",
		PreviousHash:  []byte{}, // Empty for first message
		Sequence:      1,
	}

	err = verifier.VerifyOrderedSignature(message1, orderedSig1)
	if err != nil {
		t.Fatalf("Failed to verify first message in hash chain: %v", err)
	}

	// Test second message with correct previous hash
	message2 := []byte("message 2")
	signature2, err := signer.Sign(message2)
	if err != nil {
		t.Fatalf("Failed to sign message 2: %v", err)
	}
	hash2 := sha256.Sum256(message2)

	orderedSig2 := &identitypb.OrderedSignature{
		Signature:     signature2,
		SignerAddress: "localhost:8081",
		PreviousHash:  hash1[:], // Correct previous hash
		Sequence:      2,
	}

	err = verifier.VerifyOrderedSignature(message2, orderedSig2)
	if err != nil {
		t.Fatalf("Failed to verify second message in hash chain: %v", err)
	}

	// Test third message with correct previous hash
	message3 := []byte("message 3")
	signature3, err := signer.Sign(message3)
	if err != nil {
		t.Fatalf("Failed to sign message 3: %v", err)
	}

	orderedSig3 := &identitypb.OrderedSignature{
		Signature:     signature3,
		SignerAddress: "localhost:8081",
		PreviousHash:  hash2[:], // Correct previous hash
		Sequence:      3,
	}

	err = verifier.VerifyOrderedSignature(message3, orderedSig3)
	if err != nil {
		t.Fatalf("Failed to verify third message in hash chain: %v", err)
	}

	// Test hash chain violation - wrong previous hash
	wrongHashSig := &identitypb.OrderedSignature{
		Signature:     signature3,
		SignerAddress: "localhost:8081",
		PreviousHash:  []byte("wrong_hash"), // Wrong previous hash
		Sequence:      4,
	}

	err = verifier.VerifyOrderedSignature(message3, wrongHashSig)
	if err == nil {
		t.Fatal("Should fail verification with wrong previous hash")
	}

	// Test sequence violation - wrong sequence number
	wrongSeqSig := &identitypb.OrderedSignature{
		Signature:     signature3,
		SignerAddress: "localhost:8081",
		PreviousHash:  hash2[:], // Correct previous hash
		Sequence:      10,       // Wrong sequence number
	}

	err = verifier.VerifyOrderedSignature(message3, wrongSeqSig)
	if err == nil {
		t.Fatal("Should fail verification with wrong sequence number")
	}

	// Test sequence 1 with non-empty previous hash should fail
	seq1WithHash := &identitypb.OrderedSignature{
		Signature:     signature1,
		SignerAddress: "localhost:8081",
		PreviousHash:  []byte("some_hash"), // Should be empty for sequence 1
		Sequence:      1,
	}

	err = verifier.VerifyOrderedSignature(message1, seq1WithHash)
	if err == nil {
		t.Fatal("Should fail verification with non-empty previous hash for sequence 1")
	}
}
