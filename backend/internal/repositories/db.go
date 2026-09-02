// Package repositories provides PostgreSQL-backed persistence for the tracker.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps a sql.DB with helpers specific to this project.
type DB struct {
	*sql.DB
}

// Connect opens and verifies a connection to PostgreSQL.
func Connect(databaseURL string) (*DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{db}, nil
}

// RunMigrations executes all .sql files in the given directory in alphabetical
// order. It is idempotent because the migration files use CREATE TABLE IF NOT
// EXISTS. A production project would use a proper migration tool (goose,
// golang-migrate); for this personal project simplicity wins.
func RunMigrations(db *DB, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, execErr := db.ExecContext(ctx, string(content))
		cancel()
		if execErr != nil {
			return fmt.Errorf("exec migration %s: %w", f, execErr)
		}
	}

	return nil
}
