

-- name: CreateAccount :one
INSERT INTO accounts (
    owner,
    balance,
    currency, 
    user_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, owner, balance, currency, user_id, created_at;

-- name: GetAccountById :one
SELECT id, owner, balance, currency, user_id, created_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByIdForUpdate :one
SELECT id, owner, balance, currency, user_id, created_at
FROM accounts
WHERE id = $1
FOR UPDATE;

-- name: IncrementAccountBalance :one
UPDATE accounts
SET balance =  balance + $2, updated_at = now()
WHERE id = $1
RETURNING id, owner, balance, currency, user_id, created_at;

-- name: CheckIsAccountWithIdExist :one
SELECT EXISTS (
    SELECT 1
    FROM accounts
    WHERE id = $1
);

-- name: ListAccountsByUserId :many
SELECT id, owner, balance, currency, user_id, created_at
FROM accounts
WHERE user_id = sqlc.arg(user_id)
ORDER BY id
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountAccountsByUserId :one
SELECT COUNT(*) FROM accounts
WHERE user_id = $1;