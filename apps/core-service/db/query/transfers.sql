

-- name: CreateTransfer :one
INSERT INTO transfers (
    from_account_id,
    to_account_id,
    amount,
    description
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, from_account_id, to_account_id, amount, created_at, description;

-- name: GetTransferById :one
SELECT id, from_account_id, to_account_id, amount, created_at, description
FROM transfers
WHERE id = $1;
