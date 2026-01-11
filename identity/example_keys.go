package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

var examplePublicKey ed25519.PublicKey

func init() {
	// Try to load example key during initialization
	loadExamplePublicKeyCache()
}

func loadExamplePublicKeyCache() {
	if key, err := loadExamplePublicKey(); err == nil {
		examplePublicKey = key
	}
}

func loadExamplePublicKey() (ed25519.PublicKey, error) {
	data, err := os.ReadFile("keys/example_ed25519.pub.pem")
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	var pubBytes any
	if block.Type == "PUBLIC KEY" || block.Type == "ED25519 PUBLIC KEY" {
		pubBytes, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unsupported public key type: %s", block.Type)
	}

	return pubBytes.(ed25519.PublicKey), nil
}

func isExampleKey(publicKey ed25519.PublicKey) bool {
	// Ensure example key is loaded (might have failed during init)
	if examplePublicKey == nil {
		loadExamplePublicKeyCache()
	}
	return bytes.Equal(publicKey, examplePublicKey)
}
