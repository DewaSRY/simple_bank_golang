

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

-- name: GetMainAccountByUserId :one
SELECT id, balance, currency, user_id, number, name, description, is_main, created_at
FROM accounts
WHERE user_id = $1 AND is_main = true AND deleted_at IS NULL;


-- name: ListAccountsSearchByUserNumber :many
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
WHERE number LIKE '%' || sqlc.arg(number) || '%'
  AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);


-- name: CountAccountsSearchByUserNumber :one
SELECT COUNT(*)
FROM account_user_details_view
WHERE number LIKE '%' || sqlc.arg(number) || '%'
  AND deleted_at IS NULL;
