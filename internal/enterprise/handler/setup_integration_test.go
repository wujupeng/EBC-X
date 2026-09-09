package handler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/wujupeng/ebcx/internal/enterprise/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/enterprise/repository"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

var (
	superDB  *sql.DB
	appDB    *sql.DB
	sharedPG *embeddedpostgres.EmbeddedPostgres

	rlsManager    *security.RLSManager
	uow           repository.UnitOfWork
	entRepo       *repository.EnterpriseRepositoryPostgreSQL
	idemRepo      *repository.CommandIdempotencyRepositoryPostgreSQL
	evWriter      *evidence_adapter.EnterpriseEvidenceWriterImpl
	outboxPub     *outbox.Publisher
	createHandler *CreateEnterpriseHandler
	updateHandler *UpdateEnterpriseHandler
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5436).
			Database("ebcx_enterprise_itest").
			Username("postgres").
			Password("itest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	superConnStr := "host=localhost port=5436 user=postgres password=itest dbname=ebcx_enterprise_itest sslmode=disable"
	sdb, err := sql.Open("postgres", superConnStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := findMigrationDir()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := runMigrations(sdb, migrationDir); err != nil {
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

	appConnStr := "host=localhost port=5436 user=ebcx_app password=apppass dbname=ebcx_enterprise_itest sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	superDB = sdb
	appDB = adb
	sharedPG = pg

	rlsManager = security.NewRLSManager(appDB)
	uow = repository.NewUnitOfWorkPostgreSQL(rlsManager)
	entRepo = repository.NewEnterpriseRepositoryPostgreSQL()
	idemRepo = repository.NewCommandIdempotencyRepositoryPostgreSQL()
	evWriter = evidence_adapter.NewEnterpriseEvidenceWriter("test-runner")
	outboxPub = outbox.NewPublisher(appDB)
	createHandler = NewCreateEnterpriseHandler(uow, entRepo, evWriter, outboxPub, idemRepo)
	updateHandler = NewUpdateEnterpriseHandler(uow, entRepo, evWriter, outboxPub, idemRepo)

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func findMigrationDir() (string, error) {
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

func runMigrations(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return migrationVersion(files[i]) < migrationVersion(files[j])
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

func migrationVersion(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func cleanupTables(t *testing.T) {
	t.Helper()
	_, err := superDB.Exec(`
		TRUNCATE business.command_idempotency, business.enterprises, evidence.evidence_ledger, outbox.events CASCADE
	`)
	if err != nil {
		t.Fatalf("failed to cleanup tables: %v", err)
	}
}

func countRows(t *testing.T, table string) int {
	t.Helper()
	var count int
	err := superDB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count rows in %s: %v", table, err)
	}
	return count
}

func getEvidenceRecords(t *testing.T, chainID string) []struct {
	SequenceNo   int64
	PreviousHash string
	EvidenceHash string
} {
	t.Helper()
	rows, err := superDB.Query(`
		SELECT sequence_no, previous_evidence_hash, evidence_hash
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query evidence records: %v", err)
	}
	defer rows.Close()
	var result []struct {
		SequenceNo   int64
		PreviousHash string
		EvidenceHash string
	}
	for rows.Next() {
		var r struct {
			SequenceNo   int64
			PreviousHash string
			EvidenceHash string
		}
		if err := rows.Scan(&r.SequenceNo, &r.PreviousHash, &r.EvidenceHash); err != nil {
			t.Fatalf("failed to scan evidence record: %v", err)
		}
		result = append(result, r)
	}
	return result
}

func getOutboxEvents(t *testing.T) []struct {
	EventID string
	Status  string
	Payload []byte
} {
	t.Helper()
	rows, err := superDB.Query(`
		SELECT event_id::text, status, payload::text
		FROM outbox.events
		ORDER BY created_at
	`)
	if err != nil {
		t.Fatalf("failed to query outbox events: %v", err)
	}
	defer rows.Close()
	var result []struct {
		EventID string
		Status  string
		Payload []byte
	}
	for rows.Next() {
		var r struct {
			EventID string
			Status  string
			Payload []byte
		}
		var payloadStr string
		if err := rows.Scan(&r.EventID, &r.Status, &payloadStr); err != nil {
			t.Fatalf("failed to scan outbox event: %v", err)
		}
		r.Payload = []byte(payloadStr)
		result = append(result, r)
	}
	return result
}

func waitForEvidence(t *testing.T, chainID string, expectedCount int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var count int
		superDB.QueryRow(`SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1`, chainID).Scan(&count)
		if count >= expectedCount {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d evidence records in chain %s", expectedCount, chainID)
}

var testCtx = context.Background()
