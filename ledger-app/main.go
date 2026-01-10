package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/project-atlas/ledger-app/internal/api"
	"github.com/project-atlas/ledger-app/internal/config"
	"github.com/project-atlas/ledger-app/internal/database"
	"github.com/project-atlas/ledger-app/internal/s3"
	"github.com/project-atlas/ledger-app/internal/sqs"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting Ledger Application")

	// Load configuration and secrets from environment variables
	cfg := config.LoadConfig()
	secrets := config.LoadSecrets()

	// Initialize database
	db, err := database.New(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		User:     secrets.DatabaseUser,
		Password: secrets.DatabasePassword,
		Timeout:  10 * time.Second,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize S3 client
	s3Client, err := s3.New(s3.Config{
		Endpoint: cfg.AWS.Endpoint,
		Region:   cfg.AWS.Region,
		Bucket:   cfg.AWS.S3Bucket,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to initialize S3 client", zap.Error(err))
	}

	// Initialize SQS client
	sqsClient, err := sqs.New(sqs.Config{
		Endpoint: cfg.AWS.Endpoint,
		Region:   cfg.AWS.Region,
		Queue:    cfg.AWS.SQSQueue,
	}, logger)
	if err != nil {
		logger.Fatal("Failed to initialize SQS client", zap.Error(err))
	}

	// Initialize HTTP handler
	handler := api.NewHandler(db, s3Client, sqsClient, cfg.App.Region, logger)

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/health", handler.Health).Methods("GET")
	router.HandleFunc("/ready", handler.Readiness).Methods("GET")
	router.HandleFunc("/live", handler.Liveness).Methods("GET")
	router.HandleFunc("/transactions", handler.CreateTransaction).Methods("POST")
	router.HandleFunc("/transactions", handler.ListTransactions).Methods("GET")
	router.HandleFunc("/transactions/{id}", handler.GetTransaction).Methods("GET")
	router.HandleFunc("/stats", handler.GetStats).Methods("GET")

	// Add middleware
	router.Use(loggingMiddleware(logger))
	router.Use(corsMiddleware())

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("HTTP server starting",
			zap.Int("port", cfg.App.Port),
			zap.String("region", cfg.App.Region),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Start SQS message processor in background
	go processSQSMessages(sqsClient, db, s3Client, cfg.App.Region, logger)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server stopped")
}

// processSQSMessages processes messages from SQS queue
func processSQSMessages(sqsClient *sqs.Client, db *database.DB, s3Client *s3.Client, region string, logger *zap.Logger) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		receivedMessages, err := sqsClient.ReceiveMessages(10, 0)
		if err != nil {
			logger.Warn("Failed to receive SQS messages", zap.Error(err))
			continue
		}

		for _, receivedMsg := range receivedMessages {
			msg := receivedMsg.Message
			logger.Info("Processing SQS message",
				zap.String("transaction_id", msg.TransactionID),
				zap.String("action", msg.Action),
			)

			// Process message based on action
			processed := false
			switch msg.Action {
			case "transaction_created":
				// Message already processed during API call, just log
				logger.Info("Transaction created message processed",
					zap.String("transaction_id", msg.TransactionID),
				)
				processed = true
			default:
				logger.Info("Unknown action", zap.String("action", msg.Action))
				processed = true // Delete unknown messages to prevent infinite retries
			}

			// Delete message from queue after successful processing
			if processed {
				if err := sqsClient.DeleteMessage(receivedMsg.ReceiptHandle); err != nil {
					logger.Error("Failed to delete SQS message after processing",
						zap.Error(err),
						zap.String("transaction_id", msg.TransactionID),
						zap.String("receipt_handle", receivedMsg.ReceiptHandle),
					)
					// Message will become visible again after visibility timeout
					// and will be retried
				} else {
					logger.Info("SQS message deleted after processing",
						zap.String("transaction_id", msg.TransactionID),
					)
				}
			}
		}
	}
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(logger *zap.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)),
			)
		})
	}
}

// corsMiddleware adds CORS headers
func corsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

