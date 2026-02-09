package server

import (
	"context"
	"fmt"
	"testing"

	"github.com/calvinmclean/ztg/config"
	identitypb "github.com/calvinmclean/ztg/gen/go/proto/identity/v1"
	"github.com/calvinmclean/ztg/identity"
	"github.com/calvinmclean/ztg/identity/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// setupTestStore creates an in-memory SQLite store for testing
func setupTestStore(t *testing.T) store.Store {
	dbConfig := config.DatabaseConfig{
		Path: ":memory:",
	}

	sqlStore, err := store.NewSQLStore(dbConfig)
	if err != nil {
		t.Fatalf("Failed to create SQL store: %v", err)
	}

	return sqlStore
}

// createNonOwnerSignature creates a signature using a non-owner private key
func createNonOwnerSignature(t *testing.T) *identitypb.Signature {
	// Create key manager with non-owner key
	nonOwnerKm, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "non-owner-server",
		OwnerName:          "non-owner",
		PrivateKeyFile:     "../keys/other_example.pem",
		ServerAddress:      "localhost:8081",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem", // Still use original owner public key for verification
	})
	if err != nil {
		t.Fatalf("Failed to create non-owner key manager: %v", err)
	}

	// Create signature for set-trust operation
	message := fmt.Appendf(nil, "set-trust:%s:%v", "localhost:8081", true)
	signer := NewSigner(nonOwnerKm, "non-owner")
	signature, err := signer.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message with non-owner key: %v", err)
	}

	return signature
}

// Test identity service methods
func TestIdentityService_AddIdentity(t *testing.T) {
	// Create test key manager
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create service with real store
	testStore := setupTestStore(t)
	defer testStore.Close()
	service := &identityService{
		keyManager:   km,
		serverConfig: config.ServerConfig{},
		registry:     &registry{},
		store:        testStore,
	}

	tests := []struct {
		name    string
		request *identitypb.AddIdentityRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "valid identity",
			request: &identitypb.AddIdentityRequest{
				Identity: &identitypb.Identity{
					PublicKey:     make([]byte, 32),
					ServerAddress: "localhost:8081",
					ServerName:    "Test Server",
					OwnerName:     "Test Owner",
					Capabilities:  []string{"game1"},
					IsTrusted:     false,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid public key size",
			request: &identitypb.AddIdentityRequest{
				Identity: &identitypb.Identity{
					PublicKey:     []byte("too-short"),
					ServerAddress: "localhost:8081",
					ServerName:    "Test Server",
					OwnerName:     "Test Owner",
					Capabilities:  []string{"game1"},
					IsTrusted:     false,
				},
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
		{
			name: "cannot create trusted identity",
			request: &identitypb.AddIdentityRequest{
				Identity: &identitypb.Identity{
					PublicKey:     make([]byte, 32),
					ServerAddress: "localhost:8081",
					ServerName:    "Test Server",
					OwnerName:     "Test Owner",
					Capabilities:  []string{"game1"},
					IsTrusted:     true,
				},
			},
			wantErr: true,
			errCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.AddIdentity(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && status.Code(err) != tt.errCode {
				t.Errorf("AddIdentity() error code = %v, want %v", status.Code(err), tt.errCode)
			}
		})
	}
}

func TestIdentityService_ListIdentities(t *testing.T) {
	// Create test key manager
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create service with real store
	testStore := setupTestStore(t)
	defer testStore.Close()
	service := &identityService{
		keyManager:   km,
		serverConfig: config.ServerConfig{},
		registry:     &registry{},
		store:        testStore,
	}

	// Add some test identities with different public keys
	identity1 := &identitypb.Identity{
		PublicKey:     []byte("12345678901234567890123456789012"), // 32 bytes
		ServerAddress: "localhost:8081",
		ServerName:    "Test Server 1",
		OwnerName:     "Test Owner 1",
		Capabilities:  []string{"game1"},
		IsTrusted:     false,
		CreatedAt:     timestamppb.Now(),
	}
	identity2 := &identitypb.Identity{
		PublicKey:     []byte("98765432109876543210987654321098"), // 32 bytes
		ServerAddress: "localhost:8082",
		ServerName:    "Test Server 2",
		OwnerName:     "Test Owner 2",
		Capabilities:  []string{"game2"},
		IsTrusted:     true,
		CreatedAt:     timestamppb.Now(),
	}

	testStore.InsertIdentity(context.Background(), identity1)
	testStore.InsertIdentity(context.Background(), identity2)

	// Test successful listing
	resp, err := service.ListIdentities(context.Background(), &identitypb.ListIdentitiesRequest{})
	if err != nil {
		t.Errorf("ListIdentities() error = %v, wantErr false", err)
		return
	}
	if len(resp.Identities) != 2 {
		t.Errorf("ListIdentities() returned %d identities, want 2", len(resp.Identities))
	}
}

func TestIdentityService_SetTrust(t *testing.T) {
	// Create test key manager
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create service with real store
	testStore := setupTestStore(t)
	defer testStore.Close()
	service := &identityService{
		keyManager:   km,
		serverConfig: config.ServerConfig{},
		registry:     &registry{},
		store:        testStore,
	}

	// Add a test identity
	testIdentity := &identitypb.Identity{
		PublicKey:     make([]byte, 32),
		ServerAddress: "localhost:8081",
		ServerName:    "Test Server",
		OwnerName:     "Test Owner",
		Capabilities:  []string{"game1"},
		IsTrusted:     false,
		CreatedAt:     timestamppb.Now(),
	}
	testStore.InsertIdentity(context.Background(), testIdentity)

	// Create a valid signature for set-trust operation
	// Use "owner" as signer address to match what verifyOwnerSignatureBytes expects
	validMessage := fmt.Appendf(nil, "set-trust:%s:%v", "localhost:8081", true)
	validSigner := NewSigner(km, "owner")
	validSignature, err := validSigner.SignMessage(validMessage)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	tests := []struct {
		name    string
		request *identitypb.SetTrustRequest
		wantErr bool
		errCode codes.Code
	}{
		{
			name: "valid set trust to true",
			request: &identitypb.SetTrustRequest{
				ServerAddress: "localhost:8081",
				Trusted:       true,
				Signature:     validSignature,
			},
			wantErr: false,
		},
		{
			name: "invalid signature",
			request: &identitypb.SetTrustRequest{
				ServerAddress: "localhost:8081",
				Trusted:       true,
				Signature: &identitypb.Signature{
					Signature:     []byte("invalid-signature"),
					SignerAddress: "owner",
				},
			},
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
		{
			name: "nil signature",
			request: &identitypb.SetTrustRequest{
				ServerAddress: "localhost:8081",
				Trusted:       true,
				Signature:     nil,
			},
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
		{
			name: "non-owner signature",
			request: &identitypb.SetTrustRequest{
				ServerAddress: "localhost:8081",
				Trusted:       true,
				Signature:     createNonOwnerSignature(t),
			},
			wantErr: true,
			errCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.SetTrust(context.Background(), tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetTrust() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && status.Code(err) != tt.errCode {
				t.Errorf("SetTrust() error code = %v, want %v", status.Code(err), tt.errCode)
			}
		})
	}
}

func TestIdentityService_StoreUnavailable(t *testing.T) {
	// Create test key manager
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create service without store
	serviceNoStore := &identityService{
		keyManager:   km,
		serverConfig: config.ServerConfig{},
		registry:     &registry{},
		store:        nil,
	}

	// Create a valid signature for set-trust operation
	validMessage := fmt.Appendf(nil, "set-trust:%s:%v", "localhost:8081", true)
	validSigner := NewSigner(km, "owner")
	validSignature, err := validSigner.SignMessage(validMessage)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	t.Run("AddIdentity without store", func(t *testing.T) {
		req := &identitypb.AddIdentityRequest{
			Identity: &identitypb.Identity{
				PublicKey:     make([]byte, 32),
				ServerAddress: "localhost:8081",
				ServerName:    "Test Server",
				OwnerName:     "Test Owner",
				Capabilities:  []string{"game1"},
				IsTrusted:     false,
			},
		}
		_, err := serviceNoStore.AddIdentity(context.Background(), req)
		if err == nil {
			t.Error("AddIdentity should fail when store is unavailable")
		}
		if status.Code(err) != codes.Unavailable {
			t.Errorf("Expected error code %v, got %v", codes.Unavailable, status.Code(err))
		}
	})

	t.Run("ListIdentities without store", func(t *testing.T) {
		_, err := serviceNoStore.ListIdentities(context.Background(), &identitypb.ListIdentitiesRequest{})
		if err == nil {
			t.Error("ListIdentities should fail when store is unavailable")
		}
		if status.Code(err) != codes.Unavailable {
			t.Errorf("Expected error code %v, got %v", codes.Unavailable, status.Code(err))
		}
	})

	t.Run("SetTrust without store", func(t *testing.T) {
		req := &identitypb.SetTrustRequest{
			ServerAddress: "localhost:8081",
			Trusted:       true,
			Signature:     validSignature,
		}
		_, err := serviceNoStore.SetTrust(context.Background(), req)
		if err == nil {
			t.Error("SetTrust should fail when store is unavailable")
		}
		if status.Code(err) != codes.Unavailable {
			t.Errorf("Expected error code %v, got %v", codes.Unavailable, status.Code(err))
		}
	})
}

func TestIdentityService_Integration(t *testing.T) {
	// Create test key manager
	km, err := identity.NewKeyManager(config.IdentityConfig{
		ServerName:         "test-server",
		OwnerName:          "test-owner",
		PrivateKeyFile:     "../keys/example_ed25519.pem",
		ServerAddress:      "localhost:8080",
		OwnerPublicKeyFile: "../keys/example_ed25519.pub.pem",
	})
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	// Create service with real store
	testStore := setupTestStore(t)
	defer testStore.Close()
	service := &identityService{
		keyManager:   km,
		serverConfig: config.ServerConfig{},
		registry:     &registry{},
		store:        testStore,
	}

	// Test adding identity
	addReq := &identitypb.AddIdentityRequest{
		Identity: &identitypb.Identity{
			PublicKey:     make([]byte, 32),
			ServerAddress: "localhost:8081",
			ServerName:    "Test Server",
			OwnerName:     "Test Owner",
			Capabilities:  []string{"game1"},
			IsTrusted:     false,
		},
	}

	_, err = service.AddIdentity(context.Background(), addReq)
	if err != nil {
		t.Fatalf("Failed to add identity: %v", err)
	}

	// Test listing identities
	listResp, err := service.ListIdentities(context.Background(), &identitypb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("Failed to list identities: %v", err)
	}
	if len(listResp.Identities) != 1 {
		t.Fatalf("Expected 1 identity, got %d", len(listResp.Identities))
	}
	if listResp.Identities[0].IsTrusted {
		t.Error("Identity should not be trusted initially")
	}

	// Test setting trust
	message := fmt.Appendf(nil, "set-trust:%s:%v", "localhost:8081", true)
	signer := NewSigner(km, "owner")
	signature, err := signer.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	setTrustReq := &identitypb.SetTrustRequest{
		ServerAddress: "localhost:8081",
		Trusted:       true,
		Signature:     signature,
	}

	_, err = service.SetTrust(context.Background(), setTrustReq)
	if err != nil {
		t.Fatalf("Failed to set trust: %v", err)
	}

	// Verify trust was set
	listResp, err = service.ListIdentities(context.Background(), &identitypb.ListIdentitiesRequest{})
	if err != nil {
		t.Fatalf("Failed to list identities after trust update: %v", err)
	}
	if len(listResp.Identities) != 1 {
		t.Fatalf("Expected 1 identity after trust update, got %d", len(listResp.Identities))
	}
	if !listResp.Identities[0].IsTrusted {
		t.Error("Identity should be trusted after SetTrust call")
	}
}
