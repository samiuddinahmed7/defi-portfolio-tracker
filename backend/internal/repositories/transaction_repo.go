package repositories

import (
	"context"
	"fmt"

	"github.com/samiuddinahmed7/defi-portfolio-tracker/backend/internal/models"
)

// TransactionRepository handles persistence for Ethereum transactions.
type TransactionRepository struct {
	db *DB
}

// NewTransactionRepository returns a new TransactionRepository.
func NewTransactionRepository(db *DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// UpsertBatch inserts or ignores a slice of transactions.
func (r *TransactionRepository) UpsertBatch(ctx context.Context, txns []models.Transaction) error {
	const q = `
		INSERT INTO transactions
			(hash, wallet_address, block_number, timestamp,
			 from_address, to_address, value, normal_value,
			 gas, gas_used, gas_price,
			 token_address, token_symbol, token_value, token_decimals,
			 tx_type, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (hash, wallet_address) DO NOTHING`

	for _, tx := range txns {
		_, err := r.db.ExecContext(ctx, q,
			tx.Hash, tx.WalletAddress, tx.BlockNumber, tx.Timestamp,
			tx.From, tx.To, tx.Value, tx.NormalValue,
			tx.Gas, tx.GasUsed, tx.GasPrice,
			tx.TokenAddress, tx.TokenSymbol, tx.TokenValue, tx.TokenDecimals,
			tx.TxType, tx.Status,
		)
		if err != nil {
			return fmt.Errorf("upsert transaction %s: %w", tx.Hash, err)
		}
	}
	return nil
}

// GetByWallet returns a page of transactions for the given wallet address,
// ordered newest first.
func (r *TransactionRepository) GetByWallet(ctx context.Context, address string, page, perPage int) ([]models.Transaction, int, error) {
	offset := (page - 1) * perPage

	const countQ = `SELECT COUNT(*) FROM transactions WHERE wallet_address = $1`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, address).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count transactions: %w", err)
	}

	const q = `
		SELECT id, hash, wallet_address, block_number, timestamp,
		       from_address, to_address, value, normal_value,
		       gas, gas_used, gas_price,
		       token_address, token_symbol, token_value, token_decimals,
		       tx_type, status
		FROM transactions
		WHERE wallet_address = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, q, address, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	var txns []models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID, &tx.Hash, &tx.WalletAddress, &tx.BlockNumber, &tx.Timestamp,
			&tx.From, &tx.To, &tx.Value, &tx.NormalValue,
			&tx.Gas, &tx.GasUsed, &tx.GasPrice,
			&tx.TokenAddress, &tx.TokenSymbol, &tx.TokenValue, &tx.TokenDecimals,
			&tx.TxType, &tx.Status,
		); err != nil {
			return nil, 0, fmt.Errorf("scan transaction: %w", err)
		}
		txns = append(txns, tx)
	}
	return txns, total, rows.Err()
}
