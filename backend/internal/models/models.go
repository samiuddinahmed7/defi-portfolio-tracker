package models

import (
	"time"
)

// Wallet represents a tracked Ethereum wallet address.
type Wallet struct {
	ID        int64     `json:"id" db:"id"`
	Address   string    `json:"address" db:"address"`
	Label     string    `json:"label,omitempty" db:"label"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Token represents an ERC-20 token (metadata only, not a balance).
type Token struct {
	ID          int64     `json:"id" db:"id"`
	Address     string    `json:"address" db:"address"`       // contract address
	Symbol      string    `json:"symbol" db:"symbol"`         // e.g. "USDC"
	Name        string    `json:"name" db:"name"`             // e.g. "USD Coin"
	Decimals    int       `json:"decimals" db:"decimals"`     // usually 18, but varies
	LogoURL     string    `json:"logo_url,omitempty" db:"logo_url"`
	CachedAt    time.Time `json:"cached_at" db:"cached_at"`
}

// Balance is a wallet's holding of a specific token (or native ETH).
type Balance struct {
	ID            int64     `json:"id" db:"id"`
	WalletAddress string    `json:"wallet_address" db:"wallet_address"`
	TokenAddress  string    `json:"token_address" db:"token_address"` // empty string = native ETH
	TokenSymbol   string    `json:"token_symbol" db:"token_symbol"`
	TokenName     string    `json:"token_name" db:"token_name"`
	Decimals      int       `json:"decimals" db:"decimals"`
	RawBalance    string    `json:"raw_balance" db:"raw_balance"`       // big integer as string
	NormalBalance float64   `json:"normal_balance" db:"normal_balance"` // divided by 10^decimals
	FetchedAt     time.Time `json:"fetched_at" db:"fetched_at"`
}

// Transaction represents a single Ethereum transaction involving a wallet.
type Transaction struct {
	ID              int64     `json:"id" db:"id"`
	Hash            string    `json:"hash" db:"hash"`
	WalletAddress   string    `json:"wallet_address" db:"wallet_address"`
	BlockNumber     int64     `json:"block_number" db:"block_number"`
	Timestamp       time.Time `json:"timestamp" db:"timestamp"`
	From            string    `json:"from" db:"from_address"`
	To              string    `json:"to" db:"to_address"`
	Value           string    `json:"value" db:"value"`             // wei as string
	NormalValue     float64   `json:"normal_value" db:"normal_value"` // ETH
	Gas             int64     `json:"gas" db:"gas"`
	GasUsed         int64     `json:"gas_used" db:"gas_used"`
	GasPrice        string    `json:"gas_price" db:"gas_price"`
	TokenAddress    string    `json:"token_address,omitempty" db:"token_address"`
	TokenSymbol     string    `json:"token_symbol,omitempty" db:"token_symbol"`
	TokenValue      string    `json:"token_value,omitempty" db:"token_value"`
	TokenDecimals   int       `json:"token_decimals,omitempty" db:"token_decimals"`
	TxType          string    `json:"type" db:"tx_type"` // "send", "receive", "token_transfer"
	Status          string    `json:"status" db:"status"` // "success", "failed"
}

// PriceSnapshot records a token price at a specific point in time.
type PriceSnapshot struct {
	ID           int64     `json:"id" db:"id"`
	Symbol       string    `json:"symbol" db:"symbol"`
	PriceUSD     float64   `json:"price_usd" db:"price_usd"`
	Source       string    `json:"source" db:"source"` // "chainlink", "etherscan", "alchemy", "demo"
	FetchedAt    time.Time `json:"fetched_at" db:"fetched_at"`
}

// PortfolioSummary is an aggregate view built for the API response.
type PortfolioSummary struct {
	Address       string          `json:"address"`
	NativeBalance float64         `json:"native_balance"` // ETH
	NativeUSD     float64         `json:"native_usd"`
	Tokens        []TokenHolding  `json:"tokens"`
	TotalUSD      float64         `json:"total_usd"`
	FetchedAt     time.Time       `json:"fetched_at"`
	IsDemo        bool            `json:"is_demo,omitempty"`
}

// TokenHolding combines a balance with a current price for dashboard display.
type TokenHolding struct {
	TokenAddress string  `json:"token_address"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	Quantity     float64 `json:"quantity"`
	PriceUSD     float64 `json:"price_usd"`
	ValueUSD     float64 `json:"value_usd"`
	PriceSource  string  `json:"price_source"`
}

// PaginationParams holds common pagination inputs.
type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// TransactionPage wraps a page of transactions with metadata.
type TransactionPage struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	HasNext      bool          `json:"has_next"`
}

// PriceResponse is returned by the /prices/:symbol endpoint.
type PriceResponse struct {
	Symbol    string    `json:"symbol"`
	PriceUSD  float64   `json:"price_usd"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}

// ErrorResponse is the standard JSON error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}
