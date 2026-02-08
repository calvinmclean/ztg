package config

import (
	_ "embed"
	"fmt"

	"cuelang.org/go/cue/cuecontext"
)

//go:embed config.cue
var cueSchema string

type ServerConfig struct {
	Port     int    `json:"port,omitzero" envconfig:"ZTG_PORT"`
	LogLevel string `json:"log_level,omitzero" envconfig:"ZTG_LOG_LEVEL"`
}

type IdentityConfig struct {
	ServerName         string `json:"server_name,omitzero" envconfig:"ZTG_SERVER_NAME"`
	OwnerName          string `json:"owner_name,omitzero" envconfig:"ZTG_OWNER_NAME"`
	OwnerPublicKey     string `json:"owner_public_key,omitzero" envconfig:"ZTG_OWNER_PUBLIC_KEY"`
	OwnerPublicKeyFile string `json:"owner_public_key_file,omitzero" envconfig:"ZTG_OWNER_PUBLIC_KEY_FILE"`
	PrivateKey         string `json:"private_key,omitzero" envconfig:"ZTG_PRIVATE_KEY"`
	PrivateKeyFile     string `json:"private_key_file,omitzero" envconfig:"ZTG_PRIVATE_KEY_FILE"`
	ServerAddress      string `json:"server_address,omitzero" envconfig:"ZTG_IDENTITY_SERVER_ADDRESS"`
	ForceExample       bool   `json:"force_example,omitzero" envconfig:"ZTG_FORCE_EXAMPLE"`
}

type DatabaseConfig struct {
	URL                string `json:"url,omitzero" envconfig:"ZTG_DATABASE_URL"`
	AuthToken          string `json:"auth_token,omitzero" envconfig:"ZTG_DATABASE_AUTH_TOKEN"`
	Path               string `json:"path,omitzero" envconfig:"ZTG_DATABASE_PATH"`
	PollTimeoutMs      int    `json:"long_poll_timeout_ms,omitzero" envconfig:"ZTG_DATABASE_LONG_POLL_TIMEOUT_MS"`
	IfEmpty            bool   `json:"bootstrap_if_empty,omitzero" envconfig:"ZTG_DATABASE_BOOTSTRAP_IF_EMPTY"`
	UseEmbeddedReplica bool   `json:"use_embedded_replica,omitzero" envconfig:"ZTG_DATABASE_USE_EMBEDDED_REPLICA"`
}

type Config struct {
	Server   ServerConfig   `json:"server,omitzero"`
	Identity IdentityConfig `json:"identity,omitzero"`
	Database DatabaseConfig `json:"database,omitzero"`
}

func (c *Config) Validate() error {
	ctx := cuecontext.New()

	// Compile the CUE schema
	schema := ctx.CompileString(cueSchema)
	if schema.Err() != nil {
		return fmt.Errorf("failed to compile CUE schema: %w", schema.Err())
	}

	// Encode the config as CUE value
	configValue := ctx.Encode(c)
	if configValue.Err() != nil {
		return fmt.Errorf("failed to encode config: %w", configValue.Err())
	}

	// Unify config with schema to apply constraints
	unified := schema.Unify(configValue)
	if unified.Err() != nil {
		return fmt.Errorf("config validation failed: %w", unified.Err())
	}

	// Validate the unified value
	if err := unified.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}
