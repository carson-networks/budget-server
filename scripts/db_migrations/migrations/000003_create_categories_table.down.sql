ALTER TABLE transactions DROP CONSTRAINT IF EXISTS fk_transactions_category_id;
DROP TABLE IF EXISTS categories;
