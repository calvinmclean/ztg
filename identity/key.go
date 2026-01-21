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
	serverAddress  string
	serverName     string
	ownerName      string
	isExample      bool
}

func NewKeyManager(cfg config.IdentityConfig) (*KeyManager, error) {
	// Handle owner public key from string or file
	var ownerPublicKeyStr string
	switch {
	case cfg.OwnerPublicKey != "":
		ownerPublicKeyStr = cfg.OwnerPublicKey
	case cfg.OwnerPublicKeyFile != "":
		ownerKey, err := os.ReadFile(cfg.OwnerPublicKeyFile)
		if err != nil {
			return nil, fmt.Errorf("error reading owner public key file: %w", err)
		}
		ownerPublicKeyStr = string(ownerKey)
	default:
		fmt.Println("WARNING: Using example owner key - suitable for testing only")
		ownerKey, err := os.ReadFile(examplePubKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error reading example owner key: %w", err)
		}
		ownerPublicKeyStr = string(ownerKey)
	}

	ownerKey, err := readPublicKey(ownerPublicKeyStr)
	if err != nil {
		return nil, fmt.Errorf("error reading owner's public key: %w", err)
	}

	// Handle private key from string or file
	var privKey ed25519.PrivateKey
	var pubKey ed25519.PublicKey
	switch {
	case cfg.PrivateKey != "":
		privKey, pubKey, err = loadKeyFromString(cfg.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("error loading private key from string: %w", err)
		}
	case cfg.PrivateKeyPath != "":
		privKey, pubKey, err = loadKeyFromFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error loading private key from file: %w", err)
		}
	default:
		fmt.Println("WARNING: Using example key - suitable for testing only")
		privKey, pubKey, err = loadKeyFromFile(exampleKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error loading example private key: %w", err)
		}
	}

	return &KeyManager{
		ownerPublicKey: ownerKey,
		privateKey:     privKey,
		publicKey:      pubKey,
		serverAddress:  cfg.ServerAddress,
		serverName:     cfg.ServerName,
		ownerName:      cfg.OwnerName,
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

func loadKeyFromString(keyString string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	block, _ := pem.Decode([]byte(keyString))
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode PEM block from string")
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

func loadKeyFromFile(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %w", err)
	}

	return loadKeyFromString(string(data))
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

func (km *KeyManager) ServerAddress() string {
	return km.serverAddress
}

func (km *KeyManager) ServerName() string {
	return km.serverName
}

func (km *KeyManager) OwnerName() string {
	return km.ownerName
}

func (km *KeyManager) IsExample() bool {
	return km.isExample
}

func ValidateKeyUsage(publicKey ed25519.PublicKey) error {
	if isExampleKey(publicKey) {
		fmt.Println("WARNING: Using example key - suitable for testing only")
	}

	return nil
}
