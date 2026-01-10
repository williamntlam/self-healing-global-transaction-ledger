package database

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/project-atlas/ledger-app/internal/models"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// setupTestDB creates a mock database connection for testing
func setupTestDB(t *testing.T) (*DB, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	logger := zap.NewNop()
	testDB := &DB{
		conn:   db,
		logger: logger,
	}

	cleanup := func() {
		db.Close()
	}

	return testDB, mock, cleanup
}

func TestCreateTransaction_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()
	now := time.Now()
	amount := decimal.NewFromInt(10050).Div(decimal.NewFromInt(100)) // 100.50

	tx := &models.Transaction{
		ID:          txID,
		Region:      "us-east-1",
		Amount:      amount,
		FromAccount: "acc1",
		ToAccount:   "acc2",
		Status:      "pending",
		Timestamp:   now,
	}

	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"}).
		AddRow(txID, "us-east-1", amount, "acc1", "acc2", "pending", now)

	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs(txID, "us-east-1", amount, "acc1", "acc2", "pending", now).
		WillReturnRows(rows)

	err := db.CreateTransaction(tx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestCreateTransaction_DatabaseError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()
	now := time.Now()
	amount := decimal.NewFromInt(10050).Div(decimal.NewFromInt(100))

	tx := &models.Transaction{
		ID:          txID,
		Region:      "us-east-1",
		Amount:      amount,
		FromAccount: "acc1",
		ToAccount:   "acc2",
		Status:      "pending",
		Timestamp:   now,
	}

	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs(txID, "us-east-1", amount, "acc1", "acc2", "pending", now).
		WillReturnError(errors.New("database connection failed"))

	err := db.CreateTransaction(tx)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "failed to create transaction: database connection failed" {
		t.Errorf("Expected specific error message, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransaction_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()
	now := time.Now()
	amount := decimal.NewFromInt(10050).Div(decimal.NewFromInt(100))

	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"}).
		AddRow(txID, "us-east-1", amount, "acc1", "acc2", "pending", now)

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(txID).
		WillReturnRows(rows)

	tx, err := db.GetTransaction(txID)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if tx == nil {
		t.Fatal("Expected transaction, got nil")
	}

	if tx.ID != txID {
		t.Errorf("Expected ID %s, got %s", txID, tx.ID)
	}

	if tx.Region != "us-east-1" {
		t.Errorf("Expected region us-east-1, got %s", tx.Region)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransaction_NotFound(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(txID).
		WillReturnError(sql.ErrNoRows)

	tx, err := db.GetTransaction(txID)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if tx != nil {
		t.Errorf("Expected nil transaction, got %v", tx)
	}

	expectedError := "transaction not found: " + txID.String()
	if err.Error() != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransaction_DatabaseError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(txID).
		WillReturnError(errors.New("database error"))

	tx, err := db.GetTransaction(txID)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if tx != nil {
		t.Errorf("Expected nil transaction, got %v", tx)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestListTransactions_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID1 := uuid.New()
	txID2 := uuid.New()
	now := time.Now()
	amount1 := decimal.NewFromInt(10050).Div(decimal.NewFromInt(100))
	amount2 := decimal.NewFromInt(20000).Div(decimal.NewFromInt(100))

	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"}).
		AddRow(txID1, "us-east-1", amount1, "acc1", "acc2", "pending", now).
		AddRow(txID2, "eu-central-1", amount2, "acc3", "acc4", "completed", now.Add(time.Hour))

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	transactions, err := db.ListTransactions(10, 0)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(transactions) != 2 {
		t.Errorf("Expected 2 transactions, got %d", len(transactions))
	}

	if transactions[0].ID != txID1 {
		t.Errorf("Expected first transaction ID %s, got %s", txID1, transactions[0].ID)
	}

	if transactions[1].ID != txID2 {
		t.Errorf("Expected second transaction ID %s, got %s", txID2, transactions[1].ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestListTransactions_EmptyResult(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"})

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	transactions, err := db.ListTransactions(10, 0)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(transactions) != 0 {
		t.Errorf("Expected 0 transactions, got %d", len(transactions))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestListTransactions_DatabaseError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(10, 0).
		WillReturnError(errors.New("database error"))

	transactions, err := db.ListTransactions(10, 0)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if transactions != nil {
		t.Errorf("Expected nil transactions, got %v", transactions)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestListTransactions_ScanError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Return rows with invalid data type to cause scan error
	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"}).
		AddRow("invalid-uuid", "us-east-1", "invalid-amount", "acc1", "acc2", "pending", "invalid-time")

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	transactions, err := db.ListTransactions(10, 0)
	// The function continues on scan errors, so we should get empty result
	if err != nil {
		t.Errorf("Expected no error (scan errors are logged but not returned), got: %v", err)
	}

	// Should have 0 transactions due to scan error
	if len(transactions) != 0 {
		t.Errorf("Expected 0 transactions due to scan error, got %d", len(transactions))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestListTransactions_RowsError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "region", "amount", "from_account", "to_account", "status", "timestamp"}).
		AddRow(uuid.New(), "us-east-1", decimal.NewFromInt(100), "acc1", "acc2", "pending", time.Now()).
		RowError(0, errors.New("row error"))

	mock.ExpectQuery(`SELECT id, region, amount, from_account, to_account, status, timestamp`).
		WithArgs(10, 0).
		WillReturnRows(rows)

	transactions, err := db.ListTransactions(10, 0)
	if err == nil {
		t.Error("Expected error from rows.Err(), got nil")
	}

	if transactions != nil {
		t.Errorf("Expected nil transactions, got %v", transactions)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUpdateTransactionStatus_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	mock.ExpectExec(`UPDATE transactions`).
		WithArgs("completed", txID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := db.UpdateTransactionStatus(txID, "completed")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUpdateTransactionStatus_NotFound(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	mock.ExpectExec(`UPDATE transactions`).
		WithArgs("completed", txID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := db.UpdateTransactionStatus(txID, "completed")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	expectedError := "transaction not found: " + txID.String()
	if err.Error() != expectedError {
		t.Errorf("Expected error message %s, got %s", expectedError, err.Error())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUpdateTransactionStatus_DatabaseError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	mock.ExpectExec(`UPDATE transactions`).
		WithArgs("completed", txID).
		WillReturnError(errors.New("database error"))

	err := db.UpdateTransactionStatus(txID, "completed")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestUpdateTransactionStatus_RowsAffectedError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	txID := uuid.New()

	result := sqlmock.NewErrorResult(errors.New("rows affected error"))
	mock.ExpectExec(`UPDATE transactions`).
		WithArgs("completed", txID).
		WillReturnResult(result)

	err := db.UpdateTransactionStatus(txID, "completed")
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Mock total transactions query
	totalRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnRows(totalRows)

	// Mock status stats query
	statusRows := sqlmock.NewRows([]string{"status", "count"}).
		AddRow("pending", 30).
		AddRow("completed", 70)
	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
		WillReturnRows(statusRows)

	// Mock region stats query
	regionRows := sqlmock.NewRows([]string{"region", "count"}).
		AddRow("us-east-1", 60).
		AddRow("eu-central-1", 40)
	mock.ExpectQuery(`SELECT region, COUNT\(\*\) as count`).
		WillReturnRows(regionRows)

	stats, err := db.GetTransactionStats()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if total, ok := stats["total_transactions"].(int); !ok || total != 100 {
		t.Errorf("Expected total_transactions 100, got %v", stats["total_transactions"])
	}

	byStatus, ok := stats["by_status"].(map[string]int)
	if !ok {
		t.Fatal("Expected by_status to be map[string]int")
	}
	if byStatus["pending"] != 30 {
		t.Errorf("Expected pending count 30, got %d", byStatus["pending"])
	}
	if byStatus["completed"] != 70 {
		t.Errorf("Expected completed count 70, got %d", byStatus["completed"])
	}

	byRegion, ok := stats["by_region"].(map[string]int)
	if !ok {
		t.Fatal("Expected by_region to be map[string]int")
	}
	if byRegion["us-east-1"] != 60 {
		t.Errorf("Expected us-east-1 count 60, got %d", byRegion["us-east-1"])
	}
	if byRegion["eu-central-1"] != 40 {
		t.Errorf("Expected eu-central-1 count 40, got %d", byRegion["eu-central-1"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_TotalQueryError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnError(errors.New("database error"))

	stats, err := db.GetTransactionStats()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if stats != nil {
		t.Errorf("Expected nil stats, got %v", stats)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_StatusQueryError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Mock total transactions query (success)
	totalRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnRows(totalRows)

	// Mock status stats query (error)
	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
		WillReturnError(errors.New("database error"))

	stats, err := db.GetTransactionStats()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if stats != nil {
		t.Errorf("Expected nil stats, got %v", stats)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_RegionQueryError(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Mock total transactions query (success)
	totalRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnRows(totalRows)

	// Mock status stats query (success)
	statusRows := sqlmock.NewRows([]string{"status", "count"}).
		AddRow("pending", 30)
	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
		WillReturnRows(statusRows)

	// Mock region stats query (error)
	mock.ExpectQuery(`SELECT region, COUNT\(\*\) as count`).
		WillReturnError(errors.New("database error"))

	stats, err := db.GetTransactionStats()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if stats != nil {
		t.Errorf("Expected nil stats, got %v", stats)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_EmptyResults(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Mock total transactions query
	totalRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnRows(totalRows)

	// Mock status stats query (empty)
	statusRows := sqlmock.NewRows([]string{"status", "count"})
	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
		WillReturnRows(statusRows)

	// Mock region stats query (empty)
	regionRows := sqlmock.NewRows([]string{"region", "count"})
	mock.ExpectQuery(`SELECT region, COUNT\(\*\) as count`).
		WillReturnRows(regionRows)

	stats, err := db.GetTransactionStats()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if total, ok := stats["total_transactions"].(int); !ok || total != 0 {
		t.Errorf("Expected total_transactions 0, got %v", stats["total_transactions"])
	}

	byStatus, ok := stats["by_status"].(map[string]int)
	if !ok || len(byStatus) != 0 {
		t.Errorf("Expected empty by_status, got %v", byStatus)
	}

	byRegion, ok := stats["by_region"].(map[string]int)
	if !ok || len(byRegion) != 0 {
		t.Errorf("Expected empty by_region, got %v", byRegion)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestGetTransactionStats_ScanErrorInStatus(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	// Mock total transactions query
	totalRows := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM transactions`).
		WillReturnRows(totalRows)

	// Mock status stats query with invalid data (will cause scan error, but function continues)
	statusRows := sqlmock.NewRows([]string{"status", "count"}).
		AddRow("pending", 30).
		AddRow(nil, "invalid") // Invalid data
	mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count`).
		WillReturnRows(statusRows)

	// Mock region stats query
	regionRows := sqlmock.NewRows([]string{"region", "count"}).
		AddRow("us-east-1", 100)
	mock.ExpectQuery(`SELECT region, COUNT\(\*\) as count`).
		WillReturnRows(regionRows)

	stats, err := db.GetTransactionStats()
	// Function should still succeed, just skip invalid rows
	if err != nil {
		t.Errorf("Expected no error (scan errors are skipped), got: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	byStatus, ok := stats["by_status"].(map[string]int)
	if !ok {
		t.Fatal("Expected by_status to be map[string]int")
	}
	// Should only have "pending" since the invalid row was skipped
	if byStatus["pending"] != 30 {
		t.Errorf("Expected pending count 30, got %d", byStatus["pending"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}
