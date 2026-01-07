package database

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/project-atlas/ledger-app/internal/models"
	"go.uber.org/zap"
)

// CreateTransaction creates a new transaction in the database
func (db *DB) CreateTransaction(tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (id, region, amount, from_account, to_account, status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, region, amount, from_account, to_account, status, timestamp
	`

	err := db.conn.QueryRow(
		query,
		tx.ID,
		tx.Region,
		tx.Amount,
		tx.FromAccount,
		tx.ToAccount,
		tx.Status,
		tx.Timestamp,
	).Scan(
		&tx.ID,
		&tx.Region,
		&tx.Amount,
		&tx.FromAccount,
		&tx.ToAccount,
		&tx.Status,
		&tx.Timestamp,
	)

	if err != nil {
		db.logger.Error("Failed to create transaction",
			zap.Error(err),
			zap.String("transaction_id", tx.ID.String()),
		)
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	db.logger.Info("Transaction created",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("region", tx.Region),
		zap.String("status", tx.Status),
	)

	return nil
}

// GetTransaction retrieves a transaction by ID
func (db *DB) GetTransaction(id uuid.UUID) (*models.Transaction, error) {
	var tx models.Transaction
	query := `
		SELECT id, region, amount, from_account, to_account, status, timestamp
		FROM transactions
		WHERE id = $1
	`

	err := db.conn.QueryRow(query, id).Scan(
		&tx.ID,
		&tx.Region,
		&tx.Amount,
		&tx.FromAccount,
		&tx.ToAccount,
		&tx.Status,
		&tx.Timestamp,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found: %s", id.String())
	}
	if err != nil {
		db.logger.Error("Failed to get transaction",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
		)
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

// ListTransactions retrieves transactions with pagination
func (db *DB) ListTransactions(limit, offset int) ([]*models.Transaction, error) {
	query := `
		SELECT id, region, amount, from_account, to_account, status, timestamp
		FROM transactions
		ORDER BY timestamp DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		db.logger.Error("Failed to list transactions", zap.Error(err))
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.Region,
			&tx.Amount,
			&tx.FromAccount,
			&tx.ToAccount,
			&tx.Status,
			&tx.Timestamp,
		); err != nil {
			db.logger.Error("Failed to scan transaction", zap.Error(err))
			continue
		}
		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transactions: %w", err)
	}

	return transactions, nil
}

// UpdateTransactionStatus updates the status of a transaction
func (db *DB) UpdateTransactionStatus(id uuid.UUID, status string) error {
	query := `
		UPDATE transactions
		SET status = $1
		WHERE id = $2
	`

	result, err := db.conn.Exec(query, status, id)
	if err != nil {
		db.logger.Error("Failed to update transaction status",
			zap.Error(err),
			zap.String("transaction_id", id.String()),
			zap.String("status", status),
		)
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found: %s", id.String())
	}

	db.logger.Info("Transaction status updated",
		zap.String("transaction_id", id.String()),
		zap.String("status", status),
	)

	return nil
}

// GetTransactionStats returns statistics about transactions
func (db *DB) GetTransactionStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total transactions
	var total int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total transactions: %w", err)
	}
	stats["total_transactions"] = total

	// Transactions by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM transactions
		GROUP BY status
	`
	rows, err := db.conn.Query(statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get status stats: %w", err)
	}
	defer rows.Close()

	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		statusCounts[status] = count
	}
	stats["by_status"] = statusCounts

	// Transactions by region
	regionQuery := `
		SELECT region, COUNT(*) as count
		FROM transactions
		GROUP BY region
	`
	rows, err = db.conn.Query(regionQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get region stats: %w", err)
	}
	defer rows.Close()

	regionCounts := make(map[string]int)
	for rows.Next() {
		var region string
		var count int
		if err := rows.Scan(&region, &count); err != nil {
			continue
		}
		regionCounts[region] = count
	}
	stats["by_region"] = regionCounts

	return stats, nil
}

