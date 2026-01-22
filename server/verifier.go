package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	identitypb "ztg/gen/go/identity/v1"
	"ztg/grpcutil"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// IdentityCache represents a cached peer identity with expiration
type IdentityCache struct {
	Identity  *identitypb.Identity
	PublicKey ed25519.PublicKey
	ExpiresAt *time.Time
}

// HashChainEntry represents an entry in the hash chain for verification
type HashChainEntry struct {
	Hash      []byte
	Sequence  uint64
	ExpiresAt time.Time
}

// HashChainManager handles thread-safe tracking of hash chains for each peer
type HashChainManager struct {
	chains map[string]*HashChainEntry
	mutex  sync.RWMutex
	ttl    time.Duration
}

// NewHashChainManager creates a new hash chain manager with specified TTL
func NewHashChainManager(ttl time.Duration) *HashChainManager {
	return &HashChainManager{
		chains: make(map[string]*HashChainEntry),
		ttl:    ttl,
	}
}

// VerifyAndAdd verifies a new hash chain entry and adds it if valid
func (hm *HashChainManager) VerifyAndAdd(peerAddr string, previousHash []byte, currentHash []byte, sequence uint64) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	// For sequence 1, there should be no previous hash (or it should be empty/zero)
	if sequence == 1 {
		if len(previousHash) != 0 {
			return fmt.Errorf("sequence 1 should have empty previous hash")
		}
	} else {
		// For sequences > 1, verify the previous hash matches what we have stored
		lastEntry, exists := hm.chains[peerAddr]
		if !exists {
			return fmt.Errorf("no previous hash entry found for peer %s", peerAddr)
		}

		// Check if the stored entry has expired
		if time.Now().After(lastEntry.ExpiresAt) {
			delete(hm.chains, peerAddr)
			return fmt.Errorf("previous hash entry has expired for peer %s", peerAddr)
		}

		// Verify the previous hash matches what we expect
		if !bytes.Equal(lastEntry.Hash, previousHash) {
			return fmt.Errorf("previous hash mismatch for peer %s: expected %x, got %x",
				peerAddr, lastEntry.Hash, previousHash)
		}

		// Verify sequence is sequential
		if sequence != lastEntry.Sequence+1 {
			return fmt.Errorf("sequence number mismatch for peer %s: expected %d, got %d",
				peerAddr, lastEntry.Sequence+1, sequence)
		}
	}

	// Add the new entry to the chain
	hm.chains[peerAddr] = &HashChainEntry{
		Hash:      currentHash,
		Sequence:  sequence,
		ExpiresAt: time.Now().Add(hm.ttl),
	}

	return nil
}

// Clear removes all entries from the hash chain
func (hm *HashChainManager) Clear() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.chains = make(map[string]*HashChainEntry)
}

// Cleanup removes expired entries from the hash chains
func (hm *HashChainManager) Cleanup() {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	now := time.Now()
	for addr, entry := range hm.chains {
		if now.After(entry.ExpiresAt) {
			delete(hm.chains, addr)
		}
	}
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
	if cached.ExpiresAt != nil && time.Now().After(*cached.ExpiresAt) {
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

	var expires *time.Time
	if cm.ttl > 0 {
		e := time.Now().Add(cm.ttl)
		expires = &e
	}

	cm.cache[peerAddr] = &IdentityCache{
		Identity:  identity,
		PublicKey: identity.PublicKey,
		ExpiresAt: expires,
	}
}

// SetPublicKey stores just a public key in the cache with expiration
func (cm *IdentityCacheManager) SetPublicKey(peerAddr string, publicKey ed25519.PublicKey) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var expires *time.Time
	if cm.ttl > 0 {
		e := time.Now().Add(cm.ttl)
		expires = &e
	}

	cm.cache[peerAddr] = &IdentityCache{
		PublicKey: publicKey,
		ExpiresAt: expires,
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
		if cached.ExpiresAt != nil && now.After(*cached.ExpiresAt) {
			delete(cm.cache, addr)
		}
	}
}

// Verifier handles signature verification and identity management
type Verifier struct {
	identityCache *IdentityCacheManager
	hashChain     *HashChainManager
}

// NewVerifier creates a new Verifier instance
func NewVerifier(ttl time.Duration) *Verifier {
	return &Verifier{
		identityCache: NewIdentityCacheManager(ttl),
		hashChain:     NewHashChainManager(ttl),
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
		return fmt.Errorf("address mismatch: identity reports %q, signature claims %q",
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

		// SECURITY: Verify hash chain
		currentHash := hash[:]
		if err := v.hashChain.VerifyAndAdd(signature.SignerAddress, signature.PreviousHash, currentHash, signature.Sequence); err != nil {
			return fmt.Errorf("hash chain verification failed for %s: %w", signature.SignerAddress, err)
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
		return fmt.Errorf("address mismatch in ordered signature: identity reports %q, signature claims %q",
			peerIdentity.ServerAddress, signature.SignerAddress)
	}

	// Verify signature with fresh public key
	hash := sha256.Sum256(message)
	if !ed25519.Verify(peerIdentity.PublicKey, hash[:], signature.Signature) {
		return fmt.Errorf("invalid ordered signature for %s", signature.SignerAddress)
	}

	// Cache ONLY after successful verification
	v.identityCache.Set(signature.SignerAddress, peerIdentity)

	// SECURITY: Verify hash chain
	currentHash := hash[:]
	if err := v.hashChain.VerifyAndAdd(signature.SignerAddress, signature.PreviousHash, currentHash, signature.Sequence); err != nil {
		return fmt.Errorf("hash chain verification failed for %s: %w", signature.SignerAddress, err)
	}

	return nil
}

func (v *Verifier) VerifyOrderedSignatureProto(msg proto.Message, signature *identitypb.OrderedSignature) error {
	if v == nil {
		return nil
	}

	if signature == nil {
		return fmt.Errorf("message is not signed but verifier is configured")
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	return v.VerifyOrderedSignature(msgBytes, signature)
}

func (v *Verifier) VerifySignatureProto(msg proto.Message, signature *identitypb.Signature) error {
	if v == nil {
		return nil
	}

	if signature == nil {
		return fmt.Errorf("message is not signed but verifier is configured")
	}

	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	return v.VerifyMessageSignature(msgBytes, signature)
}

// getPeerIdentity fetches peer identity from their gRPC service
func (v *Verifier) getPeerIdentity(peerAddr string) (*identitypb.Identity, error) {
	// Connect to peer's identity service
	conn, err := grpcutil.NewClient(peerAddr)
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

// ClearHashChain clears all hash chain entries
func (v *Verifier) ClearHashChain() {
	v.hashChain.Clear()
}

// CleanupHashChain removes expired hash chain entries
func (v *Verifier) CleanupHashChain() {
	v.hashChain.Cleanup()
}
