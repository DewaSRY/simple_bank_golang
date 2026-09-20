BEGIN;

CREATE INDEX idx_entries_account_id_created_at_id ON entries (account_id, created_at, id);

COMMIT;
