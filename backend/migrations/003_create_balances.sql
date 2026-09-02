-- Migration 003: create balances table
-- Records the most recently fetched balance for a wallet/token pair.
CREATE TABLE IF NOT EXISTS balances (
    id             BIGSERIAL PRIMARY KEY,
    wallet_address VARCHAR(42)  NOT NULL,
    token_address  VARCHAR(42)  NOT NULL DEFAULT '',  -- empty = native ETH
    token_symbol   VARCHAR(50)  NOT NULL,
    token_name     TEXT         NOT NULL DEFAULT '',
    decimals       INT          NOT NULL DEFAULT 18,
    raw_balance    TEXT         NOT NULL DEFAULT '0',
    normal_balance DOUBLE PRECISION NOT NULL DEFAULT 0,
    fetched_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    UNIQUE (wallet_address, token_address)
);

CREATE INDEX IF NOT EXISTS idx_balances_wallet ON balances(wallet_address);
