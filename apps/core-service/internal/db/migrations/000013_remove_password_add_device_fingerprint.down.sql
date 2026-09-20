BEGIN;

ALTER TABLE users DROP COLUMN device_fingerprint_hash;
ALTER TABLE users ADD COLUMN hashed_password VARCHAR(255) NOT NULL DEFAULT '';

COMMIT;
