-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    hashed_password
) VALUES (
    $1, $2, $3
)
RETURNING id, username, email, created_at;

-- name: GetUserByEmail :one
SELECT id, username, email, hashed_password, created_at
FROM users
WHERE email = $1;

-- name: CheckIsUsernameExist :one
SELECT EXISTS (
    SELECT 1
    FROM users
    WHERE username = $1
);
