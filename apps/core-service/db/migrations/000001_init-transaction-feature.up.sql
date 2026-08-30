BEGIN;

CREATE TABLE IF NOT EXISTS "accounts"  (
    id BIGSERIAL PRIMARY KEY,
    "owner" VARCHAR(255) NOT NULL,
    "balance" NUMERIC(20,2) NOT NULL,
    "currency" VARCHAR(3) NOT NULL,
    "created_at" timestamptz  NOT NULL DEFAULT now(),
    "updated_at" timestamptz  NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS "transfers" (
    id BIGSERIAL PRIMARY KEY,
    from_account_id BIGINT NOT NULL,
    to_account_id BIGINT NOT NULL,
    amount NUMERIC(20,2) NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT now()
);


CREATE TABLE IF NOT EXISTS "entries" (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(20,2) NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT now()
);



ALTER TABLE "entries" ADD FOREIGN KEY ("account_id") REFERENCES "accounts" ("id");

ALTER TABLE "transfers" ADD FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id");
ALTER TABLE "transfers" ADD FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id");


CREATE INDEX ON "accounts" ("owner");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");


COMMENT ON TABLE "accounts" IS 'Table to store account information';
COMMENT ON COLUMN "accounts"."balance" IS 'Balance cannot be negative, it is the amount of money in the account';

COMMENT ON TABLE "entries" IS 'Table to store account entries for each transaction';
COMMENT ON COLUMN "entries"."amount" IS 'Amount cannot be negative, it is the amount of money added or subtracted from the account';
COMMENT ON COLUMN "entries"."type" IS 'Type use to define the purpose of the entry';

COMMENT ON TABLE "transfers" IS 'Table to store transfer transactions between accounts';    
COMMENT ON COLUMN "transfers"."amount" IS 'Amount cannot be negative, it is the amount of money transferred from one account to another';


COMMIT;