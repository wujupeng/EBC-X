package handler

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/person"
)

func TestIntegration_UpdatePerson_CASConcurrencyConflict(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "CASPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := getPersonID(t, tenantID)

	var wg sync.WaitGroup
	var errs [2]error

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			updateCmd := person.UpdatePersonCommand{
				CommandID:       uuid.NewString(),
				PersonID:        personID,
				NewName:         "CASUpdatedPerson",
				NewEmployeeNo:   "EMP002",
				ExpectedVersion: 1,
				TenantID:        tenantID,
			}
			_, errs[idx] = updateHandler.Handle(testCtx, updateCmd)
		}(i)
	}
	wg.Wait()

	successCount := 0
	conflictCount := 0
	for i := 0; i < 2; i++ {
		if errs[i] == nil {
			successCount++
		} else if strings.Contains(errs[i].Error(), "CAS-CONCURRENCY-CONFLICT") || strings.Contains(errs[i].Error(), "VERSION-CONFLICT") || strings.Contains(errs[i].Error(), "23505") {
			conflictCount++
		}
	}

	if successCount != 1 {
		t.Errorf("expected exactly 1 CAS success, got %d (errs: %v, %v)", successCount, errs[0], errs[1])
	}
	if conflictCount != 1 {
		t.Errorf("expected exactly 1 CAS conflict, got %d (errs: %v, %v)", conflictCount, errs[0], errs[1])
	}

	var finalVersion int64
	err = superDB.QueryRow(`SELECT version FROM business.persons WHERE person_id::text = $1`, personID).Scan(&finalVersion)
	if err != nil {
		t.Fatalf("failed to query final version: %v", err)
	}
	if finalVersion != 2 {
		t.Errorf("expected final version=2, got %d", finalVersion)
	}
}

func TestIntegration_UpdatePerson_SameTransactionAtomic(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "UpdateAtomicPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := getPersonID(t, tenantID)
	updateCmd := person.UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		NewName:         "UpdatedName",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 1,
		TenantID:        tenantID,
	}
	agg, err := updateHandler.Handle(testCtx, updateCmd)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if agg.Version != 2 {
		t.Errorf("expected version=2, got %d", agg.Version)
	}

	if c := countRows(t, "business.persons"); c != 1 {
		t.Errorf("expected 1 person row, got %d", c)
	}
	if c := countRows(t, "evidence.evidence_ledger"); c != 2 {
		t.Errorf("expected 2 evidence records (create+update), got %d", c)
	}
	if c := countRows(t, "outbox.events"); c != 2 {
		t.Errorf("expected 2 outbox events (create+update), got %d", c)
	}
	if c := countRows(t, "business.command_idempotency"); c != 2 {
		t.Errorf("expected 2 idempotency records, got %d", c)
	}
}

func TestIntegration_EvidenceOutbox_SameTransactionAtomic(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "RollbackTestPerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := getPersonID(t, tenantID)

	duplicateCmd := person.UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		NewName:         "RollbackName",
		NewEmployeeNo:   "EMP001",
		ExpectedVersion: 99,
		TenantID:        tenantID,
	}
	_, err = updateHandler.Handle(testCtx, duplicateCmd)
	if err == nil {
		t.Fatal("expected CAS conflict error, got nil")
	}

	if c := countRows(t, "evidence.evidence_ledger"); c != 1 {
		t.Errorf("expected 1 evidence record (no new on rollback), got %d", c)
	}
	if c := countRows(t, "outbox.events"); c != 1 {
		t.Errorf("expected 1 outbox event (no new on rollback), got %d", c)
	}
}
