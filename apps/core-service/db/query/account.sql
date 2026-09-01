

-- name: CreateAccount :one
INSERT INTO accounts (
    number,
    name,
    description,
    balance,
    currency,
    user_id,
    is_main
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, balance, currency, user_id, number, name, description, is_main, created_at;

-- name: UpdateAccountNumber :one
UPDATE accounts
SET number = $2, updated_at = now()
WHERE id = $1
RETURNING id, balance, currency, user_id, number, name, description, is_main, created_at;

-- name: UpdateAccount :one
UPDATE accounts
SET name = $2, description = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, balance, currency, user_id, number, name, description, is_main, created_at;

-- name: SoftDeleteAccount :one
UPDATE accounts
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, balance, currency, user_id, number, name, description, is_main, created_at;

-- name: GetAccountById :one
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAccountByIdForUpdate :one
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: GetMainAccountByUserId :one
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE user_id = $1 AND is_main = true AND deleted_at IS NULL;

-- name: FindAccountByNumber :one
SELECT id, name, number
FROM accounts
WHERE number = $1 AND deleted_at IS NULL;

-- name: ListRecentTransferDestinations :many
SELECT id, name, number, last_used_at FROM (
    SELECT DISTINCT ON (a.id) a.id, a.name, a.number, t.created_at AS last_used_at
    FROM transfers t
    JOIN accounts a ON a.id = t.to_account_id
    WHERE t.from_account_id = sqlc.arg(from_account_id) AND a.deleted_at IS NULL
    ORDER BY a.id, t.created_at DESC
) recent_destinations
ORDER BY last_used_at DESC
LIMIT sqlc.arg(limit_count);

-- name: IncrementAccountBalance :one
UPDATE accounts
SET balance =  balance + $2, updated_at = now()
WHERE id = $1
RETURNING id, balance, currency, user_id, number, name, description, is_main, created_at;

-- name: CheckIsAccountWithIdExist :one
SELECT EXISTS (
    SELECT 1
    FROM accounts
    WHERE id = $1 AND deleted_at IS NULL
);

-- name: ListAccountsByUserId :many
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY id
LIMIT $2 OFFSET $3;

-- name: CountAccountsByUserId :one
SELECT COUNT(*) FROM accounts
WHERE user_id = $1 AND deleted_at IS NULL;
