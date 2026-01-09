package api

import (
	"github.com/google/uuid"
	"github.com/project-atlas/ledger-app/internal/models"
	"github.com/project-atlas/ledger-app/internal/sqs"
)

// DBInterface defines the database operations needed by handlers
type DBInterface interface {
	CreateTransaction(tx *models.Transaction) error
	GetTransaction(id uuid.UUID) (*models.Transaction, error)
	ListTransactions(limit, offset int) ([]*models.Transaction, error)
	UpdateTransactionStatus(id uuid.UUID, status string) error
	GetTransactionStats() (map[string]interface{}, error)
	Health() error
}

// S3Interface defines the S3 operations needed by handlers
type S3Interface interface {
	WriteAuditLog(key string, content []byte) error
	Health() error
}

// SQSInterface defines the SQS operations needed by handlers
type SQSInterface interface {
	SendMessage(msg *sqs.Message) error
	Health() error
}
