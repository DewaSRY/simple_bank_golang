-- name: ListMeAccountsByUserId :many
SELECT sqlc.embed(v)
FROM account_user_details_view v
WHERE v.user_id = sqlc.arg(user_id)
    AND v.name LIKE '%' || sqlc.arg(name) || '%' ESCAPE '\'
ORDER BY v.created_at ASC
LIMIT  sqlc.arg(limit_count) OFFSET  sqlc.arg(offset_count);

-- name: ListMeAccountsByUserIdCount :one
SELECT
    COUNT(*)
FROM account_user_details_view
WHERE user_id = sqlc.arg(user_id)
    AND name LIKE '%' || sqlc.arg(name) || '%' ESCAPE '\';

-- name: CountAccountsByUserId :one
SELECT COUNT(*) FROM accounts
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL;
