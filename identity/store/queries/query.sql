-- name: InsertIdentity :one
INSERT INTO identities (
    public_key, server_address, server_name, owner_name, capabilities, 
    created_at, last_seen, is_trusted, trust_updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetIdentityByKey :one
SELECT * FROM identities WHERE public_key = ? LIMIT 1;

-- name: GetIdentityByAddress :one
SELECT * FROM identities WHERE server_address = ? LIMIT 1;

-- name: UpdateLastSeen :exec
UPDATE identities SET last_seen = ? WHERE server_address = ?;

-- name: ListIdentities :many
SELECT * FROM identities 
WHERE (EXCLUDED.sqlite_arg('trusted_only', false) = false OR is_trusted = ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: SetTrustStatus :exec
UPDATE identities 
SET is_trusted = ?, trust_updated_at = ?
WHERE public_key = ?;

-- name: DeleteIdentity :exec
DELETE FROM identities WHERE public_key = ?;

-- name: CountIdentities :one
SELECT COUNT(*) as count FROM identities 
WHERE (EXCLUDED.sqlite_arg('trusted_only', false) = false OR is_trusted = ?);