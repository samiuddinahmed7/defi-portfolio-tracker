# Provider Reliability & Failure Modes

This document describes how the DeFi Portfolio Tracker handles failures across its three external data sources: Etherscan, Alchemy, and Chainlink oracle feeds.

## Oracle Problem Background

Blockchains are deterministic, sandboxed environments. Smart contracts cannot call external HTTP APIs — to use real-world data (like an ETH/USD price), the chain needs an *oracle*: a service that writes external data on-chain as a transaction, making it accessible to contracts.

The **oracle problem** is about trust: who decides what the "correct" price is, and what happens if they lie or go offline?

Chainlink solves this with a decentralised oracle network. Multiple independent node operators each submit a price observation to an aggregator contract. The contract applies an outlier-elimination algorithm and publishes a median answer. This means:

- No single node can corrupt the feed.
- A subset of nodes going offline does not stop the feed.
- Historical round data is permanently on-chain and auditable.

This application reads the `ETH/USD` Chainlink feed directly via JSON-RPC (`eth_call`) rather than through an intermediary API, demonstrating the oracle pattern end-to-end.

## Failure Modes

### 1. Chainlink feed is stale

**Cause:** Network congestion, all oracles offline, or the RPC node is lagging.

**Detection:** Each `latestRoundData()` response includes an `updatedAt` timestamp. The provider returns `ErrStalePrice` if the data is older than 2 hours (`stalePriceThreshold`).

**Handling:**
- The price service falls back to the in-memory cache (last known good price).
- If the cache is also empty, it falls back to the demo price table.
- The `source` field in `PriceResponse` tells the caller which path was used (`chainlink`, `cache`, or `demo`).

**Risk:** A stale price shown on the dashboard is labelled with its source so users know it is not live.

---

### 2. Alchemy API key is invalid or rate-limited

**Cause:** Expired key, billing issue, or burst traffic exceeding the free-tier limit.

**Detection:** HTTP 401 or 429 from the Alchemy JSON-RPC endpoint.

**Handling:**
- 429 responses are retried with exponential back-off (up to 3 attempts, max delay 5 s).
- On exhausted retries, the server responds with HTTP 502 and an error message that distinguishes rate-limiting from authentication failure.
- If `ALCHEMY_API_KEY` is not configured, the server automatically falls back to Etherscan.

---

### 3. Etherscan API key is missing or returns an error

**Cause:** Missing key, wrong network, or Etherscan maintenance.

**Detection:** `"NOTOK"` status field or non-200 HTTP status in the response.

**Handling:**
- Retried up to 3 times for transient HTTP errors (5xx).
- On permanent failure, falls back to demo mode if `DEMO_MODE=true`.
- `name()` on the provider is logged with every error so operators can see which provider failed.

---

### 4. PostgreSQL is unavailable

**Cause:** Database container not started, network partition, or pool exhaustion.

**Detection:** Connection error on startup (`repositories.Connect`).

**Handling:**
- The server logs a fatal error and exits — portfolio data cannot be persisted reliably without a database in production mode.
- In demo mode (`DEMO_MODE=true`), all DB operations are skipped (nil DB guard throughout the codebase), so the server runs in-memory only.
- All DB writes from `PortfolioService` are wrapped in nil-checks (`best-effort`): a write failure is logged but does not break the API response.

---

### 5. Ethereum RPC node is unreachable

**Cause:** The configured `ETHEREUM_RPC_URL` node is down or the URL is wrong.

**Detection:** Network dial error or non-200 HTTP response from `eth_call`.

**Handling:**
- Chainlink provider returns an error.
- Price service falls back to cache, then demo.
- If `ETHEREUM_RPC_URL` is not set, the Chainlink provider is not registered and prices fall back gracefully.

---

### 6. Token price is unavailable

**Cause:** A token in the wallet is not listed in Chainlink feeds or the demo price table (e.g., an obscure ERC-20).

**Detection:** All price providers return `ErrPriceNotFound`.

**Handling:**
- The holding is included in the response with `usd_value: 0` and `price: null`.
- The frontend renders "N/A" in the price column rather than "$0.00" to avoid misleading the user.
- Total portfolio value excludes these tokens.

---

## Retry Configuration

| Parameter         | Value        |
|-------------------|-------------|
| `MaxAttempts`     | 3           |
| `BaseDelay`       | 500 ms      |
| `MaxDelay`        | 5 s         |
| Retryable codes   | 429, 502, 503, 504 |

The retry wrapper is in `backend/internal/providers/retry.go` and is applied at the service layer, not in individual providers, keeping provider code simple.

## In-Memory Cache TTL

| Cache bucket                 | TTL     |
|-----------------------------|---------|
| Portfolio summary            | 2 min   |
| Native + token balances      | 2 min   |
| Transactions (per address)   | 5 min   |
| Chainlink price (implicit)   | Until stale threshold (2 h) |

Cache hits are logged at debug level with the key and remaining TTL. A cache miss triggers a live provider call.

## Demo Mode

Set `DEMO_MODE=true` in `.env` to run entirely from fixture data with no API keys or database needed. Demo mode uses a well-known Ethereum Foundation address (`0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae`) with hardcoded balances, token holdings, and transactions. All prices come from a static map in `backend/internal/providers/demo/demo.go`.
