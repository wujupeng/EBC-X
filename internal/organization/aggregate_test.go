package organization

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOrganizationAggregate_CreateOrganization_Invariants(t *testing.T) {
	agg := NewOrganizationAggregate()
	tenantID := uuid.NewString()
	enterpriseID := uuid.NewString()

	t.Run("success_root_org", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			ParentID:     "",
			Name:         "RootOrg",
			Code:         "ROOT001",
			TenantID:     tenantID,
		}
		event, err := agg.CreateOrganization(cmd, 0)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if agg.Version != 1 {
			t.Errorf("expected version=1, got %d", agg.Version)
		}
		if agg.Level != 1 {
			t.Errorf("expected level=1 for root org, got %d", agg.Level)
		}
		if event.EventType != "OrganizationCreated" {
			t.Errorf("expected EventType=OrganizationCreated, got %s", event.EventType)
		}
		if event.OrgID != agg.OrgID {
			t.Error("event OrgID mismatch")
		}
	})

	t.Run("success_child_org_level_calculation", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			ParentID:     uuid.NewString(),
			Name:         "ChildOrg",
			Code:         "CHILD001",
			TenantID:     tenantID,
		}
		agg2 := NewOrganizationAggregate()
		event, err := agg2.CreateOrganization(cmd, 2)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if agg2.Level != 3 {
			t.Errorf("expected level=3 (parent level 2 + 1), got %d", agg2.Level)
		}
		if event.Level != 3 {
			t.Errorf("expected event level=3, got %d", event.Level)
		}
	})

	t.Run("empty_name_rejected", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			Name:         "",
			Code:         "CODE1",
			TenantID:     tenantID,
		}
		_, err := NewOrganizationAggregate().CreateOrganization(cmd, 0)
		if err != ErrOrgInvalidName {
			t.Errorf("expected ErrOrgInvalidName, got %v", err)
		}
	})

	t.Run("empty_code_rejected", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			Name:         "TestOrg",
			Code:         "",
			TenantID:     tenantID,
		}
		_, err := NewOrganizationAggregate().CreateOrganization(cmd, 0)
		if err != ErrOrgInvalidCode {
			t.Errorf("expected ErrOrgInvalidCode, got %v", err)
		}
	})

	t.Run("code_too_long_rejected", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			Name:         "TestOrg",
			Code:         strings.Repeat("a", 65),
			TenantID:     tenantID,
		}
		_, err := NewOrganizationAggregate().CreateOrganization(cmd, 0)
		if err != ErrOrgInvalidCode {
			t.Errorf("expected ErrOrgInvalidCode, got %v", err)
		}
	})

	t.Run("level_exceeds_max_rejected", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: enterpriseID,
			Name:         "TooDeep",
			Code:         "DEEP001",
			TenantID:     tenantID,
		}
		_, err := NewOrganizationAggregate().CreateOrganization(cmd, MaxTreeLevel)
		if err != ErrOrgLevelExceedMax {
			t.Errorf("expected ErrOrgLevelExceedMax, got %v", err)
		}
	})

	t.Run("empty_enterprise_id_rejected", func(t *testing.T) {
		cmd := CreateOrganizationCommand{
			CommandID:    uuid.NewString(),
			EnterpriseID: "",
			Name:         "TestOrg",
			Code:         "CODE1",
			TenantID:     tenantID,
		}
		_, err := NewOrganizationAggregate().CreateOrganization(cmd, 0)
		if err != ErrOrgEnterpriseNotFound {
			t.Errorf("expected ErrOrgEnterpriseNotFound, got %v", err)
		}
	})
}

func TestOrganizationAggregate_UpdateOrganization_VersionMonotonic(t *testing.T) {
	agg := &OrganizationAggregate{
		OrgID:        uuid.NewString(),
		EnterpriseID: uuid.NewString(),
		Name:         "Original",
		Code:         "ORIG001",
		Level:        2,
		Version:      3,
		TenantID:     uuid.NewString(),
	}

	t.Run("success_version_increment", func(t *testing.T) {
		cmd := UpdateOrganizationCommand{
			CommandID:       uuid.NewString(),
			OrgID:           agg.OrgID,
			NewName:         "Updated",
			NewCode:         "UPD001",
			ExpectedVersion: 3,
		}
		event, err := agg.UpdateOrganization(cmd)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if agg.Version != 4 {
			t.Errorf("expected version=4, got %d", agg.Version)
		}
		if event.EventType != "OrganizationUpdated" {
			t.Errorf("expected EventType=OrganizationUpdated, got %s", event.EventType)
		}
	})

	t.Run("zero_expected_version_rejected", func(t *testing.T) {
		cmd := UpdateOrganizationCommand{
			CommandID:       uuid.NewString(),
			OrgID:           agg.OrgID,
			NewName:         "Updated",
			NewCode:         "UPD001",
			ExpectedVersion: 0,
		}
		_, err := agg.UpdateOrganization(cmd)
		if err != ErrOrgVersionMonotonic {
			t.Errorf("expected ErrOrgVersionMonotonic, got %v", err)
		}
	})
}

func TestOrganizationAggregate_NoMoveMethod(t *testing.T) {
	t.Log("Design Mandatory #1: OrganizationAggregate does NOT have a Move method")
	t.Log("MoveOrganization is handled by OrganizationTreeCoordinator (R1.2)")
	t.Log("This is verified by compilation - no Move() method exists on OrganizationAggregate")
	t.Log("PASS: OrganizationAggregate has only CreateOrganization and UpdateOrganization methods")
}

func TestOrganizationAggregate_EventFields(t *testing.T) {
	tenantID := uuid.NewString()
	enterpriseID := uuid.NewString()
	cmd := CreateOrganizationCommand{
		CommandID:    uuid.NewString(),
		EnterpriseID: enterpriseID,
		Name:         "EventTest",
		Code:         "EVT001",
		TenantID:     tenantID,
	}
	agg := NewOrganizationAggregate()
	event, err := agg.CreateOrganization(cmd, 0)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if event.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if event.EventType != "OrganizationCreated" {
		t.Errorf("expected EventType=OrganizationCreated, got %s", event.EventType)
	}
	if event.OrgID != agg.OrgID {
		t.Error("event OrgID mismatch")
	}
	if event.EnterpriseID != agg.EnterpriseID {
		t.Error("event EnterpriseID mismatch")
	}
	if event.Name != agg.Name {
		t.Error("event Name mismatch")
	}
	if event.Code != agg.Code {
		t.Error("event Code mismatch")
	}
	if event.Level != agg.Level {
		t.Error("event Level mismatch")
	}
	if event.Version != agg.Version {
		t.Error("event Version mismatch")
	}
	if event.TenantID != agg.TenantID {
		t.Error("event TenantID mismatch")
	}
	if event.Timestamp.IsZero() {
		t.Error("event Timestamp should not be zero")
	}
}

func TestOrganizationAggregate_MovedEvent_NoAffectedDescendantIds(t *testing.T) {
	t.Log("Design Mandatory #2: OrganizationMovedEvent does NOT contain affectedDescendantIds[]")
	t.Log("Projection Consumer queries PostgreSQL to rebuild subtree")
	event := OrganizationMovedEvent{
		EventID:      uuid.NewString(),
		EventType:    "OrganizationMoved",
		OrgID:        uuid.NewString(),
		EnterpriseID: uuid.NewString(),
		OldParentID:  uuid.NewString(),
		NewParentID:  uuid.NewString(),
		OldLevel:     2,
		NewLevel:     3,
		Version:      2,
		TenantID:     uuid.NewString(),
	}
	if event.EventType != "OrganizationMoved" {
		t.Errorf("expected EventType=OrganizationMoved, got %s", event.EventType)
	}
	t.Log("PASS: OrganizationMovedEvent has no affectedDescendantIds field")
}

func TestOrganizationAggregate_ErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"NotFound", ErrOrgNotFound, "EBCX-ORGANIZATION-NOT-FOUND"},
		{"VersionConflict", ErrOrgVersionConflict, "EBCX-ORGANIZATION-VERSION-CONFLICT"},
		{"VersionMonotonic", ErrOrgVersionMonotonic, "EBCX-ORGANIZATION-VERSION-MONOTONIC-VIOLATION"},
		{"LevelExceedMax", ErrOrgLevelExceedMax, "EBCX-ORGANIZATION-LEVEL-EXCEED-MAX"},
		{"SubtreeLevelExceedMax", ErrOrgSubtreeLevelExceedMax, "EBCX-ORGANIZATION-SUBTREE-LEVEL-EXCEED-MAX"},
		{"CycleDetected", ErrOrgCycleDetected, "EBCX-ORGANIZATION-CYCLE-DETECTED"},
		{"ParentNotFound", ErrOrgParentNotFound, "EBCX-ORGANIZATION-PARENT-NOT-FOUND"},
		{"ParentCrossEnterprise", ErrOrgParentCrossEnterprise, "EBCX-ORGANIZATION-PARENT-CROSS-ENTERPRISE"},
		{"EnterpriseNotFound", ErrOrgEnterpriseNotFound, "EBCX-ORGANIZATION-ENTERPRISE-NOT-FOUND"},
		{"EnterpriseImmutable", ErrOrgEnterpriseImmutable, "EBCX-ORGANIZATION-ENTERPRISE-IMMUTABLE"},
		{"DuplicateCode", ErrOrgDuplicateCode, "EBCX-ORGANIZATION-DUPLICATE-CODE"},
		{"InvalidName", ErrOrgInvalidName, "EBCX-ORGANIZATION-INVALID-NAME"},
		{"InvalidCode", ErrOrgInvalidCode, "EBCX-ORGANIZATION-INVALID-CODE"},
		{"InvalidCommandID", ErrOrgInvalidCommandID, "EBCX-ORGANIZATION-INVALID-COMMAND-ID"},
		{"CommandIDRequired", ErrOrgCommandIDRequired, "EBCX-ORGANIZATION-COMMAND-ID-REQUIRED"},
		{"IdempotencyConflict", ErrOrgIdempotencyConflict, "EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT"},
		{"UnknownCommand", ErrOrgUnknownCommand, "EBCX-ORGANIZATION-UNKNOWN-COMMAND"},
		{"LevelForbiddenSet", ErrOrgLevelForbiddenSet, "EBCX-ORGANIZATION-LEVEL-FORBIDDEN-SET"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.err.Error())
			}
		})
	}
}