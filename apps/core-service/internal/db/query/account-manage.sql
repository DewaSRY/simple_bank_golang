-- name: ListAccountsByUserId :many
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
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;;

-- name: CountAccountsByUserId :one
SELECT COUNT(*) FROM accounts
WHERE user_id = $1 AND deleted_at IS NULL;