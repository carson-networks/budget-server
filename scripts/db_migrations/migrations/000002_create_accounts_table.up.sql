CREATE TABLE accounts (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             TEXT NOT NULL,
    type             SMALLINT NOT NULL,
    sub_type         TEXT NOT NULL,
    balance          DECIMAL(100, 4) NOT NULL DEFAULT 0,
    starting_balance DECIMAL(100, 4) NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
