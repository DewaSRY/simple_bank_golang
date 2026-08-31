BEGIN;

-- Remove the old owner column
ALTER TABLE accounts
DROP COLUMN IF EXISTS owner;

-- Add the new columns
ALTER TABLE accounts
ADD COLUMN IF NOT EXISTS number VARCHAR(50),
ADD COLUMN IF NOT EXISTS name VARCHAR(255),
ADD COLUMN IF NOT EXISTS description TEXT;

COMMIT;