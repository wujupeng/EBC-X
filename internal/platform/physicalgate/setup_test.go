//go:build integration

package physicalgate

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

var (
	sharedPG *embeddedpostgres.EmbeddedPostgres
	sharedDB *sql.DB
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5434).
			Database("ebcx_phy").
			Username("postgres").
			Password("phytest").
			StartTimeout(60 * time.Second),
	)

	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", "host=localhost port=5434 user=postgres password=phytest dbname=ebcx_phy sslmode=disable")
	if err != nil {
		pg.Stop()
		fmt.Printf("failed to open db: %v\n", err)
		os.Exit(1)
	}

	migrationDir, err := findMigrationDir()
	if err != nil {
		db.Close()
		pg.Stop()
		fmt.Printf("failed to find migrations: %v\n", err)
		os.Exit(1)
	}

	if err := runMigrations(db, migrationDir); err != nil {
		db.Close()
		pg.Stop()
		fmt.Printf("failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	sharedPG = pg
	sharedDB = db

	code := m.Run()

	db.Close()
	pg.Stop()
	os.Exit(code)
}

func getDB(t *testing.T) *sql.DB {
	t.Helper()
	if sharedDB == nil {
		t.Fatal("sharedDB not initialized")
	}
	return sharedDB
}

func findMigrationDir() (string, error) {
	candidates := []string{
		"../../../db/migrations",
		"../../../../db/migrations",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs, nil
		}
	}
	return "", fmt.Errorf("could not find db/migrations directory")
}

func runMigrations(db *sql.DB, migrationDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationDir, "V*.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", f, err)
		}

		_, err = db.Exec(string(sqlBytes))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

func getPGVersion(t *testing.T) string {
	t.Helper()
	var version string
	err := sharedDB.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		t.Fatalf("failed to get PG version: %v", err)
	}
	return version
}

func tableExists(t *testing.T, db *sql.DB, schema, table string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)`,
		schema, table,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	return exists
}

func rlsEnabled(t *testing.T, db *sql.DB, schema, table string) bool {
	t.Helper()
	var enabled bool
	err := db.QueryRow(
		`SELECT c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = $1 AND c.relname = $2`,
		schema, table,
	).Scan(&enabled)
	if err != nil {
		t.Fatalf("failed to check RLS: %v", err)
	}
	return enabled
}

func triggerExists(t *testing.T, db *sql.DB, schema, triggerName, tableName string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM pg_trigger t JOIN pg_class c ON t.tgrelid = c.oid JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = $1 AND t.tgname = $2 AND c.relname = $3)`,
		schema, triggerName, tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check trigger: %v", err)
	}
	return exists
}

func policyExists(t *testing.T, db *sql.DB, policyName, tableName string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM pg_policies WHERE policyname = $1 AND tablename = $2)`,
		policyName, tableName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("failed to check policy: %v", err)
	}
	return exists
}

func countRowsAsTenant(t *testing.T, db *sql.DB, tenantID, query string, args ...interface{}) int {
	t.Helper()

	_, err := db.Exec("SET ROLE ebcx_runtime")
	if err != nil {
		t.Fatalf("SET ROLE error: %v", err)
	}
	defer db.Exec("RESET ROLE")

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin error: %v", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
	if err != nil {
		t.Fatalf("SET LOCAL error: %v", err)
	}

	var count int
	err = tx.QueryRow(query, args...).Scan(&count)
	if err != nil {
		t.Fatalf("QueryRow error: %v", err)
	}
	return count
}

func execShouldFail(t *testing.T, db *sql.DB, desc string, query string, args ...interface{}) {
	t.Helper()
	_, err := db.Exec(query, args...)
	if err == nil {
		t.Errorf("PHYSICAL NEGATIVE TEST FAIL: %s - expected error but succeeded", desc)
	} else {
		t.Logf("PHYSICAL NEGATIVE TEST PASS: %s - rejected by PostgreSQL: %v", desc, err)
	}
}

func execShouldSucceed(t *testing.T, db *sql.DB, desc string, query string, args ...interface{}) sql.Result {
	t.Helper()
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("PHYSICAL TEST FAIL: %s - unexpected error: %v", desc, err)
	}
	return result
}
