package migrations

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed sql/*.sql
var migrationFS embed.FS

// Run executes all pending migrations
func Run(db *sql.DB, dbName string) error {
	log.Println("🔄 Running database migrations...")

	// Create source driver from embedded files
	sourceDriver, err := iofs.New(migrationFS, "sql")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create database driver
	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	if dirty {
		log.Printf("⚠️  Migration version %d is dirty", version)
	} else if err == migrate.ErrNilVersion {
		log.Println("✅ No migrations to run")
	} else {
		log.Printf("✅ Migrations complete (version %d)", version)
	}

	return nil
}

// Rollback rolls back the last migration
func Rollback(db *sql.DB, dbName string) error {
	log.Println("🔄 Rolling back last migration...")

	sourceDriver, err := iofs.New(migrationFS, "sql")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rollback failed: %w", err)
	}

	log.Println("✅ Rollback complete")
	return nil
}

// Status returns current migration version
func Status(db *sql.DB, dbName string) (uint, bool, error) {
	sourceDriver, err := iofs.New(migrationFS, "sql")
	if err != nil {
		return 0, false, err
	}

	dbDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return 0, false, err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver)
	if err != nil {
		return 0, false, err
	}

	return m.Version()
}
