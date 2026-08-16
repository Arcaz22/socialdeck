package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jmoiron/sqlx"
)

func Migrate(db *sqlx.DB, migrationsDir string) error {
	// Buat tabel tracking migration jika belum ada
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	pattern := filepath.Join(migrationsDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)

		var count int
		db.QueryRow(`SELECT COUNT(1) FROM schema_migrations WHERE filename = $1`, filename).Scan(&count)
		if count > 0 {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read %s: %w", filename, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", filename, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("exec %s: %w", filename, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, filename,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record %s: %w", filename, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit %s: %w", filename, err)
		}

		fmt.Printf("[migrate] applied: %s\n", filename)
	}

	return nil
}
