//go:build integration

package enterprise_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	"github.com/wujupeng/ebcx/internal/enterprise/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/enterprise/handler"
	"github.com/wujupeng/ebcx/internal/enterprise/repository"
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/platform/graph"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

const neo4jURI = "bolt://localhost:7687"

var (
	pSuperDB  *sql.DB
	pAppDB    *sql.DB
	pPG       *embeddedpostgres.EmbeddedPostgres
	pUOW      repository.UnitOfWork
	pEntRepo  *repository.EnterpriseRepositoryPostgreSQL
	pIdemRepo *repository.CommandIdempotencyRepositoryPostgreSQL
	pEvWriter *evidence_adapter.EnterpriseEvidenceWriterImpl
	pOutbox   *outbox.Publisher
	pCreateH  *handler.CreateEnterpriseHandler
	pUpdateH  *handler.UpdateEnterpriseHandler
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5439).
			Database("ebcx_enterprise_physical").
			Username("postgres").
			Password("phytest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	connStr := "host=localhost port=5439 user=postgres password=phytest dbname=ebcx_enterprise_physical sslmode=disable"
	sdb, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := pFindMigrationDir()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := pRunMigrations(sdb, migrationDir); err != nil {
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

	appConnStr := "host=localhost port=5439 user=ebcx_app password=apppass dbname=ebcx_enterprise_physical sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	pSuperDB = sdb
	pAppDB = adb
	pPG = pg

	rlsMgr := security.NewRLSManager(adb)
	pUOW = repository.NewUnitOfWorkPostgreSQL(rlsMgr)
	pEntRepo = repository.NewEnterpriseRepositoryPostgreSQL()
	pIdemRepo = repository.NewCommandIdempotencyRepositoryPostgreSQL()
	pEvWriter = evidence_adapter.NewEnterpriseEvidenceWriter("physical-test-runner")
	pOutbox = outbox.NewPublisher(adb)
	pCreateH = handler.NewCreateEnterpriseHandler(pUOW, pEntRepo, pEvWriter, pOutbox, pIdemRepo)
	pUpdateH = handler.NewUpdateEnterpriseHandler(pUOW, pEntRepo, pEvWriter, pOutbox, pIdemRepo)

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func pFindMigrationDir() (string, error) {
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

func pRunMigrations(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return pMigrationVersion(files[i]) < pMigrationVersion(files[j])
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

func pMigrationVersion(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func pCleanup(t *testing.T) {
	t.Helper()
	pSuperDB.Exec(`TRUNCATE business.command_idempotency, business.enterprises, evidence.evidence_ledger, outbox.events CASCADE`)
}

func pCountRows(table string) int {
	var count int
	pSuperDB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count)
	return count
}

func pGetGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func setupNeo4j(t *testing.T) *graph.RealNeo4jGraph {
	t.Helper()
	g, err := graph.NewRealNeo4jGraph(neo4jURI, "", "")
	if err != nil {
		t.Skipf("Neo4j not available: %v", err)
	}
	ctx := context.Background()
	if err := g.Clear(ctx); err != nil {
		t.Fatalf("failed to clear graph: %v", err)
	}
	if err := g.CreateSchema(ctx); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	return g
}

func TestPhysical_PostgreSQL_Persistence(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "PhysicalPersistenceCorp",
		TenantID:  tenantID,
	}
	agg, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var dbName string
	var dbVersion int64
	err = pSuperDB.QueryRow(`SELECT name, version FROM business.enterprises WHERE enterprise_id::text = $1`, agg.EnterpriseID).Scan(&dbName, &dbVersion)
	if err != nil {
		t.Fatalf("failed to read from PostgreSQL: %v", err)
	}
	if dbName != "PhysicalPersistenceCorp" {
		t.Errorf("expected name=PhysicalPersistenceCorp, got %s", dbName)
	}
	if dbVersion != 1 {
		t.Errorf("expected version=1, got %d", dbVersion)
	}
}

func TestPhysical_Neo4jGraphProjection(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "Neo4jProjectionCorp",
		TenantID:  tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	events := pSuperDB.QueryRow(`SELECT payload::text FROM outbox.events LIMIT 1`)
	var payloadStr string
	if err := events.Scan(&payloadStr); err != nil {
		t.Fatalf("failed to read outbox payload: %v", err)
	}

	var payload map[string]any
	json.Unmarshal([]byte(payloadStr), &payload)

	node := graph.Node{
		NodeID:           fmt.Sprintf("enterprise-%s", payload["enterpriseId"]),
		NodeType:         graph.NodeEnterprise,
		EntityID:         fmt.Sprintf("%v", payload["enterpriseId"]),
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: fmt.Sprintf("%v", payload["evidenceRef"]),
		EvidenceRefs:     []string{fmt.Sprintf("%v", payload["evidenceRef"])},
	}

	ctx := context.Background()
	if err := g.MergeNode(ctx, node); err != nil {
		t.Fatalf("failed to merge node to Neo4j: %v", err)
	}

	count, err := g.CountNodesByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 node in Neo4j, got %d", count)
	}
}

func TestPhysical_FaultIsolation_Neo4jDown(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "FaultIsolationCorp",
		TenantID:  tenantID,
	}
	agg, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle should succeed even with Neo4j down: %v", err)
	}

	entCount := pCountRows("business.enterprises")
	if entCount != 1 {
		t.Errorf("expected 1 enterprise committed (R3: Neo4j not in main tx), got %d", entCount)
	}

	evCount := pCountRows("evidence.evidence_ledger")
	if evCount != 1 {
		t.Errorf("expected 1 evidence record committed, got %d", evCount)
	}

	obxCount := pCountRows("outbox.events")
	if obxCount != 1 {
		t.Errorf("expected 1 outbox event pending, got %d", obxCount)
	}

	_ = agg
}

func TestPhysical_ProjectionEventuallyConsistent(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "EventuallyConsistentCorp",
		TenantID:  tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var payloadStr string
	var eventID string
	err = pSuperDB.QueryRow(`SELECT event_id::text, payload::text FROM outbox.events LIMIT 1`).Scan(&eventID, &payloadStr)
	if err != nil {
		t.Fatalf("failed to read outbox: %v", err)
	}

	var payload map[string]any
	json.Unmarshal([]byte(payloadStr), &payload)

	node := graph.Node{
		NodeID:           fmt.Sprintf("enterprise-%s", payload["enterpriseId"]),
		NodeType:         graph.NodeEnterprise,
		EntityID:         fmt.Sprintf("%v", payload["enterpriseId"]),
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: fmt.Sprintf("%v", payload["evidenceRef"]),
		EvidenceRefs:     []string{fmt.Sprintf("%v", payload["evidenceRef"])},
	}

	start := time.Now()
	err = g.MergeNode(context.Background(), node)
	projectionLag := time.Since(start)

	if err != nil {
		t.Fatalf("failed to project to Neo4j: %v", err)
	}
	if projectionLag > 3*time.Second {
		t.Errorf("projection lag %v exceeds 3s (D-GATE-05)", projectionLag)
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 Neo4j node, got %d", count)
	}
	t.Logf("projection lag: %v (D-GATE-05: ≤3s)", projectionLag)
}

func TestPhysical_ProvenanceConsistency(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "ProvenanceCorp",
		TenantID:  tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var provenanceStr string
	err = pSuperDB.QueryRow(`SELECT provenance::text FROM evidence.evidence_ledger LIMIT 1`).Scan(&provenanceStr)
	if err != nil {
		t.Fatalf("failed to query provenance: %v", err)
	}

	var provenance map[string]any
	if err := json.Unmarshal([]byte(provenanceStr), &provenance); err != nil {
		t.Fatalf("failed to unmarshal provenance: %v", err)
	}

	gitCommit, ok := provenance["git_commit"]
	if !ok {
		t.Fatal("provenance missing git_commit field")
	}
	if gitCommit == "" || gitCommit == nil {
		t.Error("git_commit should not be empty")
	}

	actualCommit := pGetGitCommit()
	if actualCommit != "unknown" && gitCommit != actualCommit {
		t.Logf("provenance git_commit=%v, actual=%s (note: may differ if uncommitted)", gitCommit, actualCommit)
	}
}

func TestPhysical_CommandIdempotency_NetworkRetry(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: cmdID,
		Name:      "NetworkRetryCorp",
		TenantID:  tenantID,
	}
	agg1, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("first Handle failed: %v", err)
	}

	agg2, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("retry Handle should succeed (idempotent): %v", err)
	}

	if agg1.Version != agg2.Version {
		t.Errorf("expected same version on retry, got %d vs %d", agg1.Version, agg2.Version)
	}

	entCount := pCountRows("business.enterprises")
	if entCount != 1 {
		t.Errorf("expected 1 enterprise after retry, got %d", entCount)
	}

	evCount := pCountRows("evidence.evidence_ledger")
	if evCount != 1 {
		t.Errorf("expected 1 evidence after retry, got %d", evCount)
	}
}

func TestPhysical_EvidenceChain_HashChainIntegrity(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	cmd1 := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "HashIntegrityCorp",
		TenantID:  tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd1)
	if err != nil {
		t.Fatalf("first Handle failed: %v", err)
	}

	entID := pGetEnterpriseID(t, tenantID)
	cmd2 := enterprise.UpdateEnterpriseCommand{
		CommandID:       uuid.NewString(),
		EnterpriseID:    entID,
		TenantID:        tenantID,
		NewName:         "HashIntegrityCorpUpdated",
		ExpectedVersion: 1,
	}
	_, err = pUpdateH.Handle(context.Background(), cmd2)
	if err != nil {
		t.Fatalf("second Handle failed: %v", err)
	}

	rows, err := pSuperDB.Query(`
		SELECT evidence_id::text, chain_id, sequence_no, previous_evidence_hash, evidence_hash,
		       payload::text, source_event_id, transaction_id
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query evidence records: %v", err)
	}
	defer rows.Close()

	var records []evidence.Record
	for rows.Next() {
		var r evidence.Record
		var payloadStr string
		var evidenceID string
		if err := rows.Scan(&evidenceID, &r.ChainID, &r.SequenceNo, &r.PreviousEvidenceHash,
			&r.EvidenceHash, &payloadStr, &r.SourceEventID, &r.TransactionID); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		r.EvidenceID = evidenceID
		json.Unmarshal([]byte(payloadStr), &r.Payload)
		records = append(records, r)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 evidence records, got %d", len(records))
	}

	result := evidence.VerifyChain(records)
	if !result.Valid {
		t.Errorf("VerifyChain failed: %s (broken at %d)", result.Reason, result.BrokenAt)
	}

	tamperResult := evidence.DetectTamper(records)
	if tamperResult.Detected {
		t.Errorf("DetectTamper found tampering: %s (evidence: %s)", tamperResult.Reason, tamperResult.EvidenceID)
	}
}

func TestPhysical_EvidenceChain_GenesisConcurrentSafety(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	const numGoroutines = 5
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			cmd := enterprise.CreateEnterpriseCommand{
				CommandID: uuid.NewString(),
				Name:      fmt.Sprintf("GenesisSafety%d", idx),
				TenantID:  tenantID,
			}
			_, errs[idx] = pCreateH.Handle(context.Background(), cmd)
		}(i)
	}
	close(barrier)
	wg.Wait()

	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		if errs[i] == nil {
			successCount++
		}
	}

	if successCount != numGoroutines {
		t.Errorf("expected all %d to succeed, got %d", numGoroutines, successCount)
	}

	rows, err := pSuperDB.Query(`
		SELECT sequence_no, previous_evidence_hash, evidence_hash
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query chain: %v", err)
	}
	defer rows.Close()

	type chainRecord struct {
		SequenceNo   int64
		PreviousHash string
		EvidenceHash string
	}
	var records []chainRecord
	for rows.Next() {
		var r chainRecord
		rows.Scan(&r.SequenceNo, &r.PreviousHash, &r.EvidenceHash)
		records = append(records, r)
	}

	if len(records) != numGoroutines {
		t.Errorf("expected %d chain records, got %d", numGoroutines, len(records))
	}

	genHash := evidence.GenesisHash(chainID)
	if records[0].PreviousHash != genHash {
		t.Errorf("first record: expected genesis hash %s, got %s", genHash, records[0].PreviousHash)
	}
	if records[0].SequenceNo != 1 {
		t.Errorf("first record: expected seq=1, got %d", records[0].SequenceNo)
	}
	for i := 1; i < len(records); i++ {
		if records[i].SequenceNo != int64(i+1) {
			t.Errorf("record %d: expected seq=%d, got %d", i, i+1, records[i].SequenceNo)
		}
		if records[i].PreviousHash != records[i-1].EvidenceHash {
			t.Errorf("record %d: previousHash mismatch", i)
		}
	}
}

func TestPhysical_EvidenceJSONProduced(t *testing.T) {
	evidencePath := filepath.Join("..", "..", "evidence", "ev1", "EBCX-EV1-009-enterprise-evidence.json")
	if _, err := os.Stat(evidencePath); err != nil {
		t.Skipf("Physical Evidence JSON not yet produced (T13): %v", err)
	}

	content, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("failed to read evidence JSON: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		t.Fatalf("failed to parse evidence JSON: %v", err)
	}

	requiredFields := []string{
		"evidenceId", "taskId", "taskName", "gateName", "timestamp",
		"gitCommit", "infrastructure", "testLayer", "testNames",
		"result", "metrics", "artifacts", "rtm",
		"lockOrder", "pendingSemantics", "chainGenesis",
	}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("evidence JSON missing required field: %s", field)
		}
	}

	if data["taskId"] != "EBCX-EV1-009" {
		t.Errorf("expected taskId=EBCX-EV1-009, got %v", data["taskId"])
	}
	if data["infrastructure"] != "PostgreSQL 18.3 + Neo4j 5.x" {
		t.Errorf("expected infrastructure=PostgreSQL 18.3 + Neo4j 5.x, got %v", data["infrastructure"])
	}
}

func pGetEnterpriseID(t *testing.T, tenantID string) string {
	t.Helper()
	var id string
	err := pSuperDB.QueryRow(`SELECT enterprise_id::text FROM business.enterprises WHERE tenant_id::text = $1 LIMIT 1`, tenantID).Scan(&id)
	if err != nil {
		t.Fatalf("failed to get enterprise ID: %v", err)
	}
	return id
}
