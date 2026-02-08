-- identities table
CREATE TABLE identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    public_key BLOB UNIQUE NOT NULL,
    server_address TEXT NOT NULL,
    server_name TEXT NOT NULL,
    owner_name TEXT NOT NULL,
    capabilities TEXT, -- JSON array
    created_at INTEGER NOT NULL,
    last_seen INTEGER NOT NULL,
    is_trusted BOOLEAN NOT NULL DEFAULT FALSE,
    trust_updated_at INTEGER
);

-- Indexes for performance
CREATE INDEX idx_identities_public_key ON identities(public_key);
CREATE INDEX idx_identities_server_address ON identities(server_address);