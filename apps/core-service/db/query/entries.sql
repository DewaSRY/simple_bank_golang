


-- name: CreateEntries :one
INSERT INTO entries (
    account_id,
    type,
    amount
) VALUES (
    $1, $2, $3
)
RETURNING id, account_id, type, amount, created_at;

-- name: ListEntriesByAccount :many
SELECT id, account_id, type, amount, created_at
FROM entries
WHERE account_id = sqlc.arg(account_id)
ORDER BY id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountEntriesByAccount :one
SELECT COUNT(*) FROM entries
WHERE account_id = $1;