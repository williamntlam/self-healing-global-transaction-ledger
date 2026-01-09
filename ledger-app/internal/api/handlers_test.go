package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/project-atlas/ledger-app/internal/models"
	"github.com/project-atlas/ledger-app/internal/sqs"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Mock implementations for testing

type mockDB struct {
	createTransactionFunc    func(tx *models.Transaction) error
	getTransactionFunc        func(id uuid.UUID) (*models.Transaction, error)
	listTransactionsFunc      func(limit, offset int) ([]*models.Transaction, error)
	updateTransactionStatusFunc func(id uuid.UUID, status string) error
	getTransactionStatsFunc   func() (map[string]interface{}, error)
	healthFunc                func() error
}

func (m *mockDB) CreateTransaction(tx *models.Transaction) error {
	if m.createTransactionFunc != nil {
		return m.createTransactionFunc(tx)
	}
	return nil
}

func (m *mockDB) GetTransaction(id uuid.UUID) (*models.Transaction, error) {
	if m.getTransactionFunc != nil {
		return m.getTransactionFunc(id)
	}
	return nil, errors.New("transaction not found")
}

func (m *mockDB) ListTransactions(limit, offset int) ([]*models.Transaction, error) {
	if m.listTransactionsFunc != nil {
		return m.listTransactionsFunc(limit, offset)
	}
	return []*models.Transaction{}, nil
}

func (m *mockDB) UpdateTransactionStatus(id uuid.UUID, status string) error {
	if m.updateTransactionStatusFunc != nil {
		return m.updateTransactionStatusFunc(id, status)
	}
	return nil
}

func (m *mockDB) GetTransactionStats() (map[string]interface{}, error) {
	if m.getTransactionStatsFunc != nil {
		return m.getTransactionStatsFunc()
	}
	return map[string]interface{}{}, nil
}

func (m *mockDB) Health() error {
	if m.healthFunc != nil {
		return m.healthFunc()
	}
	return nil
}

type mockS3 struct {
	writeAuditLogFunc func(key string, content []byte) error
	healthFunc        func() error
}

func (m *mockS3) WriteAuditLog(key string, content []byte) error {
	if m.writeAuditLogFunc != nil {
		return m.writeAuditLogFunc(key, content)
	}
	return nil
}

func (m *mockS3) Health() error {
	if m.healthFunc != nil {
		return m.healthFunc()
	}
	return nil
}

type mockSQS struct {
	sendMessageFunc func(msg *sqs.Message) error
	healthFunc      func() error
}

func (m *mockSQS) SendMessage(msg *sqs.Message) error {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(msg)
	}
	return nil
}

func (m *mockSQS) Health() error {
	if m.healthFunc != nil {
		return m.healthFunc()
	}
	return nil
}

// Helper functions

func createTestHandler() (*Handler, *mockDB, *mockS3, *mockSQS) {
	mockDB := &mockDB{}
	mockS3 := &mockS3{}
	mockSQS := &mockSQS{}
	logger := zap.NewNop()
	handler := NewHandler(mockDB, mockS3, mockSQS, "us-east-1", logger)
	return handler, mockDB, mockS3, mockSQS
}

func createTestRouter(handler *Handler) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/transactions", handler.CreateTransaction).Methods("POST")
	router.HandleFunc("/transactions", handler.ListTransactions).Methods("GET")
	router.HandleFunc("/transactions/{id}", handler.GetTransaction).Methods("GET")
	router.HandleFunc("/stats", handler.GetStats).Methods("GET")
	router.HandleFunc("/health", handler.Health).Methods("GET")
	router.HandleFunc("/ready", handler.Readiness).Methods("GET")
	router.HandleFunc("/live", handler.Liveness).Methods("GET")
	return router
}

// Test CreateTransaction

func TestCreateTransaction_Success(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	txID := uuid.New()
	mockDB.createTransactionFunc = func(tx *models.Transaction) error {
		tx.ID = txID
		return nil
	}

	reqBody := models.TransactionRequest{
		FromAccount: "acc1",
		ToAccount:   "acc2",
		Amount:      "100.50",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.TransactionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Transaction == nil {
		t.Fatal("Expected transaction in response")
	}
	if response.Transaction.FromAccount != "acc1" {
		t.Errorf("Expected FromAccount 'acc1', got '%s'", response.Transaction.FromAccount)
	}
	if response.Message != "Transaction created successfully" {
		t.Errorf("Expected success message, got '%s'", response.Message)
	}
}

func TestCreateTransaction_InvalidJSON(t *testing.T) {
	handler, _, _, _ := createTestHandler()
	router := createTestRouter(handler)

	req := httptest.NewRequest("POST", "/transactions", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestCreateTransaction_MissingFields(t *testing.T) {
	handler, _, _, _ := createTestHandler()
	router := createTestRouter(handler)

	testCases := []struct {
		name string
		body models.TransactionRequest
	}{
		{"missing from_account", models.TransactionRequest{ToAccount: "acc2", Amount: "100.50"}},
		{"missing to_account", models.TransactionRequest{FromAccount: "acc1", Amount: "100.50"}},
		{"missing amount", models.TransactionRequest{FromAccount: "acc1", ToAccount: "acc2"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestCreateTransaction_InvalidAmount(t *testing.T) {
	handler, _, _, _ := createTestHandler()
	router := createTestRouter(handler)

	testCases := []struct {
		name  string
		amount string
	}{
		{"invalid format", "not-a-number"},
		{"zero amount", "0"},
		{"negative amount", "-10.50"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := models.TransactionRequest{
				FromAccount: "acc1",
				ToAccount:   "acc2",
				Amount:      tc.amount,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func TestCreateTransaction_DatabaseError(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.createTransactionFunc = func(tx *models.Transaction) error {
		return errors.New("database error")
	}

	reqBody := models.TransactionRequest{
		FromAccount: "acc1",
		ToAccount:   "acc2",
		Amount:      "100.50",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test GetTransaction

func TestGetTransaction_Success(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	txID := uuid.New()
	expectedTx := &models.Transaction{
		ID:          txID,
		Region:      "us-east-1",
		Amount:      decimal.NewFromInt(10050).Div(decimal.NewFromInt(100)),
		FromAccount: "acc1",
		ToAccount:   "acc2",
		Status:      "pending",
		Timestamp:   time.Now().UTC(),
	}

	mockDB.getTransactionFunc = func(id uuid.UUID) (*models.Transaction, error) {
		if id == txID {
			return expectedTx, nil
		}
		return nil, errors.New("not found")
	}

	req := httptest.NewRequest("GET", "/transactions/"+txID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.TransactionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Transaction == nil {
		t.Fatal("Expected transaction in response")
	}
	if response.Transaction.ID != txID {
		t.Errorf("Expected transaction ID %s, got %s", txID, response.Transaction.ID)
	}
}

func TestGetTransaction_InvalidID(t *testing.T) {
	handler, _, _, _ := createTestHandler()
	router := createTestRouter(handler)

	req := httptest.NewRequest("GET", "/transactions/invalid-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetTransaction_NotFound(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.getTransactionFunc = func(id uuid.UUID) (*models.Transaction, error) {
		return nil, errors.New("transaction not found")
	}

	txID := uuid.New()
	req := httptest.NewRequest("GET", "/transactions/"+txID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// Test ListTransactions

func TestListTransactions_Success(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	expectedTxs := []*models.Transaction{
		{
			ID:          uuid.New(),
			Region:      "us-east-1",
			Amount:      decimal.NewFromInt(10000).Div(decimal.NewFromInt(100)),
			FromAccount: "acc1",
			ToAccount:   "acc2",
			Status:      "pending",
			Timestamp:   time.Now().UTC(),
		},
	}

	mockDB.listTransactionsFunc = func(limit, offset int) ([]*models.Transaction, error) {
		return expectedTxs, nil
	}

	req := httptest.NewRequest("GET", "/transactions?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["transactions"] == nil {
		t.Fatal("Expected transactions in response")
	}
}

func TestListTransactions_DefaultPagination(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.listTransactionsFunc = func(limit, offset int) ([]*models.Transaction, error) {
		if limit != 50 {
			t.Errorf("Expected default limit 50, got %d", limit)
		}
		if offset != 0 {
			t.Errorf("Expected default offset 0, got %d", offset)
		}
		return []*models.Transaction{}, nil
	}

	req := httptest.NewRequest("GET", "/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestListTransactions_InvalidPagination(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.listTransactionsFunc = func(limit, offset int) ([]*models.Transaction, error) {
		return []*models.Transaction{}, nil
	}

	testCases := []struct {
		name string
		url  string
	}{
		{"negative limit", "/transactions?limit=-1"},
		{"limit too high", "/transactions?limit=200"},
		{"negative offset", "/transactions?offset=-1"},
		{"invalid limit format", "/transactions?limit=abc"},
		{"invalid offset format", "/transactions?offset=xyz"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should still return 200, but with default/validated values
			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestListTransactions_DatabaseError(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.listTransactionsFunc = func(limit, offset int) ([]*models.Transaction, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest("GET", "/transactions", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test GetStats

func TestGetStats_Success(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	expectedStats := map[string]interface{}{
		"total_transactions": 10,
		"by_status": map[string]int{
			"pending": 5,
			"completed": 5,
		},
		"by_region": map[string]int{
			"us-east-1": 6,
			"eu-central-1": 4,
		},
	}

	mockDB.getTransactionStatsFunc = func() (map[string]interface{}, error) {
		return expectedStats, nil
	}

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["total_transactions"] == nil {
		t.Fatal("Expected total_transactions in response")
	}
}

func TestGetStats_DatabaseError(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.getTransactionStatsFunc = func() (map[string]interface{}, error) {
		return nil, errors.New("database error")
	}

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test Health

func TestHealth_AllHealthy(t *testing.T) {
	handler, mockDB, mockS3, mockSQS := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return nil }
	mockS3.healthFunc = func() error { return nil }
	mockSQS.healthFunc = func() error { return nil }

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}

func TestHealth_DatabaseUnhealthy(t *testing.T) {
	handler, mockDB, mockS3, mockSQS := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return errors.New("database down") }
	mockS3.healthFunc = func() error { return nil }
	mockSQS.healthFunc = func() error { return nil }

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("Expected status 'unhealthy', got '%s'", response["status"])
	}
	if response["database"] != "unhealthy" {
		t.Errorf("Expected database 'unhealthy', got '%s'", response["database"])
	}
}

func TestHealth_S3Unhealthy(t *testing.T) {
	handler, mockDB, mockS3, mockSQS := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return nil }
	mockS3.healthFunc = func() error { return errors.New("S3 down") }
	mockSQS.healthFunc = func() error { return nil }

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestHealth_SQSUnhealthy(t *testing.T) {
	handler, mockDB, mockS3, mockSQS := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return nil }
	mockS3.healthFunc = func() error { return nil }
	mockSQS.healthFunc = func() error { return errors.New("SQS down") }

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test Readiness

func TestReadiness_Ready(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return nil }

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestReadiness_NotReady(t *testing.T) {
	handler, mockDB, _, _ := createTestHandler()
	router := createTestRouter(handler)

	mockDB.healthFunc = func() error { return errors.New("database down") }

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test Liveness

func TestLiveness_Alive(t *testing.T) {
	handler, _, _, _ := createTestHandler()
	router := createTestRouter(handler)

	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got '%s'", response["status"])
	}
}
