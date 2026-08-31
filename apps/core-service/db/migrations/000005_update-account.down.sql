BEGIN;

-- Add the old owner column back
ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS owner VARCHAR(255);

-- Remove the new columns
ALTER TABLE accounts
DROP COLUMN IF EXISTS number,
DROP COLUMN IF EXISTS name,
DROP COLUMN IF EXISTS description;

COMMIT;