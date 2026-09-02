// Package providers defines the blockchain data provider abstraction.
// Using an interface here means the rest of the application is not coupled
// to any single data source — Etherscan, Alchemy, and Chainlink can all
// be swapped or combined without changing service-layer code.
package providers

import (
	"context"
	"time"
)

// NativeBalance holds the raw and human-readable ETH balance for a wallet.
type NativeBalance struct {
	Address     string
	WeiBalance  string  // raw wei as decimal string
	ETHBalance  float64 // balance divided by 1e18
}

// TokenBalance represents a single ERC-20 token holding.
type TokenBalance struct {
	TokenAddress string
	Symbol       string
	Name         string
	Decimals     int
	RawBalance   string  // raw token units
	Balance      float64 // normalised by decimals
}

// RawTransaction is a transaction as returned by a provider, before DB storage.
type RawTransaction struct {
	Hash          string
	BlockNumber   int64
	Timestamp     time.Time
	From          string
	To            string
	Value         string // wei string
	NormalValue   float64
	Gas           int64
	GasUsed       int64
	GasPrice      string
	TokenAddress  string
	TokenSymbol   string
	TokenValue    string
	TokenDecimals int
	TxType        string // send | receive | token_transfer
	Status        string // success | failed
}

// PriceResult contains a token price and metadata about the source.
type PriceResult struct {
	Symbol    string
	PriceUSD  float64
	Source    string
	FetchedAt time.Time
}

// BlockchainProvider is the central interface every data provider must satisfy.
// Providers that cannot fulfil a method should return a descriptive error rather
// than silently returning empty data.
type BlockchainProvider interface {
	// GetNativeBalance returns the ETH balance for the given wallet address.
	GetNativeBalance(ctx context.Context, address string) (*NativeBalance, error)

	// GetTokenBalances returns all ERC-20 token holdings for a wallet.
	GetTokenBalances(ctx context.Context, address string) ([]TokenBalance, error)

	// GetTransactions returns up to limit recent transactions for a wallet.
	GetTransactions(ctx context.Context, address string, page, perPage int) ([]RawTransaction, error)

	// Name returns a short identifier for logging and error attribution.
	Name() string
}

// PriceProvider fetches current token prices from an external oracle or API.
type PriceProvider interface {
	// GetPrice returns the current USD price for the given token symbol.
	GetPrice(ctx context.Context, symbol string) (*PriceResult, error)

	// Name returns a short identifier.
	Name() string
}
