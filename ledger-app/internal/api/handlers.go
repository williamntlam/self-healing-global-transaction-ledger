package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/project-atlas/ledger-app/internal/database"
	"github.com/project-atlas/ledger-app/internal/models"
	"github.com/project-atlas/ledger-app/internal/s3"
	"github.com/project-atlas/ledger-app/internal/sqs"
	"go.uber.org/zap"
)

// Handler holds all HTTP handlers
type Handler struct {
	db      *database.DB
	s3      *s3.Client
	sqs     *sqs.Client
	region  string
	logger  *zap.Logger
}

// NewHandler creates a new handler instance
func NewHandler(db *database.DB, s3Client *s3.Client, sqsClient *sqs.Client, region string, logger *zap.Logger) *Handler {
	return &Handler{
		db:     db,
		s3:     s3Client,
		sqs:    sqsClient,
		region: region,
		logger: logger,
	}
}

// CreateTransaction handles POST /transactions
func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate request
	if req.FromAccount == "" || req.ToAccount == "" || req.Amount == "" {
		h.respondError(w, http.StatusBadRequest, "Missing required fields", nil)
		return
	}

	// Create transaction
	tx := &models.Transaction{
		ID:          uuid.New(),
		Region:      h.region,
		Amount:      req.Amount,
		FromAccount: req.FromAccount,
		ToAccount:   req.ToAccount,
		Status:      "pending",
		Timestamp:   time.Now().UTC(),
	}

	// Save to database
	if err := h.db.CreateTransaction(tx); err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create transaction", err)
		return
	}

	// Write audit log to S3
	auditLog := &models.AuditLog{
		TransactionID: tx.ID,
		Region:        h.region,
		Action:        "transaction_created",
		Timestamp:     time.Now().UTC(),
		Details:       "Transaction created via API",
	}
	auditJSON, err := auditLog.ToJSON()
	if err == nil {
		key := fmt.Sprintf("transactions/%s/%s.json", h.region, tx.ID.String())
		h.s3.WriteAuditLog(key, []byte(auditJSON))
	}

	// Send message to SQS
	sqsMsg := &sqs.Message{
		TransactionID: tx.ID.String(),
		Region:        h.region,
		Action:        "transaction_created",
		Timestamp:     time.Now().UTC(),
		Data:          auditJSON,
	}
	if err := h.sqs.SendMessage(sqsMsg); err != nil {
		h.logger.Warn("Failed to send SQS message", zap.Error(err))
	}

	h.respondJSON(w, http.StatusCreated, models.TransactionResponse{
		Transaction: tx,
		Message:     "Transaction created successfully",
	})
}

// GetTransaction handles GET /transactions/{id}
func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid transaction ID", err)
		return
	}

	tx, err := h.db.GetTransaction(id)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "Transaction not found", err)
		return
	}

	h.respondJSON(w, http.StatusOK, models.TransactionResponse{
		Transaction: tx,
	})
}

// ListTransactions handles GET /transactions
func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	transactions, err := h.db.ListTransactions(limit, offset)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to list transactions", err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"transactions": transactions,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetStats handles GET /stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.db.GetTransactionStats()
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to get statistics", err)
		return
	}

	h.respondJSON(w, http.StatusOK, stats)
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	health := map[string]string{
		"status": "healthy",
		"region": h.region,
	}

	// Check database
	if err := h.db.Health(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "unhealthy"
		h.respondJSON(w, http.StatusServiceUnavailable, health)
		return
	}
	health["database"] = "healthy"

	// Check S3
	if err := h.s3.Health(); err != nil {
		health["status"] = "unhealthy"
		health["s3"] = "unhealthy"
		h.respondJSON(w, http.StatusServiceUnavailable, health)
		return
	}
	health["s3"] = "healthy"

	// Check SQS
	if err := h.sqs.Health(); err != nil {
		health["status"] = "unhealthy"
		health["sqs"] = "unhealthy"
		h.respondJSON(w, http.StatusServiceUnavailable, health)
		return
	}
	health["sqs"] = "healthy"

	h.respondJSON(w, http.StatusOK, health)
}

// Readiness handles GET /ready
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	// Check if database is ready
	if err := h.db.Health(); err != nil {
		h.respondJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"reason": "database unavailable",
		})
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// Liveness handles GET /live
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]string{
		"status": "alive",
	})
}

// Helper methods

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", zap.Error(err))
	}
}

func (h *Handler) respondError(w http.ResponseWriter, status int, message string, err error) {
	response := models.TransactionResponse{
		Error: message,
	}
	if err != nil {
		h.logger.Error(message, zap.Error(err))
	}
	h.respondJSON(w, status, response)
}

