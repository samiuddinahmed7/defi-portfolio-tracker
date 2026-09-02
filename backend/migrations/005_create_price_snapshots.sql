-- Migration 005: create price_snapshots table
-- Stores historical and current token prices from various sources.
CREATE TABLE IF NOT EXISTS price_snapshots (
    id         BIGSERIAL PRIMARY KEY,
    symbol     VARCHAR(50)      NOT NULL,
    price_usd  DOUBLE PRECISION NOT NULL,
    source     VARCHAR(50)      NOT NULL,   -- chainlink | etherscan | alchemy | demo
    fetched_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_price_snapshots_symbol     ON price_snapshots(symbol);
CREATE INDEX IF NOT EXISTS idx_price_snapshots_fetched_at ON price_snapshots(fetched_at DESC);
