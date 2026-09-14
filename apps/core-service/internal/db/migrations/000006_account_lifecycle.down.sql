BEGIN;

DROP INDEX IF EXISTS accounts_one_main_per_user_idx;
DROP INDEX IF EXISTS accounts_user_id_active_idx;

ALTER TABLE accounts
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_main;

COMMIT;
