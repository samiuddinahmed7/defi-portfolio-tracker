// Command server starts the DeFi portfolio tracker API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/cache"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/config"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/handlers"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/middleware"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers/alchemy"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers/chainlink"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers/demo"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/providers/etherscan"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/repositories"
	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/services"
)

func main() {
	// Load .env if present (ignored in production where env vars are injected).
	_ = godotenv.Load()

	log := newLogger()

	cfg, err := config.Load()
	if err != nil {
		log.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	log.Info("starting defi portfolio tracker",
		"port", cfg.Port,
		"demo_mode", cfg.DemoMode,
		"log_level", cfg.LogLevel,
	)

	// — Blockchain data provider -------------------------------------------
	// When DEMO_MODE is enabled we use fixture data; otherwise we try Alchemy
	// first (richer API), then fall back to Etherscan for any missing data.
	var blockchainProvider providers.BlockchainProvider
	if cfg.DemoMode {
		log.Info("demo mode enabled — using fixture data")
		blockchainProvider = demo.New()
	} else {
		if cfg.AlchemyAPIKey != "" {
			log.Info("using Alchemy as primary blockchain provider")
			blockchainProvider = alchemy.New(cfg.AlchemyAPIKey, cfg.ProviderTimeout, log)
		} else if cfg.EtherscanAPIKey != "" {
			log.Info("using Etherscan as primary blockchain provider")
			blockchainProvider = etherscan.New(cfg.EtherscanAPIKey, cfg.ProviderTimeout, log)
		} else {
			log.Warn("no API keys configured — falling back to demo mode")
			cfg.DemoMode = true
			blockchainProvider = demo.New()
		}
	}

	// — Price provider --------------------------------------------------------
	// Chainlink is used when an RPC URL is available (reads directly from the
	// on-chain aggregator contract). Demo prices are always the fallback.
	var chainlinkProvider providers.PriceProvider
	if cfg.EthereumRPCURL != "" && !cfg.DemoMode {
		log.Info("Chainlink price feeds enabled", "rpc_url", sanitizeURL(cfg.EthereumRPCURL))
		chainlinkProvider = chainlink.New(
			cfg.EthereumRPCURL,
			map[string]string{
				"ETH":  cfg.ChainlinkETHUSDFeed,
				"BTC":  cfg.ChainlinkBTCUSDFeed,
				"LINK": cfg.ChainlinkLINKUSDFeed,
			},
			cfg.ProviderTimeout,
			log,
		)
	} else {
		log.Info("Chainlink disabled (no ETHEREUM_RPC_URL) — using demo prices")
	}
	demoPriceProvider := demo.NewPriceProvider()

	// — Caching ---------------------------------------------------------------
	appCache := cache.New()
	stopEviction := make(chan struct{})
	appCache.StartEviction(10*time.Minute, stopEviction)
	defer close(stopEviction)

	// — Database (optional — skipped in demo mode) ----------------------------
	var balanceRepo *repositories.BalanceRepository
	var txRepo *repositories.TransactionRepository

	if !cfg.DemoMode && cfg.DatabaseURL != "" {
		db, err := repositories.Connect(cfg.DatabaseURL)
		if err != nil {
			log.Error("database connection failed", "err", err)
			os.Exit(1)
		}
		defer db.Close()

		// Run migrations from the migrations directory next to the binary.
		_, filename, _, _ := runtime.Caller(0)
		migrationsDir := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
		if err := repositories.RunMigrations(db, migrationsDir); err != nil {
			log.Error("migrations failed", "err", err)
			os.Exit(1)
		}
		log.Info("database connected and migrations applied")

		balanceRepo = repositories.NewBalanceRepository(db)
		txRepo = repositories.NewTransactionRepository(db)
	}

	// — Services --------------------------------------------------------------
	priceService := services.NewPriceService(chainlinkProvider, demoPriceProvider, appCache, log)
	portfolioService := services.NewPortfolioService(
		blockchainProvider, priceService,
		balanceRepo, txRepo,
		appCache, cfg.DemoMode, log,
	)

	// — HTTP server -----------------------------------------------------------
	mux := http.NewServeMux()
	h := handlers.New(portfolioService, priceService, cfg.DemoMode, log)
	h.RegisterRoutes(mux)

	// Apply middleware (outermost first).
	var handler http.Handler = mux
	handler = middleware.Logger(log)(handler)
	handler = middleware.Recoverer(log)(handler)
	handler = middleware.CORS(handler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutdown signal received")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
	}()

	log.Info(fmt.Sprintf("server listening on :%s", cfg.Port))
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// sanitizeURL redacts API keys from URLs for safe logging.
func sanitizeURL(u string) string {
	if len(u) > 30 {
		return u[:30] + "..."
	}
	return u
}
