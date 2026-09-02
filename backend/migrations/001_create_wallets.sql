-- Migration 001: create wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id         BIGSERIAL PRIMARY KEY,
    address    VARCHAR(42) NOT NULL UNIQUE,
    label      TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wallets_address ON wallets(address);
