package config

type ServerConfig struct {
	Port     int    `json:"port" envconfig:"ZTG_PORT"`
	LogLevel string `json:"log_level" envconfig:"ZTG_LOG_LEVEL"`
}

type IdentityConfig struct {
	ServerName         string `json:"server_name" envconfig:"ZTG_SERVER_NAME"`
	OwnerName          string `json:"owner_name" envconfig:"ZTG_OWNER_NAME"`
	OwnerPublicKey     string `json:"owner_public_key" envconfig:"ZTG_OWNER_PUBLIC_KEY"`
	OwnerPublicKeyFile string `json:"owner_public_key_file" envconfig:"ZTG_OWNER_PUBLIC_KEY_FILE"`
	PrivateKey         string `json:"private_key" envconfig:"ZTG_PRIVATE_KEY"`
	PrivateKeyPath     string `json:"private_key_path" envconfig:"ZTG_KEY_PATH"`
	ServerAddress      string `json:"server_address" envconfig:"ZTG_IDENTITY_SERVER_ADDRESS"`
	ForceExample       bool   `json:"force_example" envconfig:"ZTG_FORCE_EXAMPLE"`
}

type Config struct {
	Server   ServerConfig   `json:"server"`
	Identity IdentityConfig `json:"identity"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     50052,
			LogLevel: "info",
		},
		Identity: IdentityConfig{
			ServerName:         "ztg-server",
			OwnerName:          "ztg-user",
			OwnerPublicKey:     "",                         // Set this to the base64 or hex encoded owner Ed25519 public key
			OwnerPublicKeyFile: "",                         // Alternatively, set this to a file path containing the owner public key
			PrivateKey:         "",                         // Set this to the PEM encoded private key string
			PrivateKeyPath:     "keys/example_ed25519.pem", // Alternatively, set this to a file path containing the private key
			ServerAddress:      "",
			ForceExample:       false,
		},
	}
}
