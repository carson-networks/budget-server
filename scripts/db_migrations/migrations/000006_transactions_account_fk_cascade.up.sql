-- Orphan transactions block adding the FK (account_id had no referential integrity).
-- Deleting them also removes plaid_transaction_links via ON DELETE CASCADE on transaction_id.
DELETE FROM transactions t
WHERE NOT EXISTS (
    SELECT 1 FROM accounts a WHERE a.id = t.account_id
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions (account_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_account_id
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE;
