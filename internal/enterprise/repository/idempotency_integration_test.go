package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/enterprise"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

var (
	iSuperDB  *sql.DB
	iAppDB    *sql.DB
	iPG       *embeddedpostgres.EmbeddedPostgres
	iRLS      *security.RLSManager
	iUOW      *UnitOfWorkPostgreSQL
	iIdemRepo *CommandIdempotencyRepositoryPostgreSQL
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5438).
			Database("ebcx_idem_itest").
			Username("postgres").
			Password("itest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	connStr := "host=localhost port=5438 user=postgres password=itest dbname=ebcx_idem_itest sslmode=disable"
	sdb, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := findMigrationDirI()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := runMigrationsI(sdb, migrationDir); err != nil {
		fmt.Printf("failed to run migrations: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	_, err = sdb.Exec(`CREATE ROLE ebcx_app LOGIN PASSWORD 'apppass' IN ROLE ebcx_runtime`)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		fmt.Printf("failed to create ebcx_app role: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	appConnStr := "host=localhost port=5438 user=ebcx_app password=apppass dbname=ebcx_idem_itest sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	iSuperDB = sdb
	iAppDB = adb
	iPG = pg
	iRLS = security.NewRLSManager(adb)
	iUOW = NewUnitOfWorkPostgreSQL(iRLS)
	iIdemRepo = NewCommandIdempotencyRepositoryPostgreSQL()

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func findMigrationDirI() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "db", "migrations")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("db/migrations not found")
}

func runMigrationsI(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return migrationVersionI(files[i]) < migrationVersionI(files[j])
	})
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("exec %s: %w", f, err)
		}
	}
	return nil
}

func migrationVersionI(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func iCleanup(t *testing.T) {
	t.Helper()
	iSuperDB.Exec(`TRUNCATE business.command_idempotency CASCADE`)
}

func TestIntegration_Idempotency_CheckAndReserve_FreshCommand(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	tx, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}

	result, err := iIdemRepo.CheckAndReserve(context.Background(), tx, cmdID, tenantID, "CreateEnterprise", "")
	if err != nil {
		t.Fatalf("CheckAndReserve failed: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for fresh command, got %+v", result)
	}

	tx.Rollback()

	var count int
	iSuperDB.QueryRow(`SELECT count(*) FROM business.command_idempotency`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 persisted records after rollback, got %d", count)
	}
}

func TestIntegration_Idempotency_MarkSuccess(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	tx, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}

	_, err = iIdemRepo.CheckAndReserve(context.Background(), tx, cmdID, tenantID, "CreateEnterprise", "")
	if err != nil {
		t.Fatalf("CheckAndReserve failed: %v", err)
	}

	eventID := uuid.NewString()
	evidenceID := uuid.NewString()
	err = iIdemRepo.MarkSuccess(context.Background(), tx, cmdID, tenantID, 1, eventID, evidenceID)
	if err != nil {
		t.Fatalf("MarkSuccess failed: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	var status string
	var resultVersion int64
	var resultEventID, resultEvidenceID string
	err = iSuperDB.QueryRow(`
		SELECT status, result_version, result_event_id::text, result_evidence_id::text
		FROM business.command_idempotency
		WHERE command_id::text = $1
	`, cmdID).Scan(&status, &resultVersion, &resultEventID, &resultEvidenceID)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if status != "success" {
		t.Errorf("expected status=success, got %s", status)
	}
	if resultVersion != 1 {
		t.Errorf("expected result_version=1, got %d", resultVersion)
	}
	if resultEventID != eventID {
		t.Errorf("expected result_event_id=%s, got %s", eventID, resultEventID)
	}
	if resultEvidenceID != evidenceID {
		t.Errorf("expected result_evidence_id=%s, got %s", evidenceID, resultEvidenceID)
	}
}

func TestIntegration_Idempotency_DuplicateReturnsFirstResult(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	tx1, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("tx1 begin failed: %v", err)
	}
	_, err = iIdemRepo.CheckAndReserve(context.Background(), tx1, cmdID, tenantID, "CreateEnterprise", "")
	if err != nil {
		t.Fatalf("tx1 CheckAndReserve failed: %v", err)
	}
	eventID := uuid.NewString()
	evidenceID := uuid.NewString()
	err = iIdemRepo.MarkSuccess(context.Background(), tx1, cmdID, tenantID, 1, eventID, evidenceID)
	if err != nil {
		t.Fatalf("tx1 MarkSuccess failed: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("tx1 commit failed: %v", err)
	}

	tx2, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("tx2 begin failed: %v", err)
	}
	result, err := iIdemRepo.CheckAndReserve(context.Background(), tx2, cmdID, tenantID, "CreateEnterprise", "")
	if err != nil {
		t.Fatalf("tx2 CheckAndReserve failed: %v", err)
	}
	tx2.Rollback()

	if result == nil {
		t.Fatal("expected non-nil result for duplicate command")
	}
	if result.ResultVersion != 1 {
		t.Errorf("expected result_version=1, got %d", result.ResultVersion)
	}
	if result.ResultEventID != eventID {
		t.Errorf("expected result_event_id=%s, got %s", eventID, result.ResultEventID)
	}
	if result.ResultEvidenceID != evidenceID {
		t.Errorf("expected result_evidence_id=%s, got %s", evidenceID, result.ResultEvidenceID)
	}
}

func TestIntegration_Idempotency_ConcurrentSameCommandId(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	var wg sync.WaitGroup
	var errs [2]error
	var results [2]*CommandFirstResult

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx, err := iUOW.BeginTenantTx(context.Background(), tenantID)
			if err != nil {
				errs[idx] = fmt.Errorf("tx%d begin: %w", idx, err)
				return
			}
			r, err := iIdemRepo.CheckAndReserve(context.Background(), tx, cmdID, tenantID, "CreateEnterprise", "")
			results[idx] = r
			if err != nil {
				errs[idx] = err
				_ = tx.Rollback()
				return
			}
			errs[idx] = tx.Commit()
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for i := 0; i < 2; i++ {
		if errs[i] == nil && results[i] == nil {
			successCount++
		} else if errs[i] != nil && (strings.Contains(errs[i].Error(), "IDEMPOTENCY-CONFLICT") || strings.Contains(errs[i].Error(), "23505")) {
			conflictCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected 1 success, got %d (errs: %v, %v)", successCount, errs[0], errs[1])
	}
	if conflictCount != 1 {
		t.Errorf("expected 1 conflict, got %d (errs: %v, %v)", conflictCount, errs[0], errs[1])
	}
}

func TestIntegration_Idempotency_MarkFailed(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	tx, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	_, err = iIdemRepo.CheckAndReserve(context.Background(), tx, cmdID, tenantID, "CreateEnterprise", "")
	if err != nil {
		t.Fatalf("CheckAndReserve failed: %v", err)
	}
	err = iIdemRepo.MarkFailed(context.Background(), tx, cmdID, tenantID)
	if err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	var status string
	err = iSuperDB.QueryRow(`SELECT status FROM business.command_idempotency WHERE command_id::text = $1`, cmdID).Scan(&status)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if status != "failed" {
		t.Errorf("expected status=failed, got %s", status)
	}
}

func TestIntegration_Idempotency_EmptyCommandIDRejected(t *testing.T) {
	iCleanup(t)
	tenantID := uuid.NewString()

	tx, err := iUOW.BeginTenantTx(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	defer tx.Rollback()

	_, err = iIdemRepo.CheckAndReserve(context.Background(), tx, "", tenantID, "CreateEnterprise", "")
	if err != enterprise.ErrEnterpriseCommandIDRequired {
		t.Errorf("expected ErrEnterpriseCommandIDRequired, got %v", err)
	}
}
