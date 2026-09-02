package services

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/cache"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/models"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
)

const priceCacheTTL = 5 * time.Minute

// PriceService fetches token prices from a chain of providers, falling back
// gracefully when upstream sources are unavailable.
//
// Priority order:
//   1. Chainlink on-chain feed (most trustless — price comes from the chain itself)
//   2. In-memory cache (avoid redundant API calls within the TTL window)
//   3. Demo/fallback prices when no live source succeeds
type PriceService struct {
	chainlinkProvider providers.PriceProvider // may be nil if RPC URL not configured
	fallbackProvider  providers.PriceProvider // demo or static prices
	cache             *cache.Cache
	log               *slog.Logger
}

// NewPriceService returns a PriceService with optional Chainlink integration.
// chainlinkProvider may be nil; fallback will be used in that case.
func NewPriceService(
	chainlinkProvider providers.PriceProvider,
	fallbackProvider providers.PriceProvider,
	cache *cache.Cache,
	log *slog.Logger,
) *PriceService {
	return &PriceService{
		chainlinkProvider: chainlinkProvider,
		fallbackProvider:  fallbackProvider,
		cache:             cache,
		log:               log.With("service", "price"),
	}
}

// GetPrice returns the best available price for symbol.
func (s *PriceService) GetPrice(ctx context.Context, symbol string) (*models.PriceResponse, error) {
	price, source := s.GetPriceWithFallback(ctx, symbol)
	return &models.PriceResponse{
		Symbol:    symbol,
		PriceUSD:  price,
		Source:    source,
		FetchedAt: time.Now(),
	}, nil
}

// GetPriceWithFallback returns a price and its source, never returning an error
// so callers can always continue with a best-effort price. This is a deliberate
// design choice: a missing price should not prevent the rest of the portfolio
// from rendering.
func (s *PriceService) GetPriceWithFallback(ctx context.Context, symbol string) (float64, string) {
	symbol = strings.ToUpper(symbol)

	// 1. Check cache.
	cacheKey := "price:" + symbol
	if cached, ok := s.cache.Get(cacheKey); ok {
		pr := cached.(*providers.PriceResult)
		return pr.PriceUSD, pr.Source
	}

	// 2. Try Chainlink (on-chain oracle — most reliable source for supported feeds).
	if s.chainlinkProvider != nil {
		result, err := s.chainlinkProvider.GetPrice(ctx, symbol)
		if err == nil {
			s.cache.Set(cacheKey, result, priceCacheTTL)
			return result.PriceUSD, result.Source
		}
		s.log.Warn("chainlink price unavailable, falling back",
			"symbol", symbol, "err", err)
	}

	// 3. Fallback (demo prices or static table).
	if s.fallbackProvider != nil {
		result, err := s.fallbackProvider.GetPrice(ctx, symbol)
		if err == nil {
			s.cache.Set(cacheKey, result, priceCacheTTL)
			return result.PriceUSD, result.Source
		}
	}

	// Last resort: return 0 with an unknown source so the caller can decide
	// how to display the missing price.
	s.log.Warn("no price available for symbol", "symbol", symbol)
	return 0, "unavailable"
}
