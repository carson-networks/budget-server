CREATE TABLE categories (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                 TEXT NOT NULL,
    is_group             BOOLEAN NOT NULL,
    parent_id            UUID NULL,
    should_be_budgeted   BOOLEAN NOT NULL,
    is_disabled          BOOLEAN NOT NULL,
    category_type        SMALLINT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories(id)
);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_category_id
    FOREIGN KEY (category_id) REFERENCES categories(id);
