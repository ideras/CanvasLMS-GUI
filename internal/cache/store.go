package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGo required
)

// Store is the low-level cache persistence interface.
// Keeping it behind an interface makes it trivially testable
// and swappable (e.g. in-memory store for unit tests).
type Store interface {
	// Get deserialises a cached value into dest.
	// Returns (false, nil) on a clean cache miss.
	// Returns (false, err) if the entry exists but is expired or corrupt.
	Get(key string, dest any) (hit bool, err error)

	// Set serialises value and stores it with the given TTL.
	Set(key string, value any, ttl time.Duration) error

	// Delete removes all entries whose key matches the given SQL LIKE pattern.
	// Use exact keys ("courses:all") or prefix patterns ("course:123:%").
	Delete(pattern string) error

	// Purge removes all expired entries. Call on startup.
	Purge() error

	// Close releases the underlying database connection.
	Close() error
}

// sqliteStore is the production SQLite-backed Store.
type sqliteStore struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS cache (
    key        TEXT PRIMARY KEY,
    value      BLOB    NOT NULL,
    expires_at INTEGER NOT NULL   -- Unix timestamp (seconds)
);
CREATE INDEX IF NOT EXISTS idx_cache_expires ON cache(expires_at);
`

// NewSQLiteStore opens (or creates) the SQLite cache database at dbPath,
// applies the schema, enables WAL mode, and purges stale entries.
func NewSQLiteStore(dbPath string) (Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cache: open db: %w", err)
	}

	// WAL gives much better concurrent read performance.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("cache: set WAL mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL;`); err != nil {
		return nil, fmt.Errorf("cache: set synchronous: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("cache: apply schema: %w", err)
	}

	s := &sqliteStore{db: db}

	// Remove stale rows from previous sessions on startup.
	if err := s.Purge(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) Get(key string, dest any) (bool, error) {
	row := s.db.QueryRow(
		`SELECT value, expires_at FROM cache WHERE key = ?`, key,
	)
	var raw []byte
	var expiresAt int64
	if err := row.Scan(&raw, &expiresAt); err != nil {
		if err == sql.ErrNoRows {
			return false, nil // clean miss
		}
		return false, fmt.Errorf("cache: get %q: %w", key, err)
	}
	if time.Now().Unix() > expiresAt {
		// Expired — remove lazily and report miss.
		_, _ = s.db.Exec(`DELETE FROM cache WHERE key = ?`, key)
		return false, nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return false, fmt.Errorf("cache: unmarshal %q: %w", key, err)
	}
	return true, nil
}

func (s *sqliteStore) Set(key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: marshal %q: %w", key, err)
	}
	expiresAt := time.Now().Add(ttl).Unix()
	_, err = s.db.Exec(
		`INSERT INTO cache(key, value, expires_at) VALUES(?,?,?)
         ON CONFLICT(key) DO UPDATE SET value=excluded.value, expires_at=excluded.expires_at`,
		key, raw, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("cache: set %q: %w", key, err)
	}
	return nil
}

func (s *sqliteStore) Delete(pattern string) error {
	_, err := s.db.Exec(`DELETE FROM cache WHERE key LIKE ?`, pattern)
	if err != nil {
		return fmt.Errorf("cache: delete %q: %w", pattern, err)
	}
	return nil
}

func (s *sqliteStore) Purge() error {
	_, err := s.db.Exec(`DELETE FROM cache WHERE expires_at < ?`, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("cache: purge: %w", err)
	}
	return nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
