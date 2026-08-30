-- ============================================================
-- Accounts: nullable user_id
-- ============================================================

ALTER TABLE "accounts"
ADD COLUMN IF NOT EXISTS "user_id" BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'accounts_user_id_fkey'
          AND conrelid = 'accounts'::regclass
    ) THEN
        ALTER TABLE "accounts"
        ADD CONSTRAINT "accounts_user_id_fkey"
        FOREIGN KEY ("user_id")
        REFERENCES "users" ("id");
    END IF;
END $$;

-- ============================================================
-- Entries: nullable user_id
-- ============================================================

ALTER TABLE "entries"
ADD COLUMN IF NOT EXISTS "user_id" BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'entries_user_id_fkey'
          AND conrelid = 'entries'::regclass
    ) THEN
        ALTER TABLE "entries"
        ADD CONSTRAINT "entries_user_id_fkey"
        FOREIGN KEY ("user_id")
        REFERENCES "users" ("id");
    END IF;
END $$;



