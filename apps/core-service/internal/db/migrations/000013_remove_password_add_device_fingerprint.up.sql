BEGIN;

ALTER TABLE users DROP COLUMN hashed_password;
ALTER TABLE users ADD COLUMN device_fingerprint_hash VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN users.device_fingerprint_hash IS 'SHA-256 hex of the browser/device fingerprint bound to this account. Empty means unbound (pre-migration row) -- the next successful login binds it.';

COMMIT;
