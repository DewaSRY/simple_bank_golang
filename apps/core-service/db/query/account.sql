

-- name: CreateAccount :one
INSERT INTO accounts (
    owner,
    balance,
    currency
) VALUES (
    $1, $2, $3
)
RETURNING id, owner, balance, currency, created_at;

-- name: GetAccountById :one
SELECT id, owner, balance, currency, created_at
FROM accounts
WHERE id = $1;

-- name: GetAccountByIdForUpdate :one
SELECT id, owner, balance, currency, created_at
FROM accounts
WHERE id = $1
FOR UPDATE;

-- name: IncrementAccountBalance :one
UPDATE accounts
SET balance =  balance + $2, updated_at = now()
WHERE id = $1
RETURNING id, owner, balance, currency, created_at;

-- name: CheckIsAccountWithIdExist :one
SELECT EXISTS (
    SELECT 1
    FROM accounts
    WHERE id = $1
);

-- name: ListAccountsByOwner :many
SELECT id, owner, balance, currency, created_at
FROM accounts
WHERE owner = sqlc.arg(owner)
ORDER BY id
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountAccountsByOwner :one
SELECT COUNT(*) FROM accounts
WHERE owner = $1;