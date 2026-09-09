package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/enterprise"
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/platform/security"
)

func TestIntegration_SameTransactionAtomic(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "AtomicTestCorp",
		TenantID:  tenantID,
	}

	agg, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if agg.Version != 1 {
		t.Errorf("expected version=1, got %d", agg.Version)
	}

	entCount := countRows(t, "business.enterprises")
	if entCount != 1 {
		t.Errorf("expected 1 enterprise row, got %d", entCount)
	}

	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)
	evCount := countRows(t, "evidence.evidence_ledger")
	if evCount != 1 {
		t.Errorf("expected 1 evidence record, got %d", evCount)
	}

	obxCount := countRows(t, "outbox.events")
	if obxCount != 1 {
		t.Errorf("expected 1 outbox event, got %d", obxCount)
	}

	idemCount := countRows(t, "business.command_idempotency")
	if idemCount != 1 {
		t.Errorf("expected 1 idempotency record, got %d", idemCount)
	}

	_ = chainID
}

func TestIntegration_EvidenceHashChain(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	chainID := fmt.Sprintf("enterprise-mutation-chain-%s", tenantID)

	cmd1 := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "HashChainCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd1)
	if err != nil {
		t.Fatalf("first Handle failed: %v", err)
	}

	cmd2 := enterprise.UpdateEnterpriseCommand{
		CommandID:       uuid.NewString(),
		EnterpriseID:    getEnterpriseID(t, tenantID),
		TenantID:        tenantID,
		NewName:         "HashChainCorpUpdated",
		ExpectedVersion: 1,
	}
	_, err = updateHandler.Handle(testCtx, cmd2)
	if err != nil {
		t.Fatalf("second Handle failed: %v", err)
	}

	records := getEvidenceRecords(t, chainID)
	if len(records) != 2 {
		t.Fatalf("expected 2 evidence records, got %d", len(records))
	}

	genHash := evidence.GenesisHash(chainID)
	if records[0].PreviousHash != genHash {
		t.Errorf("first record previousHash: expected genesis %s, got %s", genHash, records[0].PreviousHash)
	}
	if records[0].SequenceNo != 1 {
		t.Errorf("first record sequenceNo: expected 1, got %d", records[0].SequenceNo)
	}
	if records[1].PreviousHash != records[0].EvidenceHash {
		t.Errorf("second record previousHash: expected %s, got %s", records[0].EvidenceHash, records[1].PreviousHash)
	}
	if records[1].SequenceNo != 2 {
		t.Errorf("second record sequenceNo: expected 2, got %d", records[1].SequenceNo)
	}
}

func TestIntegration_OutboxWriteAndPublish(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "OutboxTestCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var eventType string
	var status string
	err = superDB.QueryRow(`SELECT event_type, status FROM outbox.events LIMIT 1`).Scan(&eventType, &status)
	if err != nil {
		t.Fatalf("failed to query outbox event: %v", err)
	}
	if eventType != "enterprise.created" {
		t.Errorf("expected event_type=enterprise.created, got %s", eventType)
	}
	if status != "pending" {
		t.Errorf("expected status=pending, got %s", status)
	}

	events := getOutboxEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}
	var payload map[string]any
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload["name"] != "OutboxTestCorp" {
		t.Errorf("expected name=OutboxTestCorp in payload, got %v", payload["name"])
	}
}

func TestIntegration_RLSTenantIsolation(t *testing.T) {
	cleanupTables(t)
	tenantA := uuid.NewString()
	tenantB := uuid.NewString()

	cmdA := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "TenantACorp",
		TenantID:  tenantA,
	}
	_, err := createHandler.Handle(testCtx, cmdA)
	if err != nil {
		t.Fatalf("create for tenant A failed: %v", err)
	}

	var countB int
	err = appDB.QueryRow(`
		SET LOCAL app.tenant_id = $1;
		SELECT count(*) FROM business.enterprises;
	`, tenantB).Scan(&countB)
	if err == nil {
		if countB != 0 {
			t.Errorf("tenant B should see 0 enterprises, got %d", countB)
		}
	}

	tx, err := rlsManager.BeginTenantTransaction(testCtx, securityRLSCtx(tenantB))
	if err != nil {
		t.Fatalf("failed to begin tenant tx for B: %v", err)
	}
	var countB2 int
	err = tx.QueryRowContext(testCtx, `SELECT count(*) FROM business.enterprises`).Scan(&countB2)
	tx.Rollback()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if countB2 != 0 {
		t.Errorf("tenant B should see 0 enterprises via RLS, got %d", countB2)
	}

	tx2, err := rlsManager.BeginTenantTransaction(testCtx, securityRLSCtx(tenantA))
	if err != nil {
		t.Fatalf("failed to begin tenant tx for A: %v", err)
	}
	var countA int
	err = tx2.QueryRowContext(testCtx, `SELECT count(*) FROM business.enterprises`).Scan(&countA)
	tx2.Rollback()
	if err != nil {
		t.Fatalf("query failed for tenant A: %v", err)
	}
	if countA != 1 {
		t.Errorf("tenant A should see 1 enterprise, got %d", countA)
	}
}

func TestIntegration_UniqueNameConstraint(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	cmd1 := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "UniqueNameCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cmd2 := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "UniqueNameCorp",
		TenantID:  tenantID,
	}
	_, err = createHandler.Handle(testCtx, cmd2)
	if err == nil {
		t.Fatal("expected duplicate name error, got nil")
	}
	if !strings.Contains(err.Error(), "DUPLICATE-NAME") && !strings.Contains(err.Error(), "23505") {
		t.Errorf("expected duplicate name error, got: %v", err)
	}
}

func TestIntegration_CommandIdempotency_DuplicateReturnsFirstResult(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: cmdID,
		Name:      "IdempotentCorp",
		TenantID:  tenantID,
	}
	agg1, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("first Handle failed: %v", err)
	}

	agg2, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("second Handle failed: %v", err)
	}

	if agg1.Version != agg2.Version {
		t.Errorf("expected same version on duplicate, got %d vs %d", agg1.Version, agg2.Version)
	}

	evCount := countRows(t, "evidence.evidence_ledger")
	if evCount != 1 {
		t.Errorf("expected 1 evidence record after duplicate, got %d", evCount)
	}

	obxCount := countRows(t, "outbox.events")
	if obxCount != 1 {
		t.Errorf("expected 1 outbox event after duplicate, got %d", obxCount)
	}
}

func TestIntegration_CommandIdempotency_SameTransactionAtomic(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "AtomicIdemCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	entCount := countRows(t, "business.enterprises")
	evCount := countRows(t, "evidence.evidence_ledger")
	obxCount := countRows(t, "outbox.events")
	idemCount := countRows(t, "business.command_idempotency")

	if entCount != 1 {
		t.Errorf("expected 1 enterprise, got %d", entCount)
	}
	if evCount != 1 {
		t.Errorf("expected 1 evidence, got %d", evCount)
	}
	if obxCount != 1 {
		t.Errorf("expected 1 outbox, got %d", obxCount)
	}
	if idemCount != 1 {
		t.Errorf("expected 1 idempotency, got %d", idemCount)
	}

	var idemStatus string
	err = superDB.QueryRow(`SELECT status FROM business.command_idempotency LIMIT 1`).Scan(&idemStatus)
	if err != nil {
		t.Fatalf("failed to query idempotency status: %v", err)
	}
	if idemStatus != "success" {
		t.Errorf("expected idempotency status=success, got %s", idemStatus)
	}
}

func TestIntegration_CommandIdempotency_ConcurrentSameCommandId(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	cmdID := uuid.NewString()

	const numGoroutines = 10
	barrier := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, numGoroutines)
	aggs := make([]*enterprise.EnterpriseAggregate, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-barrier
			cmd := enterprise.CreateEnterpriseCommand{
				CommandID: cmdID,
				Name:      "ConcurrentIdemCorp",
				TenantID:  tenantID,
			}
			agg, err := createHandler.Handle(testCtx, cmd)
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

	entCount := countRows(t, "business.enterprises")
	evCount := countRows(t, "evidence.evidence_ledger")
	obxCount := countRows(t, "outbox.events")
	idemCount := countRows(t, "business.command_idempotency")

	t.Logf("success=%d, conflict=%d, ent=%d, ev=%d, obx=%d, idem=%d", successCount, conflictCount, entCount, evCount, obxCount, idemCount)

	if entCount != 1 {
		t.Errorf("expected exactly 1 enterprise, got %d", entCount)
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

func TestIntegration_OutboxEvidenceRef_Contract(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	cmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "EvidenceRefCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var evidenceID string
	err = superDB.QueryRow(`
		SELECT evidence_id::text FROM evidence.evidence_ledger LIMIT 1
	`).Scan(&evidenceID)
	if err != nil {
		t.Fatalf("failed to query evidence_id: %v", err)
	}

	events := getOutboxEvents(t)
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}

	var payload map[string]any
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal outbox payload: %v", err)
	}

	evRef, ok := payload["evidenceRef"]
	if !ok {
		t.Fatal("payload missing evidenceRef field")
	}
	evRefStr, ok := evRef.(string)
	if !ok {
		t.Fatalf("evidenceRef is not string: %T", evRef)
	}
	if evRefStr != evidenceID {
		t.Errorf("evidenceRef mismatch: outbox=%s, evidence=%s", evRefStr, evidenceID)
	}

	if _, err := uuid.Parse(evRefStr); err != nil {
		t.Errorf("evidenceRef is not valid UUID v4: %s", evRefStr)
	}
}

func getEnterpriseID(t *testing.T, tenantID string) string {
	t.Helper()
	var id string
	err := superDB.QueryRow(`
		SELECT enterprise_id::text FROM business.enterprises WHERE tenant_id::text = $1 LIMIT 1
	`, tenantID).Scan(&id)
	if err != nil {
		t.Fatalf("failed to get enterprise ID: %v", err)
	}
	return id
}

func securityRLSCtx(tenantID string) security.RLSContext {
	return security.RLSContext{TenantID: tenantID, UserID: "test-user"}
}
