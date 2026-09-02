-- Migration 002: create tokens metadata table
CREATE TABLE IF NOT EXISTS tokens (
    id        BIGSERIAL PRIMARY KEY,
    address   VARCHAR(42) NOT NULL UNIQUE,
    symbol    VARCHAR(50) NOT NULL,
    name      TEXT NOT NULL DEFAULT '',
    decimals  INT NOT NULL DEFAULT 18,
    logo_url  TEXT NOT NULL DEFAULT '',
    cached_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tokens_symbol ON tokens(symbol);
