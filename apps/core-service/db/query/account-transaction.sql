

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

-- name: GetAccountViewById :one
SELECT 
    id,
    balance,
    currency,
    created_at,
    updated_at,
    user_id,
    name,
    description,
    is_main,
    username,
    number
FROM account_user_details_view
WHERE id = $1;

-- name: GetAccountByIdForUpdate :one
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

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