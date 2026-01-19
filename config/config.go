package config

type ServerConfig struct {
	Address    string `json:"address" envconfig:"ZTG_ADDRESS"`
	ServerName string `json:"server_name" envconfig:"ZTG_SERVER_NAME"`
	OwnerName  string `json:"owner_name" envconfig:"ZTG_OWNER_NAME"`
	Version    string `json:"version" envconfig:"ZTG_VERSION"`
	Signed     bool   `json:"signed" envconfig:"ZTG_SIGNED"`
}

type KeyConfig struct {
	PrivateKeyPath string `json:"private_key_path" envconfig:"ZTG_KEY_PATH"`
	ServerAddress  string `json:"server_address" envconfig:"ZTG_KEY_SERVER_ADDRESS"`
	ForceExample   bool   `json:"force_example" envconfig:"ZTG_FORCE_EXAMPLE"`
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
			PrivateKeyPath: "keys/server_ed25519.pem",
			ServerAddress:  ":50052",
			ForceExample:   false,
		},
	}
}
