-- +migrate Down
-- Drop identities table
DROP TABLE IF EXISTS ztg.identities;

-- Drop ztg schema if empty (optional)
DROP SCHEMA IF EXISTS ztg;
