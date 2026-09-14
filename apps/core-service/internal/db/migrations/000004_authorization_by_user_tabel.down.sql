-- ============================================================
-- Entries: remove constraints and user_id
-- ============================================================

ALTER TABLE "entries"
DROP CONSTRAINT IF EXISTS "entries_user_id_fkey";

ALTER TABLE "entries"
DROP COLUMN IF EXISTS "user_id";


-- ============================================================
-- Accounts: remove constraint and user_id
-- ============================================================

ALTER TABLE "accounts"
DROP CONSTRAINT IF EXISTS "accounts_user_id_fkey";

ALTER TABLE "accounts"
DROP COLUMN IF EXISTS "user_id";
