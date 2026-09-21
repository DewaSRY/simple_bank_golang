-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    device_fingerprint_hash
) VALUES (
    $1, $2, $3
)
RETURNING id, username, email, created_at;

-- name: GetUserByEmail :one
SELECT id, username, email, device_fingerprint_hash, created_at
FROM users
WHERE email = $1;

-- name: GetUserById :one
SELECT id, username, email, created_at
FROM users
WHERE id = $1;

-- name: CheckIsUsernameExist :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE username = $1
);

-- name: BindUserDeviceFingerprint :one
-- Binds an unbound (pre-migration, device_fingerprint_hash = '') user row to
-- the fingerprint presented on its first login after the migration. Never
-- overwrites an already-bound row.
UPDATE users
SET device_fingerprint_hash = $2, updated_at = now()
WHERE id = $1 AND device_fingerprint_hash = ''
RETURNING id, username, email, created_at;
