

-- name: CreateEntries :one
INSERT INTO entries (
    account_id,
    type,
    amount,
    description,
    transfer_id
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING id, account_id, type, amount, description, transfer_id, created_at;

-- name: ListEntriesByAccount :many
SELECT id, account_id, type, amount, description, transfer_id, created_at
FROM entries
WHERE account_id = sqlc.arg(account_id)
ORDER BY id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountEntriesByAccount :one
SELECT COUNT(*) FROM entries
WHERE account_id = $1;

-- name: ListAccountTransactionHistory :many
SELECT
    e.id,
    e.type,
    e.amount,
    e.description,
    e.created_at,
    ca.id     AS counterparty_account_id,
    ca.name   AS counterparty_account_name,
    ca.number AS counterparty_account_number
FROM entries e
LEFT JOIN transfers t ON t.id = e.transfer_id
LEFT JOIN accounts ca ON ca.id = CASE e.type
    WHEN 'SEND'     THEN t.to_account_id
    WHEN 'RECEIVED' THEN t.from_account_id
    ELSE NULL
END
WHERE e.account_id = sqlc.arg(account_id)
  AND e.created_at >= sqlc.arg(period_start)
  AND e.created_at < sqlc.arg(period_end)
ORDER BY e.created_at DESC, e.id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountAccountTransactionHistory :one
SELECT COUNT(*) FROM entries
WHERE account_id = sqlc.arg(account_id)
  AND created_at >= sqlc.arg(period_start)
  AND created_at < sqlc.arg(period_end);
