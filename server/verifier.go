package server

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	identitypb "ztg/gen/go/identity/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

// IdentityCache represents a cached peer identity with expiration
type IdentityCache struct {
	Identity  *identitypb.Identity
	PublicKey ed25519.PublicKey
	ExpiresAt time.Time
}

// IdentityCacheManager handles thread-safe caching of peer identities
type IdentityCacheManager struct {
	cache map[string]*IdentityCache
	mutex sync.RWMutex
	ttl   time.Duration
}

// NewIdentityCacheManager creates a new cache manager with specified TTL
func NewIdentityCacheManager(ttl time.Duration) *IdentityCacheManager {
	return &IdentityCacheManager{
		cache: make(map[string]*IdentityCache),
		ttl:   ttl,
	}
}

// Get retrieves a cached identity, returning nil if not found or expired
func (cm *IdentityCacheManager) Get(peerAddr string) (*IdentityCache, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	cached, exists := cm.cache[peerAddr]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		// Remove expired entry
		cm.mutex.RUnlock()
		cm.mutex.Lock()
		delete(cm.cache, peerAddr)
		cm.mutex.Unlock()
		cm.mutex.RLock()
		return nil, false
	}

	return cached, true
}

// GetWithAddressCheck retrieves a cached identity with address verification
// Ensures cached identity's address matches the lookup key (prevents cache poisoning)
func (cm *IdentityCacheManager) GetWithAddressCheck(peerAddr string) (*IdentityCache, bool) {
	cached, exists := cm.Get(peerAddr)
	if !exists {
		return nil, false
	}

	// SECURITY: Verify cached identity's address matches the lookup key
	// This prevents scenarios where a cached key might be associated with wrong address
	if cached.Identity != nil && cached.Identity.ServerAddress != peerAddr {
		// Remove inconsistent entry to prevent cache pollution
		cm.mutex.Lock()
		delete(cm.cache, peerAddr)
		cm.mutex.Unlock()
		return nil, false
	}

	return cached, true
}

// Set stores an identity in the cache with expiration
func (cm *IdentityCacheManager) Set(peerAddr string, identity *identitypb.Identity) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache[peerAddr] = &IdentityCache{
		Identity:  identity,
		PublicKey: identity.PublicKey,
		ExpiresAt: time.Now().Add(cm.ttl),
	}
}

// SetPublicKey stores just a public key in the cache with expiration
func (cm *IdentityCacheManager) SetPublicKey(peerAddr string, publicKey ed25519.PublicKey) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache[peerAddr] = &IdentityCache{
		PublicKey: publicKey,
		ExpiresAt: time.Now().Add(cm.ttl),
	}
}

// Clear removes all entries from the cache
func (cm *IdentityCacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache = make(map[string]*IdentityCache)
}

// Size returns the number of cached entries
func (cm *IdentityCacheManager) Size() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.cache)
}

// Cleanup removes expired entries from the cache
func (cm *IdentityCacheManager) Cleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	for addr, cached := range cm.cache {
		if now.After(cached.ExpiresAt) {
			delete(cm.cache, addr)
		}
	}
}

// Verifier handles signature verification and identity management
type Verifier struct {
	identityCache *IdentityCacheManager
}

// NewVerifier creates a new Verifier instance
func NewVerifier(ttl time.Duration) *Verifier {
	return &Verifier{
		identityCache: NewIdentityCacheManager(ttl),
	}
}

// VerifyMessageSignature verifies a message signature
func (v *Verifier) VerifyMessageSignature(message []byte, signature *identitypb.Signature) error {
	if signature == nil {
		return fmt.Errorf("message is not signed")
	}

	if len(signature.Signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	if signature.SignerAddress == "" {
		return fmt.Errorf("empty signer address")
	}

	// Check cache first with address verification
	cached, exists := v.identityCache.GetWithAddressCheck(signature.SignerAddress)
	if exists {
		// SECURITY: Verify signature with cached public key
		hash := sha256.Sum256(message)
		if !ed25519.Verify(cached.PublicKey, hash[:], signature.Signature) {
			return fmt.Errorf("invalid signature for cached identity %s", signature.SignerAddress)
		}
		return nil
	}

	// Not cached: fetch fresh identity and verify
	peerIdentity, err := v.getPeerIdentity(signature.SignerAddress)
	if err != nil {
		return fmt.Errorf("failed to fetch peer identity: %w", err)
	}

	// SECURITY: Verify fetched identity's address matches the claimed address
	if peerIdentity.ServerAddress != signature.SignerAddress {
		return fmt.Errorf("address mismatch: identity reports %s, signature claims %s",
			peerIdentity.ServerAddress, signature.SignerAddress)
	}

	// Verify signature with fresh public key
	hash := sha256.Sum256(message)
	if !ed25519.Verify(peerIdentity.PublicKey, hash[:], signature.Signature) {
		return fmt.Errorf("invalid signature for %s", signature.SignerAddress)
	}

	// Cache ONLY after successful verification
	v.identityCache.Set(signature.SignerAddress, peerIdentity)
	return nil
}

// VerifyOrderedSignature verifies an ordered signature
func (v *Verifier) VerifyOrderedSignature(message []byte, signature *identitypb.OrderedSignature) error {
	if signature == nil {
		return fmt.Errorf("message is not signed")
	}

	if len(signature.Signature) == 0 {
		return fmt.Errorf("empty signature")
	}

	if signature.SignerAddress == "" {
		return fmt.Errorf("empty signer address")
	}

	if signature.Sequence == 0 {
		return fmt.Errorf("sequence must be greater than 0")
	}

	// Check cache first with address verification
	cached, exists := v.identityCache.GetWithAddressCheck(signature.SignerAddress)
	if exists {
		// SECURITY: Verify signature with cached public key
		hash := sha256.Sum256(message)
		if !ed25519.Verify(cached.PublicKey, hash[:], signature.Signature) {
			return fmt.Errorf("invalid ordered signature for cached identity %s", signature.SignerAddress)
		}

		// TODO: implement hash chain verification for PreviousHash
		// For now, we'll just accept any previous hash
		return nil
	}

	// Not cached: fetch fresh identity and verify
	peerIdentity, err := v.getPeerIdentity(signature.SignerAddress)
	if err != nil {
		return fmt.Errorf("failed to fetch peer identity: %w", err)
	}

	// SECURITY: Verify fetched identity's address matches the claimed address
	if peerIdentity.ServerAddress != signature.SignerAddress {
		return fmt.Errorf("address mismatch in ordered signature: identity reports %s, signature claims %s",
			peerIdentity.ServerAddress, signature.SignerAddress)
	}

	// Verify signature with fresh public key
	hash := sha256.Sum256(message)
	if !ed25519.Verify(peerIdentity.PublicKey, hash[:], signature.Signature) {
		return fmt.Errorf("invalid ordered signature for %s", signature.SignerAddress)
	}

	// Cache ONLY after successful verification
	v.identityCache.Set(signature.SignerAddress, peerIdentity)

	// TODO: implement hash chain verification for PreviousHash
	// For now, we'll just accept any previous hash

	return nil
}

// getPeerIdentity fetches peer identity from their gRPC service
func (v *Verifier) getPeerIdentity(peerAddr string) (*identitypb.Identity, error) {
	// Connect to peer's identity service
	conn, err := grpc.NewClient(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
	}
	defer conn.Close()

	identityClient := identitypb.NewIdentityServiceClient(conn)
	identity, err := identityClient.GetIdentity(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, fmt.Errorf("failed to get identity from peer %s: %w", peerAddr, err)
	}

	return identity, nil
}

// AddPeerIdentity adds a peer identity to the verifier's cache
func (v *Verifier) AddPeerIdentity(peerAddr string, publicKey ed25519.PublicKey) {
	v.identityCache.SetPublicKey(peerAddr, publicKey)
}

// ClearIdentityCache clears all cached identities
func (v *Verifier) ClearIdentityCache() {
	v.identityCache.Clear()
}

// CleanupIdentityCache removes expired entries from the cache
func (v *Verifier) CleanupIdentityCache() {
	v.identityCache.Cleanup()
}

// GetCacheSize returns the number of cached identities
func (v *Verifier) GetCacheSize() int {
	return v.identityCache.Size()
}
