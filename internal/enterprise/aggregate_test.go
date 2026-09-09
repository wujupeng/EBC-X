package enterprise

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEnterpriseAggregate_CreateEnterprise_Invariants(t *testing.T) {
	agg := NewEnterpriseAggregate()

	t.Run("success", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: uuid.NewString(),
			Name:      "TestEnterprise",
			TenantID:  uuid.NewString(),
		}
		event, err := agg.CreateEnterprise(cmd)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if agg.Version != 1 {
			t.Errorf("expected version=1, got %d", agg.Version)
		}
		if event.EventType != "EnterpriseCreated" {
			t.Errorf("expected EventType=EnterpriseCreated, got %s", event.EventType)
		}
		if event.EnterpriseID != agg.EnterpriseID {
			t.Error("event EnterpriseID mismatch")
		}
	})

	t.Run("empty_name_rejected", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: uuid.NewString(),
			Name:      "",
			TenantID:  uuid.NewString(),
		}
		_, err := NewEnterpriseAggregate().CreateEnterprise(cmd)
		if err != ErrEnterpriseInvalidName {
			t.Errorf("expected ErrEnterpriseInvalidName, got %v", err)
		}
	})

	t.Run("name_too_long_rejected", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: uuid.NewString(),
			Name:      strings.Repeat("a", 257),
			TenantID:  uuid.NewString(),
		}
		_, err := NewEnterpriseAggregate().CreateEnterprise(cmd)
		if err != ErrEnterpriseInvalidName {
			t.Errorf("expected ErrEnterpriseInvalidName, got %v", err)
		}
	})

	t.Run("empty_command_id_rejected", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: "",
			Name:      "Test",
			TenantID:  uuid.NewString(),
		}
		_, err := NewEnterpriseAggregate().CreateEnterprise(cmd)
		if err != ErrEnterpriseCommandIDRequired {
			t.Errorf("expected ErrEnterpriseCommandIDRequired, got %v", err)
		}
	})

	t.Run("invalid_command_id_rejected", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: "not-a-uuid",
			Name:      "Test",
			TenantID:  uuid.NewString(),
		}
		_, err := NewEnterpriseAggregate().CreateEnterprise(cmd)
		if err != ErrEnterpriseInvalidCommandID {
			t.Errorf("expected ErrEnterpriseInvalidCommandID, got %v", err)
		}
	})

	t.Run("empty_tenant_id_rejected", func(t *testing.T) {
		cmd := CreateEnterpriseCommand{
			CommandID: uuid.NewString(),
			Name:      "Test",
			TenantID:  "",
		}
		_, err := NewEnterpriseAggregate().CreateEnterprise(cmd)
		if err != ErrTenantContextMismatch {
			t.Errorf("expected ErrTenantContextMismatch, got %v", err)
		}
	})
}

func TestEnterpriseAggregate_UpdateEnterprise_VersionMonotonic(t *testing.T) {
	agg := &EnterpriseAggregate{
		EnterpriseID: uuid.NewString(),
		Name:         "Original",
		Version:      3,
		TenantID:     uuid.NewString(),
	}

	t.Run("success_version_increment", func(t *testing.T) {
		cmd := UpdateEnterpriseCommand{
			CommandID:       uuid.NewString(),
			EnterpriseID:    agg.EnterpriseID,
			NewName:         "Updated",
			ExpectedVersion: 3,
		}
		event, err := agg.UpdateEnterprise(cmd)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if agg.Version != 4 {
			t.Errorf("expected version=4, got %d", agg.Version)
		}
		if event.EventType != "EnterpriseUpdated" {
			t.Errorf("expected EventType=EnterpriseUpdated, got %s", event.EventType)
		}
	})

	t.Run("zero_expected_version_rejected", func(t *testing.T) {
		cmd := UpdateEnterpriseCommand{
			CommandID:       uuid.NewString(),
			EnterpriseID:    agg.EnterpriseID,
			NewName:         "Updated",
			ExpectedVersion: 0,
		}
		_, err := agg.UpdateEnterprise(cmd)
		if err != ErrEnterpriseVersionMonotonic {
			t.Errorf("expected ErrEnterpriseVersionMonotonic, got %v", err)
		}
	})

	t.Run("negative_expected_version_rejected", func(t *testing.T) {
		cmd := UpdateEnterpriseCommand{
			CommandID:       uuid.NewString(),
			EnterpriseID:    agg.EnterpriseID,
			NewName:         "Updated",
			ExpectedVersion: -1,
		}
		_, err := agg.UpdateEnterprise(cmd)
		if err != ErrEnterpriseVersionMonotonic {
			t.Errorf("expected ErrEnterpriseVersionMonotonic, got %v", err)
		}
	})
}

func TestEnterpriseAggregate_EventFields(t *testing.T) {
	tenantID := uuid.NewString()
	cmd := CreateEnterpriseCommand{
		CommandID: uuid.NewString(),
		Name:      "EventTest",
		TenantID:  tenantID,
	}
	agg := NewEnterpriseAggregate()
	event, err := agg.CreateEnterprise(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if event.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if event.EventType != "EnterpriseCreated" {
		t.Errorf("expected EventType=EnterpriseCreated, got %s", event.EventType)
	}
	if event.EnterpriseID != agg.EnterpriseID {
		t.Error("event EnterpriseID mismatch")
	}
	if event.Name != agg.Name {
		t.Error("event Name mismatch")
	}
	if event.Version != agg.Version {
		t.Error("'event Version mismatch")
	}
	if event.TenantID != agg.TenantID {
		t.Error("event TenantID mismatch")
	}
	if event.Timestamp.IsZero() {
		t.Error("event Timestamp should not be zero")
	}
}

func TestEnterpriseAggregate_ErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"DuplicateName", ErrEnterpriseDuplicateName, "EBCX-ENTERPRISE-DUPLICATE-NAME"},
		{"NotFound", ErrEnterpriseNotFound, "EBCX-ENTERPRISE-NOT-FOUND"},
		{"VersionConflict", ErrEnterpriseVersionConflict, "EBCX-ENTERPRISE-VERSION-CONFLICT"},
		{"VersionMonotonic", ErrEnterpriseVersionMonotonic, "EBCX-ENTERPRISE-VERSION-MONOTONIC-VIOLATION"},
		{"InvalidName", ErrEnterpriseInvalidName, "EBCX-ENTERPRISE-INVALID-NAME"},
		{"InvalidCommandID", ErrEnterpriseInvalidCommandID, "EBCX-ENTERPRISE-INVALID-COMMAND-ID"},
		{"CommandIDRequired", ErrEnterpriseCommandIDRequired, "EBCX-ENTERPRISE-COMMAND-ID-REQUIRED"},
		{"IdempotencyConflict", ErrEnterpriseIdempotencyConflict, "EBCX-ENTERPRISE-IDEMPOTENCY-CONFLICT"},
		{"TenantContextMismatch", ErrTenantContextMismatch, "EBCX-TENANT-CONTEXT-MISMATCH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.err.Error())
			}
		})
	}
}
