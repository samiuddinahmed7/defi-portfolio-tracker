// Package alchemy wraps the Alchemy API for wallet data. Alchemy's API
// surface for wallet balances is richer than Etherscan's — it can return
// all token balances in a single call via alchemy_getTokenBalances.
package alchemy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
)

const moduleName = "alchemy"

// Provider fetches on-chain data via the Alchemy JSON-RPC and REST APIs.
type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	log     *slog.Logger
}

// New returns a new Alchemy provider targeting Ethereum mainnet.
func New(apiKey string, timeout time.Duration, log *slog.Logger) *Provider {
	baseURL := fmt.Sprintf("https://eth-mainnet.g.alchemy.com/v2/%s", apiKey)
	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log.With("provider", moduleName),
	}
}

func (p *Provider) Name() string { return moduleName }

// — JSON-RPC helpers ----------------------------------------------------------

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (p *Provider) rpc(ctx context.Context, method string, params []interface{}) (*rpcResponse, error) {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alchemy request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("parse rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("alchemy rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return &rpcResp, nil
}

// — BlockchainProvider implementation -----------------------------------------

// GetNativeBalance returns the ETH balance using eth_getBalance.
func (p *Provider) GetNativeBalance(ctx context.Context, address string) (*providers.NativeBalance, error) {
	resp, err := p.rpc(ctx, "eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return nil, fmt.Errorf("eth_getBalance: %w", err)
	}

	var hexBal string
	if err := json.Unmarshal(resp.Result, &hexBal); err != nil {
		return nil, fmt.Errorf("parse balance: %w", err)
	}

	wei := hexToBigInt(hexBal)
	weiStr := wei.String()
	eth := weiToFloat(wei)

	return &providers.NativeBalance{
		Address:    address,
		WeiBalance: weiStr,
		ETHBalance: eth,
	}, nil
}

// alchemyTokenBalancesResult is the shape returned by alchemy_getTokenBalances.
type alchemyTokenBalancesResult struct {
	Address       string `json:"address"`
	TokenBalances []struct {
		ContractAddress string `json:"contractAddress"`
		TokenBalance    string `json:"tokenBalance"` // hex string
		Error           string `json:"error,omitempty"`
	} `json:"tokenBalances"`
}

// alchemyTokenMetaResult is the shape returned by alchemy_getTokenMetadata.
type alchemyTokenMetaResult struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Logo     string `json:"logo"`
}

// GetTokenBalances uses alchemy_getTokenBalances to return all ERC-20 holdings.
func (p *Provider) GetTokenBalances(ctx context.Context, address string) ([]providers.TokenBalance, error) {
	resp, err := p.rpc(ctx, "alchemy_getTokenBalances", []interface{}{address, "erc20"})
	if err != nil {
		return nil, fmt.Errorf("alchemy_getTokenBalances: %w", err)
	}

	var result alchemyTokenBalancesResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse token balances: %w", err)
	}

	var holdings []providers.TokenBalance
	for _, tb := range result.TokenBalances {
		if tb.Error != "" {
			p.log.Warn("token balance error", "contract", tb.ContractAddress, "err", tb.Error)
			continue
		}

		rawWei := hexToBigInt(tb.TokenBalance)
		if rawWei.Sign() == 0 {
			continue // skip zero balances
		}

		// Fetch token metadata for symbol/decimals.
		meta, err := p.fetchTokenMeta(ctx, tb.ContractAddress)
		if err != nil {
			p.log.Warn("failed to fetch token metadata", "contract", tb.ContractAddress, "err", err)
			// Include with unknown metadata rather than dropping entirely.
			meta = &alchemyTokenMetaResult{Symbol: "?", Name: "Unknown", Decimals: 18}
		}

		decimals := meta.Decimals
		if decimals == 0 {
			decimals = 18
		}
		normalised := normaliseBigInt(rawWei, decimals)

		holdings = append(holdings, providers.TokenBalance{
			TokenAddress: strings.ToLower(tb.ContractAddress),
			Symbol:       meta.Symbol,
			Name:         meta.Name,
			Decimals:     decimals,
			RawBalance:   rawWei.String(),
			Balance:      normalised,
		})
	}
	return holdings, nil
}

func (p *Provider) fetchTokenMeta(ctx context.Context, contractAddress string) (*alchemyTokenMetaResult, error) {
	resp, err := p.rpc(ctx, "alchemy_getTokenMetadata", []interface{}{contractAddress})
	if err != nil {
		return nil, err
	}
	var meta alchemyTokenMetaResult
	if err := json.Unmarshal(resp.Result, &meta); err != nil {
		return nil, fmt.Errorf("parse token metadata: %w", err)
	}
	return &meta, nil
}

// alchemyTransfer is one entry in an alchemy_getAssetTransfers response.
type alchemyTransfer struct {
	Hash      string `json:"hash"`
	BlockNum  string `json:"blockNum"`
	From      string `json:"from"`
	To        string `json:"to"`
	Value     *float64 `json:"value"`
	Asset     string `json:"asset"`
	Category  string `json:"category"`
	RawValue  struct {
		Hex string `json:"hex"`
	} `json:"rawContract"`
}

// GetTransactions uses alchemy_getAssetTransfers which covers both ETH and
// token transfers in a unified stream.
func (p *Provider) GetTransactions(ctx context.Context, address string, page, perPage int) ([]providers.RawTransaction, error) {
	params := map[string]interface{}{
		"fromAddress": address,
		"category":    []string{"external", "erc20"},
		"maxCount":    fmt.Sprintf("0x%x", perPage),
		"order":       "desc",
	}

	resp, err := p.rpc(ctx, "alchemy_getAssetTransfers", []interface{}{params})
	if err != nil {
		return nil, fmt.Errorf("alchemy_getAssetTransfers: %w", err)
	}

	var result struct {
		Transfers []alchemyTransfer `json:"transfers"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse transfers: %w", err)
	}

	txns := make([]providers.RawTransaction, 0, len(result.Transfers))
	for _, t := range result.Transfers {
		tx := providers.RawTransaction{
			Hash:      t.Hash,
			From:      strings.ToLower(t.From),
			To:        strings.ToLower(t.To),
			Timestamp: time.Now(), // Alchemy transfers don't include a timestamp; callers should enrich
			TxType:    "send",
			Status:    "success",
		}
		if t.Value != nil {
			tx.NormalValue = *t.Value
		}
		if t.Category == "erc20" {
			tx.TxType = "token_transfer"
			tx.TokenSymbol = t.Asset
		}
		txns = append(txns, tx)
	}
	return txns, nil
}

// — Math helpers --------------------------------------------------------------

func hexToBigInt(hex string) *big.Int {
	hex = strings.TrimPrefix(hex, "0x")
	hex = strings.TrimPrefix(hex, "0X")
	i := new(big.Int)
	i.SetString(hex, 16)
	return i
}

func weiToFloat(wei *big.Int) float64 {
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(wei), divisor).Float64()
	return f
}

func normaliseBigInt(val *big.Int, decimals int) float64 {
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	f, _ := new(big.Float).Quo(new(big.Float).SetInt(val), divisor).Float64()
	return f
}
