package server

import (
	"ztg/identity"
)

// CreateSignerVerifierPair creates a signer and verifier pair if signed mode is enabled
func CreateSignerVerifierPair(keyManager *identity.KeyManager, serverAddr string) (*Signer, *Verifier) {
	return NewSigner(keyManager, serverAddr), NewVerifier(DefaultVerifierTTL)
}
