package tests

import (
	"context"
	"testing"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers/demo"
)

// TestDemoProvider verifies that the demo provider returns consistent fixture data.
func TestDemoProvider(t *testing.T) {
	ctx := context.Background()
	p := demo.New()

	if p.Name() != "demo" {
		t.Errorf("provider name = %q, want %q", p.Name(), "demo")
	}

	t.Run("native balance", func(t *testing.T) {
		bal, err := p.GetNativeBalance(ctx, "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bal.ETHBalance <= 0 {
			t.Errorf("expected positive ETH balance, got %f", bal.ETHBalance)
		}
		if bal.WeiBalance == "" {
			t.Error("expected non-empty WeiBalance")
		}
	})

	t.Run("token balances", func(t *testing.T) {
		tokens, err := p.GetTokenBalances(ctx, "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) == 0 {
			t.Error("expected at least one token balance")
		}
		for _, tok := range tokens {
			if tok.Symbol == "" {
				t.Error("token symbol should not be empty")
			}
			if tok.Decimals == 0 {
				t.Errorf("token %s has zero decimals — likely missing metadata", tok.Symbol)
			}
			if tok.Balance <= 0 {
				t.Errorf("token %s has non-positive balance %f", tok.Symbol, tok.Balance)
			}
		}
	})

	t.Run("transactions", func(t *testing.T) {
		txns, err := p.GetTransactions(ctx, "0xde0b295669a9fd93d5f28d9ec85e40f4cb697bae", 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(txns) == 0 {
			t.Error("expected at least one transaction")
		}
		for _, tx := range txns {
			if tx.Hash == "" {
				t.Error("transaction hash should not be empty")
			}
			if tx.TxType == "" {
				t.Error("transaction type should not be empty")
			}
		}
	})
}

// TestDemoPriceProvider checks that demo prices are returned for common symbols.
func TestDemoPriceProvider(t *testing.T) {
	ctx := context.Background()
	p := demo.NewPriceProvider()

	symbols := []string{"ETH", "USDC", "LINK", "DAI"}
	for _, sym := range symbols {
		result, err := p.GetPrice(ctx, sym)
		if err != nil {
			t.Errorf("GetPrice(%s) error: %v", sym, err)
			continue
		}
		if result.PriceUSD < 0 {
			t.Errorf("GetPrice(%s) returned negative price %f", sym, result.PriceUSD)
		}
		if result.Source == "" {
			t.Errorf("GetPrice(%s) returned empty source", sym)
		}
	}
}
