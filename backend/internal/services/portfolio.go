// Package services contains the business logic for the portfolio tracker.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/cache"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/models"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/repositories"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/validation"
)

const (
	balanceCacheTTL     = 2 * time.Minute
	portfolioCacheTTL   = 2 * time.Minute
	transactionCacheTTL = 5 * time.Minute
)

// PortfolioService orchestrates data fetching from providers and persists
// results to the database.
type PortfolioService struct {
	blockchainProvider providers.BlockchainProvider
	priceService       *PriceService
	balanceRepo        *repositories.BalanceRepository
	txRepo             *repositories.TransactionRepository
	cache              *cache.Cache
	isDemo             bool
	log                *slog.Logger
}

// NewPortfolioService returns a new PortfolioService.
func NewPortfolioService(
	blockchainProvider providers.BlockchainProvider,
	priceService *PriceService,
	balanceRepo *repositories.BalanceRepository,
	txRepo *repositories.TransactionRepository,
	cache *cache.Cache,
	isDemo bool,
	log *slog.Logger,
) *PortfolioService {
	return &PortfolioService{
		blockchainProvider: blockchainProvider,
		priceService:       priceService,
		balanceRepo:        balanceRepo,
		txRepo:             txRepo,
		cache:              cache,
		isDemo:             isDemo,
		log:                log.With("service", "portfolio"),
	}
}

// GetPortfolio returns the full portfolio summary for the given wallet address.
func (s *PortfolioService) GetPortfolio(ctx context.Context, address string) (*models.PortfolioSummary, error) {
	if err := validation.ValidateAddress(address); err != nil {
		return nil, err
	}
	address = validation.NormalizeAddress(address)

	cacheKey := "portfolio:" + address
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*models.PortfolioSummary), nil
	}

	// Fetch native ETH balance.
	nativeBal, err := s.blockchainProvider.GetNativeBalance(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("get native balance: %w", err)
	}

	// Persist native balance.
	if s.balanceRepo != nil {
		_ = s.balanceRepo.Upsert(ctx, models.Balance{
			WalletAddress: address,
			TokenAddress:  "",
			TokenSymbol:   "ETH",
			TokenName:     "Ether",
			Decimals:      18,
			RawBalance:    nativeBal.WeiBalance,
			NormalBalance: nativeBal.ETHBalance,
			FetchedAt:     time.Now(),
		})
	}

	// Fetch ERC-20 token balances.
	tokenBals, err := s.blockchainProvider.GetTokenBalances(ctx, address)
	if err != nil {
		s.log.Warn("failed to fetch token balances", "err", err)
		// Partial failure: continue with empty token list.
		tokenBals = nil
	}

	// Persist token balances.
	if s.balanceRepo != nil {
		for _, tb := range tokenBals {
			_ = s.balanceRepo.Upsert(ctx, models.Balance{
				WalletAddress: address,
				TokenAddress:  tb.TokenAddress,
				TokenSymbol:   tb.Symbol,
				TokenName:     tb.Name,
				Decimals:      tb.Decimals,
				RawBalance:    tb.RawBalance,
				NormalBalance: tb.Balance,
				FetchedAt:     time.Now(),
			})
		}
	}

	// Price the native balance.
	ethPrice, ethSource := s.priceService.GetPriceWithFallback(ctx, "ETH")
	nativeUSD := nativeBal.ETHBalance * ethPrice
	totalUSD := nativeUSD

	// Build token holdings with prices.
	holdings := make([]models.TokenHolding, 0, len(tokenBals))
	for _, tb := range tokenBals {
		price, priceSource := s.priceService.GetPriceWithFallback(ctx, tb.Symbol)
		valueUSD := tb.Balance * price
		totalUSD += valueUSD

		holdings = append(holdings, models.TokenHolding{
			TokenAddress: tb.TokenAddress,
			Symbol:       tb.Symbol,
			Name:         tb.Name,
			Quantity:     tb.Balance,
			PriceUSD:     price,
			ValueUSD:     valueUSD,
			PriceSource:  priceSource,
		})

		// Persist price snapshot.
		if s.balanceRepo != nil {
			_ = s.balanceRepo.SavePriceSnapshot(ctx, tb.Symbol, price, priceSource)
		}
	}

	_ = ethSource // suppress unused warning; source is used implicitly

	summary := &models.PortfolioSummary{
		Address:       address,
		NativeBalance: nativeBal.ETHBalance,
		NativeUSD:     nativeUSD,
		Tokens:        holdings,
		TotalUSD:      totalUSD,
		FetchedAt:     time.Now(),
		IsDemo:        s.isDemo,
	}

	s.cache.Set(cacheKey, summary, portfolioCacheTTL)
	return summary, nil
}

// GetNativeBalance returns only the ETH balance for an address.
func (s *PortfolioService) GetNativeBalance(ctx context.Context, address string) (*models.Balance, error) {
	if err := validation.ValidateAddress(address); err != nil {
		return nil, err
	}
	address = validation.NormalizeAddress(address)

	cacheKey := "native:" + address
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*models.Balance), nil
	}

	nb, err := s.blockchainProvider.GetNativeBalance(ctx, address)
	if err != nil {
		return nil, err
	}

	bal := &models.Balance{
		WalletAddress: address,
		TokenAddress:  "",
		TokenSymbol:   "ETH",
		TokenName:     "Ether",
		Decimals:      18,
		RawBalance:    nb.WeiBalance,
		NormalBalance: nb.ETHBalance,
		FetchedAt:     time.Now(),
	}

	s.cache.Set(cacheKey, bal, balanceCacheTTL)
	return bal, nil
}

// GetTokenBalances returns ERC-20 holdings for an address.
func (s *PortfolioService) GetTokenBalances(ctx context.Context, address string) ([]models.Balance, error) {
	if err := validation.ValidateAddress(address); err != nil {
		return nil, err
	}
	address = validation.NormalizeAddress(address)

	cacheKey := "tokens:" + address
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]models.Balance), nil
	}

	tokenBals, err := s.blockchainProvider.GetTokenBalances(ctx, address)
	if err != nil {
		return nil, err
	}

	balances := make([]models.Balance, 0, len(tokenBals))
	for _, tb := range tokenBals {
		balances = append(balances, models.Balance{
			WalletAddress: address,
			TokenAddress:  tb.TokenAddress,
			TokenSymbol:   tb.Symbol,
			TokenName:     tb.Name,
			Decimals:      tb.Decimals,
			RawBalance:    tb.RawBalance,
			NormalBalance: tb.Balance,
			FetchedAt:     time.Now(),
		})
	}

	s.cache.Set(cacheKey, balances, balanceCacheTTL)
	return balances, nil
}

// GetTransactions returns a paginated page of transactions.
func (s *PortfolioService) GetTransactions(ctx context.Context, address string, page, perPage int) (*models.TransactionPage, error) {
	if err := validation.ValidateAddress(address); err != nil {
		return nil, err
	}
	address = validation.NormalizeAddress(address)
	page, perPage = validation.ValidatePaginationParams(page, perPage)

	cacheKey := fmt.Sprintf("txns:%s:%d:%d", address, page, perPage)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*models.TransactionPage), nil
	}

	rawTxns, err := s.blockchainProvider.GetTransactions(ctx, address, page, perPage)
	if err != nil {
		return nil, err
	}

	// Persist to DB (best-effort; non-nil repo only).
	dbTxns := make([]models.Transaction, 0, len(rawTxns))
	for _, rt := range rawTxns {
		dbTxns = append(dbTxns, models.Transaction{
			Hash:          rt.Hash,
			WalletAddress: address,
			BlockNumber:   rt.BlockNumber,
			Timestamp:     rt.Timestamp,
			From:          rt.From,
			To:            rt.To,
			Value:         rt.Value,
			NormalValue:   rt.NormalValue,
			Gas:           rt.Gas,
			GasUsed:       rt.GasUsed,
			GasPrice:      rt.GasPrice,
			TokenAddress:  rt.TokenAddress,
			TokenSymbol:   rt.TokenSymbol,
			TokenValue:    rt.TokenValue,
			TokenDecimals: rt.TokenDecimals,
			TxType:        rt.TxType,
			Status:        rt.Status,
		})
	}
	if s.txRepo != nil {
		_ = s.txRepo.UpsertBatch(ctx, dbTxns)
	}

	result := &models.TransactionPage{
		Transactions: dbTxns,
		Total:        len(dbTxns),
		Page:         page,
		PerPage:      perPage,
		HasNext:      len(dbTxns) == perPage,
	}

	s.cache.Set(cacheKey, result, transactionCacheTTL)
	return result, nil
}

// NormalizeAddress returns a lowercase Ethereum address.
// Exported so handlers can use it for path parameter normalization.
func NormalizeAddress(addr string) string {
	return strings.ToLower(addr)
}
