# DeFi Portfolio Tracker

A learning project that tracks an Ethereum wallet's token holdings and transaction history, with live prices from Chainlink oracle feeds. Built to understand how on-chain price oracles work in practice.

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go) ![React](https://img.shields.io/badge/React-18-61DAFB?logo=react) ![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?logo=typescript) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?logo=postgresql)

---

## Why Chainlink?

On a blockchain, smart contracts can't make HTTP calls. To get an ETH/USD price, the chain needs an **oracle** — a service that writes external data on-chain as a transaction. The **oracle problem** is about trust: a single source can be wrong, manipulated, or go offline.

Chainlink solves this with a decentralised network of independent node operators. Each submits a price observation; the aggregator contract discards outliers and publishes a median. No single party can corrupt the feed, and the data is permanently on-chain.

This project reads the Chainlink `ETH/USD` aggregator contract directly via JSON-RPC (`eth_call`) — no SDK, just raw ABI encoding. See [`backend/internal/providers/chainlink/chainlink.go`](backend/internal/providers/chainlink/chainlink.go) for the implementation. See [`docs/reliability.md`](docs/reliability.md) for how failures are handled.

---

## Architecture

```
Browser
  │
  ▼
┌─────────────────────────────────────────────────────┐
│  Frontend  (React + TypeScript + Vite, port 3000)   │
│  Dashboard → WalletInput → PortfolioOverview        │
│           → TokenAllocation → HoldingsTable         │
│           → TransactionHistory                      │
└────────────────────────┬────────────────────────────┘
                         │  /api/v1/*  (proxy or direct)
                         ▼
┌─────────────────────────────────────────────────────┐
│  Backend  (Go 1.24, stdlib net/http, port 8080)     │
│                                                     │
│  Middleware: Logger → Recoverer → CORS              │
│  Handlers:  portfolio · balance · tokens · tx · prices │
│                                                     │
│  Services:                                          │
│   PortfolioService  ←→  BlockchainProvider          │
│   PriceService      ←→  PriceProvider               │
│   (in-memory cache, 2–5 min TTL)                    │
│                                                     │
│  Providers (selected by available env vars):        │
│   Alchemy ──┐                                       │
│   Etherscan─┤ BlockchainProvider interface          │
│   Demo ─────┘                                       │
│                                                     │
│   Chainlink ─┐ PriceProvider interface              │
│   Demo ──────┘                                      │
│                                                     │
│  Repositories: balances · transactions · prices     │
└──────────┬──────────────────┬───────────────────────┘
           │                  │
           ▼                  ▼
     PostgreSQL          Ethereum RPC
     (balances,          (Chainlink eth_call,
      transactions,       Alchemy/Etherscan APIs)
      price snapshots)
```

---

## Features

- **Multi-provider blockchain data** — prefers Alchemy (richer token metadata), falls back to Etherscan, then demo fixtures.
- **Live oracle prices** — reads the Chainlink `ETH/USD` aggregator contract directly via `eth_call`. Stale feeds (>2 h) fall back to cache, then demo prices.
- **Portfolio overview** — total USD value, ETH balance, token count in a responsive stat grid.
- **Token allocation bar** — stacked horizontal bar grouping small holdings (<2%) into "Other".
- **Holdings table** — sortable by USD value with inline percentage bars.
- **Transaction history** — paginated table with type badges, amounts, and Etherscan links.
- **Demo mode** — runs entirely from fixture data; no API keys or database required.
- **Dark/light theme** — follows `prefers-color-scheme`, with manual override support.

---

## Quick Start (Demo Mode)

No API keys needed:

```bash
# Clone
git clone https://github.com/samiuddinahmed7/defi-portfolio-tracker.git
cd defi-portfolio-tracker

# Copy env template and enable demo mode
cp .env.example .env
# Edit .env and set DEMO_MODE=true

# Start everything
docker compose up --build

# Open http://localhost:3000
# Enter any Ethereum address, or click "Load demo wallet"
```

---

## Local Development (with real data)

### Prerequisites

- Go 1.24+
- Node 20+
- PostgreSQL 16 (or use `docker compose up postgres`)
- Etherscan API key: https://etherscan.io/apis
- Alchemy API key: https://dashboard.alchemy.com

### Backend

```bash
cd backend

# Install dependencies
go mod download

# Copy and fill in .env (see .env.example)
cp ../.env.example ../.env
# Set ETHERSCAN_API_KEY, ALCHEMY_API_KEY, DATABASE_URL

# Run migrations and start the server
go run ./cmd/server
# Server starts on :8080, runs migrations automatically
```

### Frontend

```bash
cd frontend
npm install
npm run dev          # dev server on :3000 with /api proxy to :8080
npm test             # run Vitest unit tests
npm run build        # production build
```

---

## Environment Variables

See [`.env.example`](.env.example) for all variables. Key ones:

| Variable                  | Description                                          | Required      |
|---------------------------|------------------------------------------------------|---------------|
| `ETHERSCAN_API_KEY`       | Etherscan v2 API key                                 | For real data |
| `ALCHEMY_API_KEY`         | Alchemy API key (preferred provider)                 | For real data |
| `DATABASE_URL`            | PostgreSQL connection string                         | Unless demo   |
| `ETHEREUM_RPC_URL`        | JSON-RPC endpoint for Chainlink reads                | For live prices |
| `CHAINLINK_ETH_USD_FEED`  | ETH/USD aggregator address (defaults to mainnet)     | Optional      |
| `DEMO_MODE`               | `true` to use fixture data, skip DB                  | Optional      |
| `LOG_LEVEL`               | `debug`, `info`, `warn`, `error`                     | Optional      |
| `PROVIDER_TIMEOUT_SECONDS`| HTTP timeout for external calls (default 10)         | Optional      |

---

## API Reference

All endpoints are under `/api/v1`.

### `GET /api/v1/health`

Returns server status.

```json
{ "status": "ok", "provider": "alchemy", "demo": false }
```

### `GET /api/v1/portfolio/{address}`

Full portfolio summary with balances priced in USD.

```json
{
  "address": "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae",
  "total_usd": 312450.72,
  "is_demo": true,
  "holdings": [
    {
      "symbol": "ETH",
      "name": "Ethereum",
      "quantity": 100.0,
      "price_usd": 2450.50,
      "usd_value": 245050.00,
      "pct_of_portfolio": 78.4
    }
  ]
}
```

### `GET /api/v1/portfolio/{address}/transactions?page=1&per_page=25`

Paginated transaction history.

```json
{
  "transactions": [...],
  "page": 1,
  "per_page": 25,
  "has_next": true
}
```

### `GET /api/v1/prices/{symbol}`

Single asset price from Chainlink (or fallback).

```json
{
  "symbol": "ETH",
  "price_usd": 2450.50,
  "source": "chainlink",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

---

## Project Structure

```
defi-portfolio-tracker/
├── backend/
│   ├── cmd/server/main.go          # Entry point, provider wiring, graceful shutdown
│   ├── internal/
│   │   ├── cache/                  # Thread-safe in-memory cache with TTL
│   │   ├── config/                 # Config from environment variables
│   │   ├── handlers/               # HTTP handlers (Go 1.22 ServeMux)
│   │   ├── middleware/             # Logger, CORS, panic recoverer
│   │   ├── models/                 # Domain types
│   │   ├── providers/
│   │   │   ├── alchemy/            # Alchemy JSON-RPC client
│   │   │   ├── chainlink/          # Direct eth_call ABI reader
│   │   │   ├── demo/               # Fixture data for offline development
│   │   │   ├── etherscan/          # Etherscan v2 REST client
│   │   │   ├── provider.go         # BlockchainProvider / PriceProvider interfaces
│   │   │   └── retry.go            # Exponential back-off retry wrapper
│   │   ├── repositories/           # PostgreSQL data access layer
│   │   ├── services/               # Business logic + caching
│   │   └── validation/             # Ethereum address validation
│   ├── migrations/                 # SQL migration files (run on startup)
│   ├── tests/                      # Go unit tests
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/             # React components
│   │   ├── hooks/usePortfolio.ts   # Data fetching hook
│   │   ├── pages/Dashboard.tsx     # Main page
│   │   ├── services/api.ts         # Typed fetch client
│   │   ├── styles/index.css        # Design system (CSS custom properties)
│   │   ├── types/                  # TypeScript domain types
│   │   └── utils/                  # Formatting, address validation
│   ├── Dockerfile
│   └── nginx.conf
├── docs/
│   └── reliability.md              # Failure modes and fallback chains
├── docker-compose.yml
├── .env.example
└── README.md
```

---

## Running Tests

```bash
# Backend (Go)
cd backend
go test ./... -v

# Frontend (Vitest)
cd frontend
npm test
```

---

## What I Learned

Building this project clarified several things about DeFi infrastructure:

**Oracles are not magic.** Reading a Chainlink feed is just an `eth_call` to a specific contract address. The decentralisation lives in how that contract was updated, not in how you read it. Writing the ABI encoder from scratch (encoding a 4-byte selector, decoding five 32-byte words) made the data layout concrete.

**Provider abstractions pay off immediately.** The `BlockchainProvider` interface let me swap Alchemy for Etherscan for demo fixtures without touching the service layer. The retry wrapper stays outside the providers so each provider stays simple.

**Stale data is as dangerous as wrong data.** The `updatedAt` staleness check on the Chainlink feed — and the fallback chain (Chainlink → cache → demo) — shows why oracle consumers need to validate freshness, not just presence.

---

## License

MIT — see [LICENSE](LICENSE).
