package config

#Server: {
	port!:     int & >1023 | *50052
	log_level: string | *"info"
}

#Identity: {
	server_name!: string | *"ztg-server"
	owner_name!:  string | *"ztg-user"

	// Exactly one of owner_public_key or owner_public_key_file must be set
	{
		owner_public_key!:      string & !=""
		owner_public_key_file?: string & ==""
	} |
	{
		owner_public_key_file!: string & !=""
		owner_public_key?:      string & ==""
	}

	// Exactly one of private_key or private_key_file must be set
	{
		private_key!:      string & !=""
		private_key_file?: string & ==""
	} |
	{
		private_key_file!: string & !=""
		private_key?:      string & ==""
	}

	server_address!: string
	force_example!:  bool | *false
}

server!:   #Server
identity!: #Identity
