package handler

import (
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/wujupeng/ebcx/internal/enterprise"
)

func TestIntegration_CASConcurrencyConflict(t *testing.T) {
	cleanupTables(t)
	tenantID := uuid.NewString()

	createCmd := enterprise.CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "CASConcurrentCorp",
		TenantID:  tenantID,
	}
	_, err := createHandler.Handle(testCtx, createCmd)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	entID := getEnterpriseID(t, tenantID)

	var wg sync.WaitGroup
	var errs [2]error

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			updateCmd := enterprise.UpdateEnterpriseCommand{
				CommandID:       uuid.NewString(),
				EnterpriseID:    entID,
				TenantID:        tenantID,
				NewName:         "CASUpdatedCorp",
				ExpectedVersion: 1,
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
		} else if strings.Contains(errs[i].Error(), "VERSION-CONFLICT") {
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
	err = superDB.QueryRow(`SELECT version FROM business.enterprises WHERE enterprise_id::text = $1`, entID).Scan(&finalVersion)
	if err != nil {
		t.Fatalf("failed to query final version: %v", err)
	}
	if finalVersion != 2 {
		t.Errorf("expected final version=2, got %d", finalVersion)
	}
}
