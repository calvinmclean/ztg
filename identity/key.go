package identity

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"ztg/config"
)

const (
	exampleKeyPath    = "keys/example_ed25519.pem"
	examplePubKeyPath = "keys/example_ed25519.pub.pem"
)

type KeyManager struct {
	ownerPublicKey ed25519.PublicKey
	privateKey     ed25519.PrivateKey
	publicKey      ed25519.PublicKey
	isExample      bool
}

func NewKeyManager(cfg config.KeyConfig) (*KeyManager, error) {
	if cfg.OwnerPublicKey == "" {
		fmt.Println("WARNING: Using example owner key - suitable for testing only")
		ownerKey, err := os.ReadFile(examplePubKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error reading example owner key: %w", err)
		}

		cfg.OwnerPublicKey = string(ownerKey)
	}

	ownerKey, err := readPublicKey(cfg.OwnerPublicKey)
	if err != nil {
		return nil, fmt.Errorf("error reading owner's public key: %w", err)
	}

	if cfg.PrivateKeyPath == "" {
		fmt.Println("WARNING: Using example key - suitable for testing only")
		cfg.PrivateKeyPath = exampleKeyPath
	}
	privKey, pubKey, err := loadKeyFromFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, err
	}

	return &KeyManager{
		ownerPublicKey: ownerKey,
		privateKey:     privKey,
		publicKey:      pubKey,
		isExample:      isExampleKey(pubKey),
	}, nil
}

func readPublicKey(key string) (ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an Ed25519 public key")
	}

	return edPub, nil
}

func loadKeyFromFile(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block")
	}

	var privateKey ed25519.PrivateKey
	if block.Type == "PRIVATE KEY" || block.Type == "ED25519 PRIVATE KEY" {
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse Ed25519 private key: %w", err)
		}

		var ok bool
		privateKey, ok = keyInterface.(ed25519.PrivateKey)
		if !ok {
			return nil, nil, fmt.Errorf("key is not Ed25519 private key")
		}
	} else {
		return nil, nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	publicKey := privateKey.Public().(ed25519.PublicKey)
	return privateKey, publicKey, nil
}

func (km *KeyManager) PrivateKey() ed25519.PrivateKey {
	return km.privateKey
}

func (km *KeyManager) PublicKey() ed25519.PublicKey {
	return km.publicKey
}

func (km *KeyManager) OwnerPublicKey() ed25519.PublicKey {
	return km.ownerPublicKey
}

func (km *KeyManager) IsExample() bool {
	return km.isExample
}

func ValidateKeyUsage(publicKey ed25519.PublicKey, env string) error {
	if isExampleKey(publicKey) && env == "production" {
		return fmt.Errorf("EXAMPLE KEY DETECTED: Cannot use example key in production")
	}

	if isExampleKey(publicKey) {
		fmt.Println("WARNING: Using example key - suitable for testing only")
	}

	return nil
}
