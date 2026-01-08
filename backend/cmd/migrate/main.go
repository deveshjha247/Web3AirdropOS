package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const migrationsTable = "schema_migrations"

type Migration struct {
	Version     string
	Name        string
	Filename    string
	AppliedAt   *time.Time
	Checksum    string
	ExecutionMs int
}

func main() {
	// Load environment
	godotenv.Load()

	// Parse flags
	var (
		command       = flag.String("cmd", "up", "Command: up, down, status, create, reset")
		migrationName = flag.String("name", "", "Name for new migration (with -cmd=create)")
		steps         = flag.Int("steps", 0, "Number of migrations to apply (0 = all)")
		databaseURL   = flag.String("db", "", "Database URL (or set DATABASE_URL env)")
		migrationsDir = flag.String("dir", "migrations", "Migrations directory")
		dryRun        = flag.Bool("dry-run", false, "Show what would be done without executing")
	)
	flag.Parse()

	// Get database URL
	dbURL := *databaseURL
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required (set via -db flag or DATABASE_URL env)")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Ensure migrations table exists
	if err := ensureMigrationsTable(db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	// Execute command
	switch *command {
	case "up":
		if err := migrateUp(db, *migrationsDir, *steps, *dryRun); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
	case "down":
		if err := migrateDown(db, *migrationsDir, *steps, *dryRun); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
	case "status":
		if err := showStatus(db, *migrationsDir); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	case "create":
		if *migrationName == "" {
			log.Fatal("Migration name is required (use -name flag)")
		}
		if err := createMigration(*migrationsDir, *migrationName); err != nil {
			log.Fatalf("Create migration failed: %v", err)
		}
	case "reset":
		fmt.Print("⚠️  This will drop and recreate all tables. Type 'yes' to confirm: ")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			log.Fatal("Aborted")
		}
		if err := resetDatabase(db, *migrationsDir, *dryRun); err != nil {
			log.Fatalf("Reset failed: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}

func ensureMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			applied_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64),
			execution_time_ms INTEGER
		);
	`
	_, err := db.Exec(query)
	return err
}

func getAppliedMigrations(db *sql.DB) (map[string]*Migration, error) {
	rows, err := db.Query(`
		SELECT version, name, applied_at, checksum, COALESCE(execution_time_ms, 0)
		FROM schema_migrations
		ORDER BY version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	migrations := make(map[string]*Migration)
	for rows.Next() {
		m := &Migration{}
		if err := rows.Scan(&m.Version, &m.Name, &m.AppliedAt, &m.Checksum, &m.ExecutionMs); err != nil {
			return nil, err
		}
		migrations[m.Version] = m
	}
	return migrations, rows.Err()
}

func getPendingMigrations(migrationsDir string, applied map[string]*Migration) ([]*Migration, error) {
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var pending []*Migration
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		// Skip down migrations
		if strings.Contains(f.Name(), ".down.") {
			continue
		}

		// Extract version and name from filename (e.g., 001_initial_schema.sql)
		parts := strings.SplitN(strings.TrimSuffix(f.Name(), ".sql"), "_", 2)
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		name := parts[1]

		if _, ok := applied[version]; !ok {
			// Read file content for checksum
			content, err := ioutil.ReadFile(filepath.Join(migrationsDir, f.Name()))
			if err != nil {
				return nil, err
			}
			hash := sha256.Sum256(content)

			pending = append(pending, &Migration{
				Version:  version,
				Name:     name,
				Filename: f.Name(),
				Checksum: hex.EncodeToString(hash[:]),
			})
		}
	}

	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

func migrateUp(db *sql.DB, migrationsDir string, steps int, dryRun bool) error {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	pending, err := getPendingMigrations(migrationsDir, applied)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		fmt.Println("✅ No pending migrations")
		return nil
	}

	// Limit steps if specified
	if steps > 0 && steps < len(pending) {
		pending = pending[:steps]
	}

	fmt.Printf("📦 Found %d pending migration(s)\n\n", len(pending))

	for _, m := range pending {
		fmt.Printf("⬆️  Applying: %s (%s)\n", m.Version, m.Name)

		if dryRun {
			fmt.Println("   [DRY RUN - not executed]")
			continue
		}

		content, err := ioutil.ReadFile(filepath.Join(migrationsDir, m.Filename))
		if err != nil {
			return fmt.Errorf("failed to read migration file: %w", err)
		}

		start := time.Now()

		// Execute migration in transaction
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}

		// Record migration
		executionMs := int(time.Since(start).Milliseconds())
		_, err = tx.Exec(`
			INSERT INTO schema_migrations (version, name, checksum, execution_time_ms)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (version) DO UPDATE SET
				applied_at = CURRENT_TIMESTAMP,
				checksum = $3,
				execution_time_ms = $4
		`, m.Version, m.Name, m.Checksum, executionMs)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		fmt.Printf("   ✅ Applied in %dms\n", executionMs)
	}

	fmt.Printf("\n✅ All migrations applied successfully\n")
	return nil
}

func migrateDown(db *sql.DB, migrationsDir string, steps int, dryRun bool) error {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println("✅ No migrations to roll back")
		return nil
	}

	// Get applied migrations in reverse order
	var versions []string
	for v := range applied {
		versions = append(versions, v)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	if steps <= 0 {
		steps = 1 // Default to rolling back one migration
	}
	if steps > len(versions) {
		steps = len(versions)
	}

	toRollback := versions[:steps]

	for _, version := range toRollback {
		m := applied[version]
		downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.down.sql", m.Version, m.Name))

		fmt.Printf("⬇️  Rolling back: %s (%s)\n", m.Version, m.Name)

		if dryRun {
			fmt.Println("   [DRY RUN - not executed]")
			continue
		}

		// Check if down migration exists
		if _, err := os.Stat(downFile); os.IsNotExist(err) {
			return fmt.Errorf("down migration not found: %s", downFile)
		}

		content, err := ioutil.ReadFile(downFile)
		if err != nil {
			return fmt.Errorf("failed to read down migration: %w", err)
		}

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("rollback %s failed: %w", m.Version, err)
		}

		if _, err := tx.Exec("DELETE FROM schema_migrations WHERE version = $1", m.Version); err != nil {
			tx.Rollback()
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		fmt.Printf("   ✅ Rolled back\n")
	}

	return nil
}

func showStatus(db *sql.DB, migrationsDir string) error {
	applied, err := getAppliedMigrations(db)
	if err != nil {
		return err
	}

	pending, err := getPendingMigrations(migrationsDir, applied)
	if err != nil {
		return err
	}

	fmt.Println("📊 Migration Status")
	fmt.Println("==================")
	fmt.Printf("\n✅ Applied: %d\n", len(applied))

	// Sort applied by version
	var appliedVersions []string
	for v := range applied {
		appliedVersions = append(appliedVersions, v)
	}
	sort.Strings(appliedVersions)

	for _, v := range appliedVersions {
		m := applied[v]
		appliedAt := "unknown"
		if m.AppliedAt != nil {
			appliedAt = m.AppliedAt.Format("2006-01-02 15:04:05")
		}
		fmt.Printf("   • %s - %s (applied: %s, %dms)\n", m.Version, m.Name, appliedAt, m.ExecutionMs)
	}

	fmt.Printf("\n⏳ Pending: %d\n", len(pending))
	for _, m := range pending {
		fmt.Printf("   • %s - %s\n", m.Version, m.Name)
	}

	return nil
}

func createMigration(migrationsDir string, name string) error {
	// Ensure directory exists
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return err
	}

	// Find next version number
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	maxVersion := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(f.Name(), "_", 2)
		if len(parts) >= 1 {
			var v int
			if _, err := fmt.Sscanf(parts[0], "%d", &v); err == nil && v > maxVersion {
				maxVersion = v
			}
		}
	}

	version := fmt.Sprintf("%03d", maxVersion+1)
	safeName := strings.ReplaceAll(strings.ToLower(name), " ", "_")

	// Create up migration
	upFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.sql", version, safeName))
	upContent := fmt.Sprintf(`-- Migration: %s_%s
-- Description: %s
-- Created: %s

-- Your migration SQL here

-- Record this migration
INSERT INTO schema_migrations (version, name, checksum) 
VALUES ('%s', '%s', 'auto-generated')
ON CONFLICT (version) DO NOTHING;
`, version, safeName, name, time.Now().Format("2006-01-02"), version, safeName)

	if err := ioutil.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return err
	}
	fmt.Printf("✅ Created: %s\n", upFile)

	// Create down migration
	downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.down.sql", version, safeName))
	downContent := fmt.Sprintf(`-- Rollback Migration: %s_%s
-- Description: Rollback %s
-- Created: %s

-- Your rollback SQL here

-- Remove migration record
DELETE FROM schema_migrations WHERE version = '%s';
`, version, safeName, name, time.Now().Format("2006-01-02"), version)

	if err := ioutil.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		return err
	}
	fmt.Printf("✅ Created: %s\n", downFile)

	return nil
}

func resetDatabase(db *sql.DB, migrationsDir string, dryRun bool) error {
	fmt.Println("🔄 Resetting database...")

	if !dryRun {
		// Drop all tables (except system tables)
		_, err := db.Exec(`
			DO $$ DECLARE
				r RECORD;
			BEGIN
				FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
					EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
				END LOOP;
			END $$;
		`)
		if err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}

		// Drop all types
		_, err = db.Exec(`
			DO $$ DECLARE
				r RECORD;
			BEGIN
				FOR r IN (SELECT typname FROM pg_type WHERE typtype = 'e' AND typnamespace = 'public'::regnamespace) LOOP
					EXECUTE 'DROP TYPE IF EXISTS ' || quote_ident(r.typname) || ' CASCADE';
				END LOOP;
			END $$;
		`)
		if err != nil {
			return fmt.Errorf("failed to drop types: %w", err)
		}

		// Drop all functions
		_, err = db.Exec(`
			DO $$ DECLARE
				r RECORD;
			BEGIN
				FOR r IN (SELECT proname, oidvectortypes(proargtypes) as args FROM pg_proc WHERE pronamespace = 'public'::regnamespace) LOOP
					EXECUTE 'DROP FUNCTION IF EXISTS ' || quote_ident(r.proname) || '(' || r.args || ') CASCADE';
				END LOOP;
			END $$;
		`)
		if err != nil {
			// Ignore errors for functions
		}
	}

	fmt.Println("✅ All tables dropped")

	// Re-run all migrations
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	return migrateUp(db, migrationsDir, 0, dryRun)
}
