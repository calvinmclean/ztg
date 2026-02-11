-- name: InsertIdentity :one
INSERT INTO ztg.identities (
    public_key, server_address, server_name, owner_name, capabilities, is_trusted
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetIdentityByKey :one
SELECT * FROM ztg.identities WHERE public_key = $1 LIMIT 1;

-- name: GetIdentityByAddress :one
SELECT * FROM ztg.identities WHERE server_address = $1 LIMIT 1;

-- name: UpdateLastSeen :exec
UPDATE ztg.identities SET last_seen = NOW() WHERE server_address = $1;

-- name: ListIdentities :many
SELECT * FROM ztg.identities
ORDER BY created_at DESC;

-- name: SetTrustStatus :exec
UPDATE ztg.identities
SET is_trusted = $1, trust_updated_at = NOW()
WHERE server_address = $2;

-- name: DeleteIdentity :exec
DELETE FROM ztg.identities WHERE public_key = $1;

-- name: CountIdentities :one
SELECT COUNT(*) as count FROM ztg.identities
WHERE (EXCLUDED.arg_trusted_only = false OR is_trusted = $1);
