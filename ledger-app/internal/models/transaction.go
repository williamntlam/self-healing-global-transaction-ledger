package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Transaction represents a financial transaction in the ledger
type Transaction struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Region      string    `json:"region" db:"region"`
	Amount      string    `json:"amount" db:"amount"`
	FromAccount string    `json:"from_account" db:"from_account"`
	ToAccount   string    `json:"to_account" db:"to_account"`
	Status      string    `json:"status" db:"status"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
}

// TransactionRequest represents an incoming transaction request
type TransactionRequest struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
	Amount      string `json:"amount"`
}

// TransactionResponse represents the API response
type TransactionResponse struct {
	Transaction *Transaction `json:"transaction,omitempty"`
	Message     string       `json:"message,omitempty"`
	Error       string       `json:"error,omitempty"`
}

// AuditLog represents an audit log entry for S3
type AuditLog struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	Region        string    `json:"region"`
	Action        string    `json:"action"`
	Timestamp     time.Time `json:"timestamp"`
	Details       string    `json:"details"`
}

// ToJSON converts AuditLog to JSON string
func (a *AuditLog) ToJSON() (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UUIDArray is a custom type for PostgreSQL UUID arrays
type UUIDArray []uuid.UUID

// Value implements the driver.Valuer interface
func (u UUIDArray) Value() (driver.Value, error) {
	if len(u) == 0 {
		return "{}", nil
	}
	result := "{"
	for i, id := range u {
		if i > 0 {
			result += ","
		}
		result += id.String()
	}
	result += "}"
	return result, nil
}

