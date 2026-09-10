package handler

import (
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/organization"
)

func createOrgTree(t *testing.T, tenantID, entID string, names ...string) []*organization.OrganizationAggregate {
	t.Helper()
	var aggs []*organization.OrganizationAggregate
	var parentID string
	for i, name := range names {
		cmd := organization.CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: entID,
			ParentID:     parentID,
			Name:         name,
			Code:         "CODE" + uuid.NewString()[:8],
			TenantID:     tenantID,
		}
		agg, err := orgCreateHandler.Handle(orgTestCtx, cmd)
		if err != nil {
			t.Fatalf("create org %d (%s) failed: %v", i, name, err)
		}
		aggs = append(aggs, agg)
		parentID = agg.OrgID
	}
	return aggs
}

func getOrgLevel(t *testing.T, orgID string) int {
	t.Helper()
	var level int
	err := orgSuperDB.QueryRow(`SELECT level FROM business.organizations WHERE org_id = $1`, orgID).Scan(&level)
	if err != nil {
		t.Fatalf("failed to get level for %s: %v", orgID, err)
	}
	return level
}

func getOrgVersion(t *testing.T, orgID string) int64 {
	t.Helper()
	var version int64
	err := orgSuperDB.QueryRow(`SELECT version FROM business.organizations WHERE org_id = $1`, orgID).Scan(&version)
	if err != nil {
		t.Fatalf("failed to get version for %s: %v", orgID, err)
	}
	return version
}

func TestIntegration_MoveSubtree_RecursiveUpdate(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	chainA := createOrgTree(t, tenantID, entID, "A", "B", "C")
	chainE := createOrgTree(t, tenantID, entID, "E", "F")

	A, B, C := chainA[0], chainA[1], chainA[2]
	E, F := chainE[0], chainE[1]

	_ = A
	_ = E

	if getOrgLevel(t, B.OrgID) != 2 {
		t.Fatalf("expected B level=2, got %d", getOrgLevel(t, B.OrgID))
	}
	if getOrgLevel(t, C.OrgID) != 3 {
		t.Fatalf("expected C level=3, got %d", getOrgLevel(t, C.OrgID))
	}

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           B.OrgID,
		TenantID:        tenantID,
		NewParentID:     F.OrgID,
		ExpectedVersion: B.Version,
	}
	_, err := orgMoveHandler.Handle(orgTestCtx, moveCmd)
	if err != nil {
		t.Fatalf("move B under F failed: %v", err)
	}

	if level := getOrgLevel(t, B.OrgID); level != 3 {
		t.Errorf("expected B level=3 (F.level+1), got %d", level)
	}
	if level := getOrgLevel(t, C.OrgID); level != 4 {
		t.Errorf("expected C level=4 (B.level+1, recursive), got %d", level)
	}

	t.Log("PASS: MoveSubtree recursive update - B level 2→3, C level 3→4 (descendant recursive)")
}

func TestIntegration_MoveSubtree_SubtreeLevelValidation(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	chainA := createOrgTree(t, tenantID, entID, "A", "B", "C")
	chainD := createOrgTree(t, tenantID, entID, "D", "E", "F", "G")

	A := chainA[0]
	G := chainD[3]

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           A.OrgID,
		TenantID:        tenantID,
		NewParentID:     G.OrgID,
		ExpectedVersion: A.Version,
	}
	_, err := orgMoveHandler.Handle(orgTestCtx, moveCmd)
	if err != organization.ErrOrgSubtreeLevelExceedMax {
		t.Errorf("expected ErrOrgSubtreeLevelExceedMax, got %v", err)
	}

	if level := getOrgLevel(t, A.OrgID); level != 1 {
		t.Errorf("A level should remain 1 after rejected move, got %d", level)
	}

	t.Log("PASS: MoveSubtree level validation - subtreeMaxDepth + newLevel > 5 rejected")
}

func TestIntegration_MoveSubtree_AtomicRollback(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	chainA := createOrgTree(t, tenantID, entID, "A", "B", "C")
	chainD := createOrgTree(t, tenantID, entID, "D")

	A, B, C := chainA[0], chainA[1], chainA[2]
	D := chainD[0]
	_ = A

	bOrigLevel := getOrgLevel(t, B.OrgID)
	cOrigLevel := getOrgLevel(t, C.OrgID)
	bOrigVersion := getOrgVersion(t, B.OrgID)
	cOrigVersion := getOrgVersion(t, C.OrgID)
	evCountBefore := countOrgRows(t, "evidence.evidence_ledger")

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           B.OrgID,
		TenantID:        tenantID,
		NewParentID:     D.OrgID,
		ExpectedVersion: B.Version + 999,
	}
	_, err := orgMoveHandler.Handle(orgTestCtx, moveCmd)
	if err == nil {
		t.Fatal("expected CAS conflict error, got nil")
	}

	if level := getOrgLevel(t, B.OrgID); level != bOrigLevel {
		t.Errorf("B level should remain %d after rollback, got %d", bOrigLevel, level)
	}
	if level := getOrgLevel(t, C.OrgID); level != cOrigLevel {
		t.Errorf("C level should remain %d after rollback, got %d", cOrigLevel, level)
	}
	if version := getOrgVersion(t, B.OrgID); version != bOrigVersion {
		t.Errorf("B version should remain %d after rollback, got %d", bOrigVersion, version)
	}
	if version := getOrgVersion(t, C.OrgID); version != cOrigVersion {
		t.Errorf("C version should remain %d after rollback, got %d", cOrigVersion, version)
	}

	if count := countOrgRows(t, "evidence.evidence_ledger"); count != evCountBefore {
		t.Errorf("expected %d evidence records after rollback (no new), got %d", evCountBefore, count)
	}

	t.Log("PASS: MoveSubtree atomic rollback - CAS conflict, no partial state change")
}

func TestIntegration_MoveSubtree_TwoEvidenceSameTransaction(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	chainA := createOrgTree(t, tenantID, entID, "A", "B")
	chainD := createOrgTree(t, tenantID, entID, "D")

	B := chainA[1]
	D := chainD[0]

	mutationChainID := "organization-mutation-chain-" + tenantID
	structuralChainID := "organization-tree-structural-chain-" + tenantID

	mutationCountBefore := 0
	structuralCountBefore := 0
	err := orgSuperDB.QueryRow(`SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1`, mutationChainID).Scan(&mutationCountBefore)
	if err != nil {
		t.Fatalf("failed to count mutation chain before: %v", err)
	}
	err = orgSuperDB.QueryRow(`SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1`, structuralChainID).Scan(&structuralCountBefore)
	if err != nil {
		t.Fatalf("failed to count structural chain before: %v", err)
	}

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           B.OrgID,
		TenantID:        tenantID,
		NewParentID:     D.OrgID,
		ExpectedVersion: B.Version,
	}
	_, err = orgMoveHandler.Handle(orgTestCtx, moveCmd)
	if err != nil {
		t.Fatalf("move B under D failed: %v", err)
	}

	var mutationCount, structuralCount int
	err = orgSuperDB.QueryRow(`SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1`, mutationChainID).Scan(&mutationCount)
	if err != nil {
		t.Fatalf("failed to count mutation chain: %v", err)
	}
	err = orgSuperDB.QueryRow(`SELECT count(*) FROM evidence.evidence_ledger WHERE chain_id = $1`, structuralChainID).Scan(&structuralCount)
	if err != nil {
		t.Fatalf("failed to count structural chain: %v", err)
	}

	if mutationCount != mutationCountBefore+1 {
		t.Errorf("expected %d evidence in mutation chain (before+1), got %d", mutationCountBefore+1, mutationCount)
	}
	if structuralCount != structuralCountBefore+1 {
		t.Errorf("expected %d evidence in structural chain (before+1), got %d", structuralCountBefore+1, structuralCount)
	}

	var mutationEventID, structuralEventID string
	err = orgSuperDB.QueryRow(`SELECT source_event_id FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no DESC LIMIT 1`, mutationChainID).Scan(&mutationEventID)
	if err != nil {
		t.Fatalf("failed to get latest mutation event_id: %v", err)
	}
	err = orgSuperDB.QueryRow(`SELECT source_event_id FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no DESC LIMIT 1`, structuralChainID).Scan(&structuralEventID)
	if err != nil {
		t.Fatalf("failed to get latest structural event_id: %v", err)
	}

	if mutationEventID != structuralEventID {
		t.Errorf("expected same source_event_id (correlation), got mutation=%s, structural=%s", mutationEventID, structuralEventID)
	}

	t.Log("PASS: MoveSubtree 2 Evidence same transaction - mutation chain + structural chain, same CorrelationID")
}

func TestIntegration_MoveSubtree_DescendantVersionNotIncrement(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	chainA := createOrgTree(t, tenantID, entID, "A", "B", "C")
	chainD := createOrgTree(t, tenantID, entID, "D", "E")

	B := chainA[1]
	C := chainA[2]
	E := chainD[1]

	bVersionBefore := getOrgVersion(t, B.OrgID)
	cVersionBefore := getOrgVersion(t, C.OrgID)

	moveCmd := organization.MoveOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           B.OrgID,
		TenantID:        tenantID,
		NewParentID:     E.OrgID,
		ExpectedVersion: B.Version,
	}
	_, err := orgMoveHandler.Handle(orgTestCtx, moveCmd)
	if err != nil {
		t.Fatalf("move B under E failed: %v", err)
	}

	bVersionAfter := getOrgVersion(t, B.OrgID)
	cVersionAfter := getOrgVersion(t, C.OrgID)

	if bVersionAfter != bVersionBefore+1 {
		t.Errorf("B version should increment from %d to %d, got %d", bVersionBefore, bVersionBefore+1, bVersionAfter)
	}
	if cVersionAfter != cVersionBefore {
		t.Errorf("C (descendant) version should NOT increment, expected %d, got %d", cVersionBefore, cVersionAfter)
	}

	t.Log("PASS: MoveSubtree descendant version not incremented (R1.3 / Q3-NORM)")
}
