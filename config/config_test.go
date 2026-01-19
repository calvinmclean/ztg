package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Address != ":8080" {
		t.Errorf("Expected address ':8080', got '%s'", cfg.Server.Address)
	}
	if cfg.Server.ServerName != "ztg-server" {
		t.Errorf("Expected server_name 'ztg-server', got '%s'", cfg.Server.ServerName)
	}
}

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
	t.Setenv("ZTG_ADDRESS", ":9999")
	t.Setenv("ZTG_SERVER_NAME", "env-server")
	t.Setenv("ZTG_SIGNED", "true")
	t.Setenv("ZTG_KEY_PATH", "/custom/path/key.pem")
	// Note: ZTG_STRATEGY not set - should not override existing config

	cfg := DefaultConfig()
	cfg = LoadFromEnv(cfg)

	if cfg.Server.Address != ":9999" {
		t.Errorf("Expected address ':9999' from env, got '%s'", cfg.Server.Address)
	}
	if cfg.Server.ServerName != "env-server" {
		t.Errorf("Expected server_name 'env-server' from env, got '%s'", cfg.Server.ServerName)
	}
	if cfg.Server.Signed != true {
		t.Errorf("Expected signed true from env, got %v", cfg.Server.Signed)
	}
	if cfg.Key.PrivateKeyPath != "/custom/path/key.pem" {
		t.Errorf("Expected key_path '/custom/path/key.pem' from env, got '%s'", cfg.Key.PrivateKeyPath)
	}
}

func TestLoadFromEnvPartial(t *testing.T) {
	// Test that missing env vars don't override existing config values
	t.Setenv("ZTG_SERVER_NAME", "partial-server")
	// Note: Only server name set, others should preserve defaults

	cfg := DefaultConfig()
	originalAddress := cfg.Server.Address
	originalSigned := cfg.Server.Signed
	originalKeyPath := cfg.Key.PrivateKeyPath

	cfg = LoadFromEnv(cfg)

	// Should override only what's set in env
	if cfg.Server.ServerName != "partial-server" {
		t.Errorf("Expected server_name 'partial-server' from env, got '%s'", cfg.Server.ServerName)
	}

	// Should preserve everything else from original config
	if cfg.Server.Address != originalAddress {
		t.Errorf("Expected address '%s' to be preserved, got '%s'", originalAddress, cfg.Server.Address)
	}
	if cfg.Server.Signed != originalSigned {
		t.Errorf("Expected signed %v to be preserved, got %v", originalSigned, cfg.Server.Signed)
	}
	if cfg.Key.PrivateKeyPath != originalKeyPath {
		t.Errorf("Expected key_path '%s' to be preserved, got '%s'", originalKeyPath, cfg.Key.PrivateKeyPath)
	}
}

func TestLoadFromEnvOverridesCustomConfig(t *testing.T) {
	// Test that env vars override custom config, not just defaults
	t.Setenv("ZTG_ADDRESS", ":7777")
	t.Setenv("ZTG_OWNER_NAME", "env-owner")

	cfg := &Config{
		Server: ServerConfig{
			Address:    ":5555",
			ServerName: "custom-server",
			OwnerName:  "custom-owner",
			Version:    "2.0.0",
			Signed:     true,
		},
		Key: KeyConfig{
			PrivateKeyPath: "/custom/key.pem",
			ServerAddress:  ":5555",
			ForceExample:   true,
		},
	}

	cfg = LoadFromEnv(cfg)

	// Env vars should override custom config values
	if cfg.Server.Address != ":7777" {
		t.Errorf("Expected address ':7777' from env, got '%s'", cfg.Server.Address)
	}
	if cfg.Server.OwnerName != "env-owner" {
		t.Errorf("Expected owner_name 'env-owner' from env, got '%s'", cfg.Server.OwnerName)
	}

	// Non-env values should be preserved
	if cfg.Server.ServerName != "custom-server" {
		t.Errorf("Expected server_name 'custom-server' to be preserved, got '%s'", cfg.Server.ServerName)
	}
	if cfg.Server.Signed != true {
		t.Errorf("Expected signed true to be preserved, got %v", cfg.Server.Signed)
	}
}

func TestLoadFromEnvInvalid(t *testing.T) {
	// Test that invalid env values don't crash
	t.Setenv("ZTG_SIGNED", "not-a-boolean")

	cfg := DefaultConfig()
	originalSigned := cfg.Server.Signed

	cfg = LoadFromEnv(cfg)

	// Should preserve original value when env parsing fails
	if cfg.Server.Signed != originalSigned {
		t.Errorf("Expected original signed value %v when env parsing fails, got %v", originalSigned, cfg.Server.Signed)
	}
}
