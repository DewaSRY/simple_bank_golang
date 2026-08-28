ALTER TABLE "accounts" ADD CONSTRAINT accounts_balance_non_negative CHECK (balance >= 0);
ALTER TABLE "transfers" ADD CONSTRAINT transfers_amount_positive CHECK (amount > 0);
