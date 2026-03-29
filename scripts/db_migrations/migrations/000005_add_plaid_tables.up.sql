CREATE TABLE syncs (
    account_id UUID PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    sync_type  SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_syncs_sync_type ON syncs (sync_type);

CREATE TABLE plaid_items (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    access_token     TEXT NOT NULL,
    plaid_item_id    TEXT NOT NULL UNIQUE,
    institution_id   TEXT NOT NULL,
    institution_name TEXT NOT NULL,
    cursor           TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE plaid_account_links (
    plaid_account_id TEXT NOT NULL PRIMARY KEY,
    account_id       UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    plaid_item_id    UUID NOT NULL REFERENCES plaid_items(id) ON DELETE CASCADE
);

CREATE INDEX idx_plaid_account_links_account_id    ON plaid_account_links(account_id);
CREATE INDEX idx_plaid_account_links_plaid_item_id ON plaid_account_links(plaid_item_id);

CREATE TABLE plaid_transaction_links (
    plaid_transaction_id TEXT NOT NULL PRIMARY KEY,
    transaction_id       UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    plaid_account_id     TEXT NOT NULL
);

CREATE INDEX idx_plaid_transaction_links_transaction_id ON plaid_transaction_links(transaction_id);

ALTER TABLE transactions ALTER COLUMN category_id DROP NOT NULL;
