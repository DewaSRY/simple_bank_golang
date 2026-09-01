BEGIN;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS is_main BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN accounts.is_main IS 'True for the single auto-created account a user gets at registration; cannot be deleted';
COMMENT ON COLUMN accounts.deleted_at IS 'Soft-delete marker; NULL means active';

-- Speeds up "list/get active accounts for a user" queries.
CREATE INDEX IF NOT EXISTS accounts_user_id_active_idx
    ON accounts (user_id)
    WHERE deleted_at IS NULL;

-- At most one active main account per user.
CREATE UNIQUE INDEX IF NOT EXISTS accounts_one_main_per_user_idx
    ON accounts (user_id)
    WHERE is_main = true AND deleted_at IS NULL;

COMMIT;
