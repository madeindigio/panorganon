package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // SQLite driver
	"go.uber.org/zap"
)

// DB wraps the database connection
type DB struct {
	conn   *sql.DB
	logger *zap.Logger
	path   string
}

// NewDB creates a new database connection
func NewDB(path string, logger *zap.Logger) (*DB, error) {
	logger = logger.With(zap.String("component", "database"))

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open SQLite database
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set pragmas for better performance
	if _, err := conn.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA foreign_keys=ON;
	`); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set pragmas: %w", err)
	}

	db := &DB{
		conn:   conn,
		logger: logger,
		path:   path,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Database initialized", zap.String("path", path))
	return db, nil
}

// initSchema creates the database tables if they don't exist
func (db *DB) initSchema() error {
	db.logger.Info("Initializing database schema")

	if _, err := db.conn.Exec(Schema); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	db.logger.Info("Closing database connection")
	return db.conn.Close()
}

// GetConnection returns the underlying database connection
func (db *DB) GetConnection() *sql.DB {
	return db.conn
}
