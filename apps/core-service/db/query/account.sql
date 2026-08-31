

-- name: CreateAccount :one
INSERT INTO accounts (
    number, 
    name,
    description,
    balance,
    currency, 
    user_id
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING id, balance, currency, user_id, number, name, description, created_at;

-- name: GetAccountById :one
SELECT id, balance, currency, user_id, number, name, description, created_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByIdForUpdate :one
SELECT id, balance, currency, user_id, number, name, description, created_at
FROM accounts
WHERE id = $1
FOR UPDATE;

-- name: IncrementAccountBalance :one
UPDATE accounts
SET balance =  balance + $2, updated_at = now()
WHERE id = $1
RETURNING id, balance, currency, user_id, number, name, description, created_at;

-- name: CheckIsAccountWithIdExist :one
SELECT EXISTS (
    SELECT 1
    FROM accounts
    WHERE id = $1
);

-- name: ListAccountsByUserId :many
SELECT id, balance, currency, user_id, number, name, description, created_at
FROM accounts
WHERE user_id = $1
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: CountAccountsByUserId :one
SELECT COUNT(*) FROM accounts
WHERE user_id = $1;