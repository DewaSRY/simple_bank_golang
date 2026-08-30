BEGIN;

-- Add accounts balance constraint only if it does not exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'accounts_balance_non_negative'
          AND conrelid = 'accounts'::regclass
    ) THEN
        ALTER TABLE "accounts"
        ADD CONSTRAINT accounts_balance_non_negative
        CHECK (balance >= 0);
    END IF;
END $$;

-- Add transfers amount constraint only if it does not exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'transfers_amount_positive'
          AND conrelid = 'transfers'::regclass
    ) THEN
        ALTER TABLE "transfers"
        ADD CONSTRAINT transfers_amount_positive
        CHECK (amount > 0);
    END IF;
END $$;

COMMIT;
