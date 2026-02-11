-- +migrate Up
-- Create ztg schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS ztg;

-- Create identities table
CREATE TABLE IF NOT EXISTS ztg.identities (
    id SERIAL PRIMARY KEY,
    public_key BYTEA UNIQUE NOT NULL,
    server_address TEXT NOT NULL,
    server_name TEXT UNIQUE NOT NULL,
    owner_name TEXT NOT NULL,
    capabilities TEXT NOT NULL, -- JSON array
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_trusted BOOLEAN NOT NULL DEFAULT FALSE,
    trust_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_identities_public_key ON ztg.identities(public_key);
CREATE INDEX IF NOT EXISTS idx_identities_server_address ON ztg.identities(server_address);
