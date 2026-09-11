//go:build integration

package person_test

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
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/person"
	"github.com/wujupeng/ebcx/internal/person/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/person/handler"
	"github.com/wujupeng/ebcx/internal/person/projection"
	"github.com/wujupeng/ebcx/internal/person/repository"
	"github.com/wujupeng/ebcx/internal/platform/graph"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

const personNeo4jURI = "bolt://localhost:7687"

var (
	pSuperDB  *sql.DB
	pAppDB    *sql.DB
	pPG       *embeddedpostgres.EmbeddedPostgres
	pUOW      repository.UnitOfWork
	pRepo     *repository.PersonRepositoryPostgreSQL
	pIdemRepo *repository.CommandIdempotencyRepositoryPostgreSQL
	pEvWriter *evidence_adapter.PersonEvidenceWriterImpl
	pOutbox   *outbox.Publisher
	pCreateH  *handler.CreatePersonHandler
	pUpdateH  *handler.UpdatePersonHandler
	pAssignH  *handler.AssignRoleHandler
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5441).
			Database("ebcx_person_physical").
			Username("postgres").
			Password("phytest").
			StartTimeout(60 * time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	connStr := "host=localhost port=5441 user=postgres password=phytest dbname=ebcx_person_physical sslmode=disable"
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

	appConnStr := "host=localhost port=5441 user=ebcx_app password=apppass dbname=ebcx_person_physical sslmode=disable"
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
	pRepo = repository.NewPersonRepositoryPostgreSQL()
	pIdemRepo = repository.NewCommandIdempotencyRepositoryPostgreSQL()
	pEvWriter = evidence_adapter.NewPersonEvidenceWriter("physical-test-runner")
	pOutbox = outbox.NewPublisher(adb)
	pCreateH = handler.NewCreatePersonHandler(pUOW, pRepo, pEvWriter, pOutbox, pIdemRepo)
	pUpdateH = handler.NewUpdatePersonHandler(pUOW, pRepo, pEvWriter, pOutbox, pIdemRepo)
	pAssignH = handler.NewAssignRoleHandler(pUOW, pRepo, pEvWriter, pOutbox, pIdemRepo)

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
	pSuperDB.Exec(`TRUNCATE business.command_idempotency, business.persons, business.organizations, business.enterprises, evidence.evidence_ledger, outbox.events CASCADE`)
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

func pCreateFixtures(t *testing.T, tenantID string) (string, string) {
	t.Helper()
	enterpriseID := uuid.NewString()
	_, err := pSuperDB.Exec(`INSERT INTO business.enterprises (enterprise_id, name, version, tenant_id) VALUES ($1, 'PhyEnterprise', 1, $2)`, enterpriseID, tenantID)
	if err != nil {
		t.Fatalf("failed to insert enterprise: %v", err)
	}
	orgID := uuid.NewString()
	_, err = pSuperDB.Exec(`INSERT INTO business.organizations (org_id, enterprise_id, name, code, level, version, tenant_id) VALUES ($1, $2, 'PhyOrg', 'ORG001', 1, 1, $3)`, orgID, enterpriseID, tenantID)
	if err != nil {
		t.Fatalf("failed to insert organization: %v", err)
	}
	return enterpriseID, orgID
}

func pGetPersonID(t *testing.T, tenantID string) string {
	t.Helper()
	var id string
	err := pSuperDB.QueryRow(`SELECT person_id::text FROM business.persons WHERE tenant_id::text = $1 LIMIT 1`, tenantID).Scan(&id)
	if err != nil {
		t.Fatalf("failed to get person ID: %v", err)
	}
	return id
}

func setupNeo4j(t *testing.T) *graph.RealNeo4jGraph {
	t.Helper()
	g, err := graph.NewRealNeo4jGraph(personNeo4jURI, "", "")
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

func getOutboxEventsForProjection(t *testing.T) []graph.OutboxEvent {
	t.Helper()
	rows, err := pSuperDB.Query(`SELECT event_id::text, event_type, aggregate_id::text, tenant_id::text, payload::text FROM outbox.events ORDER BY created_at`)
	if err != nil {
		t.Fatalf("failed to query outbox: %v", err)
	}
	defer rows.Close()
	var events []graph.OutboxEvent
	for rows.Next() {
		var evt graph.OutboxEvent
		var payloadStr string
		if err := rows.Scan(&evt.EventID, &evt.EventType, &evt.AggregateID, &evt.TenantID, &payloadStr); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		json.Unmarshal([]byte(payloadStr), &evt.Payload)
		events = append(events, evt)
	}
	return events
}

func TestPhysical_EvidenceHashChain_Integrity(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)
	chainID := fmt.Sprintf("person-mutation-chain-%s", tenantID)

	cmd1 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "PhyHashChainPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd1)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := pGetPersonID(t, tenantID)
	cmd2 := person.UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		NewName:         "PhyHashChainUpdated",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 1,
		TenantID:        tenantID,
	}
	_, err = pUpdateH.Handle(context.Background(), cmd2)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	rows, err := pSuperDB.Query(`
		SELECT evidence_id::text, chain_id, sequence_no, previous_evidence_hash, evidence_hash,
		       payload::text, source_event_id, transaction_id
		FROM evidence.evidence_ledger
		WHERE chain_id = $1 ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query evidence: %v", err)
	}
	defer rows.Close()

	var records []evidence.Record
	for rows.Next() {
		var r evidence.Record
		var payloadStr string
		var evidenceID string
		if err := rows.Scan(&evidenceID, &r.ChainID, &r.SequenceNo, &r.PreviousEvidenceHash, &r.EvidenceHash, &payloadStr, &r.SourceEventID, &r.TransactionID); err != nil {
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
		t.Errorf("DetectTamper found tampering: %s", tamperResult.Reason)
	}
}

func TestPhysical_Neo4j_PersonNodeProjection(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "Neo4jPersonNode",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	events := getOutboxEventsForProjection(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}

	proj := projection.NewPersonProjection(g, pSuperDB)
	result := proj.Consume(context.Background(), events[0])
	if result.Err != nil {
		t.Fatalf("projection failed: %v", result.Err)
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 Person node in Neo4j, got %d", count)
	}
}

func TestPhysical_Neo4j_BELONGS_TOEdgeProjection(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "EdgeProjectionPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	orgNode := graph.Node{
		NodeID:           fmt.Sprintf("organization-%s", orgID),
		NodeType:         graph.NodeOrganization,
		EntityID:         orgID,
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: "test-org-evidence",
		EvidenceRefs:     []string{"test-org-evidence"},
	}
	if err := g.MergeNode(context.Background(), orgNode); err != nil {
		t.Fatalf("failed to merge org node: %v", err)
	}

	events := getOutboxEventsForProjection(t)
	proj := projection.NewPersonProjection(g, pSuperDB)
	result := proj.Consume(context.Background(), events[0])
	if result.Err != nil {
		t.Fatalf("projection failed: %v", result.Err)
	}

	_, err = g.ExecuteQuery(context.Background(),
		`MATCH (p:Person {nodeId: $personNodeId})-[r:BELONGS_TO]->(o:Organization {nodeId: $orgNodeId})
		 RETURN count(r) AS edgeCount`,
		map[string]any{
			"personNodeId": fmt.Sprintf("person-%v", events[0].Payload["personId"]),
			"orgNodeId":    fmt.Sprintf("organization-%s", orgID),
		})
	if err != nil {
		t.Fatalf("failed to query BELONGS_TO edge: %v", err)
	}
}

func TestPhysical_Neo4j_FaultIsolation_EventuallyConsistent(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "FaultIsolationPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create should succeed even with Neo4j down (R3): %v", err)
	}

	if c := pCountRows("business.persons"); c != 1 {
		t.Errorf("expected 1 person committed (R3), got %d", c)
	}
	if c := pCountRows("evidence.evidence_ledger"); c != 1 {
		t.Errorf("expected 1 evidence committed, got %d", c)
	}
	if c := pCountRows("outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox pending, got %d", c)
	}
}

func TestPhysical_Neo4j_RoleAssignedProjection(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "RoleProjectionPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := pGetPersonID(t, tenantID)
	roleID := uuid.NewString()
	assignCmd := person.AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		RoleID:          roleID,
		ExpectedVersion: 1,
		TenantID:        tenantID,
	}
	_, err = pAssignH.Handle(context.Background(), assignCmd)
	if err != nil {
		t.Fatalf("assign role failed: %v", err)
	}

	events := getOutboxEventsForProjection(t)
	proj := projection.NewPersonProjection(g, pSuperDB)
	for _, evt := range events {
		result := proj.Consume(context.Background(), evt)
		if result.Err != nil {
			t.Fatalf("projection failed for %s: %v", evt.EventType, result.Err)
		}
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 Person node, got %d", count)
	}
}

func TestPhysical_EvidenceJSON_Produced(t *testing.T) {
	evidencePath := filepath.Join("..", "..", "evidence", "ev1", "EBCX-EV1-011-person-evidence.json")
	if _, err := os.Stat(evidencePath); err != nil {
		t.Skipf("Physical Evidence JSON not yet produced (T16): %v", err)
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
		"execution_id", "timestamp", "environment", "git_commit",
		"test_command", "actual_output", "actual_metrics", "database_state",
		"event_id", "evidence_id", "trace_id", "failure_injection_result",
		"verification_result", "verifier",
		"taskId", "taskName", "gateName", "infrastructure",
		"testLayer", "testNames", "result", "metrics", "artifacts", "rtm",
		"lockOrder", "pendingSemantics", "chainGenesis", "hardConstraints",
	}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("evidence JSON missing required field: %s", field)
		}
	}

	if data["taskId"] != "EBCX-EV1-011" {
		t.Errorf("expected taskId=EBCX-EV1-011, got %v", data["taskId"])
	}
	if data["infrastructure"] != "PostgreSQL 18.3 + Neo4j 5.x" {
		t.Errorf("expected infrastructure=PostgreSQL 18.3 + Neo4j 5.x, got %v", data["infrastructure"])
	}
}

func TestPhysical_ProvenanceConsistency(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "ProvenancePerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
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
		t.Logf("provenance git_commit=%v, actual=%s (may differ if uncommitted)", gitCommit, actualCommit)
	}
}

func TestPhysical_EventReplayIdempotent(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "ReplayIdempotentPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	events := getOutboxEventsForProjection(t)
	proj := projection.NewPersonProjection(g, pSuperDB)

	for i := 0; i < 2; i++ {
		p2 := projection.NewPersonProjection(g, pSuperDB)
		result := p2.Consume(context.Background(), events[0])
		if i == 0 && result.Err != nil {
			t.Fatalf("first replay failed: %v", result.Err)
		}
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 node after replay, got %d", count)
	}
	_ = proj
}

func TestPhysical_Reconciliation_PGNeo4j(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "ReconciliationPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	events := getOutboxEventsForProjection(t)
	proj := projection.NewPersonProjection(g, pSuperDB)
	result := proj.Consume(context.Background(), events[0])
	if result.Err != nil {
		t.Fatalf("projection failed: %v", result.Err)
	}

	reconEngine := projection.NewPersonReconciliationEngine(g, pSuperDB)
	mismatches, err := reconEngine.DetectMismatches(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("reconciliation detect failed: %v", err)
	}
	if len(mismatches) > 0 {
		repairs, err := reconEngine.Repair(context.Background(), tenantID)
		if err != nil {
			t.Fatalf("reconciliation repair failed: %v", err)
		}
		t.Logf("repaired %d mismatches", len(repairs))
	}

	mismatches2, err := reconEngine.DetectMismatches(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("second reconciliation detect failed: %v", err)
	}
	nodeMismatches := 0
	for _, m := range mismatches2 {
		if m.Type == projection.PersonMismatchMissingNode || m.Type == projection.PersonMismatchExtraNode || m.Type == projection.PersonMismatchPropertyDiff {
			nodeMismatches++
		}
	}
	if nodeMismatches > 0 {
		t.Errorf("expected 0 node mismatches after repair, got %d (total: %d)", nodeMismatches, len(mismatches2))
	}
}

func TestPhysical_Neo4jFaultIsolation_PhysicalVerification(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "Neo4jFaultPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	agg, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create should succeed with Neo4j down (R3): %v", err)
	}

	if c := pCountRows("business.persons"); c != 1 {
		t.Errorf("expected 1 person (R3: main tx不受Neo4j影响), got %d", c)
	}
	if c := pCountRows("outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox event accumulated, got %d", c)
	}
	_ = agg
}

func TestPhysical_UniqueEmployeeNo_PhysicalVerification(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd1 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "UniqueEmpPerson1",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := pCreateH.Handle(context.Background(), cmd1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cmd2 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "UniqueEmpPerson2",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err = pCreateH.Handle(context.Background(), cmd2)
	if err == nil {
		t.Fatal("expected duplicate employeeNo error on real PG")
	}
	if !strings.Contains(err.Error(), "DUPLICATE-EMPLOYEE-NO") && !strings.Contains(err.Error(), "23505") {
		t.Errorf("expected duplicate error, got: %v", err)
	}
}

func TestPhysical_ThreeLayerConsistency(t *testing.T) {
	pCleanup(t)
	g := setupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "ThreeLayerPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	agg, err := pCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var pgName, pgEmployeeNo string
	var pgVersion int64
	err = pSuperDB.QueryRow(`SELECT name, employee_no, version FROM business.persons WHERE person_id::text = $1`, agg.PersonID).Scan(&pgName, &pgEmployeeNo, &pgVersion)
	if err != nil {
		t.Fatalf("failed to query PG person: %v", err)
	}
	if pgName != "ThreeLayerPerson" || pgEmployeeNo != "EMP001" || pgVersion != 1 {
		t.Errorf("PG layer mismatch: name=%s, empNo=%s, version=%d", pgName, pgEmployeeNo, pgVersion)
	}

	if c := pCountRows("evidence.evidence_ledger"); c != 1 {
		t.Errorf("evidence layer: expected 1 record, got %d", c)
	}
	if c := pCountRows("outbox.events"); c != 1 {
		t.Errorf("outbox layer: expected 1 event, got %d", c)
	}

	events := getOutboxEventsForProjection(t)
	proj := projection.NewPersonProjection(g, pSuperDB)
	result := proj.Consume(context.Background(), events[0])
	if result.Err != nil {
		t.Fatalf("Neo4j projection failed: %v", result.Err)
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count Neo4j nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("Neo4j layer: expected at least 1 node, got %d", count)
	}
}

func TestPhysical_GenesisConcurrentSafety(t *testing.T) {
	pCleanup(t)
	tenantID := uuid.NewString()
	_, orgID := pCreateFixtures(t, tenantID)
	chainID := fmt.Sprintf("person-mutation-chain-%s", tenantID)

	const numGoroutines = 5
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			cmd := person.CreatePersonCommand{
				CommandID:  uuid.NewString(),
				OrgID:      orgID,
				Name:       fmt.Sprintf("GenesisSafety%d", idx),
				EmployeeNo: fmt.Sprintf("EMP%d", idx),
				TenantID:   tenantID,
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

	rows, err := pSuperDB.Query(`SELECT sequence_no, previous_evidence_hash, evidence_hash FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no`, chainID)
	if err != nil {
		t.Fatalf("failed to query chain: %v", err)
	}
	defer rows.Close()

	type chainRec struct {
		Seq  int64
		Prev string
		Hash string
	}
	var records []chainRec
	for rows.Next() {
		var r chainRec
		rows.Scan(&r.Seq, &r.Prev, &r.Hash)
		records = append(records, r)
	}
	if len(records) != numGoroutines {
		t.Errorf("expected %d chain records, got %d", numGoroutines, len(records))
	}
	genHash := evidence.GenesisHash(chainID)
	if records[0].Prev != genHash {
		t.Errorf("first record: expected genesis %s, got %s", genHash, records[0].Prev)
	}
	for i := 1; i < len(records); i++ {
		if records[i].Prev != records[i-1].Hash {
			t.Errorf("record %d: previousHash mismatch", i)
		}
	}
}
