// Package chainlink provides a minimal read-only client for Chainlink Data
// Feeds deployed on Ethereum. It uses only the Ethereum JSON-RPC API (via
// eth_call) to query the latestRoundData() function of AggregatorV3Interface
// contracts without importing the entire go-ethereum library.
//
// How Chainlink price feeds work (educational summary):
//
//   Blockchains are deterministic state machines — every node must agree on
//   every computation. That means a smart contract cannot make an HTTP request
//   or read a web API: there is no way for all nodes to independently verify
//   the result. This is the "oracle problem".
//
//   Chainlink solves it with a network of independent node operators that each
//   fetch price data off-chain, sign it, and submit it on-chain. A reference
//   contract aggregates those submissions (using a median) into a single
//   authoritative price that any smart contract can read.
//
//   This package queries that aggregator contract directly via the standard
//   Ethereum JSON-RPC eth_call, decoding the ABI response by hand. No
//   additional libraries are required.
//
//   Decimals: Chainlink feeds return prices as integers scaled by 10^decimals.
//   The ETH/USD feed uses 8 decimal places, so a returned value of 200000000000
//   represents $2,000.00000000. The Decimals() method (selector 0x313ce567)
//   on the feed contract tells you this at runtime.
//
//   Reliability risks: feeds can go stale if Chainlink node operators fail to
//   update them within the heartbeat interval (e.g. 1 hour for ETH/USD). The
//   updatedAt field in latestRoundData should always be checked — if it is
//   more than a few hours old, treat the price as potentially unreliable.
package chainlink

import (
	"bytes"
	"context"
	"encoding/hex"
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

const (
	moduleName = "chainlink"

	// stalePriceThreshold is the maximum age we accept for a price feed update.
	// If updatedAt is older than this we log a warning and return a stale error.
	stalePriceThreshold = 2 * time.Hour

	// ABI 4-byte function selectors (keccak256 of signature, first 4 bytes).
	// These are constant for AggregatorV3Interface-compatible contracts.
	selectorLatestRoundData = "feaf968c" // latestRoundData()
	selectorDecimals        = "313ce567" // decimals()
)

// ErrStalePrice is returned when the feed has not been updated recently.
type ErrStalePrice struct {
	Symbol    string
	UpdatedAt time.Time
}

func (e ErrStalePrice) Error() string {
	return fmt.Sprintf("chainlink %s price feed is stale: last updated %s ago",
		e.Symbol, time.Since(e.UpdatedAt).Round(time.Minute))
}

// feedConfig maps a token symbol to its Chainlink feed contract address.
type feedConfig struct {
	symbol  string
	address string // Ethereum mainnet aggregator address
}

// Provider reads price data from Chainlink AggregatorV3 contracts via
// standard Ethereum JSON-RPC.
type Provider struct {
	rpcURL string
	feeds  map[string]feedConfig
	client *http.Client
	log    *slog.Logger
}

// New returns a Chainlink provider that calls rpcURL (e.g. your Alchemy
// endpoint) to read the feed contracts at the given addresses.
func New(rpcURL string, feedAddresses map[string]string, timeout time.Duration, log *slog.Logger) *Provider {
	feeds := make(map[string]feedConfig, len(feedAddresses))
	for symbol, addr := range feedAddresses {
		feeds[strings.ToUpper(symbol)] = feedConfig{symbol: symbol, address: addr}
	}
	return &Provider{
		rpcURL: rpcURL,
		feeds:  feeds,
		client: &http.Client{Timeout: timeout},
		log:    log.With("provider", moduleName),
	}
}

func (p *Provider) Name() string { return moduleName }

// GetPrice reads the current price for symbol from the configured Chainlink
// feed. Returns ErrStalePrice if the feed is older than stalePriceThreshold.
func (p *Provider) GetPrice(ctx context.Context, symbol string) (*providers.PriceResult, error) {
	feed, ok := p.feeds[strings.ToUpper(symbol)]
	if !ok {
		return nil, fmt.Errorf("no chainlink feed configured for %s", symbol)
	}

	// Fetch decimals first so we can normalise the price.
	decimals, err := p.readDecimals(ctx, feed.address)
	if err != nil {
		p.log.Warn("failed to read feed decimals, assuming 8", "symbol", symbol, "err", err)
		decimals = 8
	}

	// latestRoundData() returns (roundId, answer, startedAt, updatedAt, answeredInRound)
	round, err := p.readLatestRoundData(ctx, feed.address)
	if err != nil {
		return nil, fmt.Errorf("latestRoundData for %s: %w", symbol, err)
	}

	// Check for stale data.
	if time.Since(round.updatedAt) > stalePriceThreshold {
		p.log.Warn("chainlink price feed is stale",
			"symbol", symbol,
			"updatedAt", round.updatedAt,
			"age", time.Since(round.updatedAt).Round(time.Minute),
		)
		return nil, ErrStalePrice{Symbol: symbol, UpdatedAt: round.updatedAt}
	}

	// Normalise: price = answer / 10^decimals
	divisor := new(big.Float).SetInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil),
	)
	priceFloat, _ := new(big.Float).Quo(new(big.Float).SetInt(round.answer), divisor).Float64()

	p.log.Info("chainlink price fetched",
		"symbol", symbol,
		"price_usd", priceFloat,
		"feed", feed.address,
		"updated_at", round.updatedAt,
	)

	return &providers.PriceResult{
		Symbol:    symbol,
		PriceUSD:  priceFloat,
		Source:    moduleName,
		FetchedAt: time.Now(),
	}, nil
}

// — ABI encoding / decoding ---------------------------------------------------

// roundData holds the decoded output of latestRoundData().
type roundData struct {
	roundID         *big.Int
	answer          *big.Int // price * 10^decimals
	startedAt       time.Time
	updatedAt       time.Time
	answeredInRound *big.Int
}

// readLatestRoundData calls latestRoundData() and decodes the ABI tuple.
func (p *Provider) readLatestRoundData(ctx context.Context, contractAddr string) (*roundData, error) {
	// eth_call data: 4-byte selector only (no parameters).
	data := "0x" + selectorLatestRoundData

	result, err := p.ethCall(ctx, contractAddr, data)
	if err != nil {
		return nil, err
	}

	// ABI-decode the returned bytes. latestRoundData returns a tuple of
	// five 32-byte words: (uint80, int256, uint256, uint256, uint80).
	words, err := decodeABIWords(result, 5)
	if err != nil {
		return nil, fmt.Errorf("decode latestRoundData: %w", err)
	}

	rd := &roundData{
		roundID:         words[0],
		answer:          words[1],
		startedAt:       time.Unix(words[2].Int64(), 0).UTC(),
		updatedAt:       time.Unix(words[3].Int64(), 0).UTC(),
		answeredInRound: words[4],
	}
	return rd, nil
}

// readDecimals calls decimals() and returns the result as an int.
func (p *Provider) readDecimals(ctx context.Context, contractAddr string) (int, error) {
	data := "0x" + selectorDecimals
	result, err := p.ethCall(ctx, contractAddr, data)
	if err != nil {
		return 0, err
	}
	words, err := decodeABIWords(result, 1)
	if err != nil {
		return 0, fmt.Errorf("decode decimals: %w", err)
	}
	return int(words[0].Int64()), nil
}

// decodeABIWords strips the 0x prefix, then reads n successive 32-byte
// big-endian integers from the hex string.
func decodeABIWords(hexData string, n int) ([]*big.Int, error) {
	cleaned := strings.TrimPrefix(hexData, "0x")
	if cleaned == "" {
		return nil, fmt.Errorf("empty response")
	}

	data, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}

	if len(data) < n*32 {
		return nil, fmt.Errorf("response too short: got %d bytes, need %d", len(data), n*32)
	}

	words := make([]*big.Int, n)
	for i := 0; i < n; i++ {
		words[i] = new(big.Int).SetBytes(data[i*32 : (i+1)*32])
	}
	return words, nil
}

// — JSON-RPC helper -----------------------------------------------------------

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type jsonRPCResponse struct {
	Result string    `json:"result"`
	Error  *rpcError `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ethCall sends an eth_call RPC request to the configured node and returns
// the raw hex result.
func (p *Provider) ethCall(ctx context.Context, to, data string) (string, error) {
	if p.rpcURL == "" {
		return "", fmt.Errorf("ethereum RPC URL not configured")
	}

	payload := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{"to": to, "data": data},
			"latest",
		},
		ID: 1,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.rpcURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rpc request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read rpc response: %w", err)
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return "", fmt.Errorf("parse rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return "", fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}
