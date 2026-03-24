package migrations

import (
	"embed"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"

	"github.com/splitcore/backend/internal/database"
)

//go:embed 000001_init.sql
//go:embed 000002_settlements.sql
//go:embed 000003_phase3.sql
var migrationFS embed.FS

//go:embed seeds/001_currencies.sql
//go:embed seeds/002_exchange_rates.sql
var seedFS embed.FS

func RunAll() error {
	log.Println("Running migrations...")

	if err := createSchemaMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	migrations, err := loadMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to load migration files: %w", err)
	}

	for _, m := range migrations {
		applied, err := isMigrationApplied(m.version)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", m.version, err)
		}
		if applied {
			log.Printf("Migration %s already applied, skipping", m.version)
			continue
		}

		log.Printf("Applying migration %s...", m.version)
		if err := applyMigration(m); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", m.version, err)
		}
		if err := recordMigration(m.version); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", m.version, err)
		}
		log.Printf("Migration %s applied successfully", m.version)
	}

	if err := runSeeds(); err != nil {
		return fmt.Errorf("failed to run seeds: %w", err)
	}

	log.Println("All migrations completed successfully")
	return nil
}

func createSchemaMigrationsTable() error {
	_, err := database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

type migration struct {
	version string
	name   string
	sql    string
}

func loadMigrationFiles() ([]migration, error) {
	entries, err := migrationFS.ReadDir(".")
	if err != nil {
		return nil, err
	}

	var migrations []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := migrationFS.ReadFile(entry.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration{
			version: strings.TrimSuffix(entry.Name(), ".sql"),
			name:    entry.Name(),
			sql:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

func isMigrationApplied(version string) (bool, error) {
	var count int
	err := database.DB.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
		version,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func applyMigration(m migration) error {
	_, err := database.DB.Exec(m.sql)
	return err
}

func recordMigration(version string) error {
	_, err := database.DB.Exec(
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, CURRENT_TIMESTAMP)",
		version,
	)
	return err
}

func runSeeds() error {
	log.Println("Running seeds...")

	entries, err := seedFS.ReadDir("seeds")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		content, err := seedFS.ReadFile(filepath.Join("seeds", entry.Name()))
		if err != nil {
			return err
		}
		log.Printf("Running seed: %s", entry.Name())
		if _, err := database.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to run seed %s: %w", entry.Name(), err)
		}
	}

	log.Println("Seeds completed successfully")
	return nil
}
