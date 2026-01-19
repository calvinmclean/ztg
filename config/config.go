package config

type ServerConfig struct {
	Address    string `json:"address" envconfig:"ZTG_ADDRESS"`
	ServerName string `json:"server_name" envconfig:"ZTG_SERVER_NAME"`
	OwnerName  string `json:"owner_name" envconfig:"ZTG_OWNER_NAME"`
	Version    string `json:"version" envconfig:"ZTG_VERSION"`
	Signed     bool   `json:"signed" envconfig:"ZTG_SIGNED"`
}

type KeyConfig struct {
	OwnerPublicKey     string `json:"owner_public_key" envconfig:"ZTG_OWNER_PUBLIC_KEY"`
	OwnerPublicKeyFile string `json:"owner_public_key_file" envconfig:"ZTG_OWNER_PUBLIC_KEY_FILE"`
	PrivateKey         string `json:"private_key" envconfig:"ZTG_PRIVATE_KEY"`
	PrivateKeyPath     string `json:"private_key_path" envconfig:"ZTG_KEY_PATH"`
	ServerAddress      string `json:"server_address" envconfig:"ZTG_KEY_SERVER_ADDRESS"`
	ForceExample       bool   `json:"force_example" envconfig:"ZTG_FORCE_EXAMPLE"`
}

type Config struct {
	Server ServerConfig `json:"server"`
	Key    KeyConfig    `json:"key"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:    ":50052",
			ServerName: "ztg-server",
			OwnerName:  "ztg-user",
			Version:    "1.0.0",
			Signed:     false,
		},
		Key: KeyConfig{
			OwnerPublicKey:     "",                        // Set this to the base64 or hex encoded owner Ed25519 public key
			OwnerPublicKeyFile: "",                        // Alternatively, set this to a file path containing the owner public key
			PrivateKey:         "",                        // Set this to the PEM encoded private key string
			PrivateKeyPath:     "keys/server_ed25519.pem", // Alternatively, set this to a file path containing the private key
			ServerAddress:      ":50052",
			ForceExample:       false,
		},
	}
}
