-- name: ListRecentTransferDestinations :many
SELECT 
    id, 
    name, 
    number, 
    username, 
    user_id,
    last_used_at 
FROM (
    SELECT 
    	DISTINCT ON (a.id) a.id,
         a.name, 
         a.number, 
         t.created_at AS last_used_at, 
         u.username as username,
         u.id as user_id
    FROM transfers t
    JOIN accounts a ON a.id = t.to_account_id
    join users u on u.id = a.user_id
    WHERE t.from_account_id = sqlc.arg(from_account_id) AND a.deleted_at IS NULL
    ORDER BY a.id, t.created_at DESC
) recent_destinations
ORDER BY last_used_at desc
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: GetAccountViewById :one
SELECT sqlc.embed(v)
FROM account_user_details_view v
WHERE v.id = $1;

-- name: GetAccountByIdForUpdate :one
SELECT *
FROM accounts
WHERE id = $1 AND deleted_at IS NULL
FOR UPDATE;

-- name: IncrementAccountBalance :one
UPDATE accounts
SET balance =  balance + $2, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CheckIsAccountWithIdExist :one
SELECT EXISTS (
    SELECT 1
    FROM accounts
    WHERE id = $1 AND deleted_at IS NULL
);

-- name: ListAccountEntriesByAccountId :many
SELECT sqlc.embed(v)
FROM account_entries_view v
WHERE v.account_id = sqlc.arg(account_id)
  AND (
      sqlc.narg(period_start)::timestamptz IS NULL
      OR v.created_at >= sqlc.narg(period_start)::timestamptz
  )
  AND (
      sqlc.narg(period_end)::timestamptz IS NULL
      OR v.created_at < sqlc.narg(period_end)::timestamptz
  )
  AND (
      sqlc.narg(entry_type)::varchar IS NULL
      OR v.type = sqlc.narg(entry_type)::varchar
  )
ORDER BY v.created_at DESC, v.id DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountAccountEntriesByAccountId :one
SELECT COUNT(*)
FROM account_entries_view
WHERE account_id = sqlc.arg(account_id)
  AND (
      sqlc.narg(period_start)::timestamptz IS NULL
      OR created_at >= sqlc.narg(period_start)::timestamptz
  )
  AND (
      sqlc.narg(period_end)::timestamptz IS NULL
      OR created_at < sqlc.narg(period_end)::timestamptz
  )
  AND (
      sqlc.narg(entry_type)::varchar IS NULL
      OR type = sqlc.narg(entry_type)::varchar
  );

-- name: AccountEntriesByAccountId :one
SELECT sqlc.embed(v)
FROM account_entries_view v
WHERE v.id = $1;
