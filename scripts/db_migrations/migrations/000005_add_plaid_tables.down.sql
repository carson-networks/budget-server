ALTER TABLE transactions ALTER COLUMN category_id SET NOT NULL;

DROP TABLE IF EXISTS plaid_transaction_links;
DROP TABLE IF EXISTS plaid_account_links;
DROP TABLE IF EXISTS plaid_items;
