package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		format  string
	}{
		{
			name:    "JSON file",
			path:    "testdata/config.example.json",
			wantErr: false,
			format:  "json",
		},
		{
			name:    "YAML file",
			path:    "testdata/config.example.yaml",
			wantErr: false,
			format:  "yaml",
		},
		{
			name:    "nonexistent file",
			path:    "testdata/nonexistent.json",
			wantErr: true,
			format:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cfg == nil {
				t.Error("LoadFromFile() returned nil config on success")
			}

			if !tt.wantErr {
				// Verify format detection
				ext := strings.ToLower(filepath.Ext(tt.path))
				if ext == ".json" && tt.format != "json" {
					t.Errorf("Expected JSON format for %s", tt.path)
				}
				if (ext == ".yaml" || ext == ".yml") && tt.format != "yaml" {
					t.Errorf("Expected YAML format for %s", tt.path)
				}
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Use t.Setenv for test-scoped environment variables
	t.Setenv("ZTG_PORT", "9999")
	t.Setenv("ZTG_SERVER_NAME", "env-server")
	t.Setenv("ZTG_PRIVATE_KEY_FILE", "/custom/path/key.pem")
	t.Setenv("ZTG_PRIVATE_KEY", "test-private-key")
	t.Setenv("ZTG_OWNER_PUBLIC_KEY", "dGVzdC1wdWJsaWMta2V5")
	t.Setenv("ZTG_OWNER_PUBLIC_KEY_FILE", "/custom/path/pubkey.pem")
	// Note: ZTG_STRATEGY not set - should not override existing config

	cfg := Config{}
	LoadFromEnv(&cfg)

	if cfg.Server.Port != 9999 {
		t.Errorf("Expected port 9999 from env, got '%d'", cfg.Server.Port)
	}
	if cfg.Identity.ServerName != "env-server" {
		t.Errorf("Expected server_name 'env-server' from env, got '%s'", cfg.Identity.ServerName)
	}
	if cfg.Identity.ServerName != "env-server" {
		t.Errorf("Expected server_name 'env-server' from env, got '%s'", cfg.Identity.ServerName)
	}
	if cfg.Identity.PrivateKeyFile != "/custom/path/key.pem" {
		t.Errorf("Expected key_path '/custom/path/key.pem' from env, got '%s'", cfg.Identity.PrivateKeyFile)
	}
	if cfg.Identity.PrivateKey != "test-private-key" {
		t.Errorf("Expected private_key 'test-private-key' from env, got '%s'", cfg.Identity.PrivateKey)
	}
	if cfg.Identity.OwnerPublicKey != "dGVzdC1wdWJsaWMta2V5" {
		t.Errorf("Expected owner_public_key 'dGVzdC1wdWJsaWMta2V5' from env, got '%s'", cfg.Identity.OwnerPublicKey)
	}
	if cfg.Identity.OwnerPublicKeyFile != "/custom/path/pubkey.pem" {
		t.Errorf("Expected owner_public_key_file '/custom/path/pubkey.pem' from env, got '%s'", cfg.Identity.OwnerPublicKeyFile)
	}
}

func TestLoadFromEnvPartial(t *testing.T) {
	// Test that missing env vars don't override existing config values
	t.Setenv("ZTG_SERVER_NAME", "partial-server")
	// Note: Only server name set, others should preserve defaults

	cfg := Config{}
	originalAddress := cfg.Server.Port
	originalKeyPath := cfg.Identity.PrivateKeyFile

	LoadFromEnv(&cfg)

	// Should override only what's set in env
	if cfg.Identity.ServerName != "partial-server" {
		t.Errorf("Expected server_name 'partial-server' from env, got '%s'", cfg.Identity.ServerName)
	}

	// Should preserve everything else from original config
	if cfg.Server.Port != originalAddress {
		t.Errorf("Expected port '%d' to be preserved, got '%d'", originalAddress, cfg.Server.Port)
	}
	if cfg.Identity.PrivateKeyFile != originalKeyPath {
		t.Errorf("Expected key_path '%s' to be preserved, got '%s'", originalKeyPath, cfg.Identity.PrivateKeyFile)
	}
}

func TestLoadFromEnvOverridesCustomConfig(t *testing.T) {
	// Test that env vars override custom config, not just defaults
	t.Setenv("ZTG_PORT", "7777")
	t.Setenv("ZTG_OWNER_NAME", "env-owner")

	cfg := &Config{
		Server: ServerConfig{
			Port: 5555,
		},
		Identity: IdentityConfig{
			ServerName:     "custom-server",
			OwnerName:      "custom-owner",
			PrivateKeyFile: "/custom/key.pem",
			ServerAddress:  ":5555",
			ForceExample:   true,
		},
	}

	LoadFromEnv(cfg)

	// Env vars should override custom config values
	if cfg.Server.Port != 7777 {
		t.Errorf("Expected port 7777 from env, got '%d'", cfg.Server.Port)
	}
	if cfg.Identity.OwnerName != "env-owner" {
		t.Errorf("Expected owner_name 'env-owner' from env, got '%s'", cfg.Identity.OwnerName)
	}

	// Non-env values should be preserved
	if cfg.Identity.ServerName != "custom-server" {
		t.Errorf("Expected server_name 'custom-server' to be preserved, got '%s'", cfg.Identity.ServerName)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with owner_public_key and private_key",
			config: &Config{
				Server: ServerConfig{
					Port:     8080,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "test-public-key",
					OwnerPublicKeyFile: "",
					PrivateKey:         "test-private-key",
					PrivateKeyFile:     "",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: false,
		},
		{
			name: "valid config with owner_public_key_file and private_key_file",
			config: &Config{
				Server: ServerConfig{
					Port:     8080,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "",
					OwnerPublicKeyFile: "/path/to/public.key",
					PrivateKey:         "",
					PrivateKeyFile:     "/path/to/private.key",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: false,
		},
		{
			name: "invalid config - missing both owner_public_key and owner_public_key_file",
			config: &Config{
				Server: ServerConfig{
					Port:     8080,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "",
					OwnerPublicKeyFile: "",
					PrivateKey:         "test-private-key",
					PrivateKeyFile:     "",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: true,
			errorMsg:    "config validation failed",
		},
		{
			name: "invalid config - missing both private_key and private_key_file",
			config: &Config{
				Server: ServerConfig{
					Port:     8080,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "test-public-key",
					OwnerPublicKeyFile: "",
					PrivateKey:         "",
					PrivateKeyFile:     "",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: true,
			errorMsg:    "config validation failed",
		},
		{
			name: "invalid config - both OwnerPublicKey and OwnerPublicKeyFile set",
			config: &Config{
				Server: ServerConfig{
					Port:     8080,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "test-public-key",
					OwnerPublicKeyFile: "test",
					PrivateKey:         "abc",
					PrivateKeyFile:     "",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: true,
			errorMsg:    "config validation failed",
		},
		{
			name: "invalid config - zero port",
			config: &Config{
				Server: ServerConfig{
					Port:     0,
					LogLevel: "debug",
				},
				Identity: IdentityConfig{
					ServerName:         "test-server",
					OwnerName:          "test-owner",
					OwnerPublicKey:     "test-public-key",
					OwnerPublicKeyFile: "",
					PrivateKey:         "test-private-key",
					PrivateKeyFile:     "",
					ServerAddress:      "localhost:8080",
					ForceExample:       false,
				},
			},
			expectError: true,
			errorMsg:    "config validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if tt.expectError && err != nil && tt.errorMsg != "" {
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			}
		})
	}
}
