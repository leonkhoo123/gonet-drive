package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

// RunMigrations executes any embedded .sql migration scripts that haven't been applied yet.
func RunMigrations(db *sql.DB) error {
	// 1. Create schema_migrations table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Handle legacy databases that don't have migration records yet
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count schema_migrations: %w", err)
	}

	// 3. Get currently applied migrations
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to list applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to read migration version: %w", err)
		}
		applied[version] = true
	}

	// 4. Read embedded migration scripts
	entries, err := embeddedMigrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read embedded migrations directory: %w", err)
	}

	var toApply []string
	for _, f := range entries {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			if !applied[f.Name()] {
				toApply = append(toApply, f.Name())
			}
		}
	}

	// Sort migrations lexicographically
	sort.Strings(toApply)

	if len(toApply) == 0 {
		log.Println("No new migrations to apply.")
		return nil
	}

	// 5. Apply new migrations
	for _, file := range toApply {
		log.Printf("Applying migration: %s", file)
		content, err := embeddedMigrations.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Execute migration within a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", file, err)
		}

		_, err = tx.Exec(string(content))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", file)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", file, err)
		}
		log.Printf("Successfully applied migration: %s", file)
	}

	return nil
}
