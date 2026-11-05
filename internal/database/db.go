package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite"
)

// DB wraps a sql.DB instance with a mutex for safe SQLite concurrent operations
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// New initializes the SQLite database connection, applies PRAGMAs, and creates tables
func New(dbPath string) (*DB, error) {
	// Ensure directory exists if path contains directories
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// DSN with WAL mode and busy timeout for optimal SQLite concurrency
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(1)", dbPath)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Set connection limits suited for SQLite single-writer model
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// migrate creates the required database tables if they do not exist
func (d *DB) migrate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS tickets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			channel_id TEXT UNIQUE NOT NULL,
			user_id TEXT NOT NULL,
			status TEXT NOT NULL,
			category TEXT DEFAULT '',
			ingame_nickname TEXT DEFAULT '',
			additional_info TEXT DEFAULT '',
			problem_description TEXT DEFAULT '',
			created_at DATETIME NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_channel_id ON tickets (channel_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_user_id ON tickets (user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets (status);`,

		`CREATE TABLE IF NOT EXISTS transcripts_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ticket_id INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (ticket_id) REFERENCES tickets (id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_transcripts_ticket_id ON transcripts_log (ticket_id);`,
	}

	for _, query := range queries {
		if _, err := d.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration query (%s): %w", query, err)
		}
	}

	if err := d.ensureColumn("tickets", "category_details", "TEXT DEFAULT ''"); err != nil {
		return err
	}

	return nil
}

func (d *DB) ensureColumn(table, column, definition string) error {
	rows, err := d.conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("failed to inspect table %s: %w", table, err)
	}

	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			return fmt.Errorf("failed to inspect column in table %s: %w", table, err)
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to iterate columns in table %s: %w", table, err)
	}
	rows.Close()

	if found {
		return nil
	}

	if _, err := d.conn.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("failed to add column %s to table %s: %w", column, table, err)
	}
	return nil
}
