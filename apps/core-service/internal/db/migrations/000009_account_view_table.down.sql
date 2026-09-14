BEGIN;

-- drop view table for account details
DROP VIEW IF EXISTS account_user_details_view;

DROP INDEX IF EXISTS idx_accounts_user_id;


COMMIT;