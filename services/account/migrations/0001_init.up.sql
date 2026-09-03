CREATE TABLE IF NOT EXISTS accounts (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL,
    currency      TEXT NOT NULL,
    balance_cache BIGINT NOT NULL DEFAULT 0,
    version       BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every balance lookup by user (e.g. "list my accounts") filters on
-- user_id, so it needs its own index - the primary key index on id alone
-- doesn't help that query pattern.
CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts (user_id);
