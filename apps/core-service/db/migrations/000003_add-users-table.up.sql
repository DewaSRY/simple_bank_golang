CREATE TABLE IF NOT EXISTS "users" (
    id BIGSERIAL PRIMARY KEY,
    "username" VARCHAR(255) UNIQUE NOT NULL,
    "email" VARCHAR(255) UNIQUE NOT NULL,
    "hashed_password" VARCHAR(255) NOT NULL,
    "created_at" timestamptz NOT NULL DEFAULT now(),
    "updated_at" timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON "users" ("email");

COMMENT ON TABLE "users" IS 'Table to store user accounts used for authentication';
COMMENT ON COLUMN "users"."hashed_password" IS 'Password hash generated with bcrypt, plaintext passwords are never stored';
