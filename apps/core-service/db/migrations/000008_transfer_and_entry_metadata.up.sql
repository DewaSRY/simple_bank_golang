BEGIN;

ALTER TABLE transfers
    ADD COLUMN IF NOT EXISTS description TEXT NULL;

ALTER TABLE entries
    ADD COLUMN IF NOT EXISTS description TEXT NULL,
    ADD COLUMN IF NOT EXISTS transfer_id BIGINT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'entries_transfer_id_fkey' AND conrelid = 'entries'::regclass
    ) THEN
        ALTER TABLE entries
        ADD CONSTRAINT entries_transfer_id_fkey
        FOREIGN KEY (transfer_id) REFERENCES transfers (id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS entries_transfer_id_idx ON entries (transfer_id);

COMMIT;
