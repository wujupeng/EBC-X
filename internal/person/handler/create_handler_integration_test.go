package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/evidence"
	"github.com/wujupeng/ebcx/internal/person"
)

func TestIntegration_SameTransactionAtomic(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "AtomicTestPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}

	agg, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if agg.Version != 1 {
		t.Errorf("expected version=1, got %d", agg.Version)
	}

	if c := countRows(t, "business.persons"); c != 1 {
		t.Errorf("expected 1 person row, got %d", c)
	}
	if c := countRows(t, "evidence.evidence_ledger"); c != 1 {
		t.Errorf("expected 1 evidence record, got %d", c)
	}
	if c := countRows(t, "outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox event, got %d", c)
	}
	if c := countRows(t, "business.command_idempotency"); c != 1 {
		t.Errorf("expected 1 idempotency record, got %d", c)
	}
}

func TestIntegration_UniqueEmployeeNoConstraint(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	cmd1 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "PersonOne",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd1)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	cmd2 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "PersonTwo",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err = createHandler.Handle(testCtx, cmd2)
	if err == nil {
		t.Fatal("expected duplicate employeeNo error, got nil")
	}
	if !strings.Contains(err.Error(), "DUPLICATE-EMPLOYEE-NO") && !strings.Contains(err.Error(), "23505") {
		t.Errorf("expected duplicate employeeNo error, got: %v", err)
	}
}

func TestIntegration_EvidenceLedgerRecordFields(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "EvidenceFieldTest",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var chainID, evidenceType, createdBy string
	var sequenceNo int64
	var payloadBytes []byte
	err = superDB.QueryRow(`
		SELECT chain_id, sequence_no, evidence_type, payload::text, created_by
		FROM evidence.evidence_ledger LIMIT 1
	`).Scan(&chainID, &sequenceNo, &evidenceType, &payloadBytes, &createdBy)
	if err != nil {
		t.Fatalf("failed to query evidence record: %v", err)
	}
	expectedChainID := fmt.Sprintf("person-mutation-chain-%s", tenantID)
	if chainID != expectedChainID {
		t.Errorf("expected chainID=%s, got %s", expectedChainID, chainID)
	}
	if sequenceNo != 1 {
		t.Errorf("expected sequenceNo=1, got %d", sequenceNo)
	}
	if evidenceType != "mandatory" {
		t.Errorf("expected evidence_type=mandatory, got %s", evidenceType)
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload["mutationType"] != "CREATE" {
		t.Errorf("expected payload.mutationType=CREATE, got %v", payload["mutationType"])
	}
	if payload["name"] != "EvidenceFieldTest" {
		t.Errorf("expected payload.name=EvidenceFieldTest, got %v", payload["name"])
	}
}

func TestIntegration_OutboxWriteAndPublish(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "OutboxTestPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var eventType, status string
	err = superDB.QueryRow(`SELECT event_type, status FROM outbox.events LIMIT 1`).Scan(&eventType, &status)
	if err != nil {
		t.Fatalf("failed to query outbox event: %v", err)
	}
	if eventType != "person.created" {
		t.Errorf("expected event_type=person.created, got %s", eventType)
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
	if payload["name"] != "OutboxTestPerson" {
		t.Errorf("expected name=OutboxTestPerson in payload, got %v", payload["name"])
	}

	var evidenceID string
	err = superDB.QueryRow(`SELECT evidence_id::text FROM evidence.evidence_ledger LIMIT 1`).Scan(&evidenceID)
	if err != nil {
		t.Fatalf("failed to query evidence_id: %v", err)
	}
	evRef, ok := payload["evidenceRef"]
	if !ok {
		t.Fatal("payload missing evidenceRef field")
	}
	if evRef.(string) != evidenceID {
		t.Errorf("evidenceRef mismatch: outbox=%v, evidence=%s", evRef, evidenceID)
	}
}

func TestIntegration_CreatePerson_OrganizationValidation(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	createTestFixtures(t, tenantID)

	nonExistentOrgID := uuid.NewString()
	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      nonExistentOrgID,
		Name:       "OrgValidationTest",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd)
	if err == nil {
		t.Fatal("expected organization not found error, got nil")
	}
	if !strings.Contains(err.Error(), "ORGANIZATION-NOT-FOUND") {
		t.Errorf("expected ErrPersonOrganizationNotFound, got: %v", err)
	}
}

func TestIntegration_EvidenceHashChain_Integrity(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)
	chainID := fmt.Sprintf("person-mutation-chain-%s", tenantID)

	cmd1 := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "HashChainPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, cmd1)
	if err != nil {
		t.Fatalf("first Handle failed: %v", err)
	}

	personID := getPersonID(t, tenantID)
	cmd2 := person.UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		NewName:         "HashChainPersonUpdated",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 1,
		TenantID:        tenantID,
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

func TestIntegration_CommandIdempotency_DuplicateReturnsFirstResult(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)
	cmdID := uuid.NewString()

	cmd := person.CreatePersonCommand{
		CommandID:  cmdID,
		OrgID:      orgID,
		Name:       "IdempotentPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
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
	if c := countRows(t, "evidence.evidence_ledger"); c != 1 {
		t.Errorf("expected 1 evidence record after duplicate, got %d", c)
	}
	if c := countRows(t, "outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox event after duplicate, got %d", c)
	}
}

func TestIntegration_RLSTenantIsolation(t *testing.T) {
	cleanupTables(t)
	tenantA := uuid.NewString()
	tenantB := uuid.NewString()
	_, orgA := createTestFixtures(t, tenantA)
	createTestFixtures(t, tenantB)

	cmdA := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgA,
		Name:       "TenantAPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantA,
	}
	_, err := createHandler.Handle(testCtx, cmdA)
	if err != nil {
		t.Fatalf("create for tenant A failed: %v", err)
	}

	tx, err := rlsManager.BeginTenantTransaction(testCtx, securityRLSCtx(tenantB))
	if err != nil {
		t.Fatalf("failed to begin tenant tx for B: %v", err)
	}
	var countB int
	err = tx.QueryRowContext(testCtx, `SELECT count(*) FROM business.persons`).Scan(&countB)
	tx.Rollback()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if countB != 0 {
		t.Errorf("tenant B should see 0 persons, got %d", countB)
	}

	tx2, err := rlsManager.BeginTenantTransaction(testCtx, securityRLSCtx(tenantA))
	if err != nil {
		t.Fatalf("failed to begin tenant tx for A: %v", err)
	}
	var countA int
	err = tx2.QueryRowContext(testCtx, `SELECT count(*) FROM business.persons`).Scan(&countA)
	tx2.Rollback()
	if err != nil {
		t.Fatalf("query failed for tenant A: %v", err)
	}
	if countA != 1 {
		t.Errorf("tenant A should see 1 person, got %d", countA)
	}
}

func TestIntegration_ThreeLayerConsistency(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	cmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "ConsistencyTestPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	agg, err := createHandler.Handle(testCtx, cmd)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var pgName, pgEmployeeNo string
	var pgVersion int64
	err = superDB.QueryRow(`
		SELECT name, employee_no, version FROM business.persons WHERE person_id::text = $1
	`, agg.PersonID).Scan(&pgName, &pgEmployeeNo, &pgVersion)
	if err != nil {
		t.Fatalf("failed to query person: %v", err)
	}
	if pgName != "ConsistencyTestPerson" {
		t.Errorf("PG name mismatch: expected ConsistencyTestPerson, got %s", pgName)
	}
	if pgEmployeeNo != "EMP001" {
		t.Errorf("PG employeeNo mismatch: expected EMP001, got %s", pgEmployeeNo)
	}
	if pgVersion != 1 {
		t.Errorf("PG version mismatch: expected 1, got %d", pgVersion)
	}

	if c := countRows(t, "evidence.evidence_ledger"); c != 1 {
		t.Errorf("expected 1 evidence record, got %d", c)
	}
	if c := countRows(t, "outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox event, got %d", c)
	}

	events := getOutboxEvents(t)
	var payload map[string]any
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal outbox payload: %v", err)
	}
	if payload["personId"] != agg.PersonID {
		t.Errorf("outbox personId mismatch: expected %s, got %v", agg.PersonID, payload["personId"])
	}
	if payload["name"] != pgName {
		t.Errorf("outbox name mismatch: expected %s, got %v", pgName, payload["name"])
	}
}
