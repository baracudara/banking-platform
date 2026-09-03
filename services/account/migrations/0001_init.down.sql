-- Down migrations exist so a bad deploy can be rolled back without a manual
-- psql session. Drop order is the reverse of the up migration.
DROP INDEX IF EXISTS idx_accounts_user_id;
DROP TABLE IF EXISTS accounts;
