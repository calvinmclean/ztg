package config

#Server: {
	address:     string | *":8080"
	server_name: string | *"ztg-server"
	owner_name:  string | *"ztg-user"
	version:     string | *"1.0.0"
	signed:      bool | *false
}

#Key: {
	// Exactly one of owner_public_key or owner_public_key_file must be set
	// owner_public_key:      string | *""
	// owner_public_key_file: string | *""
	{owner_public_key: string | *""} | {owner_public_key_file: string | *""}

	// Exactly one of private_key or private_key_path must be set
	{private_key: string | *""} | {private_key_path: string | *""}

	server_address: string | *server.address
	force_example:  bool | *false
}

server: #Server
key:    #Key
