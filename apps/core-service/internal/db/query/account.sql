

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
RETURNING *;

-- name: UpdateAccountNumber :one
UPDATE accounts
SET number = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET name = $2, description = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteAccount :one
UPDATE accounts
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetAccountById :one
SELECT *
FROM accounts
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetMainAccountByUserId :one
SELECT *
FROM accounts
WHERE user_id = $1 AND is_main = true AND deleted_at IS NULL;


-- name: ListAccountsSearchByUserNumber :many
SELECT sqlc.embed(v)
FROM account_user_details_view v
WHERE v.number LIKE '%' || sqlc.arg(number) || '%' ESCAPE '\'
ORDER BY v.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);


-- name: CountAccountsSearchByUserNumber :one
SELECT COUNT(*)
FROM account_user_details_view
WHERE number LIKE '%' || sqlc.arg(number) || '%' ESCAPE '\';
