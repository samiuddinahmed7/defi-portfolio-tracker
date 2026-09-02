-- Migration 004: create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id             BIGSERIAL PRIMARY KEY,
    hash           VARCHAR(66)  NOT NULL,
    wallet_address VARCHAR(42)  NOT NULL,
    block_number   BIGINT       NOT NULL DEFAULT 0,
    timestamp      TIMESTAMPTZ  NOT NULL,
    from_address   VARCHAR(42)  NOT NULL DEFAULT '',
    to_address     VARCHAR(42)  NOT NULL DEFAULT '',
    value          TEXT         NOT NULL DEFAULT '0',  -- wei as string
    normal_value   DOUBLE PRECISION NOT NULL DEFAULT 0, -- ETH
    gas            BIGINT       NOT NULL DEFAULT 0,
    gas_used       BIGINT       NOT NULL DEFAULT 0,
    gas_price      TEXT         NOT NULL DEFAULT '0',
    token_address  VARCHAR(42)  NOT NULL DEFAULT '',
    token_symbol   VARCHAR(50)  NOT NULL DEFAULT '',
    token_value    TEXT         NOT NULL DEFAULT '',
    token_decimals INT          NOT NULL DEFAULT 0,
    tx_type        VARCHAR(30)  NOT NULL DEFAULT 'send',
    status         VARCHAR(20)  NOT NULL DEFAULT 'success',

    UNIQUE (hash, wallet_address)
);

CREATE INDEX IF NOT EXISTS idx_transactions_wallet   ON transactions(wallet_address);
CREATE INDEX IF NOT EXISTS idx_transactions_hash     ON transactions(hash);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC);
