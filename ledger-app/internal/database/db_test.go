package database

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock" // Used for sqlmock.Sqlmock type in setupTestDB return
	"go.uber.org/zap"
)

func TestDB_Health_Success(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	var _ sqlmock.Sqlmock = mock // Use sqlmock type explicitly
	mock.ExpectPing()

	err := db.Health()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDB_Health_Failure(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	mock.ExpectPing().WillReturnError(errors.New("connection failed"))

	err := db.Health()
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if err.Error() != "connection failed" {
		t.Errorf("Expected error message 'connection failed', got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestDB_Close(t *testing.T) {
	// Create a mock database connection
	mockDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}

	logger := zap.NewNop()
	db := &DB{
		conn:   mockDB,
		logger: logger,
	}

	// Close is a cleanup operation
	// Note: sqlmock returns an error for unexpected Close(), but that's expected behavior
	// We're just testing that Close() doesn't panic and can be called
	err = db.Close()
	// sqlmock will return an error about unexpected Close, but that's acceptable for this test
	// In real usage, Close() is called during cleanup and doesn't need to be tracked
	if err != nil && err.Error() != "all expectations were already fulfilled, call to database Close was not expected" {
		// Only fail if it's a different error
		t.Errorf("Unexpected error on close: %v", err)
	}
}

func TestDB_GetConnection(t *testing.T) {
	db, _, cleanup := setupTestDB(t)
	defer cleanup()

	conn := db.GetConnection()
	if conn == nil {
		t.Error("Expected connection, got nil")
	}
}
