package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"
)

// DB wraps the database connection
type DB struct {
	conn   *sql.DB
	logger *zap.Logger
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	Timeout  time.Duration
}

// New creates a new database connection
func New(config Config, logger *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		int(config.Timeout.Seconds()),
	)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
		zap.String("database", config.Database),
	)

	return &DB{
		conn:   conn,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}

// Health checks if the database is healthy
func (db *DB) Health() error {
	return db.conn.Ping()
}

