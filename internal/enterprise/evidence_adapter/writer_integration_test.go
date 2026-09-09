package evidence_adapter

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
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

var (
	wSuperDB *sql.DB
	wAppDB   *sql.DB
	wPG      *embeddedpostgres.EmbeddedPostgres
	wWriter  *EnterpriseEvidenceWriterImpl
	wRLS     *security.RLSManager
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5437).
			Database("ebcx_writer_itest").
			Username("postgres").
			Password("itest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	connStr := "host=localhost port=5437 user=postgres password=itest dbname=ebcx_writer_itest sslmode=disable"
	sdb, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := findMigrationDirW()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := runMigrationsW(sdb, migrationDir); err != nil {
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

	appConnStr := "host=localhost port=5437 user=ebcx_app password=apppass dbname=ebcx_writer_itest sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	wSuperDB = sdb
	wAppDB = adb
	wPG = pg
	wWriter = NewEnterpriseEvidenceWriter("test-runner")
	wRLS = security.NewRLSManager(adb)

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func findMigrationDirW() (string, error) {
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

func runMigrationsW(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return migrationVersionW(files[i]) < migrationVersionW(files[j])
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

func migrationVersionW(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func wCleanup(t *testing.T) {
	t.Helper()
	wSuperDB.Exec(`TRUNCATE evidence.evidence_ledger, outbox.events, business.enterprises, business.command_idempotency CASCADE`)
}

func wGetChainRecords(t *testing.T, chainID string) []struct {
	SequenceNo   int64
	PreviousHash string
	EvidenceHash string
} {
	t.Helper()
	rows, err := wSuperDB.Query(`
		SELECT sequence_no, previous_evidence_hash, evidence_hash
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query chain records: %v", err)
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
			t.Fatalf("failed to scan: %v", err)
		}
		result = append(result, r)
	}
	return result
}

func makeAggregate(tenantID string) *enterprise.EnterpriseAggregate {
	return &enterprise.EnterpriseAggregate{
		EnterpriseID: uuid.NewString(),
		Name:         "ChainTestCorp",
		Version:      1,
		TenantID:     tenantID,
	}
}

func makeEvent(agg *enterprise.EnterpriseAggregate) *enterprise.EnterpriseCreatedEvent {
	return &enterprise.EnterpriseCreatedEvent{
		EventID:      uuid.NewString(),
		EventType:    "EnterpriseCreated",
		EnterpriseID: agg.EnterpriseID,
		Name:         agg.Name,
		Version:      agg.Version,
		TenantID:     agg.TenantID,
		Timestamp:    time.Now(),
	}
}

func TestIntegration_EvidenceChain_ConcurrentOrdering(t *testing.T) {
	wCleanup(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	tx, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
	if err != nil {
		t.Fatalf("seed tx failed: %v", err)
	}
	agg := makeAggregate(tenantID)
	evt := makeEvent(agg)
	_, err = wWriter.Write(context.Background(), tx, agg, evt, "CreateEnterprise")
	if err != nil {
		t.Fatalf("seed write failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("seed commit failed: %v", err)
	}

	var wg sync.WaitGroup
	var errs [2]error

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
			if err != nil {
				errs[idx] = fmt.Errorf("tx%d begin: %w", idx, err)
				return
			}
			agg := makeAggregate(tenantID)
			agg.Name = fmt.Sprintf("ChainConcurrent%d", idx)
			evt := makeEvent(agg)
			_, err = wWriter.Write(context.Background(), tx, agg, evt, "CreateEnterprise")
			if err != nil {
				errs[idx] = fmt.Errorf("tx%d write: %w", idx, err)
				_ = tx.Rollback()
				return
			}
			errs[idx] = tx.Commit()
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Errorf("concurrent tx %d failed: %v", i, e)
		}
	}

	records := wGetChainRecords(t, chainID)
	if len(records) != 3 {
		t.Fatalf("expected 3 chain records, got %d", len(records))
	}

	for i, r := range records {
		expectedSeq := int64(i + 1)
		if r.SequenceNo != expectedSeq {
			t.Errorf("record %d: expected seq=%d, got %d", i, expectedSeq, r.SequenceNo)
		}
	}

	if records[1].PreviousHash != records[0].EvidenceHash {
		t.Errorf("record 1 previousHash mismatch: expected %s, got %s", records[0].EvidenceHash, records[1].PreviousHash)
	}
	if records[2].PreviousHash != records[1].EvidenceHash {
		t.Errorf("record 2 previousHash mismatch: expected %s, got %s", records[1].EvidenceHash, records[2].PreviousHash)
	}
}

func TestIntegration_EvidenceChain_GenesisConcurrentSerialization(t *testing.T) {
	wCleanup(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	var wg sync.WaitGroup
	var errs [2]error

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			tx, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
			if err != nil {
				errs[idx] = fmt.Errorf("tx%d begin: %w", idx, err)
				return
			}
			agg := makeAggregate(tenantID)
			agg.Name = fmt.Sprintf("GenesisConcurrent%d", idx)
			evt := makeEvent(agg)
			_, err = wWriter.Write(context.Background(), tx, agg, evt, "CreateEnterprise")
			if err != nil {
				errs[idx] = fmt.Errorf("tx%d write: %w", idx, err)
				_ = tx.Rollback()
				return
			}
			errs[idx] = tx.Commit()
		}(i)
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Errorf("genesis concurrent tx %d failed: %v", i, e)
		}
	}

	records := wGetChainRecords(t, chainID)
	if len(records) != 2 {
		t.Fatalf("expected 2 chain records (genesis concurrent), got %d", len(records))
	}

	if records[0].SequenceNo != 1 {
		t.Errorf("first record: expected seq=1, got %d", records[0].SequenceNo)
	}
	if records[1].SequenceNo != 2 {
		t.Errorf("second record: expected seq=2, got %d", records[1].SequenceNo)
	}

	genHash := evidence.GenesisHash(chainID)
	if records[0].PreviousHash != genHash {
		t.Errorf("first record previousHash: expected genesis %s, got %s", genHash, records[0].PreviousHash)
	}
	if records[1].PreviousHash != records[0].EvidenceHash {
		t.Errorf("second record previousHash: expected %s, got %s", records[0].EvidenceHash, records[1].PreviousHash)
	}
}

func TestIntegration_EvidenceChain_RollbackNoGap(t *testing.T) {
	wCleanup(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	tx1, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
	if err != nil {
		t.Fatalf("tx1 begin failed: %v", err)
	}
	agg1 := makeAggregate(tenantID)
	evt1 := makeEvent(agg1)
	_, err = wWriter.Write(context.Background(), tx1, agg1, evt1, "CreateEnterprise")
	if err != nil {
		t.Fatalf("tx1 write failed: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("tx1 commit failed: %v", err)
	}

	tx2, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
	if err != nil {
		t.Fatalf("tx2 begin failed: %v", err)
	}
	agg2 := makeAggregate(tenantID)
	agg2.Name = "RollbackCorp"
	evt2 := makeEvent(agg2)
	_, err = wWriter.Write(context.Background(), tx2, agg2, evt2, "CreateEnterprise")
	if err != nil {
		t.Fatalf("tx2 write failed: %v", err)
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("tx2 rollback failed: %v", err)
	}

	records := wGetChainRecords(t, chainID)
	if len(records) != 1 {
		t.Fatalf("expected 1 record after rollback, got %d", len(records))
	}

	tx3, err := wRLS.BeginTenantTransaction(context.Background(), security.RLSContext{TenantID: tenantID})
	if err != nil {
		t.Fatalf("tx3 begin failed: %v", err)
	}
	agg3 := makeAggregate(tenantID)
	agg3.Name = "NoGapCorp"
	evt3 := makeEvent(agg3)
	_, err = wWriter.Write(context.Background(), tx3, agg3, evt3, "CreateEnterprise")
	if err != nil {
		t.Fatalf("tx3 write failed: %v", err)
	}
	if err := tx3.Commit(); err != nil {
		t.Fatalf("tx3 commit failed: %v", err)
	}

	records = wGetChainRecords(t, chainID)
	if len(records) != 2 {
		t.Fatalf("expected 2 records after no-gap write, got %d", len(records))
	}
	if records[0].SequenceNo != 1 {
		t.Errorf("first record: expected seq=1, got %d", records[0].SequenceNo)
	}
	if records[1].SequenceNo != 2 {
		t.Errorf("second record: expected seq=2 (no gap), got %d", records[1].SequenceNo)
	}
	if records[1].PreviousHash != records[0].EvidenceHash {
		t.Errorf("second record previousHash: expected %s, got %s", records[0].EvidenceHash, records[1].PreviousHash)
	}
}
