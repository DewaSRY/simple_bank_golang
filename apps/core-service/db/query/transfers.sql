


-- name: CreateTransfer :one
INSERT INTO transfers (
    from_account_id,
    to_account_id,
    amount
) VALUES (
    $1, $2, $3
)
RETURNING id, from_account_id, to_account_id, amount, created_at;

-- name: GetTransferById :one
SELECT id, from_account_id, to_account_id, amount, created_at
FROM transfers
WHERE id = $1;

-- name: ListTransfersByOwner :many
SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.created_at
FROM transfers t
JOIN accounts fa ON fa.id = t.from_account_id
JOIN accounts ta ON ta.id = t.to_account_id
WHERE fa.owner = sqlc.arg(owner) OR ta.owner = sqlc.arg(owner)
ORDER BY t.id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountTransfersByOwner :one
SELECT COUNT(*)
FROM transfers t
JOIN accounts fa ON fa.id = t.from_account_id
JOIN accounts ta ON ta.id = t.to_account_id
WHERE fa.owner = $1 OR ta.owner = $1;

