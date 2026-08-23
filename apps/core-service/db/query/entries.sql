


-- name: CreateEntries :one
INSERT INTO entries (
    account_id,
    type,
    amount
) VALUES (
    $1, $2, $3
)
RETURNING id, account_id, type, amount, created_at;