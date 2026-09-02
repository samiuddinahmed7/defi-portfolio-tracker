package tests

import (
	"math/big"
	"testing"
)

// TestWeiToETHConversion verifies that the big.Int wei-to-ETH conversion
// produces correct results. This mirrors the logic in the etherscan provider.
func TestWeiToETHConversion(t *testing.T) {
	tests := []struct {
		name    string
		weiStr  string
		wantETH float64
	}{
		{"1 ETH",      "1000000000000000000",   1.0},
		{"0.5 ETH",    "500000000000000000",    0.5},
		{"0.1 ETH",    "100000000000000000",    0.1},
		{"0 ETH",      "0",                     0.0},
		{"10 ETH",     "10000000000000000000",  10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := weiToETH(tt.weiStr)
			// Allow small floating-point tolerance.
			diff := got - tt.wantETH
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-9 {
				t.Errorf("weiToETH(%s) = %f, want %f", tt.weiStr, got, tt.wantETH)
			}
		})
	}
}

// TestTokenNormalisation verifies balance normalisation for various decimal places.
func TestTokenNormalisation(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		decimals int
		want     float64
	}{
		{"USDC 6 decimals",    "1000000",             6,  1.0},
		{"USDC 100",           "100000000",           6,  100.0},
		{"LINK 18 decimals",   "1000000000000000000", 18, 1.0},
		{"LINK 50",            "50000000000000000000",18, 50.0},
		{"zero",               "0",                   18, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normaliseBalance(tt.raw, tt.decimals)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-6 {
				t.Errorf("normaliseBalance(%s, %d) = %f, want %f", tt.raw, tt.decimals, got, tt.want)
			}
		})
	}
}

// TestChainlinkDecimalConversion verifies that 8-decimal Chainlink prices
// (as returned by ETH/USD) are correctly converted.
func TestChainlinkDecimalConversion(t *testing.T) {
	// Simulate a latestRoundData answer of 200000000000 (= $2000.00000000)
	rawAnswer := new(big.Int)
	rawAnswer.SetString("200000000000", 10)

	decimals := int64(8)
	divisor := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(decimals), nil),
	)
	price, _ := new(big.Float).Quo(new(big.Float).SetInt(rawAnswer), divisor).Float64()

	want := 2000.0
	if price != want {
		t.Errorf("chainlink price conversion: got %f, want %f", price, want)
	}
}

// TestPortfolioValueCalculation verifies that USD value is correctly summed.
func TestPortfolioValueCalculation(t *testing.T) {
	type holding struct {
		qty   float64
		price float64
	}

	holdings := []holding{
		{1.5, 2000.0},   // ETH: $3000
		{2500.0, 1.0},   // USDC: $2500
		{50.0, 14.50},   // LINK: $725
	}

	var total float64
	for _, h := range holdings {
		total += h.qty * h.price
	}

	want := 6225.0
	if total != want {
		t.Errorf("portfolio total = %f, want %f", total, want)
	}
}

// — helper functions (mirrors of provider internals, for test isolation) ------

func weiToETH(weiStr string) float64 {
	if weiStr == "" || weiStr == "0" {
		return 0
	}
	wei := new(big.Int)
	wei.SetString(weiStr, 10)
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	eth, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), divisor).Float64()
	return eth
}

func normaliseBalance(raw string, decimals int) float64 {
	if raw == "" || raw == "0" {
		return 0
	}
	val := new(big.Int)
	val.SetString(raw, 10)
	if val.Sign() == 0 {
		return 0
	}
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result, _ := new(big.Float).Quo(new(big.Float).SetInt(val), divisor).Float64()
	return result
}
