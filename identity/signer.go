package identity

import (
	"crypto/ed25519"
	"crypto/sha256"
)

type Signer struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	address    string
}

func NewSigner(privateKey ed25519.PrivateKey, address string) *Signer {
	return &Signer{
		privateKey: privateKey,
		publicKey:  privateKey.Public().(ed25519.PublicKey),
		address:    address,
	}
}

func (s *Signer) Sign(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)
	signature := ed25519.Sign(s.privateKey, hash[:])
	return signature, nil
}

func (s *Signer) Address() string {
	return s.address
}

func (s *Signer) PublicKey() ed25519.PublicKey {
	return s.publicKey
}
