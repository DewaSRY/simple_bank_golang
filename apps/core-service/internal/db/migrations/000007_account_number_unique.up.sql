BEGIN;

-- Backfill first: the previous per-user daily-sequence generator was never
-- globally unique, so any pre-existing rows with a NULL or duplicate number
-- must be normalized before a unique index can be added. Deriving the
-- replacement from the row's own immutable id guarantees no new collisions.
UPDATE accounts
SET number = 'ACT/' || upper(to_char(created_at, 'DD-Mon-YYYY')) || '/' || lpad(id::text, 3, '0')
WHERE number IS NULL
   OR number IN (
        SELECT number FROM accounts GROUP BY number HAVING COUNT(*) > 1
   );

CREATE UNIQUE INDEX IF NOT EXISTS accounts_number_unique_idx ON accounts (number);

COMMIT;
