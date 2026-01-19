package identity

import (
	"crypto/ed25519"
	"fmt"
	"slices"
	"time"
)

type Identity struct {
	PublicKey    ed25519.PublicKey
	ServerAddr   string
	ServerName   string
	OwnerName    string
	Version      string
	Capabilities []string
	CreatedAt    time.Time
}

func NewIdentity(publicKey ed25519.PublicKey, serverAddr, serverName, ownerName, version string, capabilities []string) *Identity {
	return &Identity{
		PublicKey:    publicKey,
		ServerAddr:   serverAddr,
		ServerName:   serverName,
		OwnerName:    ownerName,
		Version:      version,
		Capabilities: capabilities,
		CreatedAt:    time.Now(),
	}
}

func (i *Identity) String() string {
	return fmt.Sprintf("Identity{Server: %s, Owner: %s, Address: %s}",
		i.ServerName, i.OwnerName, i.ServerAddr)
}

func (i *Identity) HasCapability(capability string) bool {
	return slices.Contains(i.Capabilities, capability)
}

func (i *Identity) IsExpired(maxAge time.Duration) bool {
	return time.Since(i.CreatedAt) > maxAge
}
