// Package etherscan wraps the Etherscan v2 API for wallet balance and
// transaction data. It uses only the standard library so there are no
// external HTTP-client dependencies.
package etherscan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
)

const (
	baseURL    = "https://api.etherscan.io/api"
	moduleName = "etherscan"
)

// Provider fetches on-chain data from the Etherscan API.
type Provider struct {
	apiKey  string
	client  *http.Client
	log     *slog.Logger
}

// New returns a new Etherscan provider. If apiKey is empty the provider will
// still work but Etherscan will rate-limit aggressively.
func New(apiKey string, timeout time.Duration, log *slog.Logger) *Provider {
	return &Provider{
		apiKey: apiKey,
		client: &http.Client{Timeout: timeout},
		log:    log.With("provider", moduleName),
	}
}

func (p *Provider) Name() string { return moduleName }

// — API response shapes -------------------------------------------------------

type apiResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type tokenBalanceResult struct {
	TokenName     string `json:"tokenName"`
	TokenSymbol   string `json:"tokenSymbol"`
	TokenDecimal  string `json:"tokenDecimal"`
	ContractAddr  string `json:"contractAddress"`
	TokenQuantity string `json:"tokenQuantity"`
}

type txResult struct {
	Hash              string `json:"hash"`
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasUsed           string `json:"gasUsed"`
	GasPrice          string `json:"gasPrice"`
	IsError           string `json:"isError"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	ContractAddress   string `json:"contractAddress"`
	TokenValue        string `json:"tokenValue"` // for ERC-20 txns
}

// — BlockchainProvider implementation -----------------------------------------

// GetNativeBalance returns the ETH balance for address.
func (p *Provider) GetNativeBalance(ctx context.Context, address string) (*providers.NativeBalance, error) {
	params := url.Values{
		"module":  {"account"},
		"action":  {"balance"},
		"address": {address},
		"tag":     {"latest"},
	}
	resp, err := p.get(ctx, params)
	if err != nil {
		return nil, err
	}

	var weiStr string
	if err := json.Unmarshal(resp.Result, &weiStr); err != nil {
		return nil, fmt.Errorf("parse balance result: %w", err)
	}

	eth := weiToETH(weiStr)
	return &providers.NativeBalance{
		Address:    address,
		WeiBalance: weiStr,
		ETHBalance: eth,
	}, nil
}

// GetTokenBalances returns ERC-20 holdings for address via Etherscan's
// tokenbalancehistory endpoint (we use tokentx to discover tokens).
func (p *Provider) GetTokenBalances(ctx context.Context, address string) ([]providers.TokenBalance, error) {
	// Etherscan doesn't have a single "all token balances" endpoint, so we
	// enumerate tokens the wallet has interacted with via token transfers, then
	// query each balance. To keep API usage minimal we use tokenbalance.
	params := url.Values{
		"module":  {"account"},
		"action":  {"tokentx"},
		"address": {address},
		"sort":    {"desc"},
		"page":    {"1"},
		"offset":  {"100"},
	}
	resp, err := p.get(ctx, params)
	if err != nil {
		return nil, err
	}

	// The result may be "No transactions found" string when empty.
	if string(resp.Result) == `"No transactions found"` || string(resp.Result) == `[]` {
		return nil, nil
	}

	var txns []txResult
	if err := json.Unmarshal(resp.Result, &txns); err != nil {
		return nil, fmt.Errorf("parse token txns: %w", err)
	}

	// Build a deduplicated set of contract addresses from the transfers.
	seen := make(map[string]struct{})
	var contracts []string
	for _, tx := range txns {
		ca := strings.ToLower(tx.ContractAddress)
		if ca == "" {
			continue
		}
		if _, ok := seen[ca]; !ok {
			seen[ca] = struct{}{}
			contracts = append(contracts, ca)
		}
	}

	// Fetch current balance for each unique contract.
	var holdings []providers.TokenBalance
	for _, ca := range contracts {
		bal, err := p.fetchTokenBalance(ctx, address, ca, txns)
		if err != nil {
			p.log.Warn("failed to fetch token balance", "contract", ca, "err", err)
			continue
		}
		if bal != nil && bal.Balance > 0 {
			holdings = append(holdings, *bal)
		}
	}
	return holdings, nil
}

func (p *Provider) fetchTokenBalance(ctx context.Context, wallet, contract string, txns []txResult) (*providers.TokenBalance, error) {
	params := url.Values{
		"module":          {"account"},
		"action":          {"tokenbalance"},
		"address":         {wallet},
		"contractaddress": {contract},
		"tag":             {"latest"},
	}
	resp, err := p.get(ctx, params)
	if err != nil {
		return nil, err
	}

	var rawBal string
	if err := json.Unmarshal(resp.Result, &rawBal); err != nil {
		return nil, fmt.Errorf("parse tokenbalance: %w", err)
	}

	// Find metadata from the transaction history.
	var symbol, name string
	var decimals int
	for _, tx := range txns {
		if strings.EqualFold(tx.ContractAddress, contract) {
			symbol = tx.TokenSymbol
			name = tx.TokenName
			d, _ := strconv.Atoi(tx.TokenDecimal)
			decimals = d
			break
		}
	}
	if decimals == 0 {
		decimals = 18
	}

	normalised := normaliseBalance(rawBal, decimals)
	return &providers.TokenBalance{
		TokenAddress: contract,
		Symbol:       symbol,
		Name:         name,
		Decimals:     decimals,
		RawBalance:   rawBal,
		Balance:      normalised,
	}, nil
}

// GetTransactions returns a page of ETH and token transactions for address.
func (p *Provider) GetTransactions(ctx context.Context, address string, page, perPage int) ([]providers.RawTransaction, error) {
	var all []providers.RawTransaction

	// Normal ETH transactions.
	ethTxns, err := p.fetchNormalTxns(ctx, address, page, perPage)
	if err != nil {
		p.log.Warn("etherscan normal txns error", "err", err)
	} else {
		all = append(all, ethTxns...)
	}

	// ERC-20 token transfers.
	tokenTxns, err := p.fetchTokenTxns(ctx, address, page, perPage)
	if err != nil {
		p.log.Warn("etherscan token txns error", "err", err)
	} else {
		all = append(all, tokenTxns...)
	}

	return all, nil
}

func (p *Provider) fetchNormalTxns(ctx context.Context, address string, page, perPage int) ([]providers.RawTransaction, error) {
	params := url.Values{
		"module":  {"account"},
		"action":  {"txlist"},
		"address": {address},
		"sort":    {"desc"},
		"page":    {strconv.Itoa(page)},
		"offset":  {strconv.Itoa(perPage)},
	}
	resp, err := p.get(ctx, params)
	if err != nil {
		return nil, err
	}
	if string(resp.Result) == `"No transactions found"` || string(resp.Result) == `[]` {
		return nil, nil
	}

	var raw []txResult
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("parse txlist: %w", err)
	}

	txns := make([]providers.RawTransaction, 0, len(raw))
	for _, r := range raw {
		tx := parseTx(r, address)
		tx.TxType = classifyTx(r, address)
		txns = append(txns, tx)
	}
	return txns, nil
}

func (p *Provider) fetchTokenTxns(ctx context.Context, address string, page, perPage int) ([]providers.RawTransaction, error) {
	params := url.Values{
		"module":  {"account"},
		"action":  {"tokentx"},
		"address": {address},
		"sort":    {"desc"},
		"page":    {strconv.Itoa(page)},
		"offset":  {strconv.Itoa(perPage)},
	}
	resp, err := p.get(ctx, params)
	if err != nil {
		return nil, err
	}
	if string(resp.Result) == `"No transactions found"` || string(resp.Result) == `[]` {
		return nil, nil
	}

	var raw []txResult
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("parse tokentx: %w", err)
	}

	txns := make([]providers.RawTransaction, 0, len(raw))
	for _, r := range raw {
		tx := parseTx(r, address)
		tx.TxType = "token_transfer"
		tx.TokenAddress = strings.ToLower(r.ContractAddress)
		tx.TokenSymbol = r.TokenSymbol
		tx.TokenValue = r.Value
		d, _ := strconv.Atoi(r.TokenDecimal)
		tx.TokenDecimals = d
		txns = append(txns, tx)
	}
	return txns, nil
}

// — HTTP helper ----------------------------------------------------------------

func (p *Provider) get(ctx context.Context, params url.Values) (*apiResponse, error) {
	if p.apiKey != "" {
		params.Set("apikey", p.apiKey)
	}
	u := baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpResp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("etherscan request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var ar apiResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if ar.Status == "0" && ar.Message != "No transactions found" {
		return nil, fmt.Errorf("etherscan error: %s", ar.Message)
	}

	return &ar, nil
}

// — helpers -------------------------------------------------------------------

func parseTx(r txResult, walletAddress string) providers.RawTransaction {
	blockNum, _ := strconv.ParseInt(r.BlockNumber, 10, 64)
	tsEpoch, _ := strconv.ParseInt(r.TimeStamp, 10, 64)
	gas, _ := strconv.ParseInt(r.Gas, 10, 64)
	gasUsed, _ := strconv.ParseInt(r.GasUsed, 10, 64)

	status := "success"
	if r.IsError == "1" {
		status = "failed"
	}

	return providers.RawTransaction{
		Hash:        r.Hash,
		BlockNumber: blockNum,
		Timestamp:   time.Unix(tsEpoch, 0).UTC(),
		From:        strings.ToLower(r.From),
		To:          strings.ToLower(r.To),
		Value:       r.Value,
		NormalValue: weiToETH(r.Value),
		Gas:         gas,
		GasUsed:     gasUsed,
		GasPrice:    r.GasPrice,
		Status:      status,
	}
}

func classifyTx(r txResult, walletAddress string) string {
	from := strings.ToLower(r.From)
	wallet := strings.ToLower(walletAddress)
	if from == wallet {
		return "send"
	}
	return "receive"
}

// weiToETH converts a wei string (decimal) to a float64 ETH value.
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

// normaliseBalance divides a raw token balance by 10^decimals.
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
