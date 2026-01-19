package config

#Server: {
	address:     string | *":8080"
	server_name: string | *"ztg-server"
	owner_name:  string | *"ztg-user"
	version:     string | *"1.0.0"
	signed:      bool | *false
}

#Key: {
	private_key_path: string | *"keys/server_ed25519.pem"
	server_address:   string | *server.address
	force_example:    bool | *false
}

server: #Server
key:    #Key
