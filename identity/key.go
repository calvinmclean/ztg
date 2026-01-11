package identity

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type KeyConfig struct {
	PrivateKeyPath string
	ServerAddress  string
	ForceExample   bool
}

type KeyManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	isExample  bool
}

func NewKeyManager(config KeyConfig) (*KeyManager, error) {
	if config.PrivateKeyPath != "" {
		return loadKeyFromFile(config.PrivateKeyPath)
	}

	if envPath := os.Getenv("ZTG_PRIVATE_KEY_PATH"); envPath != "" {
		return loadKeyFromFile(envPath)
	}

	if _, err := os.Stat("keys/server_ed25519.pem"); err == nil {
		return loadKeyFromFile("keys/server_ed25519.pem")
	}

	return loadExampleKey()
}

func loadKeyFromFile(path string) (*KeyManager, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	var privateKey ed25519.PrivateKey
	if block.Type == "PRIVATE KEY" || block.Type == "ED25519 PRIVATE KEY" {
		keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Ed25519 private key: %w", err)
		}

		var ok bool
		privateKey, ok = keyInterface.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not Ed25519 private key")
		}
	} else {
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}

	keyInterface, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Ed25519 private key: %w", err)
	}

	privateKey, ok := keyInterface.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not Ed25519 private key")
	}

	publicKey := privateKey.Public()

	return &KeyManager{
		privateKey: privateKey,
		publicKey:  publicKey.(ed25519.PublicKey),
		isExample:  isExampleKey(publicKey.(ed25519.PublicKey)),
	}, nil
}

func loadExampleKey() (*KeyManager, error) {
	fmt.Println("WARNING: Using example key - suitable for testing only")
	return loadKeyFromFile("keys/example_ed25519.pem")
}

func (km *KeyManager) PrivateKey() ed25519.PrivateKey {
	return km.privateKey
}

func (km *KeyManager) PublicKey() ed25519.PublicKey {
	return km.publicKey
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
