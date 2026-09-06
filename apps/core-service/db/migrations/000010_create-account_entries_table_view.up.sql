

BEGIN;

CREATE VIEW  account_entries_view AS
SELECT
    a.name AS account_name,
    a.number as account_number,
    a.is_main,
    at.name AS to_account_name,
    at.number AS to_account_number,
    e.id, 
    e.user_id,
    e.account_id,
    e.created_at,
    at.id as to_account_id,
    e.type,
    e.amount,
    e.description
FROM entries e
LEFT JOIN accounts a ON e.account_id = a.id
LEFT JOIN transfers t ON e.transfer_id = t.id
LEFT JOIN accounts at ON t.to_account_id = at.id;

COMMIT;