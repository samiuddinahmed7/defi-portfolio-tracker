package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/models"
)

// BalanceRepository handles persistence for wallet token balances.
type BalanceRepository struct {
	db *DB
}

// NewBalanceRepository returns a new BalanceRepository.
func NewBalanceRepository(db *DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

// Upsert inserts or replaces the balance for a wallet/token pair.
func (r *BalanceRepository) Upsert(ctx context.Context, b models.Balance) error {
	const q = `
		INSERT INTO balances
			(wallet_address, token_address, token_symbol, token_name, decimals,
			 raw_balance, normal_balance, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (wallet_address, token_address)
		DO UPDATE SET
			token_symbol   = EXCLUDED.token_symbol,
			token_name     = EXCLUDED.token_name,
			decimals       = EXCLUDED.decimals,
			raw_balance    = EXCLUDED.raw_balance,
			normal_balance = EXCLUDED.normal_balance,
			fetched_at     = EXCLUDED.fetched_at`

	_, err := r.db.ExecContext(ctx, q,
		b.WalletAddress, b.TokenAddress, b.TokenSymbol, b.TokenName,
		b.Decimals, b.RawBalance, b.NormalBalance, b.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert balance: %w", err)
	}
	return nil
}

// GetByWallet returns all balances for a wallet address.
func (r *BalanceRepository) GetByWallet(ctx context.Context, address string) ([]models.Balance, error) {
	const q = `
		SELECT id, wallet_address, token_address, token_symbol, token_name,
		       decimals, raw_balance, normal_balance, fetched_at
		FROM balances
		WHERE wallet_address = $1
		ORDER BY normal_balance DESC`

	rows, err := r.db.QueryContext(ctx, q, address)
	if err != nil {
		return nil, fmt.Errorf("query balances: %w", err)
	}
	defer rows.Close()

	var balances []models.Balance
	for rows.Next() {
		var b models.Balance
		if err := rows.Scan(
			&b.ID, &b.WalletAddress, &b.TokenAddress, &b.TokenSymbol,
			&b.TokenName, &b.Decimals, &b.RawBalance, &b.NormalBalance, &b.FetchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		balances = append(balances, b)
	}
	return balances, rows.Err()
}

// SavePriceSnapshot persists a price observation to the database.
func (r *BalanceRepository) SavePriceSnapshot(ctx context.Context, symbol string, priceUSD float64, source string) error {
	const q = `
		INSERT INTO price_snapshots (symbol, price_usd, source, fetched_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, q, symbol, priceUSD, source, time.Now())
	if err != nil {
		return fmt.Errorf("save price snapshot: %w", err)
	}
	return nil
}

// GetLatestPrice returns the most recent price snapshot for a symbol.
func (r *BalanceRepository) GetLatestPrice(ctx context.Context, symbol string) (*models.PriceSnapshot, error) {
	const q = `
		SELECT id, symbol, price_usd, source, fetched_at
		FROM price_snapshots
		WHERE symbol = $1
		ORDER BY fetched_at DESC
		LIMIT 1`

	row := r.db.QueryRowContext(ctx, q, symbol)
	var ps models.PriceSnapshot
	if err := row.Scan(&ps.ID, &ps.Symbol, &ps.PriceUSD, &ps.Source, &ps.FetchedAt); err != nil {
		return nil, err
	}
	return &ps, nil
}
