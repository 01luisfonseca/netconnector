package server

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"netconnector/pkg/logger"
)

// Repository defines the interface for data access
type Repository interface {
	GetClientIDBySubdomain(subdomain string) (string, error)
	AddMapping(subdomain, clientID string) error
	RemoveMapping(subdomain string) error
	ListMappings() (map[string]string, error)
	IsClientIDRegistered(clientID string) (bool, error)
	Close() error
}

type SQLiteDB struct {
	db *sql.DB
}

// NewSQLiteDB initializes a new SQLite database connection and runs migrations
func NewSQLiteDB(dsn string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	sqlDB := &SQLiteDB{db: db}
	if err := sqlDB.migrate(); err != nil {
		return nil, err
	}

	logger.Info("SQLite database initialized successfully", "dsn", dsn)
	return sqlDB, nil
}

func (s *SQLiteDB) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS mappings (
		subdomain TEXT PRIMARY KEY,
		client_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

func (s *SQLiteDB) GetClientIDBySubdomain(subdomain string) (string, error) {
	var clientID string
	query := `SELECT client_id FROM mappings WHERE subdomain = ?`
	err := s.db.QueryRow(query, subdomain).Scan(&clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Return empty string if not found
		}
		return "", fmt.Errorf("error querying subdomain: %w", err)
	}
	return clientID, nil
}

// IsClientIDRegistered checks if a client ID exists in the mappings table
func (s *SQLiteDB) IsClientIDRegistered(clientID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM mappings WHERE client_id = ?)`
	err := s.db.QueryRow(query, clientID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error verifying client ID existence: %w", err)
	}
	return exists, nil
}

func (s *SQLiteDB) AddMapping(subdomain, clientID string) error {
	query := `INSERT OR REPLACE INTO mappings (subdomain, client_id, created_at) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, subdomain, clientID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to insert mapping: %w", err)
	}
	return nil
}

func (s *SQLiteDB) RemoveMapping(subdomain string) error {
	query := `DELETE FROM mappings WHERE subdomain = ?`
	_, err := s.db.Exec(query, subdomain)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}
	return nil
}

func (s *SQLiteDB) ListMappings() (map[string]string, error) {
	query := `SELECT subdomain, client_id FROM mappings`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query mappings: %w", err)
	}
	defer rows.Close()

	mappings := make(map[string]string)
	for rows.Next() {
		var sub, clientID string
		if err := rows.Scan(&sub, &clientID); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		mappings[sub] = clientID
	}
	return mappings, nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
