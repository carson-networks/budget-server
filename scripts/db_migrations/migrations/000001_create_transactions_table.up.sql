CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE transactions (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id       UUID NOT NULL,
    category_id      UUID NOT NULL,
    amount           DECIMAL(100, 4) NOT NULL,
    transaction_name TEXT NOT NULL,
    transaction_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
