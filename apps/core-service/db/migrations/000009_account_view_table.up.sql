BEGIN;

-- create view table 
CREATE VIEW  account_user_details_view AS
SELECT
    a.id,
    a.balance,
    a.currency,
    a.created_at,
    a.updated_at,
    a.user_id,
    a.name,
    a.description,
    a.is_main,
    a.number,
    u.username
FROM accounts AS a
LEFT JOIN users AS u
    ON u.id = a.user_id
WHERE a.deleted_at IS NULL;

CREATE INDEX idx_accounts_user_id ON accounts(user_id);

COMMIT;