package config

#Server: {
	port:      int | *50052
	log_level: string | *"info"
}

#Identity: {
	server_name!: string | *"ztg-server"
	owner_name!:  string | *"ztg-user"

	// Exactly one of owner_public_key or owner_public_key_file must be set
	{owner_public_key: string} | {owner_public_key_file: string}

	// Exactly one of private_key or private_key_path must be set
	{private_key!: string} | {private_key_path: string}

	server_address!: string
	force_example!:  bool | *false
}

server!:   #Server
identity!: #Identity
