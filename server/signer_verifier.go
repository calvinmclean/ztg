package server

import (
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"
)

// CreateSignerVerifierPair creates a signer and verifier pair if signed mode is enabled
func CreateSignerVerifierPair(keyManager *identity.KeyManager, serverAddr string, sqlStore store.Store) (*Signer, *Verifier) {
	return NewSigner(keyManager, serverAddr), NewVerifier(DefaultVerifierTTL, sqlStore)
}
