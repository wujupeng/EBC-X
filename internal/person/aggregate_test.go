package person

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPersonAggregate_CreatePerson_Success(t *testing.T) {
	agg := NewPersonAggregate()
	cmd := CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "张三",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{{RoleID: uuid.NewString()}},
		TenantID:   uuid.NewString(),
	}
	event, err := agg.CreatePerson(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if agg.Version != 1 {
		t.Errorf("expected version=1, got %d", agg.Version)
	}
	if event.EventType != "PersonCreated" {
		t.Errorf("expected EventType=PersonCreated, got %s", event.EventType)
	}
	if event.PersonID != agg.PersonID {
		t.Error("event PersonID mismatch")
	}
	if len(agg.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(agg.Roles))
	}
}

func TestPersonAggregate_CreatePerson_OrgIDRequired(t *testing.T) {
	cmd := CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      "",
		Name:       "Test",
		EmployeeNo: "EMP001",
		TenantID:   uuid.NewString(),
	}
	_, err := NewPersonAggregate().CreatePerson(cmd)
	if err != ErrPersonOrganizationNotFound {
		t.Errorf("expected ErrPersonOrganizationNotFound, got %v", err)
	}
}

func TestPersonAggregate_CreatePerson_InvalidRoleRef(t *testing.T) {
	cmd := CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{{RoleID: "not-a-uuid"}},
		TenantID:   uuid.NewString(),
	}
	_, err := NewPersonAggregate().CreatePerson(cmd)
	if err != ErrPersonInvalidRoleRef {
		t.Errorf("expected ErrPersonInvalidRoleRef, got %v", err)
	}
}

func TestPersonAggregate_CreatePerson_DuplicateRoleRef(t *testing.T) {
	roleID := uuid.NewString()
	cmd := CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{{RoleID: roleID}, {RoleID: roleID}},
		TenantID:   uuid.NewString(),
	}
	_, err := NewPersonAggregate().CreatePerson(cmd)
	if err != ErrPersonRoleAlreadyAssigned {
		t.Errorf("expected ErrPersonRoleAlreadyAssigned, got %v", err)
	}
}

func TestPersonAggregate_CreatePerson_NameValidation(t *testing.T) {
	t.Run("empty_name", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  uuid.NewString(),
			OrgID:      uuid.NewString(),
			Name:       "",
			EmployeeNo: "EMP001",
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonInvalidName {
			t.Errorf("expected ErrPersonInvalidName, got %v", err)
		}
	})

	t.Run("name_too_long", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  uuid.NewString(),
			OrgID:      uuid.NewString(),
			Name:       strings.Repeat("a", 129),
			EmployeeNo: "EMP001",
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonInvalidName {
			t.Errorf("expected ErrPersonInvalidName, got %v", err)
		}
	})
}

func TestPersonAggregate_CreatePerson_EmployeeNoValidation(t *testing.T) {
	t.Run("empty_employee_no", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  uuid.NewString(),
			OrgID:      uuid.NewString(),
			Name:       "Test",
			EmployeeNo: "",
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonInvalidEmployeeNo {
			t.Errorf("expected ErrPersonInvalidEmployeeNo, got %v", err)
		}
	})

	t.Run("employee_no_too_long", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  uuid.NewString(),
			OrgID:      uuid.NewString(),
			Name:       "Test",
			EmployeeNo: strings.Repeat("a", 65),
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonInvalidEmployeeNo {
			t.Errorf("expected ErrPersonInvalidEmployeeNo, got %v", err)
		}
	})
}

func TestPersonAggregate_UpdatePerson_Success(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Original",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{},
		Version:    3,
		TenantID:   uuid.NewString(),
	}
	cmd := UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		NewName:         "Updated",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 3,
	}
	event, err := agg.UpdatePerson(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if agg.Version != 4 {
		t.Errorf("expected version=4, got %d", agg.Version)
	}
	if event.EventType != "PersonUpdated" {
		t.Errorf("expected EventType=PersonUpdated, got %s", event.EventType)
	}
	if agg.Name != "Updated" {
		t.Errorf("expected name=Updated, got %s", agg.Name)
	}
	if agg.EmployeeNo != "EMP002" {
		t.Errorf("expected employeeNo=EMP002, got %s", agg.EmployeeNo)
	}
}

func TestPersonAggregate_UpdatePerson_OrgIDImmutable(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Original",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{},
		Version:    1,
		TenantID:   uuid.NewString(),
	}
	cmd := UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		NewName:         "Updated",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 1,
	}
	_, err := agg.UpdatePerson(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestPersonAggregate_UpdatePerson_RolesImmutable(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Original",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{{RoleID: uuid.NewString()}},
		Version:    1,
		TenantID:   uuid.NewString(),
	}
	cmd := UpdatePersonCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		NewName:         "Updated",
		NewEmployeeNo:   "EMP002",
		ExpectedVersion: 1,
	}
	_, err := agg.UpdatePerson(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(agg.Roles) != 1 {
		t.Errorf("expected roles unchanged (1 role), got %d", len(agg.Roles))
	}
}

func TestPersonAggregate_AssignRole_Success(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{},
		Version:    2,
		TenantID:   uuid.NewString(),
	}
	cmd := AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		RoleID:          uuid.NewString(),
		ExpectedVersion: 2,
	}
	event, err := agg.AssignRole(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if agg.Version != 3 {
		t.Errorf("expected version=3, got %d", agg.Version)
	}
	if event.EventType != "RoleAssigned" {
		t.Errorf("expected EventType=RoleAssigned, got %s", event.EventType)
	}
	if len(agg.Roles) != 1 {
		t.Errorf("expected 1 role, got %d", len(agg.Roles))
	}
}

func TestPersonAggregate_AssignRole_InvalidRoleRef(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{},
		Version:    1,
		TenantID:   uuid.NewString(),
	}
	cmd := AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		RoleID:          "not-a-uuid",
		ExpectedVersion: 1,
	}
	_, err := agg.AssignRole(cmd)
	if err != ErrPersonInvalidRoleRef {
		t.Errorf("expected ErrPersonInvalidRoleRef, got %v", err)
	}
}

func TestPersonAggregate_AssignRole_AlreadyAssigned(t *testing.T) {
	roleID := uuid.NewString()
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{{RoleID: roleID}},
		Version:    1,
		TenantID:   uuid.NewString(),
	}
	cmd := AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		RoleID:          roleID,
		ExpectedVersion: 1,
	}
	_, err := agg.AssignRole(cmd)
	if err != ErrPersonRoleAlreadyAssigned {
		t.Errorf("expected ErrPersonRoleAlreadyAssigned, got %v", err)
	}
}

func TestPersonAggregate_AssignRole_DoesNotValidateRoleExistence(t *testing.T) {
	agg := &PersonAggregate{
		PersonID:   uuid.NewString(),
		OrgID:      uuid.NewString(),
		Name:       "Test",
		EmployeeNo: "EMP001",
		Roles:      []RoleRef{},
		Version:    1,
		TenantID:   uuid.NewString(),
	}
	nonExistentRoleID := uuid.NewString()
	cmd := AssignRoleCommand{
		CommandID:       uuid.NewString(),
		PersonID:        agg.PersonID,
		RoleID:          nonExistentRoleID,
		ExpectedVersion: 1,
	}
	_, err := agg.AssignRole(cmd)
	if err != nil {
		t.Fatalf("expected success (roleId existence not validated), got error: %v", err)
	}
}

func TestPersonAggregate_ErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"NotFound", ErrPersonNotFound, "EBCX-PERSON-NOT-FOUND"},
		{"DuplicateEmployeeNo", ErrPersonDuplicateEmployeeNo, "EBCX-PERSON-DUPLICATE-EMPLOYEE-NO"},
		{"OrganizationNotFound", ErrPersonOrganizationNotFound, "EBCX-PERSON-ORGANIZATION-NOT-FOUND"},
		{"OrgIDImmutable", ErrPersonOrgIDImmutable, "EBCX-PERSON-ORGID-IMMUTABLE"},
		{"RolesImmutableInUpdate", ErrPersonRolesImmutableInUpdate, "EBCX-PERSON-ROLES-IMMUTABLE-IN-UPDATE"},
		{"InvalidRoleRef", ErrPersonInvalidRoleRef, "EBCX-PERSON-INVALID-ROLE-REF"},
		{"RoleAlreadyAssigned", ErrPersonRoleAlreadyAssigned, "EBCX-PERSON-ROLE-ALREADY-ASSIGNED"},
		{"CASConflict", ErrPersonCASConflict, "EBCX-PERSON-CAS-CONCURRENCY-CONFLICT"},
		{"VersionMonotonic", ErrPersonVersionMonotonic, "EBCX-PERSON-VERSION-MONOTONIC-VIOLATION"},
		{"InvalidName", ErrPersonInvalidName, "EBCX-PERSON-INVALID-NAME"},
		{"InvalidEmployeeNo", ErrPersonInvalidEmployeeNo, "EBCX-PERSON-INVALID-EMPLOYEE-NO"},
		{"IdempotencyConflict", ErrPersonIdempotencyConflict, "EBCX-PERSON-IDEMPOTENCY-CONFLICT"},
		{"UnknownCommand", ErrPersonUnknownCommand, "EBCX-PERSON-UNKNOWN-COMMAND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestPersonAggregate_EventFields(t *testing.T) {
	tenantID := uuid.NewString()
	orgID := uuid.NewString()
	cmd := CreatePersonCommand{
		CommandID:  uuid.NewString(),
		OrgID:      orgID,
		Name:       "EventTest",
		EmployeeNo: "EMP001",
		TenantID:   tenantID,
	}
	agg := NewPersonAggregate()
	event, err := agg.CreatePerson(cmd)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if event.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if event.OrgID != orgID {
		t.Error("event OrgID mismatch")
	}
	if event.TenantID != tenantID {
		t.Error("event TenantID mismatch")
	}
	if event.Timestamp.IsZero() {
		t.Error("event Timestamp should not be zero")
	}
}

func TestPersonAggregate_CommandIDValidation(t *testing.T) {
	t.Run("empty_command_id", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  "",
			OrgID:      uuid.NewString(),
			Name:       "Test",
			EmployeeNo: "EMP001",
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonCommandIDRequired {
			t.Errorf("expected ErrPersonCommandIDRequired, got %v", err)
		}
	})

	t.Run("invalid_command_id", func(t *testing.T) {
		cmd := CreatePersonCommand{
			CommandID:  "not-a-uuid",
			OrgID:      uuid.NewString(),
			Name:       "Test",
			EmployeeNo: "EMP001",
			TenantID:   uuid.NewString(),
		}
		_, err := NewPersonAggregate().CreatePerson(cmd)
		if err != ErrPersonInvalidCommandID {
			t.Errorf("expected ErrPersonInvalidCommandID, got %v", err)
		}
	})
}
