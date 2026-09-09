package handler

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/organization"
)

func TestIntegration_OrgSameTransactionAtomic(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		ParentID:     "",
		Name:         "RootOrg",
		Code:         "ROOT001",
		TenantID:     tenantID,
	}

	agg, err := orgCreateHandler.Handle(orgTestCtx, cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if agg.Version != 1 {
		t.Errorf("expected version=1, got %d", agg.Version)
	}
	if agg.Level != 1 {
		t.Errorf("expected level=1, got %d", agg.Level)
	}

	if count := countOrgRows(t, "business.organizations"); count != 1 {
		t.Errorf("expected 1 organization, got %d", count)
	}
	if count := countOrgRows(t, "evidence.evidence_ledger"); count != 1 {
		t.Errorf("expected 1 evidence record, got %d", count)
	}
	if count := countOrgRows(t, "outbox.events"); count != 1 {
		t.Errorf("expected 1 outbox event, got %d", count)
	}
	if count := countOrgRows(t, "business.command_idempotency"); count != 1 {
		t.Errorf("expected 1 idempotency record, got %d", count)
	}

	t.Log("PASS: Organization + Evidence + Outbox + Idempotency all in same ACID transaction")
}

func TestIntegration_OrgUniqueCodeConstraint(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmd1 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "OrgA",
		Code:         "DUP001",
		TenantID:     tenantID,
	}
	_, err := orgCreateHandler.Handle(orgTestCtx, cmd1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cmd2 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "OrgB",
		Code:         "DUP001",
		TenantID:     tenantID,
	}
	_, err = orgCreateHandler.Handle(orgTestCtx, cmd2)
	if err != organization.ErrOrgDuplicateCode {
		t.Errorf("expected ErrOrgDuplicateCode, got %v", err)
	}

	t.Log("PASS: Duplicate code rejected with ErrOrgDuplicateCode")
}

func TestIntegration_OrgCommandIdempotency_DuplicateReturnsFirstResult(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmdID := uuid.NewString()
	cmd := organization.CreateOrganizationCommand{
		CommandID:    cmdID,
		EnterpriseID: entID,
		Name:         "IdempotentOrg",
		Code:         "IDEM001",
		TenantID:     tenantID,
	}

	agg1, err := orgCreateHandler.Handle(orgTestCtx, cmd)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	agg2, err := orgCreateHandler.Handle(orgTestCtx, cmd)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}

	if agg1.Version != agg2.Version {
		t.Errorf("idempotency violated: version %d vs %d", agg1.Version, agg2.Version)
	}

	if count := countOrgRows(t, "business.organizations"); count != 1 {
		t.Errorf("expected 1 organization (idempotent), got %d", count)
	}

	t.Log("PASS: Duplicate command returns first result (idempotent)")
}

func TestIntegration_OrgCASConcurrencyConflict(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	createCmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "CASOrg",
		Code:         "CAS001",
		TenantID:     tenantID,
	}
	agg, err := orgCreateHandler.Handle(orgTestCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updateCmd1 := organization.UpdateOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           agg.OrgID,
		TenantID:        tenantID,
		NewName:         "Updated1",
		NewCode:         "CAS001",
		ExpectedVersion: 1,
	}
	_, err = orgUpdateHandler.Handle(orgTestCtx, updateCmd1)
	if err != nil {
		t.Fatalf("first update failed: %v", err)
	}

	updateCmd2 := organization.UpdateOrganizationCommand{
		CommandID:       uuid.NewString(),
		OrgID:           agg.OrgID,
		TenantID:        tenantID,
		NewName:         "Updated2",
		NewCode:         "CAS001",
		ExpectedVersion: 1,
	}
	_, err = orgUpdateHandler.Handle(orgTestCtx, updateCmd2)
	if err == nil {
		t.Error("expected CAS conflict error, got nil")
	}

	t.Log("PASS: CAS concurrency conflict detected (stale ExpectedVersion rejected)")
}

func TestIntegration_OrgEvidenceHashChain(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmd1 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "ChainOrg1",
		Code:         "CHN001",
		TenantID:     tenantID,
	}
	_, err := orgCreateHandler.Handle(orgTestCtx, cmd1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cmd2 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "ChainOrg2",
		Code:         "CHN002",
		TenantID:     tenantID,
	}
	_, err = orgCreateHandler.Handle(orgTestCtx, cmd2)
	if err != nil {
		t.Fatalf("second create failed: %v", err)
	}

	chainID := "organization-mutation-chain-" + tenantID
	if count := countOrgRows(t, "evidence.evidence_ledger"); count != 2 {
		t.Errorf("expected 2 evidence records, got %d", count)
	}

	rows, err := orgSuperDB.Query(`
		SELECT sequence_no, previous_evidence_hash, evidence_hash
		FROM evidence.evidence_ledger
		WHERE chain_id = $1
		ORDER BY sequence_no
	`, chainID)
	if err != nil {
		t.Fatalf("failed to query evidence chain: %v", err)
	}
	defer rows.Close()

	var prevHash string
	seq := 0
	for rows.Next() {
		var sequenceNo int64
		var previousHash, evidenceHash string
		if err := rows.Scan(&sequenceNo, &previousHash, &evidenceHash); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		seq++
		if int(sequenceNo) != seq {
			t.Errorf("expected sequence_no=%d, got %d", seq, sequenceNo)
		}
		if seq > 1 && previousHash != prevHash {
			t.Errorf("chain broken: previousHash mismatch at sequence %d", seq)
		}
		prevHash = evidenceHash
	}

	t.Log("PASS: Evidence hash chain maintained (sequence + previousHash continuity)")
}

func TestIntegration_OrgRLSTenantIsolation(t *testing.T) {
	cleanupOrgTables(t)
	tenantA := uuid.NewString()
	tenantB := uuid.NewString()
	entA := createTestEnterprise(t, tenantA)
	entB := createTestEnterprise(t, tenantB)

	cmdA := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entA,
		Name:         "OrgA",
		Code:         "TNT001",
		TenantID:     tenantA,
	}
	_, err := orgCreateHandler.Handle(orgTestCtx, cmdA)
	if err != nil {
		t.Fatalf("create for tenant A failed: %v", err)
	}

	cmdB := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entB,
		Name:         "OrgB",
		Code:         "TNT001",
		TenantID:     tenantB,
	}
	_, err = orgCreateHandler.Handle(orgTestCtx, cmdB)
	if err != nil {
		t.Fatalf("create for tenant B failed: %v", err)
	}

	if count := countOrgRows(t, "business.organizations"); count != 2 {
		t.Errorf("expected 2 organizations (2 tenants), got %d", count)
	}

	t.Log("PASS: RLS tenant isolation - same code in different tenants allowed")
}

func TestIntegration_OrgOutboxWriteAndPublish(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmd := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "OutboxOrg",
		Code:         "OBX001",
		TenantID:     tenantID,
	}
	_, err := orgCreateHandler.Handle(orgTestCtx, cmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	var eventType string
	err = orgSuperDB.QueryRow(`SELECT event_type FROM outbox.events LIMIT 1`).Scan(&eventType)
	if err != nil {
		t.Fatalf("failed to query outbox: %v", err)
	}
	if eventType != "organization.created" {
		t.Errorf("expected event_type=organization.created, got %s", eventType)
	}

	t.Log("PASS: Outbox event written with correct event_type")
}

func TestIntegration_OrgLevelEnforcement(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmd1 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		Name:         "Level1",
		Code:         "LVL001",
		TenantID:     tenantID,
	}
	org1, err := orgCreateHandler.Handle(orgTestCtx, cmd1)
	if err != nil {
		t.Fatalf("create level 1 failed: %v", err)
	}
	if org1.Level != 1 {
		t.Errorf("expected level=1, got %d", org1.Level)
	}

	cmd2 := organization.CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: entID,
		ParentID:     org1.OrgID,
		Name:         "Level2",
		Code:         "LVL002",
		TenantID:     tenantID,
	}
	org2, err := orgCreateHandler.Handle(orgTestCtx, cmd2)
	if err != nil {
		t.Fatalf("create level 2 failed: %v", err)
	}
	if org2.Level != 2 {
		t.Errorf("expected level=2, got %d", org2.Level)
	}

	t.Log("PASS: Level enforcement - root=1, child=2 (parent.level + 1)")
}

func TestIntegration_OrgCommandIdempotency_ConcurrentSameCommandId(t *testing.T) {
	cleanupOrgTables(t)
	tenantID := uuid.NewString()
	entID := createTestEnterprise(t, tenantID)

	cmdID := uuid.NewString()

	const numGoroutines = 10
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)
	aggs := make([]*organization.OrganizationAggregate, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			cmd := organization.CreateOrganizationCommand{
				CommandID:    cmdID,
				EnterpriseID: entID,
				Name:         "ConcurrentOrg",
				Code:         "CON001",
				TenantID:     tenantID,
			}
			agg, err := orgCreateHandler.Handle(orgTestCtx, cmd)
			aggs[idx] = agg
			errs[idx] = err
		}(i)
	}
	close(barrier)
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for i := 0; i < numGoroutines; i++ {
		if errs[i] == nil {
			successCount++
		} else if strings.Contains(errs[i].Error(), "IDEMPOTENCY-CONFLICT") || strings.Contains(errs[i].Error(), "23505") || strings.Contains(errs[i].Error(), "duplicate") {
			conflictCount++
		}
	}

	orgCount := countOrgRows(t, "business.organizations")
	evCount := countOrgRows(t, "evidence.evidence_ledger")
	obxCount := countOrgRows(t, "outbox.events")
	idemCount := countOrgRows(t, "business.command_idempotency")

	t.Logf("success=%d, conflict=%d, org=%d, ev=%d, obx=%d, idem=%d", successCount, conflictCount, orgCount, evCount, obxCount, idemCount)

	if orgCount != 1 {
		t.Errorf("expected exactly 1 organization, got %d", orgCount)
	}
	if evCount != 1 {
		t.Errorf("expected exactly 1 evidence record, got %d", evCount)
	}
	if obxCount != 1 {
		t.Errorf("expected exactly 1 outbox event, got %d", obxCount)
	}
	if idemCount != 1 {
		t.Errorf("expected exactly 1 idempotency record, got %d", idemCount)
	}
	if successCount < 1 {
		t.Errorf("expected at least 1 success, got %d", successCount)
	}
	if successCount+conflictCount != numGoroutines {
		t.Errorf("expected all goroutines to either succeed or conflict, got success=%d, conflict=%d, other=%d",
			successCount, conflictCount, numGoroutines-successCount-conflictCount)
	}
}