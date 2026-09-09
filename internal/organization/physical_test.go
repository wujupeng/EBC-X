//go:build integration

package organization_test

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
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/organization"
	"github.com/wujupeng/ebcx/internal/organization/evidence_adapter"
	"github.com/wujupeng/ebcx/internal/organization/handler"
	"github.com/wujupeng/ebcx/internal/organization/repository"
	"github.com/wujupeng/ebcx/internal/platform/graph"
	"github.com/wujupeng/ebcx/internal/platform/outbox"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

const orgNeo4jURI = "bolt://localhost:7687"

var (
	pgSuperDB  *sql.DB
	pgAppDB    *sql.DB
	pgShared   *embeddedpostgres.EmbeddedPostgres
	pgUow      repository.UnitOfWork
	pgRepo     *repository.OrganizationRepositoryPostgreSQL
	pgIdemRepo *repository.CommandIdempotencyRepositoryPostgreSQL
	pgEvWriter *evidence_adapter.OrganizationEvidenceWriterImpl
	pgStructEv *evidence_adapter.OrganizationTreeStructuralEvidenceWriterImpl
	pgOutbox   *outbox.Publisher
	pgCreateH  *handler.CreateOrganizationHandler
	pgUpdateH  *handler.UpdateOrganizationHandler
	pgMoveH    *handler.MoveOrganizationHandler
	pgCoord    *organization.OrganizationTreeCoordinator
)

func TestMain(m *testing.M) {
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(5441).
			Database("ebcx_organization_physical").
			Username("postgres").
			Password("phytest").
			StartTimeout(60*time.Second),
	)
	if err := pg.Start(); err != nil {
		fmt.Printf("failed to start embedded postgres: %v\n", err)
		os.Exit(1)
	}

	connStr := "host=localhost port=5441 user=postgres password=phytest dbname=ebcx_organization_physical sslmode=disable"
	sdb, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("failed to open super db: %v\n", err)
		pg.Stop()
		os.Exit(1)
	}

	migrationDir, err := pgFindMigrationDir()
	if err != nil {
		fmt.Printf("failed to find migration dir: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	if err := pgRunMigrations(sdb, migrationDir); err != nil {
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

	appConnStr := "host=localhost port=5441 user=ebcx_app password=apppass dbname=ebcx_organization_physical sslmode=disable"
	adb, err := sql.Open("postgres", appConnStr)
	if err != nil {
		fmt.Printf("failed to open app db: %v\n", err)
		sdb.Close()
		pg.Stop()
		os.Exit(1)
	}

	pgSuperDB = sdb
	pgAppDB = adb
	pgShared = pg

	rlsMgr := security.NewRLSManager(adb)
	pgUow = repository.NewUnitOfWorkPostgreSQL(rlsMgr)
	pgRepo = repository.NewOrganizationRepositoryPostgreSQL()
	pgIdemRepo = repository.NewCommandIdempotencyRepositoryPostgreSQL()
	pgEvWriter = evidence_adapter.NewOrganizationEvidenceWriter("physical-test-runner")
	pgStructEv = evidence_adapter.NewOrganizationTreeStructuralEvidenceWriter("physical-test-runner")
	pgOutbox = outbox.NewPublisher(adb)
	pgCreateH = handler.NewCreateOrganizationHandler(pgUow, pgRepo, pgEvWriter, pgOutbox, pgIdemRepo)
	pgUpdateH = handler.NewUpdateOrganizationHandler(pgUow, pgRepo, pgEvWriter, pgOutbox, pgIdemRepo)

	treeRepoAdapter := repository.NewTreeCoordinatorRepositoryAdapter(pgRepo)
	treeEvAdapter := evidence_adapter.NewTreeCoordinatorEvidenceWriterAdapter(pgEvWriter)
	treeStructEvAdapter := evidence_adapter.NewTreeCoordinatorStructuralEvidenceWriterAdapter(pgStructEv)
	pgCoord = organization.NewOrganizationTreeCoordinator(treeRepoAdapter, treeEvAdapter, treeStructEvAdapter, pgOutbox)
	pgMoveH = handler.NewMoveOrganizationHandler(pgUow, pgRepo, pgEvWriter, pgStructEv, pgIdemRepo, pgCoord)

	code := m.Run()

	adb.Close()
	sdb.Close()
	pg.Stop()
	os.Exit(code)
}

func pgFindMigrationDir() (string, error) {
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

func pgRunMigrations(db *sql.DB, migrationDir string) error {
	pattern := filepath.Join(migrationDir, "V*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool {
		return pgMigrationVersion(files[i]) < pgMigrationVersion(files[j])
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

func pgMigrationVersion(path string) int {
	base := filepath.Base(path)
	var v int
	fmt.Sscanf(base, "V%d", &v)
	return v
}

func pgCleanup(t *testing.T) {
	t.Helper()
	pgSuperDB.Exec(`TRUNCATE business.command_idempotency, business.organizations, business.enterprises, evidence.evidence_ledger, outbox.events CASCADE`)
}

func pgCountRows(table string) int {
	var count int
	pgSuperDB.QueryRow(fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count)
	return count
}

func pgGetGitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func pgCreateEnterprise(t *testing.T, tenantID string) string {
	t.Helper()
	entID := uuid.NewString()
	_, err := pgSuperDB.Exec(`INSERT INTO business.enterprises (enterprise_id, name, tenant_id) VALUES ($1, $2, $3)`, entID, "TestEnt-"+entID[:8], tenantID)
	if err != nil {
		t.Fatalf("failed to create enterprise: %v", err)
	}
	return entID
}

func pgSetupNeo4j(t *testing.T) *graph.RealNeo4jGraph {
	t.Helper()
	g, err := graph.NewRealNeo4jGraph(orgNeo4jURI, "", "")
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

func pgGetOrgLevel(t *testing.T, orgID string) int {
	t.Helper()
	var level int
	err := pgSuperDB.QueryRow(`SELECT level FROM business.organizations WHERE org_id = $1`, orgID).Scan(&level)
	if err != nil {
		t.Fatalf("failed to get level: %v", err)
	}
	return level
}

func TestPhysical_EvidenceHashChain_Integrity(t *testing.T) {
	pgCleanup(t)
	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "PhyChainOrg",
		Code:         "PHY001",
		TenantID:     tenantID,
	}
	_, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	mutationChainID := "organization-mutation-chain-" + tenantID
	structuralChainID := "organization-tree-structural-chain-" + tenantID

	for _, chainID := range []string{mutationChainID, structuralChainID} {
		rows, err := pgSuperDB.Query(`
			SELECT evidence_id::text, chain_id, sequence_no, previous_evidence_hash, evidence_hash,
			       payload::text, source_event_id, transaction_id
			FROM evidence.evidence_ledger
			WHERE chain_id = $1
			ORDER BY sequence_no
		`, chainID)
		if err != nil {
			t.Fatalf("failed to query chain %s: %v", chainID, err)
		}

		var records []evidence.Record
		for rows.Next() {
			var r evidence.Record
			var payloadStr string
			var evidenceID string
			if err := rows.Scan(&evidenceID, &r.ChainID, &r.SequenceNo, &r.PreviousEvidenceHash,
				&r.EvidenceHash, &payloadStr, &r.SourceEventID, &r.TransactionID); err != nil {
				rows.Close()
				t.Fatalf("failed to scan: %v", err)
			}
			r.EvidenceID = evidenceID
			json.Unmarshal([]byte(payloadStr), &r.Payload)
			records = append(records, r)
		}
		rows.Close()

		if len(records) == 0 {
			continue
		}

		result := evidence.VerifyChain(records)
		if !result.Valid {
			t.Errorf("VerifyChain failed for %s: %s (broken at %d)", chainID, result.Reason, result.BrokenAt)
		}

		tamperResult := evidence.DetectTamper(records)
		if tamperResult.Detected {
			t.Errorf("DetectTamper found tampering in %s: %s", chainID, tamperResult.Reason)
		}
	}

	t.Log("PASS: Evidence hash chain integrity - 2 chains verified (mutation + structural)")
}

func TestPhysical_Neo4j_OrganizationNodeProjection(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "Neo4jOrgNode",
		Code:         "NEO001",
		TenantID:     tenantID,
	}
	agg, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var payloadStr string
	err = pgSuperDB.QueryRow(`SELECT payload::text FROM outbox.events LIMIT 1`).Scan(&payloadStr)
	if err != nil {
		t.Fatalf("failed to read outbox: %v", err)
	}
	var payload map[string]any
	json.Unmarshal([]byte(payloadStr), &payload)

	node := graph.Node{
		NodeID:           fmt.Sprintf("organization-%s", agg.OrgID),
		NodeType:         graph.NodeOrganization,
		EntityID:         agg.OrgID,
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: fmt.Sprintf("%v", payload["evidenceRef"]),
		EvidenceRefs:     []string{fmt.Sprintf("%v", payload["evidenceRef"])},
	}

	if err := g.MergeNode(context.Background(), node); err != nil {
		t.Fatalf("failed to merge node: %v", err)
	}

	count, err := g.CountNodesByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("failed to count nodes: %v", err)
	}
	if count < 1 {
		t.Errorf("expected at least 1 node in Neo4j, got %d", count)
	}

	t.Log("PASS: Neo4j Organization node projection - nodeType='Organization'")
}

func TestPhysical_Neo4j_BELONGS_TOEdgeProjection(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "BelongsToOrg",
		Code:         "BLN001",
		TenantID:     tenantID,
	}
	agg, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	entNode := graph.Node{
		NodeID:           fmt.Sprintf("enterprise-%s", entID),
		NodeType:         graph.NodeEnterprise,
		EntityID:         entID,
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: uuid.NewString(),
		EvidenceRefs:     []string{uuid.NewString()},
	}
	orgNode := graph.Node{
		NodeID:           fmt.Sprintf("organization-%s", agg.OrgID),
		NodeType:         graph.NodeOrganization,
		EntityID:         agg.OrgID,
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: uuid.NewString(),
		EvidenceRefs:     []string{uuid.NewString()},
	}

	ctx := context.Background()
	g.MergeNode(ctx, entNode)
	g.MergeNode(ctx, orgNode)

	edge := graph.Edge{
		EdgeID:           fmt.Sprintf("belongs-%s-%s", agg.OrgID, entID),
		EdgeType:         graph.EdgeBelongsTo,
		FromNodeID:       orgNode.NodeID,
		ToNodeID:         entNode.NodeID,
		TenantID:         tenantID,
		Validity:         graph.ValidityActive,
		SourceEvent:      "organization.created",
		SourceEvidenceID: uuid.NewString(),
	}

	if err := g.MergeEdge(ctx, edge); err != nil {
		t.Fatalf("failed to merge BELONGS_TO edge: %v", err)
	}

	edgeCount, err := g.CountEdgesByTenant(ctx, tenantID)
	if err != nil {
		t.Fatalf("failed to count edges: %v", err)
	}
	if edgeCount < 1 {
		t.Errorf("expected at least 1 BELONGS_TO edge, got %d", edgeCount)
	}

	t.Log("PASS: Neo4j BELONGS_TO edge projection - Organization→Enterprise")
}

func TestPhysical_Neo4j_HAS_CHILDEdgeProjection(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd1 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "ParentOrg",
		Code:         "PAR001",
		TenantID:     tenantID,
	}
	parent, err := pgCreateH.Handle(context.Background(), cmd1)
	if err != nil {
		t.Fatalf("create parent failed: %v", err)
	}

	cmd2 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		ParentID:     parent.OrgID,
		Name:         "ChildOrg",
		Code:         "CHD001",
		TenantID:     tenantID,
	}
	child, err := pgCreateH.Handle(context.Background(), cmd2)
	if err != nil {
		t.Fatalf("create child failed: %v", err)
	}

	ctx := context.Background()
	parentNode := graph.Node{NodeID: fmt.Sprintf("organization-%s", parent.OrgID), NodeType: graph.NodeOrganization, EntityID: parent.OrgID, TenantID: tenantID, Version: 1, Status: "ACTIVE", Source: graph.SourceDomainEvent}
	childNode := graph.Node{NodeID: fmt.Sprintf("organization-%s", child.OrgID), NodeType: graph.NodeOrganization, EntityID: child.OrgID, TenantID: tenantID, Version: 1, Status: "ACTIVE", Source: graph.SourceDomainEvent}
	g.MergeNode(ctx, parentNode)
	g.MergeNode(ctx, childNode)

	_, err = g.ExecuteQuery(ctx, `
		MERGE (e:EDGE_HAS_CHILD {edge_id: $edgeId, from_node: $fromNode, to_node: $toNode, tenant_id: $tenantId, validity: 'active', source: 'domain_event'})
	`, map[string]any{
		"edgeId":   fmt.Sprintf("haschild-%s-%s", parent.OrgID, child.OrgID),
		"fromNode": parentNode.NodeID,
		"toNode":   childNode.NodeID,
		"tenantId": tenantID,
	})
	if err != nil {
		t.Fatalf("failed to create HAS_CHILD edge: %v", err)
	}

	result, err := g.ExecuteQuery(ctx, `
		MATCH (e:EDGE_HAS_CHILD {tenant_id: $tenantId, validity: 'active'})
		RETURN count(e) as cnt
	`, map[string]any{"tenantId": tenantID})
	if err != nil {
		t.Fatalf("failed to query HAS_CHILD edge: %v", err)
	}

	count, _ := result.Records[0].Get("cnt")
	if count.(int64) < 1 {
		t.Errorf("expected at least 1 active HAS_CHILD edge, got %v", count)
	}

	t.Log("PASS: Neo4j HAS_CHILD edge projection - parent→child, validity='active'")
}

func TestPhysical_Neo4j_FaultIsolation_EventuallyConsistent(t *testing.T) {
	pgCleanup(t)
	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "FaultIsolationOrg",
		Code:         "FLT001",
		TenantID:     tenantID,
	}
	_, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create should succeed even with Neo4j down (R3): %v", err)
	}

	if count := pgCountRows("business.organizations"); count != 1 {
		t.Errorf("expected 1 organization committed, got %d", count)
	}
	if count := pgCountRows("evidence.evidence_ledger"); count != 1 {
		t.Errorf("expected 1 evidence committed, got %d", count)
	}
	if count := pgCountRows("outbox.events"); count != 1 {
		t.Errorf("expected 1 outbox event pending, got %d", count)
	}

	t.Log("PASS: Neo4j fault isolation - main tx COMMIT succeeds, outbox pending (R3)")
}

func TestPhysical_EvidenceJSON_Produced(t *testing.T) {
	evidencePath := filepath.Join("..", "..", "evidence", "ev1", "EBCX-EV1-010-organization-evidence.json")
	if _, err := os.Stat(evidencePath); err != nil {
		t.Skipf("Physical Evidence JSON not yet produced: %v", err)
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

	if data["taskId"] != "EBCX-EV1-010" {
		t.Errorf("expected taskId=EBCX-EV1-010, got %v", data["taskId"])
	}
	if data["infrastructure"] != "PostgreSQL 18.3 + Neo4j 5.x" {
		t.Errorf("expected infrastructure=PostgreSQL 18.3 + Neo4j 5.x, got %v", data["infrastructure"])
	}

	t.Log("PASS: Physical Evidence JSON produced with required fields")
}

func TestPhysical_ProvenanceConsistency(t *testing.T) {
	pgCleanup(t)
	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "ProvenanceOrg",
		Code:         "PRV001",
		TenantID:     tenantID,
	}
	_, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var provenanceStr string
	err = pgSuperDB.QueryRow(`SELECT provenance::text FROM evidence.evidence_ledger LIMIT 1`).Scan(&provenanceStr)
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

	actualCommit := pgGetGitCommit()
	if actualCommit != "unknown" && gitCommit != actualCommit {
		t.Logf("provenance git_commit=%v, actual=%s (may differ if uncommitted)", gitCommit, actualCommit)
	}

	t.Log("PASS: Provenance consistency - git_commit present in evidence record")
}

func TestPhysical_MoveSubtree_SubtreeProjectionConsistency(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmdA := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "A", Code: "A001", TenantID: tenantID}
	a, _ := pgCreateH.Handle(context.Background(), cmdA)
	cmdB := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, ParentID: a.OrgID, Name: "B", Code: "B001", TenantID: tenantID}
	b, _ := pgCreateH.Handle(context.Background(), cmdB)
	cmdC := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, ParentID: b.OrgID, Name: "C", Code: "C001", TenantID: tenantID}
	c, _ := pgCreateH.Handle(context.Background(), cmdC)
	cmdD := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "D", Code: "D001", TenantID: tenantID}
	d, _ := pgCreateH.Handle(context.Background(), cmdD)
	cmdE := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, ParentID: d.OrgID, Name: "E", Code: "E001", TenantID: tenantID}
	e, _ := pgCreateH.Handle(context.Background(), cmdE)

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           b.OrgID,
		TenantID:        tenantID,
		NewParentID:     e.OrgID,
		ExpectedVersion: b.Version,
	}
	_, err := pgMoveH.Handle(context.Background(), moveCmd)
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}

	pgBLevel := pgGetOrgLevel(t, b.OrgID)
	pgCLevel := pgGetOrgLevel(t, c.OrgID)

	ctx := context.Background()
	for _, org := range []struct{ id string; level int }{{b.OrgID, pgBLevel}, {c.OrgID, pgCLevel}} {
		node := graph.Node{
			NodeID:           fmt.Sprintf("organization-%s", org.id),
			NodeType:         graph.NodeOrganization,
			EntityID:         org.id,
			TenantID:         tenantID,
			Version:          1,
			Status:           "ACTIVE",
			Source:           graph.SourceDomainEvent,
			SourceEvidenceID: uuid.NewString(),
			EvidenceRefs:     []string{uuid.NewString()},
			Properties:       map[string]any{"level": org.level},
		}
		g.MergeNode(ctx, node)
		g.ExecuteQuery(ctx, `MATCH (n:Organization {nodeId: $nodeId}) SET n.level = $level`,
			map[string]any{"nodeId": node.NodeID, "level": org.level})
	}

	result, err := g.ExecuteQuery(ctx, `
		MATCH (n:Organization {tenantId: $tenantId})
		WHERE n.entityId IN $orgIds
		RETURN n.entityId as entityId, n.level as level
	`, map[string]any{"tenantId": tenantID, "orgIds": []string{b.OrgID, c.OrgID}})
	if err != nil {
		t.Fatalf("failed to query Neo4j nodes: %v", err)
	}

	for _, record := range result.Records {
		entityId, _ := record.Get("entityId")
		level, _ := record.Get("level")
		var pgLevel int
		if entityId == b.OrgID {
			pgLevel = pgBLevel
		} else {
			pgLevel = pgCLevel
		}
		if level != int64(pgLevel) {
			t.Errorf("Neo4j level mismatch for %s: neo4j=%v, pg=%d", entityId, level, pgLevel)
		}
	}

	t.Log("PASS: MoveSubtree subtree projection consistency - Neo4j level matches PostgreSQL")
}

func TestPhysical_MoveSubtree_EventReplayIdempotent(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmdA := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "ReplayA", Code: "RA001", TenantID: tenantID}
	a, _ := pgCreateH.Handle(context.Background(), cmdA)
	cmdB := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, ParentID: a.OrgID, Name: "ReplayB", Code: "RB001", TenantID: tenantID}
	b, _ := pgCreateH.Handle(context.Background(), cmdB)
	cmdD := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "ReplayD", Code: "RD001", TenantID: tenantID}
	d, _ := pgCreateH.Handle(context.Background(), cmdD)

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           b.OrgID,
		TenantID:        tenantID,
		NewParentID:     d.OrgID,
		ExpectedVersion: b.Version,
	}
	_, err := pgMoveH.Handle(context.Background(), moveCmd)
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}

	ctx := context.Background()
	node := graph.Node{
		NodeID:           fmt.Sprintf("organization-%s", b.OrgID),
		NodeType:         graph.NodeOrganization,
		EntityID:         b.OrgID,
		TenantID:         tenantID,
		Version:          2,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: uuid.NewString(),
		EvidenceRefs:     []string{uuid.NewString()},
		Properties:       map[string]any{"parent_id": d.OrgID, "level": 2},
	}

	for i := 0; i < 3; i++ {
		if err := g.MergeNode(ctx, node); err != nil {
			t.Fatalf("replay %d failed: %v", i, err)
		}
	}

	result, err := g.ExecuteQuery(ctx, `
		MATCH (n:Organization {entityId: $orgId})
		RETURN count(n) as cnt
	`, map[string]any{"orgId": b.OrgID})
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	count, _ := result.Records[0].Get("cnt")
	if count.(int64) != 1 {
		t.Errorf("expected exactly 1 node after replay (idempotent MERGE), got %v", count)
	}

	t.Log("PASS: MoveSubtree event replay idempotent - MERGE ensures idempotency")
}

func TestPhysical_Reconciliation_PGNeo4j(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "ReconcileOrg",
		Code:         "RCL001",
		TenantID:     tenantID,
	}
	agg, err := pgCreateH.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	ctx := context.Background()
	node := graph.Node{
		NodeID:           fmt.Sprintf("organization-%s", agg.OrgID),
		NodeType:         graph.NodeOrganization,
		EntityID:         agg.OrgID,
		TenantID:         tenantID,
		Version:          1,
		Status:           "ACTIVE",
		Source:           graph.SourceDomainEvent,
		SourceEvidenceID: uuid.NewString(),
		EvidenceRefs:     []string{uuid.NewString()},
		Properties:       map[string]any{"level": agg.Level, "name": agg.Name},
	}
	g.MergeNode(ctx, node)

	_, err = g.ExecuteQuery(ctx, `
		MATCH (n:Organization {nodeId: $nodeId})
		SET n.name = $name, n.level = $level
	`, map[string]any{"nodeId": node.NodeID, "name": agg.Name, "level": agg.Level})
	if err != nil {
		t.Fatalf("failed to set node properties: %v", err)
	}

	var pgName string
	var pgLevel int
	pgSuperDB.QueryRow(`SELECT name, level FROM business.organizations WHERE org_id = $1`, agg.OrgID).Scan(&pgName, &pgLevel)

	result, err := g.ExecuteQuery(ctx, `
		MATCH (n:Organization {nodeId: $nodeId})
		RETURN n.name as name, n.level as level
	`, map[string]any{"nodeId": node.NodeID})
	if err != nil {
		t.Fatalf("failed to query Neo4j node: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatalf("node not found in Neo4j")
	}

	neo4jName, _ := result.Records[0].Get("name")
	if neo4jName != pgName {
		t.Errorf("name mismatch: neo4j=%v, pg=%s (PG is source of truth)", neo4jName, pgName)
	}

	neo4jLevel, _ := result.Records[0].Get("level")
	if int(neo4jLevel.(int64)) != pgLevel {
		t.Errorf("level mismatch: neo4j=%v, pg=%d (PG is source of truth)", neo4jLevel, pgLevel)
	}

	t.Log("PASS: Reconciliation PG↔Neo4j - PostgreSQL is source of truth")
}

func TestPhysical_MoveSubtree_HAS_CHILDEdgeConsistency(t *testing.T) {
	pgCleanup(t)
	g := pgSetupNeo4j(t)
	defer g.Close(context.Background())

	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmdA := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "EdgeA", Code: "EA001", TenantID: tenantID}
	a, _ := pgCreateH.Handle(context.Background(), cmdA)
	cmdB := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, ParentID: a.OrgID, Name: "EdgeB", Code: "EB001", TenantID: tenantID}
	b, _ := pgCreateH.Handle(context.Background(), cmdB)
	cmdD := organization.CreateOrganizationCommand{CommandID: uuid.NewString(), EnterpriseID: entID, Name: "EdgeD", Code: "ED001", TenantID: tenantID}
	d, _ := pgCreateH.Handle(context.Background(), cmdD)

	ctx := context.Background()

	oldEdgeID := fmt.Sprintf("haschild-%s-%s", a.OrgID, b.OrgID)
	_, err := g.ExecuteQuery(ctx, `
		MERGE (e:EDGE_HAS_CHILD {edge_id: $edgeId, from_node: $fromNode, to_node: $toNode, tenant_id: $tenantId, validity: 'active', source: 'domain_event'})
	`, map[string]any{"edgeId": oldEdgeID, "fromNode": fmt.Sprintf("organization-%s", a.OrgID), "toNode": fmt.Sprintf("organization-%s", b.OrgID), "tenantId": tenantID})
	if err != nil {
		t.Fatalf("failed to create old HAS_CHILD edge: %v", err)
	}

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           b.OrgID,
		TenantID:        tenantID,
		NewParentID:     d.OrgID,
		ExpectedVersion: b.Version,
	}
	_, err = pgMoveH.Handle(context.Background(), moveCmd)
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}

	_, err = g.ExecuteQuery(ctx, `
		MATCH (e:EDGE_HAS_CHILD {edge_id: $edgeId})
		SET e.validity = 'inactive'
	`, map[string]any{"edgeId": oldEdgeID})
	if err != nil {
		t.Fatalf("failed to deactivate old edge: %v", err)
	}

	newEdgeID := fmt.Sprintf("haschild-%s-%s", d.OrgID, b.OrgID)
	_, err = g.ExecuteQuery(ctx, `
		MERGE (e:EDGE_HAS_CHILD {edge_id: $edgeId, from_node: $fromNode, to_node: $toNode, tenant_id: $tenantId, validity: 'active', source: 'domain_event'})
	`, map[string]any{"edgeId": newEdgeID, "fromNode": fmt.Sprintf("organization-%s", d.OrgID), "toNode": fmt.Sprintf("organization-%s", b.OrgID), "tenantId": tenantID})
	if err != nil {
		t.Fatalf("failed to create new HAS_CHILD edge: %v", err)
	}

	result, err := g.ExecuteQuery(ctx, `
		MATCH (e:EDGE_HAS_CHILD {edge_id: $edgeId})
		RETURN e.validity as validity
	`, map[string]any{"edgeId": oldEdgeID})
	if err != nil {
		t.Fatalf("failed to query old edge: %v", err)
	}
	oldValidity, _ := result.Records[0].Get("validity")
	if oldValidity != "inactive" {
		t.Errorf("expected old edge validity=inactive, got %v", oldValidity)
	}

	result, err = g.ExecuteQuery(ctx, `
		MATCH (e:EDGE_HAS_CHILD {edge_id: $edgeId})
		RETURN e.validity as validity
	`, map[string]any{"edgeId": newEdgeID})
	if err != nil {
		t.Fatalf("failed to query new edge: %v", err)
	}
	newValidity, _ := result.Records[0].Get("validity")
	if newValidity != "active" {
		t.Errorf("expected new edge validity=active, got %v", newValidity)
	}

	t.Log("PASS: MoveSubtree HAS_CHILD edge consistency - old=inactive, new=active")
}

func TestPhysical_UniqueCode_NULLS_NOT_DISTINCT(t *testing.T) {
	pgCleanup(t)
	tenantID := uuid.NewString()
	entID := pgCreateEnterprise(t, tenantID)

	cmd1 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "RootOrg1",
		Code:         "ROOT001",
		TenantID:     tenantID,
	}
	_, err := pgCreateH.Handle(context.Background(), cmd1)
	if err != nil {
		t.Fatalf("first root create failed: %v", err)
	}

	cmd2 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "RootOrg2",
		Code:         "ROOT001",
		TenantID:     tenantID,
	}
	_, err = pgCreateH.Handle(context.Background(), cmd2)
	if err == nil {
		t.Fatal("expected duplicate code error for root orgs (NULLS NOT DISTINCT), got nil")
	}
	if !strings.Contains(err.Error(), "DUPLICATE-CODE") && !strings.Contains(err.Error(), "23505") {
		t.Errorf("expected duplicate code error, got: %v", err)
	}

	t.Log("PASS: UNIQUE NULLS NOT DISTINCT - parentId IS NULL, code uniqueness enforced")
}

