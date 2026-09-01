BEGIN;

DROP INDEX IF EXISTS entries_transfer_id_idx;

ALTER TABLE entries
    DROP CONSTRAINT IF EXISTS entries_transfer_id_fkey,
    DROP COLUMN IF EXISTS transfer_id,
    DROP COLUMN IF EXISTS description;

ALTER TABLE transfers
    DROP COLUMN IF EXISTS description;

COMMIT;
