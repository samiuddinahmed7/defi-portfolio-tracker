package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Server
	Port     string
	LogLevel string

	// Database
	DatabaseURL string

	// Blockchain APIs
	EtherscanAPIKey string
	AlchemyAPIKey   string
	EthereumRPCURL  string

	// Chainlink price feed contract addresses
	ChainlinkETHUSDFeed  string
	ChainlinkBTCUSDFeed  string
	ChainlinkLINKUSDFeed string

	// Behaviour flags
	DemoMode bool

	// Provider timeouts
	ProviderTimeout time.Duration
}

// Load reads configuration from environment variables and returns a Config.
// It does not require all variables to be present; missing optional keys get
// sensible defaults. Required keys (DATABASE_URL) cause an error when absent
// and DEMO_MODE is false.
func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnvOrDefault("PORT", "8080"),
		LogLevel: getEnvOrDefault("LOG_LEVEL", "info"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		EtherscanAPIKey: os.Getenv("ETHERSCAN_API_KEY"),
		AlchemyAPIKey:   os.Getenv("ALCHEMY_API_KEY"),
		EthereumRPCURL:  os.Getenv("ETHEREUM_RPC_URL"),

		ChainlinkETHUSDFeed:  getEnvOrDefault("CHAINLINK_ETH_USD_FEED", "0x5f4eC3Df9cbd43714FE2740f5E3616155c5b8419"),
		ChainlinkBTCUSDFeed:  getEnvOrDefault("CHAINLINK_BTC_USD_FEED", "0xF4030086522a5bEEa4988F8cA5B36dbC97BeE88d"),
		ChainlinkLINKUSDFeed: getEnvOrDefault("CHAINLINK_LINK_USD_FEED", "0x2c1d072e956AFFC0D435Cb7AC308d97936Ed4a3"),
	}

	// Parse demo mode
	demoStr := getEnvOrDefault("DEMO_MODE", "false")
	demo, err := strconv.ParseBool(demoStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DEMO_MODE value %q: %w", demoStr, err)
	}
	cfg.DemoMode = demo

	// Parse provider timeout
	timeoutStr := getEnvOrDefault("PROVIDER_TIMEOUT_SECONDS", "10")
	timeoutSecs, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PROVIDER_TIMEOUT_SECONDS value %q: %w", timeoutStr, err)
	}
	cfg.ProviderTimeout = time.Duration(timeoutSecs) * time.Second

	// Validate required fields when not in demo mode
	if !cfg.DemoMode {
		if cfg.DatabaseURL == "" {
			return nil, fmt.Errorf("DATABASE_URL is required (or set DEMO_MODE=true)")
		}
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
