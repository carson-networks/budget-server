ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_account_id;

DROP INDEX IF EXISTS idx_transactions_account_id;
