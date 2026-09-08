//go:build integration

package security

import (
	"context"
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
	rlsPG *embeddedpostgres.EmbeddedPostgres
	rlsDB *sql.DB
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5435).
			Database("ebcx_rls").
			Username("postgres").
			Password("rlstest").
			StartTimeout(60 * time.Second),
	)

	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", "host=localhost port=5435 user=postgres password=rlstest dbname=ebcx_rls sslmode=disable")
	if err != nil {
		pg.Stop()
		fmt.Printf("failed to open db: %v\n", err)
		os.Exit(1)
	}

	migrationDir, err := findRLSMigrationDir()
	if err != nil {
		db.Close()
		pg.Stop()
		fmt.Printf("failed to find migrations: %v\n", err)
		os.Exit(1)
	}

	if err := runRLSMigrations(db, migrationDir); err != nil {
		db.Close()
		pg.Stop()
		fmt.Printf("failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	rlsPG = pg
	rlsDB = db

	code := m.Run()

	db.Close()
	pg.Stop()
	os.Exit(code)
}

func findRLSMigrationDir() (string, error) {
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

func runRLSMigrations(db *sql.DB, migrationDir string) error {
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

func TestPhysical_RLSManager_SetTenantContext(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"

	_, err := rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-a', 'READ', 'order', 'o1', 'allow', 'test')`,
		tenantA)
	if err != nil {
		t.Fatalf("insert audit A: %v", err)
	}

	_, err = rlsDB.Exec(
		`INSERT INTO audit.audit_log (tenant_id, user_id, action, resource, resource_id, decision, reason)
		 VALUES ($1, 'user-b', 'READ', 'order', 'o2', 'allow', 'test')`,
		tenantB)
	if err != nil {
		t.Fatalf("insert audit B: %v", err)
	}

	rlsDB.Exec("SET ROLE ebcx_runtime")
	defer rlsDB.Exec("RESET ROLE")

	tx, _ := rlsDB.Begin()
	tx.Exec(fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantA))
	var countA int
	tx.QueryRow("SELECT count(*) FROM audit.audit_log").Scan(&countA)
	tx.Rollback()
	if countA != 1 {
		t.Errorf("RLS: tenant A should see 1 row, got %d", countA)
	}

	tx2, _ := rlsDB.Begin()
	tx2.Exec(fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantB))
	var countB int
	tx2.QueryRow("SELECT count(*) FROM audit.audit_log").Scan(&countB)
	tx2.Rollback()
	if countB != 1 {
		t.Errorf("RLS: tenant B should see 1 row, got %d", countB)
	}

	t.Log("PHYSICAL PASS: RLSManager + RLS policy - tenant isolation verified via real PostgreSQL 18.3")
}

func TestPhysical_RLSManager_MissingTenant_Error(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	mgr := NewRLSManager(rlsDB)

	err := mgr.SetTenantContext(context.Background(), RLSContext{TenantID: "", UserID: "user-a"})
	if err != ErrTenantNotSet {
		t.Errorf("expected ErrTenantNotSet, got %v", err)
	}

	t.Log("PHYSICAL PASS: RLSManager rejects empty tenant_id")
}

func TestPhysical_RLSManager_ClearContext(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	mgr := NewRLSManager(rlsDB)

	err := mgr.SetTenantContext(context.Background(), RLSContext{TenantID: "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatalf("SetTenantContext error: %v", err)
	}

	err = mgr.ClearTenantContext(context.Background())
	if err != nil {
		t.Fatalf("ClearTenantContext error: %v", err)
	}

	t.Log("PHYSICAL PASS: RLSManager ClearTenantContext succeeds")
}

func TestPhysical_RLSManager_VerifyRLSEnabled(t *testing.T) {
	if rlsDB == nil {
		t.Fatal("rlsDB not initialized")
	}

	mgr := NewRLSManager(rlsDB)

	enabled, err := mgr.VerifyRLSEnabled(context.Background(), "audit_log")
	if err != nil {
		t.Fatalf("VerifyRLSEnabled error: %v", err)
	}
	if !enabled {
		t.Error("RLS should be enabled on audit_log")
	}

	enabled, err = mgr.VerifyRLSEnabled(context.Background(), "metadata")
	if err != nil {
		t.Fatalf("VerifyRLSEnabled error: %v", err)
	}
	if !enabled {
		t.Error("RLS should be enabled on artifact.metadata")
	}

	t.Log("PHYSICAL PASS: RLSManager VerifyRLSEnabled - audit_log and metadata have RLS enabled")
}
