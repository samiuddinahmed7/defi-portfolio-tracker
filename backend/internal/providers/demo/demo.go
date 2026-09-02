// Package demo provides fixture data for development and testing when live
// API keys are not available. The data is clearly marked as demo in all
// responses so it is never mistaken for real holdings.
package demo

import (
	"context"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
)

const moduleName = "demo"

// demoAddress is the well-known Ethereum Foundation address, used so the
// demo data feels realistic.
const demoAddress = "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae"

// Provider returns deterministic fixture data.
type Provider struct{}

func New() *Provider { return &Provider{} }
func (p *Provider) Name() string { return moduleName }

func (p *Provider) GetNativeBalance(_ context.Context, address string) (*providers.NativeBalance, error) {
	return &providers.NativeBalance{
		Address:    address,
		WeiBalance: "1234567890000000000",
		ETHBalance: 1.23456789,
	}, nil
}

func (p *Provider) GetTokenBalances(_ context.Context, _ string) ([]providers.TokenBalance, error) {
	return []providers.TokenBalance{
		{
			TokenAddress: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Symbol:       "USDC",
			Name:         "USD Coin",
			Decimals:     6,
			RawBalance:   "2500000000",
			Balance:      2500.0,
		},
		{
			TokenAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Symbol:       "USDT",
			Name:         "Tether USD",
			Decimals:     6,
			RawBalance:   "1000000000",
			Balance:      1000.0,
		},
		{
			TokenAddress: "0x514910771af9ca656af840dff83e8264ecf986ca",
			Symbol:       "LINK",
			Name:         "ChainLink Token",
			Decimals:     18,
			RawBalance:   "50000000000000000000",
			Balance:      50.0,
		},
		{
			TokenAddress: "0x6b175474e89094c44da98b954eedeac495271d0f",
			Symbol:       "DAI",
			Name:         "Dai Stablecoin",
			Decimals:     18,
			RawBalance:   "750000000000000000000",
			Balance:      750.0,
		},
	}, nil
}

func (p *Provider) GetTransactions(_ context.Context, address string, _, _ int) ([]providers.RawTransaction, error) {
	now := time.Now().UTC()
	return []providers.RawTransaction{
		{
			Hash:        "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			BlockNumber: 19000000,
			Timestamp:   now.Add(-2 * time.Hour),
			From:        address,
			To:          "0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
			Value:       "500000000000000000",
			NormalValue: 0.5,
			Gas:         21000,
			GasUsed:     21000,
			GasPrice:    "20000000000",
			TxType:      "send",
			Status:      "success",
		},
		{
			Hash:        "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			BlockNumber: 18999000,
			Timestamp:   now.Add(-24 * time.Hour),
			From:        "0xd8da6bf26964af9d7eed9e03e53415d37aa96045",
			To:          address,
			Value:       "1000000000000000000",
			NormalValue: 1.0,
			Gas:         21000,
			GasUsed:     21000,
			GasPrice:    "18000000000",
			TxType:      "receive",
			Status:      "success",
		},
		{
			Hash:          "0xdeadbeef1234567890abcdef1234567890abcdef1234567890abcdef12345678",
			BlockNumber:   18998000,
			Timestamp:     now.Add(-48 * time.Hour),
			From:          address,
			To:            "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Value:         "0",
			NormalValue:   0,
			Gas:           65000,
			GasUsed:       52000,
			GasPrice:      "15000000000",
			TokenAddress:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			TokenSymbol:   "USDC",
			TokenValue:    "500000000",
			TokenDecimals: 6,
			TxType:        "token_transfer",
			Status:        "success",
		},
	}, nil
}

// DemoPriceProvider provides fixture prices.
type DemoPriceProvider struct{}

func NewPriceProvider() *DemoPriceProvider { return &DemoPriceProvider{} }
func (p *DemoPriceProvider) Name() string  { return moduleName }

var demoPrices = map[string]float64{
	"ETH":  2350.00,
	"USDC": 1.00,
	"USDT": 1.00,
	"LINK": 14.50,
	"DAI":  1.00,
	"BTC":  43500.00,
	"WETH": 2350.00,
}

func (p *DemoPriceProvider) GetPrice(_ context.Context, symbol string) (*providers.PriceResult, error) {
	price, ok := demoPrices[symbol]
	if !ok {
		price = 1.0 // safe fallback
	}
	return &providers.PriceResult{
		Symbol:    symbol,
		PriceUSD:  price,
		Source:    moduleName,
		FetchedAt: time.Now(),
	}, nil
}
