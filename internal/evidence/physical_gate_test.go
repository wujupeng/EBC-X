package evidence

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

func startEmbeddedPostgres(t *testing.T) (*embeddedpostgres.EmbeddedPostgres, string) {
	t.Helper()
	_, repoRoot, _ := strings.Cut(os.Getenv("EBCX_REPO_ROOT"), "")
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
		for i := 0; i < 5; i++ {
			if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
				break
			}
			repoRoot = filepath.Dir(repoRoot)
		}
	}
	dbName := "ebcx_test"
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("postgres").
			Password("postgres").
			Database(dbName).
			Port(5433),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("failed to start embedded postgres: %v", err)
	}
	connStr := fmt.Sprintf("host=localhost port=5433 user=postgres password=postgres dbname=%s sslmode=disable", dbName)
	return pg, connStr
}

func execMigrations(t *testing.T, db *sql.DB, repoRoot string) {
	t.Helper()
	files := []string{
		"V1__create_schemas.sql",
		"V2__create_roles.sql",
		"V3__tenant_foundation.sql",
		"V4__evidence_ledger.sql",
	}
	for _, f := range files {
		path := filepath.Join(repoRoot, "db", "migrations", f)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := db.Exec(string(content)); err != nil {
			t.Fatalf("exec migration %s: %v", f, err)
		}
	}
}

func TestPhysicalGate_MigrationsAndPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping physical gate in short mode")
	}
	pg, connStr := startEmbeddedPostgres(t)
	defer pg.Stop()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	repoRoot, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}

	execMigrations(t, db, repoRoot)

	var schemaCount int
	if err := db.QueryRow("SELECT count(*) FROM information_schema.schemata WHERE schema_name IN ('business','evidence','outbox','audit','master_data','policy','tenant')").Scan(&schemaCount); err != nil {
		t.Fatalf("count schemas: %v", err)
	}
	if schemaCount != 7 {
		t.Fatalf("schema count = %d, want 7", schemaCount)
	}

	db.Exec("CREATE ROLE ebcx_app LOGIN PASSWORD 'apppass' IN ROLE ebcx_runtime")

	appConnStr := strings.Replace(connStr, "user=postgres password=postgres", "user=ebcx_app password=apppass", 1)
	appDB, err := sql.Open("postgres", appConnStr)
	if err != nil {
		t.Fatalf("connect as runtime: %v", err)
	}
	defer appDB.Close()

	genHash := GenesisHash("test")
	firstHash := ChainHash("test", 0, genHash, []byte(`{}`), "evt-0", "tx-0")
	appDB.Exec("SET app.tenant_id = '00000000-0000-0000-0000-000000000001'")
	if _, err := appDB.Exec("INSERT INTO evidence.evidence_ledger (chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by) VALUES ('test', 0, $1, $2, 'mandatory', '{}', 'evt-0', 'tx-0', '00000000-0000-0000-0000-000000000001', 'test')", genHash, firstHash); err != nil {
		t.Fatalf("INSERT as runtime failed (should succeed): %v", err)
	}

	if _, err := appDB.Exec("UPDATE evidence.evidence_ledger SET payload = '{\"hacked\":true}'"); err == nil {
		t.Fatal("UPDATE as runtime succeeded — TASK-H02 violation (should be rejected)")
	} else {
		t.Logf("✅ UPDATE rejected: %v", err)
	}

	if _, err := appDB.Exec("DELETE FROM evidence.evidence_ledger"); err == nil {
		t.Fatal("DELETE as runtime succeeded — TASK-H02 violation")
	} else {
		t.Logf("✅ DELETE rejected: %v", err)
	}

	if _, err := appDB.Exec("TRUNCATE evidence.evidence_ledger"); err == nil {
		t.Fatal("TRUNCATE as runtime succeeded — TASK-H02 violation")
	} else {
		t.Logf("✅ TRUNCATE rejected: %v", err)
	}
}

func TestPhysicalGate_EvidenceHashChainPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping physical gate in short mode")
	}
	pg, connStr := startEmbeddedPostgres(t)
	defer pg.Stop()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()

	repoRoot, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		repoRoot = filepath.Dir(repoRoot)
	}

	execMigrations(t, db, repoRoot)

	tenantID := "00000000-0000-0000-0000-000000000001"
	chainID := "physical-test-chain"
	prev := GenesisHash(chainID)

	for i := int64(1); i <= 5; i++ {
		payload := fmt.Sprintf(`{"seq":%d,"data":"evidence-%d"}`, i, i)
		hash := ChainHash(chainID, i, prev, []byte(payload), fmt.Sprintf("evt-%d", i), fmt.Sprintf("tx-%d", i))
		_, err := db.Exec(
			`INSERT INTO evidence.evidence_ledger (chain_id, sequence_no, previous_evidence_hash, evidence_hash, evidence_type, payload, source_event_id, transaction_id, tenant_id, created_by)
			 VALUES ($1, $2, $3, $4, 'mandatory', $5::jsonb, $6, $7, $8::uuid, 'physical-test')`,
			chainID, i, prev, hash, payload, fmt.Sprintf("evt-%d", i), fmt.Sprintf("tx-%d", i), tenantID,
		)
		if err != nil {
			t.Fatalf("insert evidence %d: %v", i, err)
		}
		prev = hash
	}

	var count int
	db.QueryRow("SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1", chainID).Scan(&count)
	if count != 5 {
		t.Fatalf("persisted count = %d, want 5", count)
	}
	t.Logf("✅ 5 evidence records persisted with hash chain")

	rows, err := db.Query("SELECT sequence_no, previous_evidence_hash, evidence_hash, payload::text FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no", chainID)
	if err != nil {
		t.Fatalf("query chain: %v", err)
	}
	var records []Record
	for rows.Next() {
		var seq int64
		var prevHash, hash, payloadStr string
		rows.Scan(&seq, &prevHash, &hash, &payloadStr)
		records = append(records, Record{
			EvidenceID:           fmt.Sprintf("seq-%d", seq),
			ChainID:              chainID,
			SequenceNo:           seq,
			PreviousEvidenceHash: prevHash,
			EvidenceHash:         hash,
		})
	}
	rows.Close()

	result := VerifyChain(records)
	if !result.Valid {
		t.Fatalf("persisted chain invalid: %s", result.Reason)
	}
	t.Logf("✅ Persisted hash chain verified: %d nodes, all links continuous", result.TotalNodes)

	if _, err := db.Exec("UPDATE evidence.evidence_ledger SET evidence_hash = 'ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff' WHERE chain_id = $1 AND sequence_no = 2", chainID); err == nil {
		t.Log("⚠️ Superuser UPDATE bypassed trigger (expected — superuser has BYPASSRLS/bypasses trigger via superuser)")
		rows2, _ := db.Query("SELECT sequence_no, previous_evidence_hash, evidence_hash FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no", chainID)
		var tampered []Record
		for rows2.Next() {
			var seq int64
			var ph, h string
			rows2.Scan(&seq, &ph, &h)
			tampered = append(tampered, Record{EvidenceID: fmt.Sprintf("s-%d", seq), ChainID: chainID, SequenceNo: seq, PreviousEvidenceHash: ph, EvidenceHash: h})
		}
		rows2.Close()
		vr := VerifyChain(tampered)
		if vr.Valid {
			t.Fatal("tampered chain (superuser modified hash) NOT detected by VerifyChain")
		}
		t.Logf("✅ Tamper detected after superuser modification: broken at sequence %d — %s", vr.BrokenAt, vr.Reason)
	}

	time.Sleep(100 * time.Millisecond)
}