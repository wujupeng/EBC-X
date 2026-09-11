package handler

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/person"
)

func TestIntegration_AssignRole_CASConcurrencyConflict(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "CASAssignPerson",
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
			assignCmd := person.AssignRoleCommand{
				CommandID:       uuid.NewString(),
				PersonID:        personID,
				RoleID:          uuid.NewString(),
				ExpectedVersion: 1,
				TenantID:        tenantID,
			}
			_, errs[idx] = assignHandler.Handle(testCtx, assignCmd)
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
}

func TestIntegration_AssignRole_RoleIdUniqueness(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()
	_, orgID := createTestFixtures(t, tenantID)

	createCmd := person.CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "RoleUniquePerson",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	_, err := createHandler.Handle(testCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	personID := getPersonID(t, tenantID)
	roleID := uuid.NewString()

	assignCmd1 := person.AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		RoleID:          roleID,
		ExpectedVersion: 1,
		TenantID:        tenantID,
	}
	_, err = assignHandler.Handle(testCtx, assignCmd1)
	if err != nil {
		t.Fatalf("first assign role failed: %v", err)
	}

	assignCmd2 := person.AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        personID,
		RoleID:          roleID,
		ExpectedVersion: 2,
		TenantID:        tenantID,
	}
	_, err = assignHandler.Handle(testCtx, assignCmd2)
	if err == nil {
		t.Fatal("expected role already assigned error, got nil")
	}
	if !strings.Contains(err.Error(), "ROLE-ALREADY-ASSIGNED") {
		t.Errorf("expected ErrPersonRoleAlreadyAssigned, got: %v", err)
	}
}
